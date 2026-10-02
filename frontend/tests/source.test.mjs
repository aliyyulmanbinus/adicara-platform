import assert from 'node:assert/strict';
import { readFile } from 'node:fs/promises';
import test from 'node:test';

test('homepage has a single primary heading and core sections', async () => {
  const source = await readFile(
    new URL('../src/pages/index.astro', import.meta.url),
    'utf8',
  );
  assert.equal((source.match(/<h1>/g) ?? []).length, 1);
  for (const sectionID of [
    'tema',
    'fitur',
    'harga',
    'inspirasi',
    'faq',
    'mulai',
  ]) {
    assert.match(source, new RegExp(`id="${sectionID}"`));
  }
});

test('private invitation route opts out of prerendering and derives noindex from data', async () => {
  const source = await readFile(
    new URL('../src/pages/i/[slug].astro', import.meta.url),
    'utf8',
  );
  assert.match(source, /export const prerender = false/);
  assert.match(source, /isPersonalized \|\| !invitation\.allowIndexing/);
});

test('template registry keeps template selection separate from invitation data', async () => {
  const source = await readFile(
    new URL('../src/invitation-templates/registry.ts', import.meta.url),
    'utf8',
  );
  assert.match(source, /editorialIvoryConfig/);
  assert.match(source, /botanicalModernConfig/);
  assert.match(source, /monochromeLuxeConfig/);
  assert.match(source, /getInvitationTemplate/);
});

test('marketing routes required by the public sitemap exist', async () => {
  const routes = [
    '../src/pages/tema/index.astro',
    '../src/pages/tema/[slug].astro',
    '../src/pages/fitur.astro',
    '../src/pages/harga.astro',
    '../src/pages/inspirasi/index.astro',
    '../src/pages/inspirasi/cara-membuat-undangan-digital.astro',
  ];

  for (const route of routes) {
    const source = await readFile(new URL(route, import.meta.url), 'utf8');
    assert.match(source, /<MarketingLayout/);
  }
});

test('authenticated application routes are noindex and use csrf protection', async () => {
  const login = await readFile(
    new URL('../src/pages/auth/masuk.astro', import.meta.url),
    'utf8',
  );
  const dashboard = await readFile(
    new URL('../src/pages/dashboard/index.astro', import.meta.url),
    'utf8',
  );
  const editor = await readFile(
    new URL('../src/pages/dashboard/undangan/baru.astro', import.meta.url),
    'utf8',
  );
  assert.match(login, /noindex={true}/);
  assert.match(dashboard, /export const prerender = false/);
  assert.match(dashboard, /X-CSRF-Token/);
  assert.match(editor, /X-CSRF-Token/);
});
