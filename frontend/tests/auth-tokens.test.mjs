import assert from 'node:assert/strict';
import test from 'node:test';

import { isFresh, jwtExpiry } from '../src/lib/auth/tokens.ts';

function jwt(payload) {
  const encode = (value) =>
    Buffer.from(JSON.stringify(value)).toString('base64url');
  return `${encode({ alg: 'HS256', typ: 'JWT' })}.${encode(payload)}.signature`;
}

test('jwtExpiry reads exp from a base64url payload', () => {
  assert.equal(jwtExpiry(jwt({ sub: 'x', exp: 1_900_000_000 })), 1_900_000_000);
});

test('jwtExpiry copes with payloads whose base64url form needs padding and - _ chars', () => {
  // '>>>???' encodes to characters that differ between base64 and base64url.
  const token = jwt({ exp: 42, pad: '>>>???~~~' });
  assert.equal(jwtExpiry(token), 42);
});

test('jwtExpiry returns null for anything that is not a JWT with a numeric exp', () => {
  for (const bad of [
    '',
    'abc',
    'a.b.c',
    'a..c',
    jwt({ sub: 'x' }),
    jwt({ exp: '99' }),
  ]) {
    assert.equal(jwtExpiry(bad), null, bad);
  }
});

test('isFresh requires the token to outlive the skew', () => {
  const token = jwt({ exp: 1000 });
  assert.equal(isFresh(token, 900, 30), true);
  assert.equal(isFresh(token, 970, 30), false); // exactly at the skew boundary
  assert.equal(isFresh(token, 1100, 30), false);
});

test('isFresh is false for a missing or unreadable token', () => {
  assert.equal(isFresh(undefined, 0), false);
  assert.equal(isFresh('garbage', 0), false);
});
