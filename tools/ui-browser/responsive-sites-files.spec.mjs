import { createServer } from 'node:http';
import { readFile } from 'node:fs/promises';
import { dirname, extname, resolve, sep } from 'node:path';
import { fileURLToPath } from 'node:url';
import AxeBuilder from '@axe-core/playwright';
import { expect, test } from '@playwright/test';

const uiRoot = resolve(dirname(fileURLToPath(import.meta.url)), '../../cmd/tongd/ui');
const viewports = [
    { width: 320, height: 640 },
    { width: 390, height: 360 },
    { width: 390, height: 844 },
    { width: 768, height: 1024 },
    { width: 1024, height: 768 },
    { width: 1280, height: 800 },
    { width: 1440, height: 900 },
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

async function openComponentPreview(page, records, nextFailures = { sites: null, files: null }) {
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
                after: url.searchParams.get('after') ?? '',
                filesPath: url.searchParams.get('path') ?? '',
                page: url.searchParams.get('page'),
                method: request.method(),
                hasCookie: Boolean(request.headers().cookie),
                hasAuthorization: Boolean(request.headers().authorization),
            };
            if (url.pathname === '/api/v1/sites') {
                records.sites.push(summary);
                const failure = nextFailures.sites;
                if (failure) {
                    nextFailures.sites = null;
                    failure.markStarted();
                    await failure.released;
                    await route.abort('failed');
                    return;
                }
                if (nextFailures.paginatedSites) {
                    const firstPageSites = Array.from({ length: 50 }, (_, index) => {
                        const siteID = `site-${String(index + 1).padStart(3, '0')}`;
                        return { site_id: siteID, primary_domain: `${siteID}.example.test`, ownership_state: 'pending' };
                    });
                    const pages = {
                        '': { sites: firstPageSites, next_after: 'site-050' },
                        'site-050': { sites: [{ site_id: 'site-051', primary_domain: 'site-051.example.test', ownership_state: 'pending' }], next_after: '' },
                    };
                    if (!Object.hasOwn(pages, summary.after)) {
                        await route.fulfill({ status: 400, body: '' });
                        return;
                    }
                    await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify(pages[summary.after]) });
                    return;
                }
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
            const siteDetailsMatch = url.pathname.match(/^\/api\/v1\/sites\/([A-Za-z0-9_-]+)$/);
            if (siteDetailsMatch) {
                const siteID = siteDetailsMatch[1];
                records.siteDetails?.push(summary);
                if (siteID !== 'responsive-site') {
                    await route.fulfill({ status: 404, contentType: 'application/json', body: '{}' });
                    return;
                }
                await route.fulfill({
                    status: 200,
                    contentType: 'application/json',
                    body: JSON.stringify({
                        SiteID: 'responsive-site',
                        TenantID: 'synthetic-tenant',
                        DocumentRoot: 'sites/responsive-site/public',
                        CreatedAt: 1,
                        Domains: [{ Name: 'responsive.example.test', Kind: 'primary', OwnershipState: 'pending' }],
                    }),
                });
                return;
            }
            if (url.pathname === '/api/v1/files') {
                records.files.push(summary);
                const failure = nextFailures.files;
                if (failure) {
                    nextFailures.files = null;
                    failure.markStarted();
                    await failure.released;
                    await route.abort('failed');
                    return;
                }
                if (nextFailures.paginatedFiles) {
                    if (summary.after === 'alpha.txt' && nextFailures.deniedPageStatus) {
                        const status = nextFailures.deniedPageStatus;
                        nextFailures.deniedPageStatus = null;
                        await route.fulfill({ status, contentType: 'application/json', body: '{}' });
                        return;
                    }
                    if (nextFailures.openFolderTouchTarget && summary.filesPath === 'documents') {
                        await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ entries: [{ name: 'inside.txt', kind: 'file', size: 3 }], next_after: '' }) });
                        return;
                    }
                    const pages = {
                        '': { entries: nextFailures.openFolderTouchTarget ? [{ name: 'alpha.txt', kind: 'file', size: 5 }, { name: 'documents', kind: 'directory', size: 0 }] : [{ name: 'alpha.txt', kind: 'file', size: 5 }], next_after: nextFailures.openFolderTouchTarget ? 'documents' : 'alpha.txt' },
                        'documents': { entries: [{ name: 'beta.txt', kind: 'file', size: 4 }], next_after: '' },
                        'alpha.txt': { entries: [{ name: 'beta.txt', kind: 'file', size: 4 }], next_after: '' },
                    };
                    if (!Object.hasOwn(pages, summary.after)) {
                        await route.fulfill({ status: 400, body: '' });
                        return;
                    }
                    await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify(pages[summary.after]) });
                    return;
                }
                if (nextFailures.emptyFolderJourney && summary.filesPath === 'empty-folder') {
                    await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ entries: [], next_after: '' }) });
                    return;
                }
                const entries = [
                    ...(nextFailures.emptyFolderJourney ? [{ name: 'empty-folder', kind: 'directory', size: 0 }] : []),
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
                const heldPreview = nextFailures.preview;
                if (heldPreview) {
                    nextFailures.preview = null;
                    heldPreview.markStarted();
                    await heldPreview.released;
                    try {
                        await route.fulfill({ status: 200, contentType: 'application/octet-stream', body: 'Synthetic stale preview marker.' });
                        heldPreview.markFinished(true);
                    } catch {
                        heldPreview.markFinished(false);
                    }
                    return;
                }
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

function createHeldNetworkFailure() {
    let markStarted;
    let releaseFailure;
    const started = new Promise(resolve => { markStarted = resolve; });
    const released = new Promise(resolve => { releaseFailure = resolve; });
    return {
        started,
        released,
        markStarted: () => markStarted(),
        release: () => releaseFailure(),
    };
}

async function expectResponsiveBounds(page, viewport, selectors) {
    const documentWidth = await page.evaluate(() => document.documentElement.scrollWidth);
    expect(documentWidth, `document overflow at ${viewport.width}Ã—${viewport.height}`).toBeLessThanOrEqual(viewport.width);
    for (const selector of selectors) {
        const element = page.locator(selector);
        await expect(element, `${selector} should be visible at ${viewport.width}Ã—${viewport.height}`).toBeVisible();
        const bounds = await element.boundingBox();
        expect(bounds, `${selector} should have a rendered box at ${viewport.width}Ã—${viewport.height}`).toBeTruthy();
        expect(bounds.x, `${selector} should not be clipped on the left at ${viewport.width}px`).toBeGreaterThanOrEqual(0);
        expect(bounds.x + bounds.width, `${selector} should not be clipped on the right at ${viewport.width}px`).toBeLessThanOrEqual(viewport.width + 1);
    }
}

async function expectNoHorizontalContentOverflow(page, selectors) {
    for (const selector of selectors) {
        const element = page.locator(selector);
        const width = await element.evaluate(node => ({ scrollWidth: node.scrollWidth, clientWidth: node.clientWidth }));
        expect(width.scrollWidth, `${selector} should reflow without internal horizontal clipping`)
            .toBeLessThanOrEqual(width.clientWidth + 1);
    }
}

async function apply200PercentTextEnlargement(page) {
    return page.evaluate(() => {
        const selectors = ['section[aria-labelledby="sites-title"]', '.files-explorer'];
        const roots = selectors.map(selector => document.querySelector(selector));
        if (roots.some(root => !root)) throw new Error('Sites or Files fixture section is missing');

        const headings = {
            sites: document.querySelector('#sites-title'),
            files: document.querySelector('#files-explorer-title'),
        };
        const baseline = {
            sites: Number.parseFloat(getComputedStyle(headings.sites).fontSize),
            files: Number.parseFloat(getComputedStyle(headings.files).fontSize),
        };
        const elements = [...new Set(roots.flatMap(root => [root, ...root.querySelectorAll('*')]))];
        const originalFontSizes = elements.map(element => Number.parseFloat(getComputedStyle(element).fontSize));
        elements.forEach((element, index) => {
            const fontSize = originalFontSizes[index];
            if (Number.isFinite(fontSize)) element.style.setProperty('font-size', String(fontSize * 2) + 'px', 'important');
        });

        return {
            sitesHeading: Number.parseFloat(getComputedStyle(headings.sites).fontSize) / baseline.sites,
            filesHeading: Number.parseFloat(getComputedStyle(headings.files).fontSize) / baseline.files,
        };
    });
}

function expectSyntheticReadOnlyRequests(records, expected, pageLoads = 1) {
    expect(records.external).toEqual([]);
    expect(records.otherAPI.map(request => request.path).sort()).toEqual(
        Array.from({ length: pageLoads }, () => ['/api/v1/version', '/healthz']).flat().sort(),
    );
    const siteDetails = records.siteDetails ?? [];
    for (const request of [...records.sites, ...siteDetails, ...records.files, ...records.previews, ...records.otherAPI]) {
        expect(request.method).toBe('GET');
        expect(request.hasCookie).toBe(false);
        expect(request.hasAuthorization).toBe(false);
    }
    expect(records.sites).toHaveLength(expected.sites);
    expect(records.files).toHaveLength(expected.files);
    expect(records.previews).toHaveLength(expected.previews);
}

async function tabUntilFocused(page, selector, viewport) {
    const target = page.locator(selector);
    for (let step = 0; step < 128; step++) {
        await page.keyboard.press('Tab');
        if (await target.evaluate(element => document.activeElement === element)) return;
    }
    throw new Error(`${selector} was not reachable by Tab at ${viewport.width}Ã—${viewport.height}`);
}

async function expectVisibleKeyboardFocus(page, selector, viewport) {
    const focus = await page.locator(selector).evaluate(element => {
        const style = getComputedStyle(element);
        const bounds = element.getBoundingClientRect();
        return {
            focusVisible: element.matches(':focus-visible'),
            outlineStyle: style.outlineStyle,
            outlineWidth: style.outlineWidth,
            left: bounds.left,
            right: bounds.right,
        };
    });
    expect(focus.focusVisible, `${selector} should use keyboard-visible focus`).toBe(true);
    expect(focus.outlineStyle, `${selector} should show a focus outline`).not.toBe('none');
    expect(Number.parseFloat(focus.outlineWidth), `${selector} should have a nonzero focus outline`).toBeGreaterThan(0);
    expect(focus.left, `${selector} focus should stay inside the viewport`).toBeGreaterThanOrEqual(0);
    expect(focus.right, `${selector} focus should stay inside the viewport`).toBeLessThanOrEqual(viewport.width + 1);
}


function cssRgbChannels(color) {
    const channels = color.match(/[\d.]+/g)?.slice(0, 3).map(Number);
    expect(channels, 'expected a rendered RGB color, got ' + color).toHaveLength(3);
    return channels;
}

function relativeLuminance([red, green, blue]) {
    const linear = [red, green, blue].map(channel => {
        const normalized = channel / 255;
        return normalized <= 0.04045 ? normalized / 12.92 : ((normalized + 0.055) / 1.055) ** 2.4;
    });
    return 0.2126 * linear[0] + 0.7152 * linear[1] + 0.0722 * linear[2];
}

function contrastRatio(first, second) {
    const lighter = Math.max(relativeLuminance(first), relativeLuminance(second));
    const darker = Math.min(relativeLuminance(first), relativeLuminance(second));
    return (lighter + 0.05) / (darker + 0.05);
}

function gradientSamples(stops) {
    if (stops.length < 2) return stops;
    const samples = [];
    for (let index = 0; index < stops.length - 1; index++) {
        for (let step = 0; step <= 20; step++) {
            const amount = step / 20;
            samples.push(stops[index].map((channel, channelIndex) =>
                channel + (stops[index + 1][channelIndex] - channel) * amount));
        }
    }
    return samples;
}

async function expectFocusIndicatorContrast(page, selector, surfaceSelector, focusState) {
    const rendered = await page.locator(selector).evaluate((element, adjacentSelector) => {
        const outline = getComputedStyle(element);
        const surface = getComputedStyle(document.querySelector(adjacentSelector));
        const gradientStops = Array.from(surface.backgroundImage.matchAll(/rgba?\(([^)]+)\)/g), match =>
            match[1].split(',').slice(0, 3).map(Number));
        return {
            focused: element.matches(':focus'),
            focusVisible: element.matches(':focus-visible'),
            outlineColor: outline.outlineColor,
            outlineStyle: outline.outlineStyle,
            outlineWidth: outline.outlineWidth,
            backgroundColor: surface.backgroundColor,
            gradientStops,
        };
    }, surfaceSelector);
    expect(rendered.focused, selector + ' should be focused').toBe(true);
    if (focusState === 'keyboard') expect(rendered.focusVisible, selector + ' should use keyboard-visible focus').toBe(true);
    expect(rendered.outlineStyle, selector + ' should render its focus indicator').not.toBe('none');
    expect(Number.parseFloat(rendered.outlineWidth), selector + ' should have a nonzero focus indicator').toBeGreaterThan(0);

    const surfaceColors = rendered.gradientStops.length
        ? gradientSamples(rendered.gradientStops)
        : [cssRgbChannels(rendered.backgroundColor)];
    const outlineColor = cssRgbChannels(rendered.outlineColor);
    const minimumContrast = Math.min(...surfaceColors.map(color => contrastRatio(outlineColor, color)));
    const theme = await page.locator('html').getAttribute('data-theme');
    expect(minimumContrast, selector + ' outline ' + rendered.outlineColor + ' against ' + surfaceSelector + ' in ' + theme + ' theme').toBeGreaterThanOrEqual(3);
}

