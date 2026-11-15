import { createServer } from 'node:http';
import { readFile } from 'node:fs/promises';
import { dirname, extname, resolve, sep } from 'node:path';
import { fileURLToPath } from 'node:url';
import { expect, test } from '@playwright/test';

const uiRoot = resolve(dirname(fileURLToPath(import.meta.url)), '../../cmd/tongd/ui');
const viewports = [
    { width: 320, height: 640 },
    { width: 390, height: 844 },
    { width: 768, height: 1024 },
    { width: 1024, height: 768 },
    { width: 1280, height: 800 },
];
const contentTypes = new Map([
    ['.css', 'text/css; charset=utf-8'],
    ['.html', 'text/html; charset=utf-8'],
    ['.js', 'text/javascript; charset=utf-8'],
    ['.mjs', 'text/javascript; charset=utf-8'],
]);

let server;
let baseURL;

function sendNotFound(response) {
    response.writeHead(404, { 'Content-Type': 'text/plain; charset=utf-8', 'Cache-Control': 'no-store' });
    response.end('Not found');
}

async function serveUI(request, response) {
    if (request.method !== 'GET') {
        response.writeHead(405, { Allow: 'GET', 'Cache-Control': 'no-store' });
        response.end();
        return;
    }

    let pathname;
    try { pathname = decodeURIComponent(new URL(request.url, 'http://127.0.0.1').pathname); }
    catch { sendNotFound(response); return; }

    let relativePath;
    if (pathname === '/') relativePath = 'index.html';
    else if (pathname.startsWith('/ui/')) relativePath = pathname.slice('/ui/'.length);
    else { sendNotFound(response); return; }

    const target = resolve(uiRoot, relativePath);
    const rootPrefix = uiRoot.endsWith(sep) ? uiRoot : uiRoot + sep;
    if (target !== uiRoot && !target.startsWith(rootPrefix)) { sendNotFound(response); return; }

    try {
        const body = await readFile(target);
        response.writeHead(200, {
            'Content-Type': contentTypes.get(extname(target)) ?? 'application/octet-stream',
            'Cache-Control': 'no-store',
            'X-Content-Type-Options': 'nosniff',
        });
        response.end(body);
    } catch { sendNotFound(response); }
}

test.beforeAll(async () => {
    server = createServer((request, response) => { void serveUI(request, response); });
    await new Promise((resolveListen, rejectListen) => {
        server.once('error', rejectListen);
        server.listen(0, '127.0.0.1', () => {
            server.off('error', rejectListen);
            resolveListen();
        });
    });
    const address = server.address();
    baseURL = `http://127.0.0.1:${address.port}/`;
});

test.afterAll(async () => {
    if (server?.listening) await new Promise((resolveClose, rejectClose) => server.close(error => error ? rejectClose(error) : resolveClose()));
});

async function openComponentPreview(page, records) {
    const origin = new URL(baseURL).origin;
    await page.route('**/*', async route => {
        const request = route.request();
        const url = new URL(request.url());
        if (url.origin !== origin) {
            records.external.push(url.href);
            await route.abort('blockedbyclient');
            return;
        }

        if (url.pathname.startsWith('/api/') || url.pathname === '/healthz') {
            const summary = {
                path: url.pathname,
                prefix: url.searchParams.get('prefix') ?? '',
                method: request.method(),
                hasCookie: Boolean(request.headers().cookie),
                hasAuthorization: Boolean(request.headers().authorization),
            };
            if (url.pathname === '/api/v1/sites') {
                records.sites.push(summary);
                await route.fulfill({
                    status: 200,
                    contentType: 'application/json',
                    body: JSON.stringify({
                        sites: [{ site_id: 'responsive-site', primary_domain: 'responsive.example.test', ownership_state: 'pending' }],
                        next_after: '',
                    }),
                });
                return;
            }
            if (url.pathname === '/api/v1/files') {
                records.files.push(summary);
                const entries = [
                    { name: 'documents', kind: 'directory', size: 0 },
                    { name: 'responsive-layout-notes.txt', kind: 'file', size: 38 },
                ].filter(entry => entry.name.startsWith(summary.prefix));
                await route.fulfill({
                    status: 200,
                    contentType: 'application/json',
                    body: JSON.stringify({
                        entries,
                        next_after: '',
                    }),
                });
                return;
            }
            if (url.pathname === '/api/v1/files/content') {
                records.previews.push(summary);
                await route.fulfill({
                    status: 200,
                    contentType: 'application/octet-stream',
                    body: 'Synthetic preview for responsive layout checks.',
                });
                return;
            }

            records.otherAPI.push(summary);
            await route.fulfill({ status: 503, contentType: 'application/json', body: '{"error":"synthetic unavailable"}' });
            return;
        }

        await route.continue();
    });

    // This serves the current checked-out UI assets. All API calls are
    // fulfilled or denied by the route above; no login or backend is used.
    await page.goto(baseURL);
    await expect(page.locator('#jobs-panel')).toBeVisible();
}

