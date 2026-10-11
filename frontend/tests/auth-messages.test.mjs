import assert from 'node:assert/strict';
import { readFile } from 'node:fs/promises';
import test from 'node:test';

import {
  SERVICE_UNAVAILABLE,
  authErrorMessage,
} from '../src/lib/auth/messages.ts';

const DEFAULT = 'Permintaan belum berhasil. Periksa kembali data Anda.';

test('login failures give one generic message so accounts cannot be probed', () => {
  assert.equal(
    authErrorMessage(401, 'unauthorized', 'login'),
    'Email/username atau kata sandi salah.',
  );
});

test('rate limiting and outages are recognised by status', () => {
  assert.match(
    authErrorMessage(429, 'rate_limited', 'login'),
    /Terlalu banyak/,
  );
  assert.match(authErrorMessage(429, undefined, 'register'), /Terlalu banyak/);
  assert.equal(authErrorMessage(500, 'internal', 'login'), SERVICE_UNAVAILABLE);
  assert.equal(
    authErrorMessage(503, undefined, 'register'),
    SERVICE_UNAVAILABLE,
  );
});

test('unknown codes fall back to a neutral message and never echo backend text', () => {
  assert.equal(authErrorMessage(400, 'something_new', 'register'), DEFAULT);
  assert.equal(authErrorMessage(400, undefined, 'register'), DEFAULT);
});

// If the backend adds a validation code the form must learn about it, otherwise
// users just see the generic fallback.
test('every client-error code the backend auth service can return has its own message', async () => {
  const source = await readFile(
    new URL(
      '../../backend/internal/modules/auth/usecase/service.go',
      import.meta.url,
    ),
    'utf8',
  );
  const codes = new Set(
    [
      ...source.matchAll(/apierr\.(?:BadRequest|Conflict)\("([a-z_]+)"/g),
      ...source.matchAll(/apierr\.New\(\d+, "([a-z_]+)"/g),
    ].map((match) => match[1]),
  );
  assert.ok(
    codes.size >= 7,
    `expected to find the backend codes, got ${[...codes]}`,
  );

  for (const code of codes) {
    assert.notEqual(
      authErrorMessage(400, code, 'register'),
      DEFAULT,
      `no frontend message for backend error code "${code}"`,
    );
  }
});