async function expectVisibleForcedColorFocus(page, selector, viewport) {
    const focus = await page.locator(selector).evaluate(element => {
        const style = getComputedStyle(element);
        const reference = document.createElement('span');
        reference.style.color = 'Highlight';
        reference.setAttribute('aria-hidden', 'true');
        document.body.append(reference);
        const highlightColor = getComputedStyle(reference).color;
        reference.remove();
        const bounds = element.getBoundingClientRect();
        return {
            forcedColors: matchMedia('(forced-colors: active)').matches,
            focusVisible: element.matches(':focus-visible'),
            outlineColor: style.outlineColor,
            outlineStyle: style.outlineStyle,
            outlineWidth: style.outlineWidth,
            highlightColor,
            left: bounds.left,
            right: bounds.right,
        };
    });
    expect(focus.forcedColors, selector + ' should be checked in forced-colors mode').toBe(true);
    expect(focus.focusVisible, selector + ' should use keyboard-visible focus').toBe(true);
    expect(focus.outlineStyle, selector + ' should show a focus outline').not.toBe('none');
    expect(Number.parseFloat(focus.outlineWidth), selector + ' should have a nonzero focus outline').toBeGreaterThan(0);
    expect(focus.outlineColor, selector + ' focus should use the system highlight color').toBe(focus.highlightColor);
    expect(focus.left, selector + ' focus should stay inside the viewport').toBeGreaterThanOrEqual(0);
    expect(focus.right, selector + ' focus should stay inside the viewport').toBeLessThanOrEqual(viewport.width + 1);
}

async function expectLiveStatus(page, selector, state, message) {
    const status = page.locator(selector);
    await expect(status).toHaveAttribute('role', 'status');
    await expect(status).toHaveAttribute('aria-live', 'polite');
    await expect(status).toHaveAttribute('aria-atomic', 'true');
    await expect(status).toHaveAttribute('data-state', state);
    await expect(status).toContainText(message);
}

async function expectReverseTabAndReturn(page, selector, previousSelector) {
    const target = page.locator(selector);
    await page.keyboard.press('Shift+Tab');
    if (previousSelector) await expect(page.locator(previousSelector)).toBeFocused();
    else await expect(target).not.toBeFocused();
    await page.keyboard.press('Tab');
    await expect(target).toBeFocused();
}

function keyboardActivation(index) {
    return index % 2 === 0 ? 'Enter' : 'Space';
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

    const sitesAxe = await new AxeBuilder({ page })
        .include('section[aria-labelledby="sites-title"]')
        .withTags(['wcag2a', 'wcag2aa'])
        .analyze();
    expect(sitesAxe.violations.map(({ id, impact, nodes }) => ({
        id,
        impact,
        nodes: nodes.map(node => ({ target: node.target, reason: node.failureSummary })),
    }))).toEqual([]);

    expectSyntheticReadOnlyRequests(records, { sites: viewports.length * 2, files: 0, previews: 0 });
    expect(records.sites.map(request => request.prefix)).toEqual(['', ...Array(viewports.length * 2 - 1).fill('res')]);
});

test('component preview: Sites details are keyboard reachable and fit a phone with synthetic data', async ({ page }) => {
    const records = { sites: [], siteDetails: [], files: [], previews: [], otherAPI: [], external: [] };
    const viewport = { width: 390, height: 844 };
    await page.setViewportSize(viewport);
    await openComponentPreview(page, records);
    await page.locator('#sites-load').click();
    await expect(page.locator('#sites-status')).toHaveAttribute('data-state', 'available');
    await expect(page.locator('#sites-list')).toContainText('responsive.example.test');

    const detailsButton = 'button[aria-label="View details for responsive-site"]';
    await tabUntilFocused(page, detailsButton, viewport);
    await expectVisibleKeyboardFocus(page, detailsButton, viewport);
    await expectResponsiveBounds(page, viewport, ['#sites-title', '#sites-load', detailsButton]);
    await page.keyboard.press('Enter');

    await expect(page.locator('#site-details')).toBeVisible();
    await expect(page.locator('#site-details-title')).toHaveText('Site details: responsive-site');
    await expect(page.locator('#site-details-title')).toBeFocused();
    await expect(page.locator('#site-details-fields')).toContainText('sites/responsive-site/public');
    await expect(page.locator('#site-details-domains')).toContainText('responsive.example.test');
    await expect(page.locator('#site-details-domains')).toContainText('pending');
    await expect(page.locator('#site-details-status')).toHaveAttribute('role', 'status');
    await expect(page.locator('#site-details-status')).toContainText('Catalog metadata only');
    await expectResponsiveBounds(page, viewport, ['#site-details-title', '#site-details-status', '#site-details-domains', '#site-details-close']);

    const detailsAxe = await new AxeBuilder({ page })
        .include('#site-details')
        .withTags(['wcag2a', 'wcag2aa'])
        .analyze();
    expect(detailsAxe.violations.map(({ id, impact, nodes }) => ({
        id,
        impact,
        nodes: nodes.map(node => ({ target: node.target, reason: node.failureSummary })),
    }))).toEqual([]);

    expectSyntheticReadOnlyRequests(records, { sites: 1, files: 0, previews: 0 });
    expect(records.siteDetails).toHaveLength(1);
    expect(records.siteDetails[0].path).toBe('/api/v1/sites/responsive-site');
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
        if (viewport.width === 320 && viewport.height === 640) {
            const previewAxe = await new AxeBuilder({ page })
                .include('#files-preview')
                .withTags(['wcag2a', 'wcag2aa'])
                .analyze();
            expect(previewAxe.violations.map(({ id, impact, nodes }) => ({
                id,
                impact,
                nodes: nodes.map(node => ({ target: node.target, reason: node.failureSummary })),
            }))).toEqual([]);
        }
        await page.locator('#files-preview-close').click();
        await expect(page.locator('#files-preview')).toBeHidden();
    }

    const filesAxe = await new AxeBuilder({ page })
        .include('.files-explorer')
        .withTags(['wcag2a', 'wcag2aa'])
        .analyze();
    expect(filesAxe.violations.map(({ id, impact, nodes }) => ({
        id,
        impact,
        nodes: nodes.map(node => ({ target: node.target, reason: node.failureSummary })),
    }))).toEqual([]);

    expectSyntheticReadOnlyRequests(records, { sites: 0, files: viewports.length * 2, previews: viewports.length });
    expect(records.files.map(request => request.prefix)).toEqual(['', ...Array(viewports.length * 2 - 1).fill('responsive')]);
});

