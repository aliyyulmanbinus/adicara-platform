import { runtimeEnv } from '@/lib/env';

export interface BackendUser {
  id: string;
  email: string;
  username: string;
  name: string | null;
}

export interface BackendTokens {
  access_token: string;
  refresh_token: string;
  expires_in: number;
}

export interface BackendError {
  code?: string;
  message?: string;
}

export interface BackendResult<T> {
  status: number;
  ok: boolean;
  data: T | null;
  error: BackendError | null;
}

/** The backend could not be reached (network error, timeout). Not a 5xx reply. */
export class BackendUnavailableError extends Error {
  constructor(cause?: unknown) {
    super('backend unavailable', { cause });
  }
}

interface BackendRequest {
  method?: 'GET' | 'POST' | 'PATCH' | 'DELETE';
  body?: unknown;
  accessToken?: string;
  /** Real client address, so the backend rate-limits per user, not per Astro server. */
  clientIP?: string;
}

/**
 * Server-to-server call to the Go API. Paths are the backend's own (`/v1/...`);
 * the `/api` prefix only exists on the public proxy and never applies here.
 * Replies that are not JSON (e.g. a proxy error page) yield `data: null`.
 */
export async function backendFetch<T = unknown>(
  path: string,
  request: BackendRequest = {},
): Promise<BackendResult<T>> {
  const base = runtimeEnv('API_BASE_URL') || 'http://localhost:8080';
  const headers: Record<string, string> = { Accept: 'application/json' };
  if (request.body !== undefined) headers['Content-Type'] = 'application/json';
  if (request.accessToken) {
    headers.Authorization = `Bearer ${request.accessToken}`;
  }
  if (request.clientIP) headers['X-Real-IP'] = request.clientIP;

  let response: Response;
  try {
    response = await fetch(new URL(path, base), {
      method: request.method ?? (request.body === undefined ? 'GET' : 'POST'),
      headers,
      body:
        request.body === undefined ? undefined : JSON.stringify(request.body),
      signal: AbortSignal.timeout(8000),
    });
  } catch (error) {
    throw new BackendUnavailableError(error);
  }

  let payload: unknown = null;
  try {
    payload = JSON.parse(await response.text());
  } catch {
    payload = null;
  }

  const error =
    !response.ok && payload && typeof payload === 'object'
      ? ((payload as { error?: BackendError }).error ?? null)
      : null;

  return {
    status: response.status,
    ok: response.ok,
    data: response.ok ? (payload as T | null) : null,
    error,
  };
}
