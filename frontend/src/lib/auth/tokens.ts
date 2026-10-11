/**
 * Reads the `exp` claim (seconds since epoch) of a JWT without verifying it.
 * This is only a hint for cookie lifetimes and refresh timing; the backend is
 * always the authority on whether a token is valid.
 */
export function jwtExpiry(token: string): number | null {
  const payload = token.split('.')[1];
  if (!payload) return null;
  try {
    const base64 = payload.replace(/-/g, '+').replace(/_/g, '/');
    const json = atob(base64.padEnd(Math.ceil(base64.length / 4) * 4, '='));
    const exp = (JSON.parse(json) as { exp?: unknown }).exp;
    return typeof exp === 'number' ? exp : null;
  } catch {
    return null;
  }
}

/** True when the token exists and will still be valid `skewSec` seconds from now. */
export function isFresh(
  token: string | undefined,
  nowSec: number,
  skewSec = 30,
): boolean {
  if (!token) return false;
  const exp = jwtExpiry(token);
  return exp !== null && exp - skewSec > nowSec;
}
