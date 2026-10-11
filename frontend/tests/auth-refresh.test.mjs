import assert from 'node:assert/strict';
import test from 'node:test';

import { createCoalescer } from '../src/lib/auth/refresh.ts';

function deferred() {
  let resolve;
  let reject;
  const promise = new Promise((res, rej) => {
    resolve = res;
    reject = rej;
  });
  return { promise, resolve, reject };
}

test('concurrent callers with the same refresh token hit the backend once', async () => {
  let calls = 0;
  const gate = deferred();
  const rotate = createCoalescer(
    async () => {
      calls++;
      return gate.promise;
    },
    { ttlMs: 10_000 },
  );

  const pending = Promise.all([1, 2, 3, 4, 5].map(() => rotate('old-rt')));
  gate.resolve({ access_token: 'new' });

  const results = await pending;
  assert.equal(calls, 1);
  for (const result of results)
    assert.deepEqual(result, { access_token: 'new' });
});

test('a late request still carrying the old token reuses the rotated pair within the window', async () => {
  let clock = 1000;
  let calls = 0;
  const rotate = createCoalescer(
    async () => {
      calls++;
      return { access_token: `pair-${calls}` };
    },
    { ttlMs: 10_000, now: () => clock },
  );

  assert.deepEqual(await rotate('old-rt'), { access_token: 'pair-1' });
  clock += 9_000;
  assert.deepEqual(await rotate('old-rt'), { access_token: 'pair-1' });
  assert.equal(calls, 1);

  clock += 2_000; // 11s after it settled: the window is over
  assert.deepEqual(await rotate('old-rt'), { access_token: 'pair-2' });
  assert.equal(calls, 2);
});

test('different refresh tokens never share a result', async () => {
  const rotate = createCoalescer(async (token) => ({ for: token }), {
    ttlMs: 10_000,
  });
  assert.deepEqual(await rotate('a'), { for: 'a' });
  assert.deepEqual(await rotate('b'), { for: 'b' });
});

test('a rejected token (null) is remembered so it is not retried in a burst', async () => {
  let calls = 0;
  const rotate = createCoalescer(
    async () => {
      calls++;
      return null;
    },
    { ttlMs: 10_000 },
  );
  assert.equal(await rotate('dead'), null);
  assert.equal(await rotate('dead'), null);
  assert.equal(calls, 1);
});

test('an error (backend unreachable) is not remembered: the next request retries', async () => {
  let calls = 0;
  const rotate = createCoalescer(
    async () => {
      calls++;
      if (calls === 1) throw new Error('backend down');
      return { access_token: 'recovered' };
    },
    { ttlMs: 10_000 },
  );

  await assert.rejects(rotate('rt'), /backend down/);
  assert.deepEqual(await rotate('rt'), { access_token: 'recovered' });
  assert.equal(calls, 2);
});

test('concurrent callers all see the error when the shared attempt fails', async () => {
  const gate = deferred();
  const rotate = createCoalescer(() => gate.promise, { ttlMs: 10_000 });
  const pending = [rotate('rt'), rotate('rt')].map((p) =>
    assert.rejects(p, /boom/),
  );
  gate.reject(new Error('boom'));
  await Promise.all(pending);
});

test('settled entries are evicted so the cache cannot grow without bound', async () => {
  let clock = 0;
  let calls = 0;
  const rotate = createCoalescer(
    async (token) => {
      calls++;
      return token;
    },
    { ttlMs: 1_000, now: () => clock },
  );
  await rotate('a');
  clock = 5_000;
  await rotate('b'); // sweeps 'a'
  await rotate('a'); // must call fn again, proving 'a' was evicted
  assert.equal(calls, 3);
});