test('component preview: Sites and Files reflow with 200% text enlargement and synthetic data', async ({ page }) => {
    const records = { sites: [], siteDetails: [], files: [], previews: [], otherAPI: [], external: [] };
    const viewport = { width: 320, height: 640 };
    await page.setViewportSize(viewport);
    await openComponentPreview(page, records);

    await page.locator('#sites-load').click();
    await expect(page.locator('#sites-list')).toContainText('responsive.example.test');
    await page.getByRole('button', { name: 'View details for responsive-site' }).click();
    await expect(page.locator('#site-details')).toBeVisible();

    await page.locator('#files-refresh').click();
    await expect(page.locator('#files-list')).toContainText('responsive-layout-notes.txt');
    await page.getByRole('button', { name: 'Preview text responsive-layout-notes.txt' }).click();
    await expect(page.locator('#files-preview')).toBeVisible();

    const scale = await apply200PercentTextEnlargement(page);
    expect(scale.sitesHeading).toBeCloseTo(2, 2);
    expect(scale.filesHeading).toBeCloseTo(2, 2);
    await expectResponsiveBounds(page, viewport, [
        '#sites-title', '#sites-load', '#sites-prefix', '#sites-filter button', '#sites-status',
        'button[aria-label="View details for responsive-site"]', '#site-details-title', '#site-details-status', '#site-details-close',
        '#files-explorer-title', '#files-refresh', '#files-prefix', '#files-filter button', '#files-breadcrumbs', '#files-status',
        'button[aria-label="Preview text responsive-layout-notes.txt"]', '#files-preview-title', '#files-preview-close', '#files-preview-content',
    ]);

    await expectNoHorizontalContentOverflow(page, [
        '#sites-list', '#site-details', '#site-details-fields', '#site-details-domains',
        '#files-breadcrumbs', '#files-list', '#files-preview-content',
    ]);

    for (const selector of ['#sites-status', '#site-details-close', '#files-status', '#files-preview-close']) {
        const target = page.locator(selector);
        await target.scrollIntoViewIfNeeded();
        const bounds = await target.boundingBox();
        expect(bounds, selector + ' should remain vertically reachable after text enlargement').toBeTruthy();
        expect(bounds.y, selector + ' should not be clipped above the viewport').toBeGreaterThanOrEqual(0);
        expect(bounds.y + bounds.height, selector + ' should not be clipped below the viewport').toBeLessThanOrEqual(viewport.height + 1);
    }

    await tabUntilFocused(page, '#site-details-close', viewport);
    await expectVisibleKeyboardFocus(page, '#site-details-close', viewport);
    await page.locator('#files-refresh').focus();
    await tabUntilFocused(page, '#files-preview-close', viewport);
    await expectVisibleKeyboardFocus(page, '#files-preview-close', viewport);

    expectSyntheticReadOnlyRequests(records, { sites: 1, files: 1, previews: 1 });
    expect(records.siteDetails).toHaveLength(1);
});

test('component preview: Sites controls work by keyboard at phone, tablet, and desktop widths', async ({ page }) => {
    const records = { sites: [], files: [], previews: [], otherAPI: [], external: [] };
    await openComponentPreview(page, records);

    for (const [index, viewport] of viewports.entries()) {
        await page.setViewportSize(viewport);
        if (index > 0) {
            await page.goto(baseURL);
            await expect(page.locator('#jobs-panel')).toBeVisible();
        }

        await tabUntilFocused(page, '#sites-load', viewport);
        await expectVisibleKeyboardFocus(page, '#sites-load', viewport);
        await expectResponsiveBounds(page, viewport, ['#sites-title', '#sites-load']);
        const currentFilesBreadcrumb = '#files-breadcrumbs button[aria-current="location"]';
        await expectReverseTabAndReturn(page, '#sites-load', currentFilesBreadcrumb);
        await expect(page.locator(currentFilesBreadcrumb)).toHaveText('Files');
        await page.keyboard.press(keyboardActivation(index));
        await expect(page.locator('#sites-status')).toHaveAttribute('data-state', 'available');
        await expect(page.locator('#sites-list')).toContainText('responsive.example.test');

        await tabUntilFocused(page, '#sites-prefix', viewport);
        await expectVisibleKeyboardFocus(page, '#sites-prefix', viewport);
        await page.keyboard.type('res');
        await page.keyboard.press('Tab');
        await expect(page.locator('#sites-filter button')).toBeFocused();
        await expectVisibleKeyboardFocus(page, '#sites-filter button', viewport);
        await expectReverseTabAndReturn(page, '#sites-filter button', '#sites-prefix');
        await page.keyboard.press(keyboardActivation(index));
        await expect(page.locator('#sites-status')).toHaveAttribute('data-state', 'available');

        const usePreview = 'button[aria-label="Use responsive-site for preview"]';
        await tabUntilFocused(page, usePreview, viewport);
        await expectVisibleKeyboardFocus(page, usePreview, viewport);
        await expectResponsiveBounds(page, viewport, ['#sites-title', '#sites-load', '#sites-filter button', usePreview]);
        await expectReverseTabAndReturn(page, usePreview, 'button[aria-label="View details for responsive-site"]');
        await page.keyboard.press(keyboardActivation(index));
        await expect(page.locator('#site-preview-id')).toHaveValue('responsive-site');
        await expect(page.locator('#site-preview-id')).toBeFocused();
        await expectVisibleKeyboardFocus(page, '#site-preview-id', viewport);
    }

    expectSyntheticReadOnlyRequests(records, { sites: viewports.length * 2, files: 0, previews: 0 }, viewports.length);
    expect(records.sites.map(request => request.prefix)).toEqual(viewports.flatMap(() => ['', 'res']));
});

test('component preview: Files controls work by keyboard at phone, tablet, and desktop widths', async ({ page }) => {
    const records = { sites: [], files: [], previews: [], otherAPI: [], external: [] };
    await openComponentPreview(page, records);

    for (const [index, viewport] of viewports.entries()) {
        await page.setViewportSize(viewport);
        if (index > 0) {
            await page.goto(baseURL);
            await expect(page.locator('#jobs-panel')).toBeVisible();
        }

        await tabUntilFocused(page, '#files-refresh', viewport);
        await expectVisibleKeyboardFocus(page, '#files-refresh', viewport);
        await expectResponsiveBounds(page, viewport, ['#files-explorer-title', '#files-refresh']);
        await expectReverseTabAndReturn(page, '#files-refresh', '#jobs-refresh');
        await page.keyboard.press(keyboardActivation(index));
        await expect(page.locator('#files-status')).toHaveAttribute('data-state', 'available');
        await expect(page.locator('#files-list')).toContainText('responsive-layout-notes.txt');

        await page.keyboard.press('Tab');
        await expect(page.locator('#files-prefix')).toBeFocused();
        await expectVisibleKeyboardFocus(page, '#files-prefix', viewport);
        await page.keyboard.type('responsive');
        await page.keyboard.press('Tab');
        await expect(page.locator('#files-filter button')).toBeFocused();
        await expectVisibleKeyboardFocus(page, '#files-filter button', viewport);
        await expectReverseTabAndReturn(page, '#files-filter button', '#files-prefix');
        await page.keyboard.press(keyboardActivation(index));
        await expect(page.locator('#files-status')).toHaveAttribute('data-state', 'available');
        await expect(page.locator('#files-list')).not.toContainText('documents');

        const preview = 'button[aria-label="Preview text responsive-layout-notes.txt"]';
        const download = 'button[aria-label="Download responsive-layout-notes.txt"]';
        await tabUntilFocused(page, preview, viewport);
        await expectVisibleKeyboardFocus(page, preview, viewport);
        await expectResponsiveBounds(page, viewport, ['#files-explorer-title', '#files-refresh', '#files-filter button', preview]);
        await expectReverseTabAndReturn(page, preview, download);
        await page.keyboard.press(keyboardActivation(index));
        await expect(page.locator('#files-preview')).toBeVisible();
        await expect(page.locator('#files-preview-content')).toHaveText('Synthetic preview for responsive layout checks.');

        await page.keyboard.press('Tab');
        await expect(page.locator('#files-preview-close')).toBeFocused();
        await expectVisibleKeyboardFocus(page, '#files-preview-close', viewport);
        await expectResponsiveBounds(page, viewport, ['#files-preview-title', '#files-preview-close', '#files-preview-content']);
        await expectReverseTabAndReturn(page, '#files-preview-close', preview);
        await page.keyboard.press(keyboardActivation(index));
        await expect(page.locator('#files-preview')).toBeHidden();
        await expect(page.locator(preview)).toBeFocused();
        await expectVisibleKeyboardFocus(page, preview, viewport);
        await page.keyboard.press('Tab');
        const focusInsideHiddenPreview = await page.locator('#files-preview').evaluate(element => element.contains(document.activeElement));
        expect(focusInsideHiddenPreview).toBe(false);
    }

    expectSyntheticReadOnlyRequests(records, { sites: 0, files: viewports.length * 2, previews: viewports.length }, viewports.length);
    expect(records.files.map(request => request.prefix)).toEqual(viewports.flatMap(() => ['', 'responsive']));
});

