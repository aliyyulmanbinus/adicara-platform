import assert from 'node:assert/strict';
import { execFileSync } from 'node:child_process';
import test from 'node:test';

function getConfiguredSite(value) {
  const env = { ...process.env };
  if (value === undefined) {
    delete env.PUBLIC_SITE_URL;
  } else {
    env.PUBLIC_SITE_URL = value;
  }
  return execFileSync(
    process.execPath,
    [
      '--input-type=module',
      '-e',
      'import config from "./astro.config.mjs"; process.stdout.write(config.site, () => process.exit(0));',
    ],
    {
      cwd: new URL('..', import.meta.url),
      env,
      encoding: 'utf8',
      timeout: 60000,
    },
  );
}

test('missing, empty, and whitespace site URLs use the local fallback', () => {
  for (const value of [undefined, '', '   ']) {
    assert.equal(getConfiguredSite(value), 'http://localhost:4321');
  }
});

test('an explicit production site URL is preserved and trimmed', () => {
  assert.equal(
    getConfiguredSite('  https://example.com  '),
    'https://example.com',
  );
});
