import { test, expect } from '@playwright/test';
import AxeBuilder from '@axe-core/playwright';
import { spawn } from 'node:child_process';
import { once } from 'node:events';
import { join } from 'node:path';
import { tmpdir } from 'node:os';
import { randomUUID } from 'node:crypto';

let child;
let fixture;

function hasExited(process) {
    return process.exitCode !== null || process.signalCode !== null;
}

function waitForExit(process, timeoutMs) {
    if (hasExited(process)) return Promise.resolve(true);
    return new Promise(resolve => {
        const timer = setTimeout(() => finish(false), timeoutMs);
        const exited = () => finish(true);
        const failed = () => {}; // A process error is not proof that it exited.
        function finish(value) {
            clearTimeout(timer);
            process.off('exit', exited);
            process.off('error', failed);
            resolve(value);
        }
        process.once('exit', exited);
        process.on('error', failed);
    });
}

async function stopChild(process) {
    // Failed spawn has no process to reap. Signal termination leaves exitCode
    // null, so signalCode must also be checked before waiting for another event.
    if (!process?.pid) return;
    if (hasExited(process)) {
        if (process.exitCode !== 0) throw new Error('Browser fixture failed');
        return;
    }
    const cleanExit = waitForExit(process, 7000);
    const ignorePipeError = () => {};
    // Retain the one-shot pipe error handler through stream close; EPIPE may
    // arrive just after the child's exit event during a concurrent shutdown.
    process.stdin?.once('error', ignorePipeError);
    try { process.stdin?.end(); } catch {} // A concurrent exit can close stdin.
    if (!await cleanExit) {
        try { process.kill('SIGKILL'); } catch {}
        if (!await waitForExit(process, 3000)) throw new Error('Browser fixture could not be reaped within bound');
        throw new Error('Browser fixture did not stop cleanly');
    }
    if (process.exitCode !== 0) throw new Error('Browser fixture failed');
}

async function stopFixture() {
    await stopChild(child);
}

test.beforeAll(async () => {
    const binary = process.env.TONG_UI_FIXTURE_BINARY;
    if (!binary) throw new Error('TONG_UI_FIXTURE_BINARY is required');
    const env = Object.fromEntries(Object.entries(process.env).filter(([key]) => !key.toUpperCase().startsWith('TONG_')));
    child = spawn(binary, ['-test.run=^TestJobsBrowserFixture$'], {
        env: { ...env, TONG_UI_BROWSER_FIXTURE: 'yes' },
        windowsHide: true,
        stdio: ['pipe', 'pipe', 'ignore'],
    });
    try {
        fixture = await new Promise((resolve, reject) => {
            let bytes = '';
            const timer = setTimeout(() => finish(new Error('Browser fixture readiness timed out')), 10000);
            function finish(error, value) {
                clearTimeout(timer);
                child.stdout.off('data', receive);
                child.off('error', failed);
                child.off('exit', exited);
                error ? reject(error) : resolve(value);
            }
            function failed() { finish(new Error('Browser fixture could not start')); }
            function exited() { finish(new Error('Browser fixture exited before readiness')); }
            function receive(chunk) {
                bytes += chunk.toString('utf8');
                if (bytes.length > 8192) return finish(new Error('Browser fixture handshake exceeded bound'));
                const newline = bytes.indexOf('\n');
                if (newline < 0) return;
                try { finish(null, JSON.parse(bytes.slice(0, newline))); }
                catch { finish(new Error('Browser fixture handshake was invalid')); }
            }
            child.stdout.on('data', receive);
            child.once('error', failed);
            child.once('exit', exited);
        });
        for (const key of ['Primary', 'Unconfigured', 'NoJobs']) {
            const url = new URL(fixture[key]);
            if (url.protocol !== 'https:' || url.hostname !== '127.0.0.1') throw new Error('Browser fixture must stay on TLS loopback');
        }
    } catch (error) {
        await stopFixture();
        throw error;
    }
});