test('component preview: navigation, Sites, and Files keyboard focus follows forced colors at responsive widths', async ({ page }) => {
    const records = { sites: [], files: [], previews: [], otherAPI: [], external: [] };
    await page.emulateMedia({ forcedColors: 'active' });
    await openComponentPreview(page, records);

    for (const [index, viewport] of viewports.entries()) {
        await page.setViewportSize(viewport);
        if (index > 0) {
            await page.goto(baseURL);
            await expect(page.locator('#jobs-panel')).toBeVisible();
        }

        const homeLink = '.sidebar nav a[aria-label="Home"]';
        await tabUntilFocused(page, homeLink, viewport);
        await expectVisibleForcedColorFocus(page, homeLink, viewport);

        const noticeLink = '.notice a';
        await tabUntilFocused(page, noticeLink, viewport);
        await expectVisibleForcedColorFocus(page, noticeLink, viewport);

        await tabUntilFocused(page, '#sites-load', viewport);
        await expectVisibleForcedColorFocus(page, '#sites-load', viewport);
        await expectResponsiveBounds(page, viewport, ['#sites-title', '#sites-load']);
        await page.keyboard.press('Enter');
        await expect(page.locator('#sites-status')).toHaveAttribute('data-state', 'available');
        await expect(page.locator('#sites-list')).toContainText('responsive.example.test');

        await tabUntilFocused(page, '#files-refresh', viewport);
        await expectVisibleForcedColorFocus(page, '#files-refresh', viewport);
        await expectResponsiveBounds(page, viewport, ['#files-explorer-title', '#files-refresh']);
        await page.keyboard.press('Enter');
        await expect(page.locator('#files-status')).toHaveAttribute('data-state', 'available');
        await expect(page.locator('#files-list')).toContainText('responsive-layout-notes.txt');

        await page.keyboard.press('Tab');
        await expect(page.locator('#files-prefix')).toBeFocused();
        await expectVisibleForcedColorFocus(page, '#files-prefix', viewport);
        await page.keyboard.press('Tab');
        await expect(page.locator('#files-filter button')).toBeFocused();
        await expectVisibleForcedColorFocus(page, '#files-filter button', viewport);
        await page.keyboard.press('Enter');
        await expect(page.locator('#files-status')).toHaveAttribute('data-state', 'available');

        const preview = 'button[aria-label="Preview text responsive-layout-notes.txt"]';
        await tabUntilFocused(page, preview, viewport);
        await expectVisibleForcedColorFocus(page, preview, viewport);
        await page.keyboard.press('Enter');
        await expect(page.locator('#files-preview')).toBeVisible();
        await expect(page.locator('#files-preview-content')).toHaveText('Synthetic preview for responsive layout checks.');

        await page.keyboard.press('Tab');
        await expect(page.locator('#files-preview-close')).toBeFocused();
        await expectVisibleForcedColorFocus(page, '#files-preview-close', viewport);
        await page.keyboard.press('Enter');
        await expect(page.locator('#files-preview')).toBeHidden();
        await expect(page.locator(preview)).toBeFocused();
        await expectVisibleForcedColorFocus(page, preview, viewport);
        await expectResponsiveBounds(page, viewport, [preview]);
    }

    expectSyntheticReadOnlyRequests(records, {
        sites: viewports.length,
        files: viewports.length * 2,
        previews: viewports.length,
    }, viewports.length);
    expect(records.files.map(request => request.prefix)).toEqual(viewports.flatMap(() => ['', '']));
});

test('component preview: reduced motion preserves keyboard Sites and Files status updates', async ({ page }) => {
    const records = { sites: [], files: [], previews: [], otherAPI: [], external: [] };
    await page.emulateMedia({ reducedMotion: 'reduce' });
    await openComponentPreview(page, records);
    const refreshTransitionProperty = await page.locator('#files-refresh').evaluate(element => getComputedStyle(element).transitionProperty);
    expect(refreshTransitionProperty, 'reduced motion should disable control transitions').toBe('none');
    expect(await page.evaluate(() => matchMedia('(prefers-reduced-motion: reduce)').matches)).toBe(true);

    for (const [index, viewport] of viewports.entries()) {
        await page.setViewportSize(viewport);
        if (index > 0) {
            await page.goto(baseURL);
            await expect(page.locator('#jobs-panel')).toBeVisible();
            expect(await page.evaluate(() => matchMedia('(prefers-reduced-motion: reduce)').matches)).toBe(true);
        }

        await tabUntilFocused(page, '#sites-load', viewport);
        await expectVisibleKeyboardFocus(page, '#sites-load', viewport);
        await page.keyboard.press(keyboardActivation(index));
        await expect(page.locator('#sites-load')).toBeFocused();
        await expectLiveStatus(page, '#sites-status', 'available', 'Page 1: 1 catalog sites.');
        await expect(page.locator('#sites-list')).toContainText('responsive.example.test');

        await tabUntilFocused(page, '#files-refresh', viewport);
        await expectVisibleKeyboardFocus(page, '#files-refresh', viewport);
        await page.keyboard.press(keyboardActivation(index + 1));
        await expectLiveStatus(page, '#files-status', 'available', 'Page 1: 2 items.');
        await expect(page.locator('#files-list')).toContainText('responsive-layout-notes.txt');

        await page.keyboard.press('Tab');
        await expect(page.locator('#files-prefix')).toBeFocused();
        await expectVisibleKeyboardFocus(page, '#files-prefix', viewport);
        await page.keyboard.type('responsive');
        await page.keyboard.press('Tab');
        await expect(page.locator('#files-filter button')).toBeFocused();
        await expectVisibleKeyboardFocus(page, '#files-filter button', viewport);
        await page.keyboard.press(keyboardActivation(index));
        await expect(page.locator('#files-filter button')).toBeFocused();
        await expectLiveStatus(page, '#files-status', 'available', 'Page 1: 1 item.');
        await expect(page.locator('#files-list')).toContainText('responsive-layout-notes.txt');
        await expect(page.locator('#files-list')).not.toContainText('documents');

        const preview = 'button[aria-label="Preview text responsive-layout-notes.txt"]';
        await tabUntilFocused(page, preview, viewport);
        await expectVisibleKeyboardFocus(page, preview, viewport);
        await page.keyboard.press('Enter');
        await expect(page.locator('#files-preview')).toBeVisible();
        await expect(page.locator('#files-preview-content')).toHaveText('Synthetic preview for responsive layout checks.');
        await expectLiveStatus(page, '#files-status', 'available', 'Text preview loaded.');

        await page.keyboard.press('Tab');
        await expect(page.locator('#files-preview-close')).toBeFocused();
        await expectVisibleKeyboardFocus(page, '#files-preview-close', viewport);
        await page.keyboard.press('Enter');
        await expect(page.locator('#files-preview')).toBeHidden();
        await expect(page.locator(preview)).toBeFocused();
        await expectVisibleKeyboardFocus(page, preview, viewport);
        await expectLiveStatus(page, '#files-status', 'available', 'Text preview loaded.');
    }

    expectSyntheticReadOnlyRequests(records, {
        sites: viewports.length,
        files: viewports.length * 2,
        previews: viewports.length,
    }, viewports.length);
    expect(records.sites.map(request => request.prefix)).toEqual(viewports.map(() => ''));
    expect(records.files.map(request => request.prefix)).toEqual(viewports.flatMap(() => ['', 'responsive']));
});

test('component preview: Sites and Files retry synthetic network failures at phone sizes', async ({ page }) => {
    const records = { sites: [], files: [], previews: [], otherAPI: [], external: [] };
    const nextFailures = { sites: null, files: null };
    const retryViewports = viewports.filter(viewport =>
        (viewport.width === 320 && viewport.height === 640)
        || (viewport.width === 390 && viewport.height === 360));
    await openComponentPreview(page, records, nextFailures);

    for (const [index, viewport] of retryViewports.entries()) {
        await page.setViewportSize(viewport);
        if (index > 0) {
            await page.goto(baseURL);
            await expect(page.locator('#jobs-panel')).toBeVisible();
        }

        const filesFailure = createHeldNetworkFailure();
        nextFailures.files = filesFailure;
        await tabUntilFocused(page, '#files-refresh', viewport);
        await expectVisibleKeyboardFocus(page, '#files-refresh', viewport);
        await expectResponsiveBounds(page, viewport, ['#files-explorer-title', '#files-refresh', '#files-status']);
        await page.keyboard.press(keyboardActivation(index));
        try {
            await filesFailure.started;
            await expect(page.locator('#files-status')).toHaveAttribute('data-state', 'loading');
            await expect(page.locator('#files-refresh')).toBeDisabled();
            await expectResponsiveBounds(page, viewport, ['#files-explorer-title', '#files-refresh', '#files-status']);
        } finally {
            filesFailure.release();
        }
        await expect(page.locator('#files-status')).toHaveAttribute('data-state', 'unavailable');
        await expect(page.locator('#files-status')).toHaveText('Files could not be reached. Check your connection and try again.');
        await expect(page.locator('#files-list li')).toHaveCount(0);
        await expect(page.locator('#files-refresh')).toBeEnabled();
        await tabUntilFocused(page, '#files-refresh', viewport);
        await expectVisibleKeyboardFocus(page, '#files-refresh', viewport);
        await expectResponsiveBounds(page, viewport, ['#files-explorer-title', '#files-refresh', '#files-status']);
        await page.keyboard.press(keyboardActivation(index + 1));
        await expect(page.locator('#files-status')).toHaveAttribute('data-state', 'available');
        await expect(page.locator('#files-list')).toContainText('responsive-layout-notes.txt');
        await expectResponsiveBounds(page, viewport, ['#files-explorer-title', '#files-refresh', '#files-status']);

        const sitesFailure = createHeldNetworkFailure();
        nextFailures.sites = sitesFailure;
        await tabUntilFocused(page, '#sites-load', viewport);
        await expectVisibleKeyboardFocus(page, '#sites-load', viewport);
        await expectResponsiveBounds(page, viewport, ['#sites-title', '#sites-load', '#sites-status']);
        await page.keyboard.press(keyboardActivation(index + 1));
        try {
            await sitesFailure.started;
            await expect(page.locator('#sites-status')).toHaveAttribute('data-state', 'loading');
            await expectResponsiveBounds(page, viewport, ['#sites-title', '#sites-load', '#sites-status']);
        } finally {
            sitesFailure.release();
        }
        await expect(page.locator('#sites-status')).toHaveAttribute('data-state', 'unavailable');
        await expect(page.locator('#sites-status')).toHaveText('Sites could not be loaded. Try again.');
        await expect(page.locator('#sites-list li')).toHaveCount(0);
        await expect(page.locator('#sites-load')).toBeEnabled();
        await expect(page.locator('#sites-load')).toBeFocused();
        await expectVisibleKeyboardFocus(page, '#sites-load', viewport);
        await expectResponsiveBounds(page, viewport, ['#sites-title', '#sites-load', '#sites-status']);
        await page.keyboard.press(keyboardActivation(index));
        await expect(page.locator('#sites-status')).toHaveAttribute('data-state', 'available');
        await expect(page.locator('#sites-list')).toContainText('responsive.example.test');
        await expectResponsiveBounds(page, viewport, ['#sites-title', '#sites-load', '#sites-status']);
    }

    expectSyntheticReadOnlyRequests(records, {
        sites: retryViewports.length * 2,
        files: retryViewports.length * 2,
        previews: 0,
    }, retryViewports.length);
    expect(records.sites.map(request => request.prefix)).toEqual(retryViewports.flatMap(() => ['', '']));
    expect(records.files.map(request => request.prefix)).toEqual(retryViewports.flatMap(() => ['', '']));
});


