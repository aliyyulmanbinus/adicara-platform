import type { AstroCookies } from 'astro';

import {
  BackendUnavailableError,
  backendFetch,
  type BackendTokens,
  type BackendUser,
} from '@/lib/api/backend';
import { runtimeEnv } from '@/lib/env';
import { createCoalescer } from './refresh';
import { isFresh, jwtExpiry } from './tokens';

export const ACCESS_COOKIE = 'adicara_at';
export const REFRESH_COOKIE = 'adicara_rt';

// Requests that still carry a just-rotated refresh token reuse its result for
// this long. Short on purpose: it is also the window in which a replay of the
// old token would succeed.
const REFRESH_REUSE_MS = 10_000;

function cookieOptions(maxAgeSec: number) {
  return {
    path: '/',
    httpOnly: true,
    sameSite: 'lax' as const,
    secure: runtimeEnv('COOKIE_SECURE') === 'true',
    maxAge: Math.max(1, Math.floor(maxAgeSec)),
  };
}

export function setSessionCookies(
  cookies: AstroCookies,
  tokens: BackendTokens,
) {
  const now = Date.now() / 1000;
  const accessExp = jwtExpiry(tokens.access_token) ?? now + tokens.expires_in;
  const refreshExp = jwtExpiry(tokens.refresh_token) ?? now + 60 * 60 * 24 * 7;
  cookies.set(
    ACCESS_COOKIE,
    tokens.access_token,
    cookieOptions(accessExp - now),
  );
  cookies.set(
    REFRESH_COOKIE,
    tokens.refresh_token,
    cookieOptions(refreshExp - now),
  );
}

export function clearSessionCookies(cookies: AstroCookies) {
  cookies.delete(ACCESS_COOKIE, { path: '/' });
  cookies.delete(REFRESH_COOKIE, { path: '/' });
}

/** Resolves to the new pair, `null` if the backend rejected the token. */
const rotateRefreshToken = createCoalescer<string | undefined, BackendTokens>(
  async (refreshToken, clientIP) => {
    const result = await backendFetch<{ tokens: BackendTokens }>(
      '/v1/auth/refresh',
      { body: { refresh_token: refreshToken }, clientIP },
    );
    if (result.status === 401) return null;
    // Anything else (5xx, 429, malformed reply) is "try again", not "logged out".
    if (!result.ok || !result.data?.tokens) throw new BackendUnavailableError();

    return result.data.tokens;
  },
  { ttlMs: REFRESH_REUSE_MS },
);

export type SessionResult =
  | { kind: 'ok'; user: BackendUser; accessToken: string }
  /** No usable session: the caller should clear cookies and send the user to login. */
  | { kind: 'anonymous' }
  /** The backend is down. The session may well be valid, so cookies must be kept. */
  | { kind: 'unavailable' };

/**
 * Turns the session cookies into a user, refreshing the access token when it
 * is missing/expired (or rejected). Refreshed cookies are written to `cookies`.
 */
export async function resolveSession(
  cookies: AstroCookies,
  clientIP?: string,
): Promise<SessionResult> {
  const refreshToken = cookies.get(REFRESH_COOKIE)?.value;
  const stored = cookies.get(ACCESS_COOKIE)?.value;
  let accessToken = isFresh(stored, Date.now() / 1000) ? stored : undefined;
  let refreshed = false;

  const refresh = async () => {
    if (!refreshToken || refreshed) return false;
    refreshed = true;
    const tokens = await rotateRefreshToken(refreshToken, clientIP);
    if (!tokens) return false;
    setSessionCookies(cookies, tokens);
    accessToken = tokens.access_token;

    return true;
  };

  try {
    if (!accessToken && !(await refresh())) return { kind: 'anonymous' };

    let me = await backendFetch<{ data: { user: BackendUser } }>('/v1/me', {
      accessToken,
      clientIP,
    });
    // Fresh by clock but refused by the backend (e.g. signing key rotated).
    if (me.status === 401 && (await refresh())) {
      me = await backendFetch<{ data: { user: BackendUser } }>('/v1/me', {
        accessToken,
        clientIP,
      });
    }
    if (me.status === 401) return { kind: 'anonymous' };

    const user = me.data?.data.user;
    if (!me.ok || !user || !accessToken) return { kind: 'unavailable' };

    return { kind: 'ok', user, accessToken };
  } catch (error) {
    if (error instanceof BackendUnavailableError)
      return { kind: 'unavailable' };
    throw error;
  }
}
