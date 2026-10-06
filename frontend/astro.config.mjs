import node from '@astrojs/node';
import sitemap from '@astrojs/sitemap';
import { defineConfig } from 'astro/config';

const site = process.env.PUBLIC_SITE_URL?.trim() || 'http://localhost:4321';

export default defineConfig({
  site,
  adapter: node({ mode: 'standalone' }),
  integrations: [
    sitemap({
      filter: (page) =>
        !page.includes('/i/') &&
        !page.includes('/auth/') &&
        !page.includes('/dashboard'),
    }),
  ],
  output: 'static',
  security: {
    checkOrigin: true,
  },
  vite: {
    server: {
      allowedHosts: ['localhost'],
      proxy: {
        '/api': 'http://127.0.0.1:8080',
      },
    },
  },
});
