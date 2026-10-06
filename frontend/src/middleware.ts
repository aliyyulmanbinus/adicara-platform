import { defineMiddleware } from 'astro:middleware';

import { clientIPOf } from '@/lib/auth/credentials';
import { clearSessionCookies, resolveSession } from '@/lib/auth/session';

function isProtected(pathname: string) {
  return pathname === '/dashboard' || pathname.startsWith('/dashboard/');
}

export const onRequest = defineMiddleware(async (context, next) => {
  if (!isProtected(context.url.pathname)) return next();

  const session = await resolveSession(context.cookies, clientIPOf(context));

  if (session.kind === 'unavailable') {
    // Do not touch the cookies: the session is probably still valid.
    return new Response(
      'Layanan sedang tidak tersedia. Coba muat ulang sebentar lagi.',
      {
        status: 503,
        headers: { 'Retry-After': '5', 'Cache-Control': 'no-store' },
      },
    );
  }
  if (session.kind === 'anonymous') {
    clearSessionCookies(context.cookies);
    return context.redirect('/auth/masuk');
  }

  context.locals.user = session.user;
  context.locals.accessToken = session.accessToken;
  const response = await next();
  response.headers.set('Cache-Control', 'private, no-store');

  return response;
});