test.afterAll(stopFixture);

test('fixture cleanup handles failed spawn and a child already terminated by signal', async () => {
    const missing = spawn(join(tmpdir(), 'tong-ui-missing-' + randomUUID(), 'fixture'), [], {
        stdio: 'ignore', windowsHide: true,
    });
    await once(missing, 'error');
    await stopChild(missing);

    const signaled = spawn(process.execPath, ['-e', 'setInterval(() => {}, 1000)'], {
        stdio: ['pipe', 'ignore', 'ignore'], windowsHide: true,
    });
    try {
        await once(signaled, 'spawn');
        signaled.kill('SIGKILL');
        expect(await waitForExit(signaled, 3000)).toBe(true);
        expect(signaled.exitCode).toBeNull();
        expect(signaled.signalCode).toBe('SIGKILL');
        await expect(stopChild(signaled)).rejects.toThrow('Browser fixture failed');
    } finally {
        if (!hasExited(signaled)) {
            signaled.kill('SIGKILL');
            if (!await waitForExit(signaled, 3000)) throw new Error('Cleanup regression child did not exit');
        }
    }
});

async function open(page, identity = 'alice', base = fixture.Primary) {
    await page.context().route('**/*', route => {
        const origin = new URL(route.request().url()).origin;
        if ([fixture.Primary, fixture.Unconfigured, fixture.NoJobs].includes(origin)) return route.continue();
        return route.abort('blockedbyclient');
    });
    if (identity) await page.context().addCookies([{
        name: '__Host-tong_session', value: fixture.Tokens[identity], url: base,
        secure: true, httpOnly: true, sameSite: 'Strict',
    }]);
    await page.goto(base);
    await expect(page.locator('#jobs-panel')).toBeVisible();
}

async function load(page) {
    await page.locator('#jobs-refresh').click();
    await expect(page.locator('#jobs-status')).not.toHaveAttribute('data-state', 'loading');
}

test('actual TLS session shows only own SQL jobs, supports keyboard details and passes scoped accessibility checks', async ({ page, browser }) => {
    const methods = [];
    page.on('request', request => {
        if (new URL(request.url()).pathname.startsWith('/api/v1/jobs')) methods.push(request.method());
    });
    await open(page);
    await expect(page.locator('#jobs-list .job-entry')).toHaveCount(0);
    await load(page);
    await expect(page.locator('#jobs-list .job-entry')).toHaveCount(2);
    await expect(page.locator('#jobs-list')).toContainText(fixture.IDs.completed);
    await expect(page.locator('#jobs-list')).toContainText(fixture.IDs.queued);
    await expect(page.locator('#jobs-list')).not.toContainText(fixture.IDs.sameTenant);
    await expect(page.locator('#jobs-list')).not.toContainText(fixture.IDs.otherTenant);
    for (const id of [fixture.IDs.sameTenant, fixture.IDs.otherTenant]) {
        const status = await page.evaluate(async id => (await fetch(`/api/v1/jobs/${id}`, { credentials: 'same-origin' })).status, id);
        expect(status).toBe(404);
    }
    const button = page.locator(`button[data-job-id="${fixture.IDs.completed}"]`);
    await button.focus();
    await page.keyboard.press('Enter');
    await expect(page.locator('#job-detail-title')).toBeFocused();
    await expect(page.locator('#job-detail-status')).toHaveAttribute('data-state', 'available');
    await expect(page.locator('#job-detail-fields')).toContainText('completed');
    const axe = await new AxeBuilder({ page }).include('#jobs-panel').withTags(['wcag2a', 'wcag2aa']).analyze();
    expect(axe.violations.map(({ id, impact, nodes }) => ({ id, impact, nodes: nodes.map(node => ({ target: node.target, reason: node.failureSummary })) }))).toEqual([]);
    expect(methods.every(method => method === 'GET')).toBe(true);
    expect(await page.evaluate(() => localStorage.length + sessionStorage.length)).toBe(0);
    test.info().annotations.push({ type: 'browser', description: browser.version() });
    await page.screenshot({ path: test.info().outputPath('jobs-desktop.png'), fullPage: true });
});

