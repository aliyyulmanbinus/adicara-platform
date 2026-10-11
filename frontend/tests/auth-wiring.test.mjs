import assert from 'node:assert/strict';
import { readFile } from 'node:fs/promises';
import test from 'node:test';

const read = (path) =>
  readFile(new URL(`../src/${path}`, import.meta.url), 'utf8');

test('the dashboard routes are guarded by the auth middleware and rendered on demand', async () => {
  const middleware = await read('middleware.ts');
  assert.match(middleware, /pathname === '\/dashboard'/);
  assert.match(middleware, /startsWith\('\/dashboard\/'\)/);

  // A prerendered page would be served as static HTML, bypassing the middleware.
  for (const page of [
    'pages/dashboard/index.astro',
    'pages/dashboard/undangan/baru.astro',
  ]) {
    assert.match(await read(page), /export const prerender = false/, page);
  }
});

test('auth endpoints are POST-only, on demand, and outside the proxy /api prefix', async () => {
  for (const name of ['login', 'register', 'logout']) {
    const source = await read(`pages/auth/${name}.ts`);
    assert.match(source, /export const prerender = false/, name);
    assert.match(source, /export const POST/, name);
    assert.doesNotMatch(source, /export const GET/, name);
  }
});

test('registering only creates the account: no login, no cookies, redirect to the login page', async () => {
  const source = await read('lib/auth/credentials.ts');
  const start = source.indexOf("if (action === 'register') {");
  const end = source.indexOf('const credentials =');
  assert.ok(start !== -1 && end > start, 'register branch not found');

  const register = source.slice(start, end);
  assert.match(register, /\/v1\/auth\/register/);
  assert.match(register, /redirect: '\/auth\/masuk'/);
  assert.doesNotMatch(register, /setSessionCookies|\/v1\/auth\/login/);

  // Session cookies are written exactly once, after the login call.
  assert.equal(source.split('setSessionCookies(cookies').length - 1, 1);
  assert.ok(source.indexOf('setSessionCookies(cookies') > end);
});

test('the register form has no username hint and hands a one-shot toast flag to the login page', async () => {
  const form = await read('components/auth/AuthForm.astro');
  assert.doesNotMatch(form, /username-help|3–30 karakter/);

  // Only a known key is stored (never text); the login form consumes it once.
  assert.match(form, /sessionStorage\.setItem\(TOAST_KEY, 'registered'\)/);
  assert.match(form, /sessionStorage\.removeItem\(TOAST_KEY\)/);
  assert.match(form, /dataset\.mode === 'login'\) showPendingToast\(\)/);

  // A live region that exists before the message is added, so it is announced.
  assert.match(form, /role="status" aria-live="polite" data-toast-region/);
});

test('session cookies are HttpOnly and SameSite=Lax', async () => {
  const session = await read('lib/auth/session.ts');
  assert.match(session, /httpOnly: true/);
  assert.match(session, /sameSite: 'lax'/);
});

test('forms and header talk to the same-origin auth endpoints with the fields the backend expects', async () => {
  const form = await read('components/auth/AuthForm.astro');
  assert.match(form, /fetch\(`\/auth\/\$\{mode\}`/);
  assert.match(form, /name="username"/);
  assert.match(form, /'email' : 'identifier'/);
  assert.doesNotMatch(form, /display_name/);

  const header = await read('components/layout/AppHeader.astro');
  assert.match(header, /fetch\('\/auth\/logout'/);
});

// A body-less POST is origin-checked by Astro as if it were a form post, which
// breaks behind a TLS-terminating proxy (Origin https vs. computed http).
test('logout sends and requires JSON so it is not treated as a form post', async () => {
  const header = await read('components/layout/AppHeader.astro');
  assert.match(header, /'Content-Type': 'application\/json'/);
  const endpoint = await read('pages/auth/logout.ts');
  assert.match(endpoint, /includes\('application\/json'\)/);
});

test('the browser never calls the backend with the pre-rewrite cookie/CSRF auth', async () => {
  for (const file of [
    'components/auth/AuthForm.astro',
    'components/layout/AppHeader.astro',
    'lib/auth/session.ts',
    'lib/auth/credentials.ts',
    'middleware.ts',
  ]) {
    const source = await read(file);
    assert.doesNotMatch(
      source,
      /X-CSRF-Token|adicara_csrf|\/api\/v1\/(auth|me)/,
      file,
    );
  }
});