test('component preview: Files refresh suppresses a held stale preview and recovers at phone sizes', async ({ page }) => {
    const records = { sites: [], files: [], previews: [], otherAPI: [], external: [] };
    const controls = { sites: null, files: null, preview: null };
    const phones = viewports.filter(viewport =>
        (viewport.width === 320 && viewport.height === 640)
        || (viewport.width === 390 && viewport.height === 360));
    await openComponentPreview(page, records, controls);

    for (const [index, viewport] of phones.entries()) {
        await page.setViewportSize(viewport);
        if (index > 0) {
            await page.goto(baseURL);
            await expect(page.locator('#jobs-panel')).toBeVisible();
        }
        await tabUntilFocused(page, '#files-refresh', viewport);
        await page.keyboard.press(keyboardActivation(index));
        await expect(page.locator('#files-status')).toHaveAttribute('data-state', 'available');
        const preview = page.getByRole('button', { name: 'Preview text responsive-layout-notes.txt' });
        await expect(preview).toBeVisible();

        let markFinished;
        const finished = new Promise(resolve => { markFinished = resolve; });
        const held = createHeldNetworkFailure();
        held.markFinished = markFinished;
        controls.preview = held;
        await preview.click();
        try {
            await held.started;
            await tabUntilFocused(page, '#files-refresh', viewport);
            await expectVisibleKeyboardFocus(page, '#files-refresh', viewport);
            await page.keyboard.press(keyboardActivation(index + 1));
            await expect(page.locator('#files-status')).toHaveAttribute('data-state', 'available');
            await expect(page.locator('#files-list')).toContainText('responsive-layout-notes.txt');
        } finally {
            held.release();
        }
        expect(await finished, 'held stale response must be fulfilled for this regression').toBe(true);
        await expect(page.locator('#files-preview')).toBeHidden();
        await expect(page.locator('#files-preview-content')).toBeEmpty();
        await expect(page.locator('#files-status')).toHaveAttribute('data-state', 'available');
        await expectResponsiveBounds(page, viewport, ['#files-explorer-title', '#files-refresh', '#files-status']);
        await tabUntilFocused(page, 'button[aria-label="Preview text responsive-layout-notes.txt"]', viewport);
        await expectVisibleKeyboardFocus(page, 'button[aria-label="Preview text responsive-layout-notes.txt"]', viewport);
        await page.keyboard.press(keyboardActivation(index));
        await expect(page.locator('#files-preview')).toBeVisible();
        await expect(page.locator('#files-preview-content')).toHaveText('Synthetic preview for responsive layout checks.');
        await expect(page.locator('#files-preview-content')).not.toContainText('Synthetic stale preview marker.');
        await expectResponsiveBounds(page, viewport, ['#files-preview-title', '#files-preview-close', '#files-preview-content']);
    }
    expectSyntheticReadOnlyRequests(records, { sites: 0, files: phones.length * 2, previews: phones.length * 2 }, phones.length);
});


test('component preview: Sites and Files controls remain vertically reachable on short screens', async ({ page }) => {
    const records = { sites: [], files: [], previews: [], otherAPI: [], external: [] };
    const shortViewports = [{ width: 320, height: 640 }, { width: 390, height: 360 }];
    await openComponentPreview(page, records);

    async function expectReachable(selector, viewport) {
        const element = page.locator(selector);
        await expect(element).toBeVisible();
        await element.scrollIntoViewIfNeeded();
        const geometry = await element.evaluate(node => {
            const rect = node.getBoundingClientRect();
            const hit = document.elementFromPoint(rect.left + rect.width / 2, rect.top + rect.height / 2);
            return { top: rect.top, bottom: rect.bottom, height: rect.height,
                hit: hit === node || node.contains(hit) };
        });
        expect(geometry.height, `${selector} has no height`).toBeGreaterThan(0);
        expect(geometry.top, `${selector} clipped above ${viewport.width}×${viewport.height}`).toBeGreaterThanOrEqual(0);
        expect(geometry.bottom, `${selector} clipped below ${viewport.width}×${viewport.height}`).toBeLessThanOrEqual(viewport.height + 1);
        expect(geometry.hit, `${selector} obscured at ${viewport.width}×${viewport.height}`).toBe(true);
    }

    for (const [index, viewport] of shortViewports.entries()) {
        await page.setViewportSize(viewport);
        if (index > 0) {
            await page.goto(baseURL);
            await expect(page.locator('#jobs-panel')).toBeVisible();
        }
        await expectReachable('#sites-load', viewport);
        await page.locator('#sites-load').click();
        await expect(page.locator('#sites-status')).toHaveAttribute('data-state', 'available');
        for (const selector of ['#sites-title', '#sites-prefix', '#sites-filter button', '#sites-status', 'button[aria-label="Use responsive-site for preview"]']) {
            await expectReachable(selector, viewport);
        }
        await expectReachable('#files-refresh', viewport);
        await page.locator('#files-refresh').click();
        await expect(page.locator('#files-status')).toHaveAttribute('data-state', 'available');
        for (const selector of ['#files-explorer-title', '#files-prefix', '#files-filter button', '#files-status', 'button[aria-label="Preview text responsive-layout-notes.txt"]']) {
            await expectReachable(selector, viewport);
        }
        await page.getByRole('button', { name: 'Preview text responsive-layout-notes.txt' }).click();
        await expect(page.locator('#files-preview-content')).toHaveText('Synthetic preview for responsive layout checks.');
        for (const selector of ['#files-preview-title', '#files-preview-close', '#files-preview-content']) {
            await expectReachable(selector, viewport);
        }
        await expectResponsiveBounds(page, viewport, ['#files-preview-close']);
        await page.locator('#files-preview-close').click();
        await expect(page.locator('#files-preview')).toBeHidden();
    }
    expectSyntheticReadOnlyRequests(records, { sites: shortViewports.length, files: shortViewports.length, previews: shortViewports.length }, shortViewports.length);
});


test('component preview: Files task survives portrait and landscape resizing without requests', async ({ page }) => {
    const records = { sites: [], files: [], previews: [], otherAPI: [], external: [] };
    const requests = [];
    page.on('request', request => requests.push({ url: request.url(), method: request.method() }));
    const portrait = { width: 390, height: 844 };
    const landscape = { width: 844, height: 390 };
    await page.setViewportSize(portrait);
    await openComponentPreview(page, records);
    await page.locator('#files-refresh').click();
    await expect(page.locator('#files-status')).toHaveAttribute('data-state', 'available');
    await page.locator('#files-prefix').fill('responsive');
    await page.locator('#files-filter button').click();
    await expect(page.locator('#files-status')).toHaveAttribute('data-state', 'available');
    await expect(page.locator('#files-list')).toContainText('responsive-layout-notes.txt');
    await expect(page.locator('#files-list')).not.toContainText('documents');
    const preview = 'button[aria-label="Preview text responsive-layout-notes.txt"]';
    await page.locator(preview).click();
    await expect(page.locator('#files-preview-content')).toHaveText('Synthetic preview for responsive layout checks.');
    await expect(page.locator('#files-preview')).toBeVisible();
    const listBefore = await page.locator('#files-list').textContent();
    const scopeBefore = await page.locator('#files-breadcrumbs').textContent();
    const urlBefore = page.url();
    const requestsBefore = requests.slice();

    for (const viewport of [landscape, portrait]) {
        await page.setViewportSize(viewport);
        await expect(page.locator('#files-prefix')).toHaveValue('responsive');
        expect(await page.locator('#files-list').textContent()).toBe(listBefore);
        expect(await page.locator('#files-breadcrumbs').textContent()).toBe(scopeBefore);
        await expect(page.locator('#files-status')).toHaveAttribute('data-state', 'available');
        await expect(page.locator('#files-preview')).toBeVisible();
        await expect(page.locator('#files-preview-content')).toHaveText('Synthetic preview for responsive layout checks.');
        expect(page.url()).toBe(urlBefore);
        expect(requests).toEqual(requestsBefore);
        await page.locator('#files-preview-close').scrollIntoViewIfNeeded();
        const bounds = await page.locator('#files-preview-close').boundingBox();
        expect(bounds.y).toBeGreaterThanOrEqual(0);
        expect(bounds.y + bounds.height).toBeLessThanOrEqual(viewport.height + 1);
        await expectResponsiveBounds(page, viewport, ['#files-preview-close']);
    }
    await tabUntilFocused(page, '#files-preview-close', portrait);
    await expectVisibleKeyboardFocus(page, '#files-preview-close', portrait);
    await page.keyboard.press('Enter');
    await expect(page.locator('#files-preview')).toBeHidden();
    await expect(page.locator(preview)).toBeFocused();
    expect(requests).toEqual(requestsBefore);
    expectSyntheticReadOnlyRequests(records, { sites: 0, files: 2, previews: 1 });
    expect(records.files.map(request => request.prefix)).toEqual(['', 'responsive']);
});