test('auditor reads their own job and an empty account has a truthful empty state', async ({ page }) => {
    await open(page, 'auditor');
    await load(page);
    await expect(page.locator('#jobs-list .job-entry')).toHaveCount(1);
    await expect(page.locator('#jobs-list')).toContainText(fixture.IDs.auditor);
    await expect(page.locator('#jobs-list')).not.toContainText(fixture.IDs.completed);
    await page.context().clearCookies();
    await page.context().addCookies([{ name: '__Host-tong_session', value: fixture.Tokens.empty, url: fixture.Primary, secure: true, httpOnly: true, sameSite: 'Strict' }]);
    await load(page);
    await expect(page.locator('#jobs-status')).toHaveAttribute('data-state', 'empty');
    await expect(page.locator('#jobs-list .job-entry')).toHaveCount(0);
});

test('real absent session, unconfigured auth and unavailable jobs expose no credential-entry flow', async ({ page }) => {
    await open(page, null);
    await load(page);
    await expect(page.locator('#jobs-status')).toHaveAttribute('data-state', 'session_required');
    await page.goto(fixture.Unconfigured);
    await load(page);
    await expect(page.locator('#jobs-status')).toHaveAttribute('data-state', 'auth_unconfigured');
    await page.context().addCookies([{ name: '__Host-tong_session', value: fixture.Tokens.alice, url: fixture.NoJobs, secure: true, httpOnly: true, sameSite: 'Strict' }]);
    await page.goto(fixture.NoJobs);
    await load(page);
    await expect(page.locator('#jobs-status')).toHaveAttribute('data-state', 'jobs_unconfigured');
    await expect(page.locator('#jobs-panel input')).toHaveCount(0);
});

test('expired access clears prior list and details using actual authentication', async ({ page }) => {
    await open(page);
    await load(page);
    await page.locator('button[data-job-id]').first().click();
    await expect(page.locator('#job-detail-status')).toHaveAttribute('data-state', 'available');
    await page.context().clearCookies();
    await page.locator('button[data-job-id]').first().click();
    await expect(page.locator('#jobs-status')).toHaveAttribute('data-state', 'session_required');
    await expect(page.locator('#jobs-list .job-entry')).toHaveCount(0);
    await expect(page.locator('#job-detail')).toBeHidden();
    await expect(page.locator('#jobs-refresh')).toBeFocused();
});

test('mobile jobs and details remain within viewport and accessible', async ({ page }) => {
    await page.setViewportSize({ width: 390, height: 844 });
    await open(page);
    await load(page);
    await page.locator('button[data-job-id]').first().click();
    await expect(page.locator('#job-detail-status')).toHaveAttribute('data-state', 'available');
    expect(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth)).toBe(true);
    const axe = await new AxeBuilder({ page }).include('#jobs-panel').withTags(['wcag2a', 'wcag2aa']).analyze();
    expect(axe.violations.map(({ id, impact, nodes }) => ({ id, impact, nodes: nodes.map(node => ({ target: node.target, reason: node.failureSummary })) }))).toEqual([]);
    await page.screenshot({ path: test.info().outputPath('jobs-mobile.png'), fullPage: true });
});

test('dark theme job details pass scoped accessibility checks', async ({ page }) => {
    await open(page);
    await load(page);
    await page.locator('button[data-job-id]').first().click();
    await expect(page.locator('#job-detail-status')).toHaveAttribute('data-state', 'available');
    await page.locator('#theme').click();
    await expect(page.locator('html')).toHaveAttribute('data-theme', 'dark');
    const axe = await new AxeBuilder({ page }).include('#jobs-panel').withTags(['wcag2a', 'wcag2aa']).analyze();
    expect(axe.violations.map(({ id, impact, nodes }) => ({ id, impact, nodes: nodes.map(node => ({ target: node.target, reason: node.failureSummary })) }))).toEqual([]);
});

