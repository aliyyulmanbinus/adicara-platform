import type { APIRoute } from 'astro';

export const prerender = true;

export const GET: APIRoute = ({ site }) => {
  const baseURL = site ?? new URL('http://localhost:4321');
  const sitemapURL = new URL('sitemap-index.xml', baseURL);

  return new Response(
    `User-agent: *\nAllow: /\nDisallow: /i/\nDisallow: /auth/\nDisallow: /dashboard/\nSitemap: ${sitemapURL.href}\n`,
    {
      headers: { 'Content-Type': 'text/plain; charset=utf-8' },
    },
  );
};
