import assert from 'node:assert/strict';
import { readFile } from 'node:fs/promises';
import test from 'node:test';
import { runInNewContext } from 'node:vm';
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
    assert.equal(result.message, 'Panel health check failed. Retry to check again.');
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

test('Refresh button announces a result, blocks duplicate clicks, then retries', async () => {
    class FakeElement {
        constructor() {
            this.attributes = new Map();
            this.children = [];
            this.dataset = {};
            this.listeners = new Map();
            this.classList = { add() {} };
            this.disabled = false;
            this.textContent = '';
        }
        addEventListener(type, callback) { this.listeners.set(type, callback); }
        setAttribute(name, value) { this.attributes.set(name, value); }
        removeAttribute(name) { this.attributes.delete(name); }
        replaceChildren(...children) { this.children = children; }
        append(...children) { this.children.push(...children); }
        appendChild(child) { this.children.push(child); }
        click() {
            if (!this.disabled)
                this.listeners.get('click')?.();
        }
        showModal() {}
        close() {}
    }
    const ids = ['#tiles', '#search', '#status', '#status-version', '#status-checked', '#refresh-status', '#theme', '#intro', '#tour', '#close-tour', '#count'];
    const elements = new Map(ids.map(id => [id, new FakeElement()]));
    const document = {
        querySelector: (selector) => elements.get(selector) ?? null,
        createElement: () => new FakeElement(),
        documentElement: { dataset: {} },
    };
    const pending = [];
    const calls = [];
    const fetcher = (path) => {
        calls.push(path);
        if (calls.length <= 2)
            return new Promise(resolve => pending.push({ path, resolve }));
        return Promise.resolve(response(path === '/healthz' ? healthy : version));
    };
    const source = (await readFile(new URL('../../cmd/tongd/ui/app.js', import.meta.url), 'utf8'))
        .replace("import { loadPanelStatus } from './panel-status.mjs';", 'const loadPanelStatus = globalThis.testLoadPanelStatus;')
        .replace(/\nexport \{\};\s*$/, '\n');
    runInNewContext(source, {
        document,
        testLoadPanelStatus: () => loadPanelStatus(fetcher),
    });

    const status = elements.get('#status');
    const refresh = elements.get('#refresh-status');
    assert.equal(refresh.disabled, true);
    assert.equal(refresh.attributes.get('aria-busy'), 'true');
    refresh.click();
    assert.equal(calls.length, 2, 'disabled button must suppress overlapping checks');

    for (const item of pending)
        item.resolve(response(item.path === '/healthz' ? healthy : version));
    await new Promise(resolve => setTimeout(resolve, 0));
    assert.equal(status.textContent, 'Panel process is responding.');
    assert.equal(refresh.disabled, false);
    assert.equal(refresh.attributes.has('aria-busy'), false);

    refresh.click();
    await new Promise(resolve => setTimeout(resolve, 0));
    assert.equal(calls.length, 4, 'retry should request both read-only endpoints again');
    assert.equal(status.dataset.state, 'available');
    assert.equal(elements.get('#status-version').textContent, 'Tong Panel · experimental-core');
});
