/**
 * Refresh tokens rotate: the backend spends the presented token and rejects it
 * if it is replayed. When several requests carrying the same expired access
 * token reach the server together (page + assets, two tabs), only the first
 * refresh may hit the backend; the rest must reuse its result, or they would
 * be rejected and the user logged out for no reason.
 *
 * `fn` runs once per key. Its outcome (including `null` = "rejected") is kept
 * for `ttlMs` so requests that still carry the old cookie get the same answer.
 * A thrown error (backend unreachable) is not remembered, so the next request
 * retries.
 */
export function createCoalescer<A, T>(
  fn: (key: string, arg: A) => Promise<T | null>,
  options: { ttlMs: number; now?: () => number },
): (key: string, arg: A) => Promise<T | null> {
  const now = options.now ?? Date.now;
  const entries = new Map<
    string,
    { promise: Promise<T | null>; settledAt: number | null }
  >();

  return (key, arg) => {
    for (const [k, entry] of entries) {
      if (entry.settledAt !== null && now() - entry.settledAt > options.ttlMs) {
        entries.delete(k);
      }
    }

    const existing = entries.get(key);
    if (existing) return existing.promise;

    const entry: { promise: Promise<T | null>; settledAt: number | null } = {
      promise: Promise.resolve(null),
      settledAt: null,
    };
    entry.promise = fn(key, arg).then(
      (value) => {
        entry.settledAt = now();
        return value;
      },
      (error: unknown) => {
        entries.delete(key);
        throw error;
      },
    );
    entries.set(key, entry);

    return entry.promise;
  };
}