test('mocked non-JSON detail 401 clears already rendered jobs and details', async ({ page }) => {
    await open(page);
    await load(page);
    const button = page.locator('button[data-job-id]').first();
    const id = await button.getAttribute('data-job-id');
    await button.click();
    await expect(page.locator('#job-detail-status')).toHaveAttribute('data-state', 'available');
    await page.route(`**/api/v1/jobs/${id}`, route => route.fulfill({
        status: 401, contentType: 'text/html', body: '<p>Synthetic non-JSON authentication failure</p>',
    }));
    await button.click();
    await expect(page.locator('#jobs-status')).toHaveAttribute('data-state', 'session_required');
    await expect(page.locator('#jobs-list .job-entry')).toHaveCount(0);
    await expect(page.locator('#job-detail')).toBeHidden();
    await expect(page.locator('#jobs-refresh')).toBeFocused();
});

test('existing file explorer remains wired when the fixture has no file backend', async ({ page }) => {
    await open(page);
    await page.locator('#files-refresh').click();
    await expect(page.locator('#files-status')).toHaveAttribute('data-state', 'unavailable');
    await expect(page.locator('#files-status')).toHaveText('File access is not enabled on this server yet.');
});

test('large actual job history navigates bounded pages and returns to the first page', async ({ page }) => {
    await open(page, 'bulk');
    const legacyBytes = await page.evaluate(async () => {
        const response = await fetch('/api/v1/jobs', { credentials: 'same-origin' });
        return new TextEncoder().encode(await response.text()).byteLength;
    });
    expect(legacyBytes).toBeGreaterThan(256 * 1024);
    await load(page);
    await expect(page.locator('#jobs-list .job-entry')).toHaveCount(50);
    await expect(page.locator('#jobs-page')).toHaveText('Page 1');
    await expect(page.locator('#jobs-previous')).toBeDisabled();
    const first = await page.locator('button[data-job-id]').evaluateAll(buttons => buttons.map(button => button.dataset.jobId));
    await page.locator('#jobs-next').click();
    await expect(page.locator('#jobs-page')).toHaveText('Page 2');
    await expect(page.locator('#jobs-status')).toBeFocused();
    await expect(page.locator('#jobs-list .job-entry')).toHaveCount(50);
    const second = await page.locator('button[data-job-id]').evaluateAll(buttons => buttons.map(button => button.dataset.jobId));
    expect(second.every(id => !first.includes(id))).toBe(true);
    await page.locator('#jobs-previous').click();
    await expect(page.locator('#jobs-page')).toHaveText('Page 1');
    expect(await page.locator('button[data-job-id]').evaluateAll(buttons => buttons.map(button => button.dataset.jobId))).toEqual(first);
    await expect(page.locator('#jobs-list')).not.toContainText(fixture.IDs.completed);
});

test('failed page navigation retains keyboard focus on the status', async ({ page }) => {
    await open(page, 'bulk');
    await load(page);
    await page.route(url => url.pathname === '/api/v1/jobs' && url.searchParams.has('after'), route => route.fulfill({
        status: 503, contentType: 'application/json', body: JSON.stringify({ error: { code: 'jobs_unavailable' } }),
    }));
    await page.locator('#jobs-next').click();
    await expect(page.locator('#jobs-status')).toHaveAttribute('data-state', 'jobs_unavailable');
    await expect(page.locator('#jobs-status')).toBeFocused();
});

