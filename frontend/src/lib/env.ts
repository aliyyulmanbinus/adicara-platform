/**
 * Server-side settings come from the real process environment at runtime
 * (Docker/Compose); `import.meta.env` covers `astro dev`, which loads `.env`.
 */
export function runtimeEnv(name: 'API_BASE_URL' | 'COOKIE_SECURE') {
  return process.env[name] ?? import.meta.env[name];
}
