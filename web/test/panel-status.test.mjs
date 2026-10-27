import assert from 'node:assert/strict';
import { readFile } from 'node:fs/promises';
import test from 'node:test';
import { loadPanelStatus } from '../../cmd/tongd/ui/panel-status.mjs';

const healthy = { status: 'ok' };
const version = { product: 'Tong Panel', stage: 'experimental-core', api: 'v1' };
const response = (data, ok = true) => ({ ok, json: async () => data });

test('reports only panel-process availability and version on success', async () => {
    const paths = [];
    const result = await loadPanelStatus(async (path, options) => {
        paths.push(path);
        assert.equal(options.cache, 'no-store');
        assert.ok(options.signal instanceof AbortSignal);
        return response(path === '/healthz' ? healthy : version);
    });
    assert.deepEqual(paths.sort(), ['/api/v1/version', '/healthz']);
    assert.equal(result.state, 'available');
    assert.equal(result.message, 'Panel process is responding.');
    assert.equal(result.version, 'Tong Panel · experimental-core');
    assert.ok(result.checkedAt instanceof Date);
    assert.equal('host' in result, false);
});

test('reports unavailable when the process health endpoint fails', async () => {
    const result = await loadPanelStatus(async (path) => response(path === '/healthz' ? {} : version, path !== '/healthz'));
    assert.equal(result.state, 'unavailable');
});

test('reports unknown when the health response is malformed or unexpected', async () => {
    const malformed = await loadPanelStatus(async (path) => ({
        ok: true,
        json: async () => {
            if (path === '/healthz')
                throw new SyntaxError('invalid JSON');
            return version;
        },
    }));
    assert.equal(malformed.state, 'unknown');

    const unexpected = await loadPanelStatus(async (path) => response(path === '/healthz' ? { status: 'degraded' } : version));
    assert.equal(unexpected.state, 'unknown');
});

test('bounds endpoint waits and reports timeout as unavailable', async () => {
    const result = await loadPanelStatus((_path, { signal }) => new Promise((_resolve, reject) => {
        signal.addEventListener('abort', () => reject(Object.assign(new Error('aborted'), { name: 'AbortError' })), { once: true });
    }), 10);
    assert.equal(result.state, 'unavailable');
});

test('keeps process availability when version details are unavailable', async () => {
    const result = await loadPanelStatus(async (path) => {
        if (path === '/api/v1/version')
            return response({}, false);
        return response(healthy);
    });
    assert.equal(result.state, 'available');
    assert.match(result.message, /version information is unavailable/);
    assert.equal(result.version, undefined);
});

test('allows a retry after a failed check', async () => {
    let checks = 0;
    const fetcher = async (path) => {
        if (checks === 0)
            return response({}, false);
        return response(path === '/healthz' ? healthy : version);
    };
    assert.equal((await loadPanelStatus(fetcher)).state, 'unavailable');
    checks++;
    assert.equal((await loadPanelStatus(fetcher)).state, 'available');
});

test('keeps navigation names and refresh control available at phone widths', async () => {
    const root = new URL('../../cmd/tongd/ui/', import.meta.url);
    const [html, css, app] = await Promise.all([
        readFile(new URL('index.html', root), 'utf8'),
        readFile(new URL('style.css', root), 'utf8'),
        readFile(new URL('app.js', root), 'utf8'),
    ]);
    for (const label of ['Home', 'Roadmap', 'Updates', 'Source code'])
        assert.ok(html.includes(`aria-label="${label}"`), `missing navigation label: ${label}`);
    assert.match(html, /id="status" role="status" aria-live="polite" aria-atomic="true"/);
    assert.match(html, /id="refresh-status" type="button">Refresh status/);
    assert.match(css, /\.sidebar nav a:nth-child\(2\),\.sidebar nav a:nth-child\(3\)\{display:flex\}/);
    assert.match(css, /\.sidebar nav\{[^}]*overflow-x:auto/);
    assert.match(app, /refreshStatusButton\?\.addEventListener\('click'/);
});