test('component preview: Sites task survives portrait and landscape resizing without requests', async ({ page }) => {
    const records = { sites: [], files: [], previews: [], otherAPI: [], external: [] };
    const requests = [];
    page.on('request', request => requests.push({ url: request.url(), method: request.method() }));
    const portrait = { width: 390, height: 844 };
    const landscape = { width: 844, height: 390 };
    await page.setViewportSize(portrait);
    await openComponentPreview(page, records);
    await page.locator('#sites-load').click();
    await expect(page.locator('#sites-status')).toHaveAttribute('data-state', 'available');
    await expect(page.locator('#sites-list')).toContainText('responsive.example.test');
    await page.locator('#sites-prefix').fill('res');
    await page.locator('#sites-filter button').click();
    await expect(page.locator('#sites-status')).toHaveAttribute('data-state', 'available');
    await expect(page.locator('#sites-list')).toContainText('responsive.example.test');
    const usePreview = 'button[aria-label="Use responsive-site for preview"]';
    await page.locator(usePreview).click();
    await expect(page.locator('#site-preview-id')).toHaveValue('responsive-site');
    const listBefore = await page.locator('#sites-list').textContent();
    const statusBefore = await page.locator('#sites-status').textContent();
    const urlBefore = page.url();
    const requestsBefore = requests.slice();

    for (const viewport of [landscape, portrait]) {
        await page.setViewportSize(viewport);
        await expect(page.locator('#sites-prefix')).toHaveValue('res');
        expect(await page.locator('#sites-list').textContent()).toBe(listBefore);
        await expect(page.locator('#sites-status')).toHaveAttribute('data-state', 'available');
        expect(await page.locator('#sites-status').textContent()).toBe(statusBefore);
        await expect(page.locator('#site-preview-id')).toHaveValue('responsive-site');
        expect(page.url()).toBe(urlBefore);
        expect(requests).toEqual(requestsBefore);
        await expectResponsiveBounds(page, viewport, ['#sites-title', '#sites-load', '#sites-prefix', '#sites-filter button', usePreview]);
    }

    expectSyntheticReadOnlyRequests(records, { sites: 2, files: 0, previews: 0 });
    expect(records.sites.map(request => request.prefix)).toEqual(['', 'res']);
});


test('component preview: blur and simulated hidden visibility clear Files previews at phone sizes', async ({ page }) => {
    const records = { sites: [], files: [], previews: [], otherAPI: [], external: [] };
    const requests = [];
    page.on('request', request => requests.push({ url: request.url(), method: request.method() }));
    const phones = [{ width: 320, height: 640 }, { width: 390, height: 360 }];
    await openComponentPreview(page, records);

    for (const [index, viewport] of phones.entries()) {
        await page.setViewportSize(viewport);
        if (index > 0) {
            await page.goto(baseURL);
            await expect(page.locator('#jobs-panel')).toBeVisible();
        }
        await page.locator('#files-refresh').click();
        await expect(page.locator('#files-status')).toHaveAttribute('data-state', 'available');
        const preview = page.getByRole('button', { name: 'Preview text responsive-layout-notes.txt' });

        for (const signal of ['blur', 'visibility']) {
            await preview.click();
            await expect(page.locator('#files-preview')).toBeVisible();
            await expect(page.locator('#files-preview-content')).toHaveText('Synthetic preview for responsive layout checks.');
            await page.locator('#files-prefix').focus();
            await expect(page.locator('#files-prefix')).toBeFocused();
            const beforeCleanup = requests.slice();
            // Deterministically simulate the event and, for visibility, the hidden
            // property. This does not claim a real browser tab/background transition.
            await page.evaluate(kind => {
                if (kind === 'blur') window.dispatchEvent(new Event('blur'));
                else {
                    Object.defineProperty(document, 'hidden', { configurable: true, value: true });
                    document.dispatchEvent(new Event('visibilitychange'));
                }
            }, signal);
            await expect(page.locator('#files-preview')).toBeHidden();
            await expect(page.locator('#files-preview-content')).toBeEmpty();
            await expect(page.locator('#files-prefix')).toBeFocused();
            await expect(preview).not.toBeFocused();
            expect(requests).toEqual(beforeCleanup);
            if (signal === 'visibility') {
                await page.evaluate(() => {
                    delete document.hidden;
                    document.dispatchEvent(new Event('visibilitychange'));
                });
            } else {
                await page.evaluate(() => window.dispatchEvent(new Event('focus')));
            }
            await expect(page.locator('#files-preview')).toBeHidden();
            await expect(page.locator('#files-preview-content')).toBeEmpty();
            await expect(page.locator('#files-prefix')).toBeFocused();
            expect(requests).toEqual(beforeCleanup);

            await preview.click();
            await expect(page.locator('#files-preview')).toBeVisible();
            await expect(page.locator('#files-preview-content')).toHaveText('Synthetic preview for responsive layout checks.');
            expect(requests).toHaveLength(beforeCleanup.length + 1);
            await page.locator('#files-preview-close').click();
        }
    }
    expectSyntheticReadOnlyRequests(records, { sites: 0, files: phones.length, previews: phones.length * 4 }, phones.length);
});


test('component preview: simulated focus loss suppresses pending Files previews at phone sizes', async ({ page }) => {
    const records = { sites: [], files: [], previews: [], otherAPI: [], external: [] };
    const controls = { sites: null, files: null, preview: null };
    const phones = [{ width: 320, height: 640 }, { width: 390, height: 360 }];
    await openComponentPreview(page, records, controls);
    for (const [index, viewport] of phones.entries()) {
        await page.setViewportSize(viewport);
        if (index > 0) {
            await page.goto(baseURL);
            await expect(page.locator('#jobs-panel')).toBeVisible();
        }
        await page.locator('#files-refresh').click();
        await expect(page.locator('#files-status')).toHaveAttribute('data-state', 'available');
        const preview = page.getByRole('button', { name: 'Preview text responsive-layout-notes.txt' });
        for (const signal of ['blur', 'visibility']) {
            let markFinished;
            const finished = new Promise(resolve => { markFinished = resolve; });
            const held = createHeldNetworkFailure();
            held.markFinished = markFinished;
            controls.preview = held;
            const response = page.waitForResponse(result => new URL(result.url()).pathname === '/api/v1/files/content');
            await preview.click();
            try {
                await held.started;
                await page.locator('#files-prefix').focus();
                // Simulated DOM events/property, not a real background-tab transition.
                await page.evaluate(kind => {
                    if (kind === 'blur') window.dispatchEvent(new Event('blur'));
                    else {
                        Object.defineProperty(document, 'hidden', { configurable: true, value: true });
                        document.dispatchEvent(new Event('visibilitychange'));
                    }
                }, signal);
                await expect(page.locator('#files-preview')).toBeHidden();
                await expect(page.locator('#files-preview-content')).toBeEmpty();
                await expect(page.locator('#files-prefix')).toBeFocused();
                if (signal === 'visibility') {
                    await page.evaluate(() => {
                        delete document.hidden;
                        document.dispatchEvent(new Event('visibilitychange'));
                    });
                } else {
                    await page.evaluate(() => window.dispatchEvent(new Event('focus')));
                }
            } finally {
                held.release();
            }
            expect(await finished, 'held preview must be fulfilled').toBe(true);
            expect((await (await response).body()).toString()).toBe('Synthetic stale preview marker.');
            await page.evaluate(() => new Promise(resolve => requestAnimationFrame(() => requestAnimationFrame(resolve))));
            await expect(page.locator('#files-preview')).toBeHidden();
            await expect(page.locator('#files-preview-content')).toBeEmpty();
            await expect(page.locator('#files-prefix')).toBeFocused();
            await expect(preview).not.toBeFocused();
            await preview.click();
            await expect(page.locator('#files-preview')).toBeVisible();
            await expect(page.locator('#files-preview-content')).toHaveText('Synthetic preview for responsive layout checks.');
            await page.locator('#files-preview-close').click();
        }
    }
    expectSyntheticReadOnlyRequests(records, { sites: 0, files: phones.length, previews: phones.length * 4 }, phones.length);
});


test('component preview: empty Files folder returns to root by keyboard at phone size with synthetic data', async ({ page }) => {
    const records = { sites: [], files: [], previews: [], otherAPI: [], external: [] };
    const fixture = { emptyFolderJourney: true };
    const viewport = { width: 320, height: 640 };
    await page.setViewportSize(viewport);
    await openComponentPreview(page, records, fixture);

    await tabUntilFocused(page, '#files-refresh', viewport);
    await expectVisibleKeyboardFocus(page, '#files-refresh', viewport);
    await page.keyboard.press('Enter');
    await expect(page.locator('#files-status')).toHaveAttribute('data-state', 'available');
    const openEmptyFolder = 'button[aria-label="Open folder empty-folder"]';
    await expect(page.locator('#files-list')).toContainText('empty-folder');
    await tabUntilFocused(page, openEmptyFolder, viewport);
    await expectVisibleKeyboardFocus(page, openEmptyFolder, viewport);
    await page.keyboard.press('Enter');

    await expect(page.locator('#files-breadcrumbs button[aria-current="location"]')).toHaveText('empty-folder');
    await expect(page.locator('#files-status')).toHaveAttribute('data-state', 'available');
    await expect(page.locator('#files-status')).toContainText('Page 1: 0 items.');
    await expect(page.locator('#files-list')).toHaveText('No items on this page.');
    await expect(page.locator('#files-previous')).toBeDisabled();
    await expect(page.locator('#files-next')).toBeDisabled();
    await expect(page.locator('#files-refresh')).toBeEnabled();
    await expect(page.locator('#files-prefix')).toBeEnabled();
    await expectResponsiveBounds(page, viewport, ['#files-explorer-title', '#files-breadcrumbs', '#files-status', '#files-refresh', '#files-prefix', '#files-filter button']);

    const rootBreadcrumb = '#files-breadcrumbs button[aria-current="false"]';
    await tabUntilFocused(page, rootBreadcrumb, viewport);
    await expectVisibleKeyboardFocus(page, rootBreadcrumb, viewport);
    await page.keyboard.press('Enter');
    await expect(page.locator('#files-breadcrumbs button[aria-current="location"]')).toHaveText('Files');
    await expect(page.locator('#files-status')).toHaveAttribute('data-state', 'available');
    await expect(page.locator('#files-status')).toContainText('Page 1: 3 items.');
    await expect(page.locator('#files-list')).toContainText('empty-folder');
    await expectResponsiveBounds(page, viewport, ['#files-explorer-title', '#files-breadcrumbs', '#files-status', '#files-refresh']);

    expectSyntheticReadOnlyRequests(records, { sites: 0, files: 3, previews: 0 });
    expect(records.files.map(({ filesPath, prefix }) => ({ filesPath, prefix }))).toEqual([
        { filesPath: '', prefix: '' },
        { filesPath: 'empty-folder', prefix: '' },
        { filesPath: '', prefix: '' },
    ]);
});