test('mocked HTML-looking job metadata stays literal text without executing script', async ({ page }) => {
    await open(page);
    const tenant = '<img src=x onerror="globalThis.tongJobInjection=true">';
    const key = '<svg onload="globalThis.tongJobInjection=true"></svg>';
    const job = {
        id: fixture.IDs.completed,
        tenant_id: tenant,
        requested_by: 'synthetic-requester',
        kind: 'validation.noop',
        state: 'completed',
        idempotency_key: key,
        created_at: '2026-10-28T01:00:00Z',
        updated_at: '2026-10-28T01:01:00Z',
    };
    await page.route(url => url.pathname === '/api/v1/jobs' && url.searchParams.has('limit'), route => route.fulfill({
        status: 200, contentType: 'application/json', body: JSON.stringify({ jobs: [job], next_after: "" }),
    }));
    await page.route(`**/api/v1/jobs/${job.id}`, route => route.fulfill({
        status: 200, contentType: 'application/json', body: JSON.stringify(job),
    }));
    await load(page);
    await expect(page.locator('#jobs-list .job-entry')).toHaveCount(1);
    await page.locator(`button[data-job-id="${job.id}"]`).click();
    await expect(page.locator('#job-detail-status')).toHaveAttribute('data-state', 'available');
    const fields = page.locator('#job-detail-fields');
    await expect(fields.locator('dd').filter({ hasText: tenant })).toHaveText(tenant);
    await expect(fields.locator('dd').filter({ hasText: key })).toHaveText(key);
    await expect(page.locator('#jobs-panel img, #jobs-panel svg, #jobs-panel script')).toHaveCount(0);
    expect(await page.evaluate(() => globalThis.tongJobInjection)).toBeUndefined();
});

test('overlapping refresh keeps actual current jobs after a held stale list is released', async ({ page }) => {
    await open(page);
    const staleID = Array.from({ length: 6 }, (_, index) => String(index).repeat(32))
        .find(id => !Object.values(fixture.IDs).includes(id));
    const staleJob = {
        id: staleID, tenant_id: 'tenant-a', requested_by: 'alice', kind: 'validation.noop', state: 'queued',
        idempotency_key: 'synthetic-stale-list-0001',
        created_at: '2026-10-28T01:00:00Z', updated_at: '2026-10-28T01:00:00Z',
    };
    let markHeld, releaseHeld, markReleased, heldRequest;
    const held = new Promise(resolve => { markHeld = resolve; });
    const release = new Promise(resolve => { releaseHeld = resolve; });
    const released = new Promise(resolve => { markReleased = resolve; });
    let first = true;
    await page.route(url => url.pathname === '/api/v1/jobs' && url.searchParams.has('limit'), async route => {
        if (!first) return route.continue(); // The second request reaches the real Go handler.
        first = false;
        heldRequest = route.request();
        markHeld();
        await release;
        try {
            await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ jobs: [staleJob], next_after: "" }) });
        } catch (error) {
            // An intentionally aborted first request may no longer accept a
            // response. Other fulfillment failures must still fail the test.
            if (!heldRequest.failure()) throw error;
        } finally {
            markReleased();
        }
    });
    try {
        await page.locator('#jobs-refresh').click();
        await held;
        await expect(page.locator('#jobs-status')).toHaveAttribute('data-state', 'loading');
        const canceled = page.waitForEvent('requestfailed', { predicate: request => request === heldRequest });
        await page.locator('#jobs-refresh').click();
        await canceled;
        await expect(page.locator('#jobs-status')).toHaveAttribute('data-state', 'available');
        await expect(page.locator('#jobs-list .job-entry')).toHaveCount(2);
        await expect(page.locator('#jobs-list')).toContainText(fixture.IDs.completed);
        await expect(page.locator('#jobs-list')).toContainText(fixture.IDs.queued);
        releaseHeld();
        await released;
        await expect(page.locator('#jobs-list .job-entry')).toHaveCount(2);
        await expect(page.locator('#jobs-list')).not.toContainText(staleID);
        await expect(page.locator('#jobs-list')).toContainText(fixture.IDs.completed);
        await expect(page.locator('#jobs-list')).toContainText(fixture.IDs.queued);
    } finally {
        releaseHeld();
    }
});
