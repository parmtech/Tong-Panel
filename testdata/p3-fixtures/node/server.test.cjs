'use strict';

const test = require('node:test');
const assert = require('node:assert/strict');
const { createServer } = require('./server.cjs');

async function withServer(options, run) {
  const server = createServer(options);
  await new Promise((resolve) => server.listen(0, '127.0.0.1', resolve));
  try {
    await run(`http://127.0.0.1:${server.address().port}`);
  } finally {
    await new Promise((resolve, reject) => server.close((err) => err ? reject(err) : resolve()));
  }
}

test('reports ready and the synthetic release identity', async () => {
  await withServer({ release: 'fixture-abc' }, async (base) => {
    assert.deepEqual(await (await fetch(`${base}/healthz`)).json(), { ready: true });
    assert.deepEqual(await (await fetch(base)).json(), { release: 'fixture-abc' });
  });
});

test('reports unhealthy, rejects other methods, and hides unknown paths', async () => {
  await withServer({ ready: false }, async (base) => {
    assert.equal((await fetch(`${base}/healthz`)).status, 503);
    assert.equal((await fetch(base, { method: 'POST' })).status, 405);
    assert.equal((await fetch(`${base}/missing`)).status, 404);
  });
});
