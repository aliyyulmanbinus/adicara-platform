/// <reference types="astro/client" />

declare namespace App {
  interface Locals {
    /** Set by the auth middleware on every /dashboard route. */
    user?: import('./lib/api/backend').BackendUser;
    accessToken?: string;
  }
}

interface ImportMetaEnv {
  readonly API_BASE_URL?: string;
  readonly COOKIE_SECURE?: string;
  readonly PUBLIC_SITE_URL?: string;
}

interface ImportMeta {
  readonly env: ImportMetaEnv;
}
