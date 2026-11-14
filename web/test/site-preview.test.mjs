import assert from 'node:assert/strict';
import test from 'node:test';
import { loadNginxPreview } from '../../cmd/tongd/ui/site-preview.mjs';

const response = (status, body, contentType = 'text/plain; charset=utf-8') => ({
    ok: status >= 200 && status < 300,
    headers: { get: (name) => name === 'Content-Type' ? contentType : null },
    text: async () => body,
    json: async () => JSON.parse(body),
});

test('loads same-origin Nginx preview with cookie credentials and no bearer token', async () => {
    let request;
    const config = await loadNginxPreview('site-a_1', async (path, options) => {
        request = { path, options };
        return response(200, 'server_name example.test;');
    });
    assert.equal(config, 'server_name example.test;');
    assert.equal(request.path, '/api/v1/sites/site-a_1/preview/nginx');
    assert.equal(request.options.method, 'GET');
    assert.equal(request.options.credentials, 'same-origin');
    assert.equal(request.options.cache, 'no-store');
    assert.equal(request.options.headers.Accept, 'text/plain');
    assert.equal('Authorization' in request.options.headers, false);
    assert.ok(request.options.signal instanceof AbortSignal);
});

test('rejects malformed site IDs before making a request', async () => {
    let called = false;
    await assert.rejects(loadNginxPreview('../other', async () => { called = true; }), /site ID/);
    assert.equal(called, false);
});

test('maps authorization and readiness failures to fixed actionable messages', async () => {
    await assert.rejects(loadNginxPreview('site-a', async () => response(401, JSON.stringify({ error: { code: 'unauthorized' } }), 'application/json')), /secure browser session/);
    await assert.rejects(loadNginxPreview('site-a', async () => response(409, JSON.stringify({ error: { code: 'site_not_ready' } }), 'application/json')), /Verify the site/);
    await assert.rejects(loadNginxPreview('site-a', async () => response(503, JSON.stringify({ error: { code: 'site_preview_unavailable' } }), 'application/json')), /not enabled/);
});

test('rejects unexpected response types and hides server error text', async () => {
    await assert.rejects(loadNginxPreview('site-a', async () => response(200, '<script>unsafe</script>', 'text/html')), /unexpected preview format/);
    await assert.rejects(loadNginxPreview('site-a', async () => response(500, '<html>secret host detail</html>', 'text/html')), error => {
        assert.match(error.message, /could not be loaded/);
        assert.doesNotMatch(error.message, /secret host detail/);
        return true;
    });
});
