import assert from 'node:assert/strict';
import test from 'node:test';
import { loadACMEPreflight } from '../../cmd/tongd/ui/acme-preflight.mjs';
const snapshot = { site_id: 'site-a', domains: ['primary.example.test'], challenge: 'http-01', snapshot_only: true, execution_enabled: false };
const response = data => new Response(JSON.stringify(data), { headers: { 'Content-Type': 'application/json' } });

test('loads snapshot using only same-origin cookie GET and fixed endpoint', async () => {
    const result = await loadACMEPreflight('site-a', async (path, options) => {
        assert.equal(path, '/api/v1/sites/site-a/preflight/acme-http01');
        assert.equal(options.method, 'GET'); assert.equal(options.credentials, 'same-origin');
        assert.equal(options.redirect, 'error'); assert.equal(options.cache, 'no-store');
        assert.deepEqual(options.headers, { Accept: 'application/json' });
        return response(snapshot);
    });
    assert.deepEqual(result, { siteID: 'site-a', domains: ['primary.example.test'] });
});

test('refuses unsafe input, execution flags, unknown fields and unsafe domains', async () => {
    await assert.rejects(loadACMEPreflight('../other', () => assert.fail('unsafe request')));
    for (const data of [
        { ...snapshot, site_id: 'other' }, { ...snapshot, execution_enabled: true },
        { ...snapshot, snapshot_only: false }, { ...snapshot, secret: 'private-canary' },
        { ...snapshot, challenge: 'dns-01' }, { ...snapshot, domains: [] },
        { ...snapshot, domains: ['<script>private-canary</script>'] },
        { ...snapshot, domains: ['127.0.0.1'] }, { ...snapshot, domains: ['UPPER.test'] },
        { ...snapshot, domains: ['*.example.test'] },
        { ...snapshot, domains: ['x.example.test', 'x.example.test'] },
    ]) await assert.rejects(loadACMEPreflight('site-a', async () => response(data)), error => !error.message.includes('private-canary'));
});

test('bounds body reads and maps failures without reflecting server bodies', async () => {
    for (const status of [401, 403, 404, 409, 503]) {
        await assert.rejects(loadACMEPreflight('site-a', async () => new Response('private-canary', { status })), error => !error.message.includes('private-canary'));
    }
    for (const value of [
        new Response('x'.repeat(16385), { headers: { 'Content-Type': 'application/json' } }),
        new Response(JSON.stringify(snapshot), { headers: { 'Content-Type': 'text/html' } }),
        new Response(JSON.stringify(snapshot), { headers: { 'Content-Type': 'application/json', 'Content-Length': '16385' } }),
        new Response(JSON.stringify(snapshot), { status: 206, headers: { 'Content-Type': 'application/json' } }),
    ]) await assert.rejects(loadACMEPreflight('site-a', async () => value));
});

test('abort settles ignored fetch and stalled body reads', async () => {
    const first = new AbortController();
    const pending = loadACMEPreflight('site-a', () => new Promise(() => {}), first.signal);
    first.abort(); await assert.rejects(pending);
    const second = new AbortController();
    const body = new ReadableStream({ start() {} });
    const reading = loadACMEPreflight('site-a', async () => new Response(body, { headers: { 'Content-Type': 'application/json' } }), second.signal);
    await new Promise(resolve => setImmediate(resolve)); second.abort(); await assert.rejects(reading);
});
