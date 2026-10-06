import type { APIRoute } from 'astro';

import { handleCredentials } from '@/lib/auth/credentials';

export const prerender = false;

export const POST: APIRoute = (context) =>
  handleCredentials(context, 'register');
