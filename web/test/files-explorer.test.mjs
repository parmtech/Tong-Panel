import assert from 'node:assert/strict';
import { readFile } from 'node:fs/promises';
import test from 'node:test';
import { downloadTenantFile, FilesExplorerError, listTenantFiles, validateFilesPath } from '../../cmd/tongd/ui/files-explorer.mjs';

const jsonResponse = (body, ok = true, status = 200) => ({ ok, status, json: async () => body });

test('validates relative paths before making a same-origin read-only listing request', async () => {
    const calls = [];
    const entries = await listTenantFiles('htdocs/site one', async (url, options) => {
        calls.push({ url, options });
        return jsonResponse({ entries: [{ name: 'index.html', kind: 'file', size: 5 }] });
    });
    assert.deepEqual(entries, [{ name: 'index.html', kind: 'file', size: 5 }]);
    assert.equal(calls.length, 1);
    assert.equal(calls[0].url, '/api/v1/files?path=htdocs%2Fsite+one');
    assert.equal(calls[0].options.method, 'GET');
    assert.equal(calls[0].options.credentials, 'same-origin');
    assert.equal(calls[0].options.cache, 'no-store');
    assert.equal(calls[0].options.headers.Accept, 'application/json');
    assert.equal('Authorization' in calls[0].options.headers, false);
    assert.ok(calls[0].options.signal instanceof AbortSignal);
});

test('rejects traversal, absolute, empty-segment and backslash paths without fetching', async () => {
    for (const path of ['../secret', 'a/../../secret', '/etc/passwd', 'a//b', 'a\\b', 'a/./b', 'a/'])
        assert.throws(() => validateFilesPath(path), FilesExplorerError, path);
    let calls = 0;
    await assert.rejects(listTenantFiles('../secret', async () => { calls++; }), FilesExplorerError);
    assert.equal(calls, 0);
});

test('maps authorization, not-found and disabled responses to fixed guidance', async () => {
    for (const [status, expected] of [
        [401, /secure browser session/],
        [403, /cannot access/],
        [404, /not available/],
        [503, /not enabled/],
    ]) {
        await assert.rejects(listTenantFiles('', async () => jsonResponse({ error: { code: 'secret-detail', message: 'raw internal path' } }, false, status)), expected);
    }
});

test('rejects malformed and unsafe listing records without reflecting names', async () => {
    for (const entries of [
        [{ name: '../secret', kind: 'file', size: 2 }],
        [{ name: 'link', kind: 'symlink', size: -1 }],
        [{ name: 'file', kind: 'html', size: 2 }],
        Array.from({ length: 1001 }, () => ({ name: 'x', kind: 'file', size: 0 })),
    ]) {
        await assert.rejects(listTenantFiles('', async () => jsonResponse({ entries })), /unexpected file listing/);
    }
    await assert.rejects(listTenantFiles('', async () => ({ ok: true, json: async () => { throw new Error('sensitive'); } })), /unexpected file listing/);
});

test('downloads only selected regular files through same-origin GET and returns an inert binary blob', async () => {
    let call;
    const result = await downloadTenantFile('public/hello world.txt', async (url, options) => {
        call = { url, options };
        return { ok: true, status: 200, headers: new Headers({ 'Content-Type': 'application/octet-stream' }), arrayBuffer: async () => new TextEncoder().encode('hello').buffer };
    });
    assert.equal(call.url, '/api/v1/files/content?path=public%2Fhello+world.txt');
    assert.equal(call.options.method, 'GET');
    assert.equal(call.options.credentials, 'same-origin');
    assert.equal(call.options.cache, 'no-store');
    assert.equal(call.options.headers.Accept, 'application/octet-stream');
    assert.equal('Authorization' in call.options.headers, false);
    assert.equal(result.name, 'hello world.txt');
    assert.equal(result.blob.type, 'application/octet-stream');
    assert.equal(await result.blob.text(), 'hello');
});

test('rejects an oversized or non-binary download response', async () => {
    const tooLarge = new ArrayBuffer(16 * 1024 * 1024 + 1);
    const binaryHeaders = new Headers({ 'Content-Type': 'application/octet-stream' });
    await assert.rejects(downloadTenantFile('large.bin', async () => ({ ok: true, headers: binaryHeaders, arrayBuffer: async () => tooLarge })), /supported download limit/);
    await assert.rejects(downloadTenantFile('bad.bin', async () => ({ ok: true, headers: binaryHeaders, arrayBuffer: async () => 'not bytes' })), /supported download limit/);
    await assert.rejects(downloadTenantFile('redirect.html', async () => ({ ok: true, headers: new Headers({ 'Content-Type': 'text/html' }), arrayBuffer: async () => new ArrayBuffer(0) })), /unexpected file format/);
});

test('uses bounded requests and maps network failures without surfacing internals', async () => {
    await assert.rejects(listTenantFiles('', async (_url, { signal }) => new Promise((_resolve, reject) => {
        signal.addEventListener('abort', () => reject(Object.assign(new Error('secret'), { name: 'AbortError' })), { once: true });
    }), 10), /timed out/);
    await assert.rejects(listTenantFiles('', async (_url, { signal }) => ({
        ok: true,
        json: () => new Promise((_resolve, reject) => signal.addEventListener('abort', () => reject(Object.assign(new Error('private'), { name: 'AbortError' })), { once: true })),
    }), 10), /timed out/);
    await assert.rejects(listTenantFiles('', async () => { throw new Error('private host path'); }), /could not be reached/);
});

test('wires an accessible read-only explorer without upload, edit or delete controls', async () => {
    const root = new URL('../../cmd/tongd/ui/', import.meta.url);
    const [html, app, css] = await Promise.all([
        readFile(new URL('index.html', root), 'utf8'),
        readFile(new URL('app.js', root), 'utf8'),
        readFile(new URL('style.css', root), 'utf8'),
    ]);
    assert.match(html, /id="files-explorer-title"/);
    assert.match(html, /id="files-refresh" type="button"/);
    assert.match(html, /id="files-status" class="files-status" role="status" aria-live="polite"/);
    assert.match(app, /import \{ downloadTenantFile, listTenantFiles \} from '\.\/files-explorer\.mjs'/);
    assert.match(app, /name\.textContent =/);
    assert.match(html, /Files are not loaded/);
    assert.match(app, /filesRefresh\?\.addEventListener\('click'/);
    assert.match(css, /\.files-list\{/);
    assert.doesNotMatch(html, /type="file"|>Upload<|>Delete<|>Edit</);
});