async function expectResponsiveBounds(page, viewport, selectors) {
    const documentWidth = await page.evaluate(() => document.documentElement.scrollWidth);
    expect(documentWidth, `document overflow at ${viewport.width}×${viewport.height}`).toBeLessThanOrEqual(viewport.width);
    for (const selector of selectors) {
        const element = page.locator(selector);
        await expect(element, `${selector} should be visible at ${viewport.width}×${viewport.height}`).toBeVisible();
        const bounds = await element.boundingBox();
        expect(bounds, `${selector} should have a rendered box at ${viewport.width}×${viewport.height}`).toBeTruthy();
        expect(bounds.x, `${selector} should not be clipped on the left at ${viewport.width}px`).toBeGreaterThanOrEqual(0);
        expect(bounds.x + bounds.width, `${selector} should not be clipped on the right at ${viewport.width}px`).toBeLessThanOrEqual(viewport.width + 1);
    }
}

function expectSyntheticReadOnlyRequests(records, expected) {
    expect(records.external).toEqual([]);
    expect(records.otherAPI.map(request => request.path).sort()).toEqual(['/api/v1/version', '/healthz']);
    for (const request of [...records.sites, ...records.files, ...records.previews, ...records.otherAPI]) {
        expect(request.method).toBe('GET');
        expect(request.hasCookie).toBe(false);
        expect(request.hasAuthorization).toBe(false);
    }
    expect(records.sites).toHaveLength(expected.sites);
    expect(records.files).toHaveLength(expected.files);
    expect(records.previews).toHaveLength(expected.previews);
}

test('component preview: Sites view fits phone, tablet, and desktop widths with synthetic data', async ({ page }) => {
    const records = { sites: [], files: [], previews: [], otherAPI: [], external: [] };
    await openComponentPreview(page, records);

    for (const viewport of viewports) {
        await page.setViewportSize(viewport);
        await page.locator('#sites-load').click();
        await expect(page.locator('#sites-status')).toHaveAttribute('data-state', 'available');
        await expect(page.locator('#sites-list')).toContainText('responsive.example.test');
        await page.locator('#sites-prefix').fill('res');
        await page.locator('#sites-filter button').click();
        await expect(page.locator('#sites-status')).toHaveAttribute('data-state', 'available');
        await expectResponsiveBounds(page, viewport, ['#sites-title', '#sites-load', '#sites-prefix', '#sites-filter button']);

        const usePreview = page.getByRole('button', { name: 'Use responsive-site for preview' });
        await expectResponsiveBounds(page, viewport, ['#sites-title', '#sites-load', '#sites-filter button', 'button[aria-label="Use responsive-site for preview"]']);
        await usePreview.click();
        await expect(page.locator('#site-preview-id')).toHaveValue('responsive-site');
    }

    expectSyntheticReadOnlyRequests(records, { sites: viewports.length * 2, files: 0, previews: 0 });
    expect(records.sites.map(request => request.prefix)).toEqual(['', ...Array(viewports.length * 2 - 1).fill('res')]);
});

test('component preview: Files view fits phone, tablet, and desktop widths with synthetic data', async ({ page }) => {
    const records = { sites: [], files: [], previews: [], otherAPI: [], external: [] };
    await openComponentPreview(page, records);

    for (const viewport of viewports) {
        await page.setViewportSize(viewport);
        await page.locator('#files-refresh').click();
        await expect(page.locator('#files-status')).toHaveAttribute('data-state', 'available');
        await expect(page.locator('#files-list')).toContainText('responsive-layout-notes.txt');
        await page.locator('#files-prefix').fill('responsive');
        await page.locator('#files-filter button').click();
        await expect(page.locator('#files-status')).toHaveAttribute('data-state', 'available');
        await expect(page.locator('#files-list')).not.toContainText('documents');
        await expectResponsiveBounds(page, viewport, ['#files-explorer-title', '#files-refresh', '#files-prefix', '#files-filter button']);

        const preview = page.getByRole('button', { name: 'Preview text responsive-layout-notes.txt' });
        await expectResponsiveBounds(page, viewport, ['#files-explorer-title', '#files-refresh', '#files-filter button', 'button[aria-label="Preview text responsive-layout-notes.txt"]']);
        await preview.click();
        await expect(page.locator('#files-preview')).toBeVisible();
        await expect(page.locator('#files-preview-content')).toHaveText('Synthetic preview for responsive layout checks.');
        await expectResponsiveBounds(page, viewport, ['#files-preview-title', '#files-preview-close', '#files-preview-content']);
        await page.locator('#files-preview-close').click();
        await expect(page.locator('#files-preview')).toBeHidden();
    }

    expectSyntheticReadOnlyRequests(records, { sites: 0, files: viewports.length * 2, previews: viewports.length });
    expect(records.files.map(request => request.prefix)).toEqual(['', ...Array(viewports.length * 2 - 1).fill('responsive')]);
});
