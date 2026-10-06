import type { APIRoute } from 'astro';

import { backendFetch } from '@/lib/api/backend';
import { clientIPOf } from '@/lib/auth/credentials';
import { REFRESH_COOKIE, clearSessionCookies } from '@/lib/auth/session';

export const prerender = false;

export const POST: APIRoute = async (context) => {
  // Same CSRF rule as login/register: a cross-site page cannot send JSON
  // without a CORS preflight, which this endpoint never grants. It also keeps
  // Astro from origin-checking the request as a form post.
  if (
    !context.request.headers.get('content-type')?.includes('application/json')
  ) {
    return new Response(null, {
      status: 415,
      headers: { 'Cache-Control': 'no-store' },
    });
  }

  const refreshToken = context.cookies.get(REFRESH_COOKIE)?.value;
  if (refreshToken) {
    // Best effort: an expired or already-revoked token must not stop the user
    // from signing out locally, and neither must an unreachable backend.
    await backendFetch('/v1/auth/logout', {
      body: { refresh_token: refreshToken },
      clientIP: clientIPOf(context),
    }).catch(() => undefined);
  }
  clearSessionCookies(context.cookies);

  return new Response(null, {
    status: 204,
    headers: { 'Cache-Control': 'no-store' },
  });
};
