import type { APIContext } from 'astro';

import {
  BackendUnavailableError,
  backendFetch,
  type BackendTokens,
  type BackendUser,
} from '@/lib/api/backend';
import {
  authErrorMessage,
  SERVICE_UNAVAILABLE,
  type AuthAction,
} from './messages';
import { setSessionCookies } from './session';

const MAX_BODY_BYTES = 8 * 1024;

export function clientIPOf(context: {
  request: Request;
  clientAddress: string;
}): string | undefined {
  // nginx overwrites X-Real-IP with the connecting address; without a proxy
  // (astro dev) fall back to the socket address.
  const header = context.request.headers.get('x-real-ip');
  if (header) return header;
  try {
    return context.clientAddress;
  } catch {
    return undefined;
  }
}

export function json(status: number, body: unknown) {
  return new Response(JSON.stringify(body), {
    status,
    headers: {
      'Content-Type': 'application/json',
      'Cache-Control': 'no-store',
    },
  });
}

function fail(status: number, code: string, message: string) {
  return json(status, { error: { code, message } });
}

async function readBody(
  request: Request,
): Promise<Record<string, unknown> | null> {
  const length = Number(request.headers.get('content-length') ?? 0);
  if (length > MAX_BODY_BYTES) return null;
  try {
    const body: unknown = await request.json();
    return body && typeof body === 'object' && !Array.isArray(body)
      ? (body as Record<string, unknown>)
      : null;
  } catch {
    return null;
  }
}

function text(value: unknown): string | null {
  return typeof value === 'string' ? value : null;
}

/**
 * Login and registration share one entry point: the browser talks only to this
 * same-origin endpoint, which calls the Go API. Login keeps the tokens in
 * HttpOnly cookies so page scripts can never read them; registration creates
 * the account and sends the user to the login page without signing them in.
 */
export async function handleCredentials(
  context: APIContext,
  action: AuthAction,
): Promise<Response> {
  const { request, cookies } = context;

  // A cross-site page cannot send application/json without a CORS preflight,
  // which this endpoint never grants — that is the CSRF defence here.
  if (!request.headers.get('content-type')?.includes('application/json')) {
    return fail(
      415,
      'unsupported_media_type',
      authErrorMessage(415, 'unsupported_media_type', action),
    );
  }
  const body = await readBody(request);
  if (!body) {
    return fail(
      400,
      'invalid_json',
      authErrorMessage(400, 'invalid_json', action),
    );
  }

  const password = text(body.password);
  const identifier = text(body.identifier)?.trim() ?? text(body.email)?.trim();
  const username = text(body.username);
  if (!password || !identifier || (action === 'register' && !username)) {
    return fail(
      400,
      'invalid_input',
      authErrorMessage(400, 'invalid_input', action),
    );
  }

  const clientIP = clientIPOf(context);

  try {
    if (action === 'register') {
      const created = await backendFetch('/v1/auth/register', {
        body: { username, email: identifier, password },
        clientIP,
      });
      if (!created.ok) {
        return fail(
          created.status >= 500 ? 503 : created.status,
          created.error?.code ?? 'request_failed',
          authErrorMessage(created.status, created.error?.code, action),
        );
      }

      // Registering only creates the account: no tokens, no cookies. The user
      // signs in on the login page, like for every later visit.
      return json(201, { redirect: '/auth/masuk' });
    }

    const credentials = identifier.includes('@')
      ? { email: identifier, password }
      : { username: identifier, password };
    const login = await backendFetch<{
      data: BackendUser;
      tokens: BackendTokens;
    }>('/v1/auth/login', { body: credentials, clientIP });

    if (!login.ok || !login.data) {
      return fail(
        login.status >= 500 ? 503 : login.status,
        login.error?.code ?? 'request_failed',
        authErrorMessage(login.status, login.error?.code, action),
      );
    }

    setSessionCookies(cookies, login.data.tokens);

    return json(200, { user: login.data.data, redirect: '/dashboard' });
  } catch (error) {
    if (error instanceof BackendUnavailableError) {
      return fail(503, 'service_unavailable', SERVICE_UNAVAILABLE);
    }
    throw error;
  }
}