test('component preview: Files no-match filter recovers by keyboard at phone sizes', async ({ page }) => {
    const records = { sites: [], files: [], previews: [], otherAPI: [], external: [] };
    const phones = [{ width: 320, height: 640 }, { width: 390, height: 360 }];
    await openComponentPreview(page, records);
    for (const [index, viewport] of phones.entries()) {
        await page.setViewportSize(viewport);
        if (index > 0) {
            await page.goto(baseURL);
            await expect(page.locator('#jobs-panel')).toBeVisible();
        }
        await tabUntilFocused(page, '#files-refresh', viewport);
        await page.keyboard.press(keyboardActivation(index));
        await expect(page.locator('#files-status')).toHaveAttribute('data-state', 'available');
        const preview = 'button[aria-label="Preview text responsive-layout-notes.txt"]';
        await tabUntilFocused(page, preview, viewport);
        await page.keyboard.press(keyboardActivation(index));
        await expect(page.locator('#files-preview-content')).toHaveText('Synthetic preview for responsive layout checks.');

        await tabUntilFocused(page, '#files-prefix', viewport);
        await page.keyboard.type('no-such-entry');
        await page.keyboard.press('Tab');
        await expect(page.locator('#files-filter button')).toBeFocused();
        await expectVisibleKeyboardFocus(page, '#files-filter button', viewport);
        await page.keyboard.press(keyboardActivation(index + 1));
        await expect(page.locator('#files-status')).toHaveAttribute('data-state', 'available');
        await expect(page.locator('#files-status')).toContainText('Page 1: 0 items.');
        await expect(page.locator('#files-list')).toHaveText('No items on this page.');
        await expect(page.locator('#files-list')).not.toContainText('responsive-layout-notes.txt');
        await expect(page.locator('#files-preview')).toBeHidden();
        await expect(page.locator('#files-preview-content')).toBeEmpty();
        await expect(page.locator('#files-previous')).toBeDisabled();
        await expect(page.locator('#files-next')).toBeDisabled();
        await expect(page.locator('#files-refresh')).toBeEnabled();
        await expect(page.locator('#files-prefix')).toBeEnabled();
        await expectResponsiveBounds(page, viewport, ['#files-refresh', '#files-prefix', '#files-filter button', '#files-status']);

        await tabUntilFocused(page, '#files-prefix', viewport);
        await expectVisibleKeyboardFocus(page, '#files-prefix', viewport);
        await page.keyboard.press('ControlOrMeta+A');
        await page.keyboard.type('responsive');
        await page.keyboard.press('Tab');
        await expect(page.locator('#files-filter button')).toBeFocused();
        await page.keyboard.press(keyboardActivation(index));
        await expect(page.locator('#files-status')).toHaveAttribute('data-state', 'available');
        await expect(page.locator('#files-status')).toContainText('Page 1: 1 item.');
        await expect(page.locator('#files-list')).toContainText('responsive-layout-notes.txt');
        await expect(page.locator('#files-list')).not.toContainText('No items on this page.');
        await expect(page.locator('#files-previous')).toBeDisabled();
        await expect(page.locator('#files-next')).toBeDisabled();
        await tabUntilFocused(page, preview, viewport);
        await expectVisibleKeyboardFocus(page, preview, viewport);
    }
    expectSyntheticReadOnlyRequests(records, { sites: 0, files: phones.length * 3, previews: phones.length }, phones.length);
    expect(records.files.map(request => request.prefix)).toEqual(phones.flatMap(() => ['', 'no-such-entry', 'responsive']));
});


test('component preview: Files pages are reachable by keyboard at phone sizes', async ({ page }) => {
    const records = { sites: [], files: [], previews: [], otherAPI: [], external: [] };
    const controls = { sites: null, files: null, paginatedFiles: true };
    const phones = [{ width: 320, height: 640 }, { width: 390, height: 360 }];
    await openComponentPreview(page, records, controls);
    async function expectVerticalReachability(selector, viewport) {
        const element = page.locator(selector);
        await element.scrollIntoViewIfNeeded();
        const box = await element.boundingBox();
        expect(box).not.toBeNull();
        expect(box.y).toBeGreaterThanOrEqual(0);
        expect(box.y + box.height).toBeLessThanOrEqual(viewport.height + 1);
    }
    for (const [index, viewport] of phones.entries()) {
        await page.setViewportSize(viewport);
        if (index > 0) {
            await page.goto(baseURL);
            await expect(page.locator('#jobs-panel')).toBeVisible();
        }
        await tabUntilFocused(page, '#files-refresh', viewport);
        await expectVisibleKeyboardFocus(page, '#files-refresh', viewport);
        await page.keyboard.press(keyboardActivation(index));
        await expect(page.locator('#files-status')).toHaveAttribute('data-state', 'available');
        await expect(page.locator('#files-status')).toContainText('Page 1: 1 item.');
        await expect(page.locator('#files-list')).toContainText('alpha.txt');
        await expect(page.locator('#files-list')).not.toContainText('beta.txt');
        await expect(page.locator('#files-previous')).toBeDisabled();
        await expect(page.locator('#files-next')).toBeEnabled();
        const scope = await page.locator('#files-breadcrumbs').textContent();
        await tabUntilFocused(page, '#files-next', viewport);
        await expectVisibleKeyboardFocus(page, '#files-next', viewport);
        await expectResponsiveBounds(page, viewport, ['#files-next', '#files-status']);
        await expectVerticalReachability('#files-status', viewport);
        await expectVerticalReachability('#files-next', viewport);
        await page.keyboard.press(keyboardActivation(index));
        await expect(page.locator('#files-status')).toHaveAttribute('data-state', 'available');
        await expect(page.locator('#files-status')).toContainText('Page 2: 1 item.');
        await expect(page.locator('#files-list')).toContainText('beta.txt');
        await expect(page.locator('#files-list')).not.toContainText('alpha.txt');
        await expect(page.locator('#files-previous')).toBeEnabled();
        await expect(page.locator('#files-next')).toBeDisabled();
        await expect(page.locator('#files-previous')).toBeFocused();
        await expectVisibleKeyboardFocus(page, '#files-previous', viewport);
        expect(await page.locator('#files-breadcrumbs').textContent()).toBe(scope);
        await expectResponsiveBounds(page, viewport, ['#files-previous', '#files-status']);
        await expectVerticalReachability('#files-status', viewport);
        await expectVerticalReachability('#files-previous', viewport);
        await page.keyboard.press(keyboardActivation(index + 1));
        await expect(page.locator('#files-status')).toHaveAttribute('data-state', 'available');
        await expect(page.locator('#files-status')).toContainText('Page 1: 1 item.');
        await expect(page.locator('#files-list')).toContainText('alpha.txt');
        await expect(page.locator('#files-list')).not.toContainText('beta.txt');
        await expect(page.locator('#files-previous')).toBeDisabled();
        await expect(page.locator('#files-next')).toBeEnabled();
        await expect(page.locator('#files-next')).toBeFocused();
        await expectVisibleKeyboardFocus(page, '#files-next', viewport);
        expect(await page.locator('#files-breadcrumbs').textContent()).toBe(scope);
    }
    expectSyntheticReadOnlyRequests(records, { sites: 0, files: phones.length * 3, previews: 0 }, phones.length);
    expect(records.files.map(({ filesPath, after, prefix, page }) => ({ filesPath, after, prefix, page }))).toEqual(
        phones.flatMap(() => ['', 'alpha.txt', ''].map(after => ({ filesPath: '', after, prefix: '', page: '1' }))),
    );
});

test('component preview: denied Files pagination returns keyboard focus to refresh', async ({ page }) => {
    const records = { sites: [], files: [], previews: [], otherAPI: [], external: [] };
    const controls = { sites: null, files: null, paginatedFiles: true, deniedPageStatus: null };
    const denials = [
        { status: 401, message: 'Sign in with a secure browser session to browse files.' },
        { status: 403, message: 'Your account cannot access these files.' },
    ];
    const phones = [{ width: 320, height: 640 }, { width: 390, height: 360 }];
    await openComponentPreview(page, records, controls);

    for (const [index, denial] of denials.entries()) {
        const viewport = phones[index];
        await page.setViewportSize(viewport);
        if (index > 0) {
            await page.goto(baseURL);
            await expect(page.locator('#jobs-panel')).toBeVisible();
        }

        await tabUntilFocused(page, '#files-refresh', viewport);
        await page.keyboard.press('Enter');
        await expect(page.locator('#files-status')).toHaveAttribute('data-state', 'available');
        await expect(page.locator('#files-list')).toContainText('alpha.txt');
        await expect(page.locator('#files-next')).toBeEnabled();

        await tabUntilFocused(page, '#files-next', viewport);
        await expectVisibleKeyboardFocus(page, '#files-next', viewport);
        controls.deniedPageStatus = denial.status;
        await page.keyboard.press('Enter');

        await expect(page.locator('#files-status')).toHaveAttribute('data-state', 'unavailable');
        await expect(page.locator('#files-status')).toHaveText(denial.message);
        await expect(page.locator('#files-list')).toBeEmpty();
        await expect(page.locator('#files-previous')).toBeDisabled();
        await expect(page.locator('#files-next')).toBeDisabled();
        await expect(page.locator('#files-refresh')).toBeEnabled();
        await expect(page.locator('#files-refresh')).toBeFocused();
        await expectVisibleKeyboardFocus(page, '#files-refresh', viewport);
    }

    expectSyntheticReadOnlyRequests(records, { sites: 0, files: denials.length * 2, previews: 0 }, phones.length);
    expect(records.files.map(({ filesPath, after, prefix, page }) => ({ filesPath, after, prefix, page }))).toEqual(
        denials.flatMap(() => ['', 'alpha.txt'].map(after => ({ filesPath: '', after, prefix: '', page: '1' }))),
    );
});

test('component preview: phone Sites and Files pointer targets meet project sizes', async ({ page }) => {
    const records = { sites: [], files: [], previews: [], otherAPI: [], external: [] };
    const controls = { sites: null, files: null, paginatedSites: true, paginatedFiles: true, openFolderTouchTarget: true };
    const phones = [{ width: 320, height: 640 }, { width: 390, height: 844 }];

    async function expectTarget(selector, minimum, viewport) {
        const target = page.locator(selector);
        await expect(target, selector + ' should be visible at ' + viewport.width + 'px').toBeVisible();
        await target.scrollIntoViewIfNeeded();
        const bounds = await target.boundingBox();
        expect(bounds, selector + ' should have a target box at ' + viewport.width + 'px').toBeTruthy();
        expect(bounds.width, selector + ' target width at ' + viewport.width + 'px').toBeGreaterThanOrEqual(minimum);
        expect(bounds.height, selector + ' target height at ' + viewport.width + 'px').toBeGreaterThanOrEqual(minimum);
        const geometry = await target.evaluate(element => {
            const rect = element.getBoundingClientRect();
            const hit = document.elementFromPoint(rect.left + rect.width / 2, rect.top + rect.height / 2);
            return { left: rect.left, right: rect.right, top: rect.top, bottom: rect.bottom,
                hit: hit === element || element.contains(hit) };
        });
        expect(geometry.left, selector + ' left edge at ' + viewport.width + 'px').toBeGreaterThanOrEqual(0);
        expect(geometry.right, selector + ' right edge at ' + viewport.width + 'px').toBeLessThanOrEqual(viewport.width + 1);
        expect(geometry.top, selector + ' top edge at ' + viewport.width + 'px').toBeGreaterThanOrEqual(0);
        expect(geometry.bottom, selector + ' bottom edge at ' + viewport.width + 'px').toBeLessThanOrEqual(viewport.height + 1);
        expect(geometry.hit, selector + ' should receive a center pointer hit at ' + viewport.width + 'px').toBe(true);
    }

    await page.setViewportSize(phones[0]);
    await openComponentPreview(page, records, controls);

    for (const [index, viewport] of phones.entries()) {
        await page.setViewportSize(viewport);
        if (index > 0) {
            await page.goto(baseURL);
            await expect(page.locator('#jobs-panel')).toBeVisible();
        }

        await expectTarget('#sites-load', 44, viewport);
        await page.locator('#sites-load').click();
        await expect(page.locator('#sites-status')).toHaveAttribute('data-state', 'available');
        await expectTarget('#sites-prefix', 44, viewport);
        await expectTarget('#sites-filter button', 44, viewport);
        await expectTarget('#sites-list button[aria-label="View details for site-001"]', 24, viewport);
        await expectTarget('#sites-list button[aria-label="Use site-001 for preview"]', 44, viewport);
        await page.locator('#sites-prefix').fill('');
        await page.locator('#sites-filter button').click();
        await expect(page.locator('#sites-status')).toHaveAttribute('data-state', 'available');
        await page.locator('#sites-list button[aria-label="Use site-001 for preview"]').click();
        await expect(page.locator('#site-preview-id')).toHaveValue('site-001');

        await expectTarget('#sites-next', 44, viewport);
        await page.locator('#sites-next').click();
        await expect(page.locator('#sites-page')).toHaveText('Page 2');
        await expect(page.locator('#sites-list')).toContainText('site-051');
        await expectTarget('#sites-previous', 44, viewport);
        await page.locator('#sites-previous').click();
        await expect(page.locator('#sites-page')).toHaveText('Page 1');

        await expectTarget('#files-refresh', 44, viewport);
        await page.locator('#files-refresh').click();
        await expect(page.locator('#files-status')).toHaveAttribute('data-state', 'available');
        await expectTarget('#files-prefix', 44, viewport);
        await expectTarget('#files-filter button', 44, viewport);
        await page.locator('#files-prefix').fill('');
        await page.locator('#files-filter button').click();
        await expect(page.locator('#files-status')).toHaveAttribute('data-state', 'available');
        await expectTarget('button[aria-label="Download alpha.txt"]', 24, viewport);
        await expectTarget('button[aria-label="Open folder documents"]', 44, viewport);
        await expectTarget('button[aria-label="Preview text alpha.txt"]', 44, viewport);
        await page.locator('button[aria-label="Preview text alpha.txt"]').click();
        await expect(page.locator('#files-preview-content')).toHaveText('Synthetic preview for responsive layout checks.');
        await expectTarget('#files-preview-close', 44, viewport);
        await page.locator('#files-preview-close').click();
        await expect(page.locator('#files-preview')).toBeHidden();

        await expectTarget('#files-next', 44, viewport);
        await page.locator('#files-next').click();
        await expect(page.locator('#files-status')).toContainText('Page 2: 1 item.');
        await expectTarget('#files-previous', 44, viewport);
        await page.locator('#files-previous').click();
        await expect(page.locator('#files-status')).toContainText('Page 1: 2 items.');
        await page.locator('button[aria-label="Open folder documents"]').click();
        await expect(page.locator('#files-breadcrumbs')).toContainText('documents');
        await expect(page.locator('#files-status')).toHaveAttribute('data-state', 'available');
        await expect(page.locator('#files-list')).toContainText('inside.txt');
        await expectResponsiveBounds(page, viewport, ['#sites-load', '#sites-filter button', '#files-refresh', '#files-filter button']);
    }

    expectSyntheticReadOnlyRequests(records, {
        sites: phones.length * 4,
        files: phones.length * 5,
        previews: phones.length,
    }, phones.length);
    expect(records.files.filter(request => request.filesPath === 'documents')).toHaveLength(phones.length);
});

test('component preview: sampled focus states meet 3:1 against adjacent surfaces in light and dark themes', async ({ page }) => {
    const records = { sites: [], files: [], previews: [], otherAPI: [], external: [] };
    const viewport = { width: 1440, height: 900 };
    const cases = [
        { selector: '.sidebar nav a[aria-label="Home"]', surface: '.sidebar' },
        { selector: '#theme', surface: 'body' },
        { selector: '.notice a', surface: '.notice' },
        { selector: '#refresh-status', surface: '.panel-status' },
        { selector: '#jobs-refresh', surface: '#jobs-panel' },
        { selector: '#jobs-status', surface: '#jobs-panel', state: 'jobs-status' },
        { selector: '#sites-load', surface: '.jobs-panel[aria-labelledby="sites-title"]' },
        { selector: '#sites-prefix', surface: '.jobs-panel[aria-labelledby="sites-title"]' },
        { selector: '#sites-filter button', surface: '.jobs-panel[aria-labelledby="sites-title"]' },
        { selector: '#files-refresh', surface: '.files-explorer' },
        { selector: '#files-prefix', surface: '.files-explorer' },
        { selector: '#files-filter button', surface: '.files-explorer' },
        { selector: 'button[aria-label="Preview text responsive-layout-notes.txt"]', surface: '.files-explorer', state: 'file-entry' },
        { selector: '#job-detail-title', surface: '.job-detail', state: 'job-detail' },
        { selector: '#search', surface: '.search' },
    ];

    await page.setViewportSize(viewport);
    await openComponentPreview(page, records);

    for (const theme of ['light', 'dark']) {
        for (const focusCase of cases) {
            await page.goto(baseURL);
            await page.evaluate(themeName => { document.documentElement.dataset.theme = themeName; }, theme);

            if (focusCase.state === 'jobs-status') {
                await page.locator('#jobs-status').evaluate(element => element.focus());
                await expect(page.locator('#jobs-status')).toBeFocused();
                await expectFocusIndicatorContrast(page, focusCase.selector, focusCase.surface, 'programmatic');
                continue;
            }
            if (focusCase.state === 'job-detail') {
                await page.evaluate(() => {
                    document.querySelector('#job-detail').hidden = false;
                    document.querySelector('#job-detail-title').focus();
                });
                await expect(page.locator('#job-detail-title')).toBeFocused();
                await expectFocusIndicatorContrast(page, focusCase.selector, focusCase.surface, 'programmatic');
                continue;
            }
            if (focusCase.state === 'file-entry') {
                await tabUntilFocused(page, '#files-refresh', viewport);
                await page.keyboard.press('Enter');
                await expect(page.locator('#files-list')).toContainText('responsive-layout-notes.txt');
            }

            await tabUntilFocused(page, focusCase.selector, viewport);
            await expectVisibleKeyboardFocus(page, focusCase.selector, viewport);
            await expectFocusIndicatorContrast(page, focusCase.selector, focusCase.surface, 'keyboard');
        }
    }
});
