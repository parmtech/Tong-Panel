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
        for (const key of ['Primary', 'Unconfigured', 'NoJobs', ...(fixture.Files ? ['Files'] : [])]) {
            const url = new URL(fixture[key]);
            if (url.protocol !== 'https:' || url.hostname !== '127.0.0.1') throw new Error('Browser fixture must stay on TLS loopback');
        }
    } catch (error) {
        await stopFixture();
        throw error;
    }
});

test.afterAll(stopFixture);

test('certificate snapshot uses real TLS session and stays read-only with truthful unavailable states', async ({ page }) => {
    const requests = [];
    page.on('request', request => {
        if (new URL(request.url()).pathname.endsWith('/preflight/acme-http01')) requests.push(request);
    });
    await open(page, 'auditor');
    await page.locator('#acme-preflight-id').fill('site-000');
    await page.locator('#acme-preflight-submit').focus();
    await page.keyboard.press('Enter');
    await expect(page.locator('#acme-preflight-status')).toHaveAttribute('data-state', 'available');
    await expect(page.locator('#acme-preflight-domains')).toHaveText('site-000.example.test');
    await expect(page.locator('#acme-preflight-status')).toContainText('Snapshot only');
    const axe = await new AxeBuilder({ page }).include('[aria-labelledby="acme-preflight-title"]').withTags(['wcag2a', 'wcag2aa']).analyze();
    expect(axe.violations.map(({ id }) => id)).toEqual([]);
    await page.locator('#acme-preflight-id').fill('site-001');
    await expect(page.locator('#acme-preflight-domains')).toBeHidden();
    await page.locator('#acme-preflight-submit').click();
    await expect(page.locator('#acme-preflight-status')).toContainText('Verify the site');
    await page.context().clearCookies();
    await page.locator('#acme-preflight-id').fill('site-000');
    await page.locator('#acme-preflight-submit').click();
    await expect(page.locator('#acme-preflight-status')).toContainText('secure browser session');
    await expect(page.locator('#acme-preflight-domains')).toBeHidden();
    expect(requests.length).toBe(3);
    expect(requests.every(request => request.method() === 'GET' && !request.headers().authorization)).toBe(true);
    expect(await page.evaluate(() => localStorage.length + sessionStorage.length)).toBe(0);
});

test('certificate snapshot clears and suppresses a held response after changing site', async ({ page }) => {
    await open(page);
    let release;
    const held = new Promise(resolve => { release = resolve; });
    let started;
    const ready = new Promise(resolve => { started = resolve; });
    await page.route('**/api/v1/sites/site-000/preflight/acme-http01', async route => {
        started();
        await held;
        try { await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({
            site_id: 'site-000', domains: ['old.example.test'], challenge: 'http-01', snapshot_only: true, execution_enabled: false,
        }) }); } catch {} // An aborted browser request may already be closed.
    });
    try {
        await page.locator('#acme-preflight-id').fill('site-000');
        await page.locator('#acme-preflight-submit').click();
        await ready;
        await page.locator('#acme-preflight-id').fill('site-001');
        release();
        await expect(page.locator('#acme-preflight-submit')).toBeEnabled();
        await expect(page.locator('#acme-preflight-domains')).toBeHidden();
        await expect(page.locator('#acme-preflight-domains')).toHaveText('');
        await expect(page.locator('#acme-preflight-status')).toHaveText('No certificate domain snapshot loaded.');
    } finally { release(); }
});

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
        if ([fixture.Primary, fixture.Unconfigured, fixture.NoJobs, fixture.Files].includes(origin)) return route.continue();
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

async function loadSites(page) {
    await page.locator('#sites-load').click();
    await expect(page.locator('#sites-status')).not.toHaveAttribute('data-state', 'loading');
}

test('real site catalog pages stay tenant scoped and selection fills preview', async ({ page }) => {
    await open(page, 'auditor');
    await loadSites(page);
    await expect(page.locator('#sites-status')).toHaveAttribute('data-state', 'available');
    await expect(page.locator('#sites-list li')).toHaveCount(50);
    await expect(page.locator('#sites-list')).toContainText('site-000.example.test');
    await expect(page.locator('#sites-list')).not.toContainText('other.example.test');
    await expect(page.locator('#sites-previous')).toBeDisabled();
    await page.locator('#sites-next').click();
    await expect(page.locator('#sites-page')).toHaveText('Page 2');
    await expect(page.locator('#sites-list li')).toHaveCount(11);
    await expect(page.locator('#sites-list')).toContainText('site-060.example.test');
    await page.locator('#sites-previous').click();
    await expect(page.locator('#sites-page')).toHaveText('Page 1');
    await page.getByRole('button', { name: 'Use site-000 for preview' }).click();
    await expect(page.locator('#site-preview-id')).toHaveValue('site-000');
    await expect(page.locator('#site-preview-id')).toBeFocused();
    await page.context().clearCookies();
    await page.context().addCookies([{ name: '__Host-tong_session', value: fixture.Tokens.carol, url: fixture.Primary, secure: true, httpOnly: true, sameSite: 'Strict' }]);
    await loadSites(page);
    await expect(page.locator('#sites-list li')).toHaveCount(1);
    await expect(page.locator('#sites-list')).toContainText('other.example.test');
    await expect(page.locator('#sites-list')).not.toContainText('site-000.example.test');
});

test('site details use actual catalog and discard stale or unauthorized responses', async ({ page }) => {
    await open(page, 'alice');
    await loadSites(page);
    await page.getByRole('button', { name: 'View details for site-000' }).click();
    await expect(page.locator('#site-details-title')).toHaveText('Site details: site-000');
    await expect(page.locator('#site-details-fields')).toContainText('sites/site-000/public');
    await expect(page.locator('#site-details-domains')).toContainText('site-000.example.test');
    await expect(page.locator('#site-details-status')).toContainText('does not confirm');
    await page.locator('#site-details-close').click();
    await expect(page.locator('#site-details')).toBeHidden();
    await expect(page.locator('#site-details-fields')).toBeEmpty();

    // Hold one real-handler request, then switch to another site; any late
    // response must not overwrite the newly selected site.
    let markHeld, releaseHeld, markFinished;
    const held = new Promise(resolve => { markHeld = resolve; });
    const release = new Promise(resolve => { releaseHeld = resolve; });
    const finished = new Promise(resolve => { markFinished = resolve; });
    await page.route('**/api/v1/sites/site-000', async route => {
        markHeld();
        await release;
        try { await route.continue(); } catch { /* aborted request */ }
        markFinished();
    });
    await page.getByRole('button', { name: 'View details for site-000' }).click();
    await held;
    await page.getByRole('button', { name: 'View details for site-001' }).click();
    await expect(page.locator('#site-details-title')).toHaveText('Site details: site-001');
    releaseHeld();
    await finished;
    await expect(page.locator('#site-details-title')).toHaveText('Site details: site-001');
    await page.unroute('**/api/v1/sites/site-000');

    let markRefreshHeld, releaseRefreshHeld, markRefreshFinished;
    const refreshHeld = new Promise(resolve => { markRefreshHeld = resolve; });
    const refreshRelease = new Promise(resolve => { releaseRefreshHeld = resolve; });
    const refreshFinished = new Promise(resolve => { markRefreshFinished = resolve; });
    await page.route('**/api/v1/sites/site-000', async route => {
        markRefreshHeld();
        await refreshRelease;
        try { await route.continue(); } catch { /* aborted request */ }
        markRefreshFinished();
    });
    await page.getByRole('button', { name: 'View details for site-000' }).click();
    await refreshHeld;
    await loadSites(page);
    releaseRefreshHeld();
    await refreshFinished;
    await expect(page.locator('#site-details')).toBeHidden();
    await expect(page.locator('#site-details-fields')).toBeEmpty();
    await page.unroute('**/api/v1/sites/site-000');

    // Only denial responses are intercepted; catalog and successful detail
    // requests above used the actual handler and SQLite session.
    for (const status of [401, 403]) {
        await page.route('**/api/v1/sites/site-000', route => route.fulfill({ status, body: '' }));
        await page.getByRole('button', { name: 'View details for site-000' }).click();
        await expect(page.locator('#site-details-status')).toContainText(status === 401 ? 'session' : 'cannot view');
        await expect(page.locator('#site-details-fields')).toBeEmpty();
        await expect(page.locator('#sites-list li')).toHaveCount(0);
        await page.unroute('**/api/v1/sites/site-000');
        if (status === 401) await loadSites(page);
    }
});

test('real site details follow tenant and session changes', async ({ page }) => {
    await open(page, 'alice');
    await loadSites(page);
    await page.getByRole('button', { name: 'View details for site-000' }).click();
    await expect(page.locator('#site-details-fields')).toContainText('sites/site-000/public');
    await page.context().clearCookies();
    await page.context().addCookies([{ name: '__Host-tong_session', value: fixture.Tokens.carol, url: fixture.Primary, secure: true, httpOnly: true, sameSite: 'Strict' }]);
    await loadSites(page);
    await expect(page.locator('#site-details')).toBeHidden();
    await expect(page.locator('#sites-list')).toContainText('other.example.test');
    await expect(page.locator('#sites-list')).not.toContainText('site-000.example.test');
    await page.getByRole('button', { name: 'View details for other-site' }).click();
    await expect(page.locator('#site-details-title')).toHaveText('Site details: other-site');
    await expect(page.locator('#site-details-fields')).toContainText('sites/other-site/public');
    const crossTenantStatus = await page.evaluate(async () => (await fetch('/api/v1/sites/site-000', { credentials: 'same-origin' })).status);
    expect(crossTenantStatus).toBe(404);
    await page.context().clearCookies();
    await page.getByRole('button', { name: 'View details for other-site' }).click();
    await expect(page.locator('#site-details-status')).toContainText('session');
    await expect(page.locator('#site-details-fields')).toBeEmpty();
    await expect(page.locator('#sites-list li')).toHaveCount(0);
});

test('site database catalog follows real tenant sessions and discards stale reads', async ({ page }) => {
    await open(page, 'alice');
    await loadSites(page);
    const select = async name => {
        await page.getByRole('button', { name: `View details for ${name}` }).click();
        await expect(page.locator('#site-details-title')).toHaveText(`Site details: ${name}`);
        await expect(page.locator('#site-databases-load')).toBeEnabled();
    };
    await select('site-000');
    await page.locator('#site-databases-load').click();
    await expect(page.locator('#site-databases-list')).toContainText('db-a');
    await expect(page.locator('#site-databases-list')).not.toContainText('db-b');
    await expect(page.locator('#site-databases-status')).toContainText('provisioning is not confirmed');
    let markHeld, releaseHeld, markFinished;
    const held = new Promise(resolve => { markHeld = resolve; });
    const release = new Promise(resolve => { releaseHeld = resolve; });
    const finished = new Promise(resolve => { markFinished = resolve; });
    await page.route('**/api/v1/sites/site-000/databases?**', async route => {
        markHeld(); await release;
        try { await route.continue(); } catch { /* aborted request */ }
        markFinished();
    });
    await page.locator('#site-databases-load').click();
    await held;
    await page.locator('#site-details-close').click();
    releaseHeld(); await finished;
    await expect(page.locator('#site-details')).toBeHidden();
    await expect(page.locator('#site-databases-list')).toBeEmpty();
    await page.unroute('**/api/v1/sites/site-000/databases?**');
    await page.context().clearCookies();
    await page.context().addCookies([{ name: '__Host-tong_session', value: fixture.Tokens.bob, url: fixture.Primary, secure: true, httpOnly: true, sameSite: 'Strict' }]);
    await loadSites(page); await select('site-000');
    await page.locator('#site-databases-load').click();
    await expect(page.locator('#site-databases-list')).toContainText('db-a');
    await page.context().clearCookies();
    await page.context().addCookies([{ name: '__Host-tong_session', value: fixture.Tokens.carol, url: fixture.Primary, secure: true, httpOnly: true, sameSite: 'Strict' }]);
    await loadSites(page); await select('other-site');
    await page.locator('#site-databases-load').click();
    await expect(page.locator('#site-databases-list')).toContainText('db-b');
    await expect(page.locator('#site-databases-list')).not.toContainText('db-a');
    const cross = await page.evaluate(async () => (await fetch('/api/v1/sites/site-000/databases', { credentials: 'same-origin' })).status);
    expect(cross).toBe(404);
    await page.context().clearCookies();
    await page.locator('#site-databases-load').click();
    await expect(page.locator('#sites-status')).toHaveAttribute('data-state', 'unavailable');
    await expect(page.locator('#sites-list li')).toHaveCount(0);
    await expect(page.locator('#site-details')).toBeHidden();
    await expect(page.locator('#site-databases-list')).toBeEmpty();
});

test('real site database pages remain tenant scoped across sessions', async ({ page }) => {
    const requests = [];
    page.on('request', request => {
        const url = new URL(request.url());
        if (url.pathname === '/api/v1/sites/site-000/databases') requests.push({ page: url.searchParams.get('page'), limit: url.searchParams.get('limit'), after: url.searchParams.get('after') });
    });
    await open(page, 'alice');
    await loadSites(page);
    await page.getByRole('button', { name: 'View details for site-000' }).click();
    await expect(page.locator('#site-databases-load')).toBeEnabled();
    await page.locator('#site-databases-load').click();
    await expect(page.locator('#site-databases-list li')).toHaveCount(50);
    await expect(page.locator('#site-databases-list')).toContainText('db-a');
    await expect(page.locator('#site-databases-list')).not.toContainText('db-b');
    await expect(page.locator('#site-databases-previous')).toBeDisabled();
    const first = await page.locator('#site-databases-list li').allTextContents();
    await page.locator('#site-databases-next').click();
    await expect(page.locator('#site-databases-page')).toHaveText('Page 2');
    await expect(page.locator('#site-databases-list li')).toHaveCount(6);
    await expect(page.locator('#site-databases-list')).toContainText('db-p054');
    await expect(page.locator('#site-databases-next')).toBeDisabled();
    await page.locator('#site-databases-previous').click();
    await expect(page.locator('#site-databases-page')).toHaveText('Page 1');
    expect(await page.locator('#site-databases-list li').allTextContents()).toEqual(first);
    expect(requests).toEqual([
        { page: '1', limit: '50', after: null },
        { page: '1', limit: '50', after: 'db-p048' },
        { page: '1', limit: '50', after: null },
    ]);
    await page.context().clearCookies();
    await page.context().addCookies([{ name: '__Host-tong_session', value: fixture.Tokens.carol, url: fixture.Primary, secure: true, httpOnly: true, sameSite: 'Strict' }]);
    await loadSites(page);
    await page.getByRole('button', { name: 'View details for other-site' }).click();
    await expect(page.locator('#site-databases-load')).toBeEnabled();
    await page.locator('#site-databases-load').click();
    await expect(page.locator('#site-databases-list li')).toHaveCount(1);
    await expect(page.locator('#site-databases-list')).toContainText('db-b');
    await expect(page.locator('#site-databases-list')).not.toContainText('db-a');
    await page.context().clearCookies();
    await page.locator('#site-databases-load').click();
    await expect(page.locator('#sites-status')).toHaveAttribute('data-state', 'unavailable');
    await expect(page.locator('#site-databases-list')).toBeEmpty();
    await expect(page.locator('#site-details')).toBeHidden();
});

test('site catalog empty and absent authentication states do not expose stale sites', async ({ page }) => {
    await open(page, 'siteempty');
    await loadSites(page);
    await expect(page.locator('#sites-status')).toHaveAttribute('data-state', 'empty');
    await expect(page.locator('#sites-list li')).toHaveCount(0);
    await page.context().clearCookies();
    await loadSites(page);
    await expect(page.locator('#sites-status')).toHaveAttribute('data-state', 'unavailable');
    await expect(page.locator('#sites-list li')).toHaveCount(0);
});

test('selecting a different site suppresses a held stale preview response', async ({ page }) => {
    await open(page);
    await loadSites(page);
    let markHeld, releaseHeld, markReleased;
    const held = new Promise(resolve => { markHeld = resolve; });
    const release = new Promise(resolve => { releaseHeld = resolve; });
    const released = new Promise(resolve => { markReleased = resolve; });
    await page.route('**/api/v1/sites/site-000/preview/nginx', async route => {
        markHeld();
        await release;
        try {
            await route.fulfill({ status: 200, contentType: 'text/plain', body: 'synthetic old site configuration' });
        } finally { markReleased(); }
    });
    try {
        await page.getByRole('button', { name: 'Use site-000 for preview' }).click();
        await page.locator('#site-preview-submit').click();
        await held;
        await expect(page.locator('#site-preview-status')).toHaveAttribute('data-state', 'loading');
        await page.getByRole('button', { name: 'Use site-001 for preview' }).click();
        await expect(page.locator('#site-preview-id')).toHaveValue('site-001');
        releaseHeld();
        await released;
        await expect(page.locator('#site-preview-submit')).toBeEnabled();
        await expect(page.locator('#site-preview-output')).toBeHidden();
        await expect(page.locator('#site-preview-output')).not.toContainText('synthetic old site configuration');
        await expect(page.locator('#site-preview-status')).toHaveText('No preview loaded.');
    } finally { releaseHeld(); }
});

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

test('manual refresh announces observed same-user job state changes and supports dismissal', async ({ page }) => {
    await open(page);
    const job = {
        id: 'f'.repeat(32), tenant_id: 'tenant-a', requested_by: 'alice', kind: 'validation.noop', state: 'queued',
        idempotency_key: 'synthetic-notification-0001', created_at: '2026-10-28T01:00:00Z', updated_at: '2026-10-28T01:00:00Z',
    };
    let state = 'queued';
    await page.route(url => url.pathname === '/api/v1/jobs' && url.searchParams.has('limit'), route => route.fulfill({
        status: 200, contentType: 'application/json', body: JSON.stringify({ jobs: [{ ...job, state }], next_after: '' }),
    }));
    await load(page);
    await expect(page.locator('#jobs-notifications li')).toHaveCount(0);
    state = 'running';
    await load(page);
    await expect(page.locator('#jobs-notifications li')).toHaveCount(1);
    await expect(page.locator('#jobs-notifications')).toContainText('changed from queued to running');
    const dismiss = page.getByRole('button', { name: `Dismiss update for job ${job.id}` });
    await dismiss.focus();
    await page.keyboard.press('Enter');
    await expect(page.locator('#jobs-notifications li')).toHaveCount(0);
    await load(page);
    await expect(page.locator('#jobs-notifications li')).toHaveCount(0);
    state = 'completed';
    await load(page);
    await expect(page.locator('#jobs-notifications li')).toHaveCount(1);
    job.requested_by = 'bob';
    state = 'queued';
    await load(page);
    await expect(page.locator('#jobs-notifications li')).toHaveCount(0);
    state = 'failed';
    await load(page);
    await expect(page.locator('#jobs-notifications li')).toHaveCount(1);
    await page.setViewportSize({ width: 390, height: 844 });
    expect(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth)).toBe(true);
    const axe = await new AxeBuilder({ page }).include('#jobs-panel').withTags(['wcag2a', 'wcag2aa']).analyze();
    expect(axe.violations.map(({ id }) => id)).toEqual([]);
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

test('real Files handler pages and previews stay scoped to the authenticated tenant', async ({ page }) => {
    test.skip(process.platform !== 'linux', 'real pinned tenant roots are Linux-only');
    expect(fixture.Files).toBeTruthy();
    await open(page, 'alice', fixture.Files);
    await page.locator('#files-refresh').click();
    await expect(page.locator('#files-status')).toHaveAttribute('data-state', 'available');
    await expect(page.locator('#files-list .file-entry')).toHaveCount(200);
    await expect(page.locator('#files-list')).toContainText('a-000.txt');
    await expect(page.locator('#files-list')).not.toContainText('b-only.txt');
    await page.getByRole('button', { name: 'Preview text a-000.txt' }).click();
    await expect(page.locator('#files-preview-content')).toHaveText('tenant A preview');
    await page.locator('#files-next').click();
    await expect(page.locator('#files-list .file-entry')).toHaveCount(1);
    await expect(page.locator('#files-list')).toContainText('a-200.txt');
    await expect(page.locator('#files-preview')).toBeHidden();
    await page.context().clearCookies();
    await page.context().addCookies([{ name: '__Host-tong_session', value: fixture.Tokens.carol, url: fixture.Files, secure: true, httpOnly: true, sameSite: 'Strict' }]);
    await page.locator('#files-refresh').click();
    await expect(page.locator('#files-list')).toContainText('b-only.txt');
    await expect(page.locator('#files-list')).not.toContainText('a-200.txt');
    await page.getByRole('button', { name: 'Preview text b-only.txt' }).click();
    await expect(page.locator('#files-preview-content')).toHaveText('tenant B preview');
    await page.context().clearCookies();
    await page.locator('#files-refresh').click();
    await expect(page.locator('#files-status')).toHaveAttribute('data-state', 'unavailable');
    await expect(page.locator('#files-preview')).toBeHidden();
    await expect(page.locator('#files-list')).not.toContainText('b-only.txt');
});

test('Files prefix filter resets cursors and authorization failures clear rows', async ({ page }) => {
    // These controlled responses exercise browser interactions only.
    const requests = [];
    let listStatus = 200;
    let contentStatus = 200;
    await page.route('**/api/v1/files?**', route => {
        const url = new URL(route.request().url());
        const prefix = url.searchParams.get('prefix');
        const after = url.searchParams.get('after');
        requests.push({ prefix, after, page: url.searchParams.get('page') });
        if (listStatus !== 200) return route.fulfill({ status: listStatus, body: '' });
        const entry = name => ({ name, kind: 'file', size: 3 });
        const body = prefix === 'a' ? after ? { entries: [entry('a-two')], next_after: '' } : { entries: [entry('a-one')], next_after: 'a-one' }
            : prefix === 'b' ? { entries: [entry('b-one')], next_after: '' }
                : { entries: [entry('all-one')], next_after: '' };
        return route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify(body) });
    });
    await page.route('**/api/v1/files/content?**', route => route.fulfill({ status: contentStatus, contentType: 'application/octet-stream', body: 'abc' }));
    await open(page);
    await page.locator('#files-refresh').click();
    await expect(page.locator('#files-list')).toContainText('all-one');
    await page.locator('#files-prefix').fill('a');
    await page.getByRole('button', { name: 'Filter files' }).click();
    await expect(page.locator('#files-list')).toContainText('a-one');
    await page.locator('#files-next').click();
    await expect(page.locator('#files-list')).toContainText('a-two');
    await page.locator('#files-prefix').fill('b');
    await page.getByRole('button', { name: 'Filter files' }).click();
    await expect(page.locator('#files-list')).toContainText('b-one');
    await expect(page.locator('#files-previous')).toBeDisabled();
    expect(requests).toEqual([
        { prefix: null, after: '', page: '1' }, { prefix: 'a', after: '', page: '1' },
        { prefix: 'a', after: 'a-one', page: '1' }, { prefix: 'b', after: '', page: '1' },
    ]);
    await page.getByRole('button', { name: 'Preview text b-one' }).click();
    await expect(page.locator('#files-preview-content')).toHaveText('abc');
    listStatus = 401;
    await page.locator('#files-refresh').click();
    await expect(page.locator('#files-status')).toHaveAttribute('data-state', 'unavailable');
    await expect(page.locator('#files-list .file-entry')).toHaveCount(0);
    await expect(page.locator('#files-preview')).toBeHidden();
    await expect(page.locator('#files-next')).toBeDisabled();
    await expect(page.locator('#files-previous')).toBeDisabled();
    listStatus = 200;
    await page.locator('#files-refresh').click();
    await expect(page.locator('#files-list')).toContainText('b-one');
    contentStatus = 403;
    await page.getByRole('button', { name: 'Preview text b-one' }).click();
    await expect(page.locator('#files-status')).toHaveAttribute('data-state', 'unavailable');
    await expect(page.locator('#files-list .file-entry')).toHaveCount(0);
    await expect(page.locator('#files-preview')).toBeHidden();
    await expect(page.locator('#files-next')).toBeDisabled();
});

test('Files text preview stays inert, bounded, and clears on navigation', async ({ page }) => {
    // Files responses are controlled here; the Linux handler is not exercised.
    let releaseSlow, markSlow, markFinished;
    const held = new Promise(resolve => { releaseSlow = resolve; });
    const started = new Promise(resolve => { markSlow = resolve; });
    const finished = new Promise(resolve => { markFinished = resolve; });
    await page.route('**/api/v1/files?**', route => route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({
        entries: ['plain.txt', 'markup.txt', 'slow.txt', 'denied.txt', 'large.txt', 'oversized.txt'].map(name => ({ name, kind: 'file', size: name === 'large.txt' ? 262145 : 10 })),
        next_after: '',
    }) }));
    await page.route('**/api/v1/files/content?**', async route => {
        const name = new URL(route.request().url()).searchParams.get('path');
        if (name === 'slow.txt') { markSlow(); await held; }
        if (name === 'denied.txt') return route.fulfill({ status: 403, body: '' });
        const body = name === 'markup.txt' ? '<img src=x onerror="window.previewInjected=true">' : name === 'slow.txt' ? 'stale secret' : name === 'oversized.txt' ? 'x'.repeat(262145) : 'ordinary text';
        try { await route.fulfill({ status: 200, contentType: 'application/octet-stream', body }); } catch { /* request may have closed */ }
        if (name === 'slow.txt') markFinished();
    });
    await open(page);
    await page.locator('#files-refresh').click();
    await expect(page.getByRole('button', { name: 'Preview text plain.txt' })).toBeVisible();
    await page.getByRole('button', { name: 'Preview text plain.txt' }).click();
    await expect(page.locator('#files-preview-content')).toHaveText('ordinary text');
    await page.locator('#files-preview-close').click();
    await expect(page.locator('#files-preview')).toBeHidden();
    await expect(page.locator('#files-preview-content')).toBeEmpty();
    await page.getByRole('button', { name: 'Preview text markup.txt' }).click();
    await expect(page.locator('#files-preview-content')).toHaveText('<img src=x onerror="window.previewInjected=true">');
    await expect(page.locator('#files-preview-content img')).toHaveCount(0);
    expect(await page.evaluate(() => window.previewInjected)).toBeUndefined();
    await expect(page.getByRole('button', { name: 'Preview text large.txt' })).toBeDisabled();
    await page.getByRole('button', { name: 'Preview text oversized.txt' }).click();
    await expect(page.locator('#files-status')).toHaveAttribute('data-state', 'unavailable');
    await expect(page.locator('#files-preview')).toBeHidden();
    await page.getByRole('button', { name: 'Preview text denied.txt' }).click();
    await expect(page.locator('#files-status')).toHaveAttribute('data-state', 'unavailable');
    await expect(page.locator('#files-preview')).toBeHidden();
    await expect(page.locator('#files-list .file-entry')).toHaveCount(0);
    await page.locator('#files-refresh').click();
    await expect(page.locator('#files-status')).toHaveAttribute('data-state', 'available');
    await page.getByRole('button', { name: 'Preview text slow.txt' }).click();
    await started;
    await page.locator('#files-refresh').click();
    await expect(page.locator('#files-status')).toHaveAttribute('data-state', 'available');
    releaseSlow();
    await finished;
    await expect(page.locator('#files-preview')).toBeHidden();
    await expect(page.locator('#files-preview-content')).toBeEmpty();
});

test('Files folder transition rejects stale row actions', async ({ page }) => {
    // Controlled responses test browser behavior, not the filesystem backend.
    let markFolder, releaseFolder;
    const folderStarted = new Promise(resolve => { markFolder = resolve; });
    const folderHeld = new Promise(resolve => { releaseFolder = resolve; });
    const contentRequests = [];
    await page.route('**/api/v1/files?**', async route => {
        const path = new URL(route.request().url()).searchParams.get('path');
        if (path === 'folder') { markFolder(); await folderHeld; }
        const entries = path === 'folder' ? [{ name: 'new.txt', kind: 'file', size: 1 }] : [
            { name: 'folder', kind: 'directory', size: 0 },
            { name: 'old.txt', kind: 'file', size: 1 },
        ];
        await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ entries, next_after: '' }) });
    });
    await page.route('**/api/v1/files/content?**', route => {
        contentRequests.push(new URL(route.request().url()).searchParams.get('path'));
        return route.fulfill({ status: 200, contentType: 'application/octet-stream', body: 'unexpected' });
    });
    await open(page);
    await page.locator('#files-refresh').click();
    await expect(page.getByRole('button', { name: 'Preview text old.txt' })).toBeVisible();
    await page.evaluate(() => { window.staleFileButton = document.querySelector('[aria-label="Preview text old.txt"]'); });
    await page.getByRole('button', { name: 'Open folder folder' }).click();
    await folderStarted;
    await expect(page.locator('#files-list')).not.toContainText('old.txt');
    await page.evaluate(() => window.staleFileButton.click());
    expect(contentRequests).toEqual([]);
    releaseFolder();
    await expect(page.locator('#files-list')).toContainText('new.txt');
    await expect(page.locator('#files-preview')).toBeHidden();
    expect(contentRequests).toEqual([]);
});

test('Files page navigation and folder resets use the expected cursors', async ({ page }) => {
    // The Files endpoint is intercepted here: this checks browser behavior,
    // while Linux backend and tenant isolation are tested separately.
    const requests = [];
    let failNext = false;
    await page.route('**/api/v1/files?**', async route => {
        const url = new URL(route.request().url());
        const path = url.searchParams.get('path');
        const after = url.searchParams.get('after');
        requests.push({ path, after, page: url.searchParams.get('page') });
        if (failNext && path === '' && after === 'b-folder') {
            failNext = false;
            return route.fulfill({ status: 503, body: '' });
        }
        const entry = (name, kind) => ({ name, kind, size: 0 });
        const body = path === 'b-folder'
            ? { entries: [entry('inside.txt', 'file')], next_after: '' }
            : after === 'b-folder'
                ? { entries: [entry('c.txt', 'file')], next_after: '' }
                : { entries: [entry('a.txt', 'file'), entry('b-folder', 'directory')], next_after: 'b-folder' };
        await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify(body) });
    });
    await open(page);
    await page.locator('#files-refresh').click();
    await expect(page.locator('#files-status')).toContainText('Page 1:');
    await expect(page.locator('#files-list .file-entry-name')).toHaveText(['a.txt', '📁 b-folder']);
    await expect(page.locator('#files-previous')).toBeDisabled();
    await expect(page.locator('#files-next')).toBeEnabled();
    await page.locator('#files-next').click();
    await expect(page.locator('#files-status')).toContainText('Page 2:');
    await expect(page.locator('#files-list .file-entry-name')).toHaveText(['c.txt']);
    await expect(page.locator('#files-next')).toBeDisabled();
    await page.locator('#files-previous').click();
    await expect(page.locator('#files-list .file-entry-name')).toHaveText(['a.txt', '📁 b-folder']);
    await expect(page.locator('#files-status')).toContainText('Page 1:');
    failNext = true;
    await page.locator('#files-next').click();
    await expect(page.locator('#files-status')).toHaveAttribute('data-state', 'unavailable');
    await expect(page.locator('#files-list .file-entry-name')).toHaveText(['a.txt', '📁 b-folder']);
    await expect(page.locator('#files-next')).toBeEnabled();
    await page.getByRole('button', { name: 'Open folder b-folder' }).click();
    await expect(page.locator('#files-list .file-entry-name')).toHaveText(['inside.txt']);
    await expect(page.locator('#files-previous')).toBeDisabled();
    await page.locator('#files-refresh').click();
    await expect(page.locator('#files-status')).toContainText('Page 1:');
    expect(requests).toEqual([
        { path: '', after: '', page: '1' },
        { path: '', after: 'b-folder', page: '1' },
        { path: '', after: '', page: '1' },
        { path: '', after: 'b-folder', page: '1' },
        { path: 'b-folder', after: '', page: '1' },
        { path: 'b-folder', after: '', page: '1' },
    ]);
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

test('home search matches the documented feature synonyms', async ({ page }) => {
    await open(page);
    const search = page.getByRole('searchbox', { name: 'Search features' });
    const tiles = page.locator('#tiles .tile');
    const count = page.locator('#count');
    const synonyms = [
        ['make email', 'Email'],
        ['new WordPress', 'Applications'],
        ['Node app', 'Applications'],
        ['PHP version', 'Websites'],
        ['SSL redirect', 'SSL & HTTPS'],
    ];

    await expect(tiles).toHaveCount(12);
    await expect(count).toHaveText('12 modules planned');
    for (const [query, title] of synonyms) {
        await search.fill(query);
        await expect(tiles).toHaveCount(1);
        await expect(tiles.locator('h3')).toHaveText(title);
        await expect(count).toHaveText('1 module planned');
    }
});

test('home search handles literal text, empty results, clear, and keyboard access', async ({ page }) => {
    await open(page);
    const search = page.getByRole('searchbox', { name: 'Search features' });
    const tiles = page.locator('#tiles .tile');
    const count = page.locator('#count');

    let foundSearchByTab = false;
    for (let tab = 0; tab < 20; tab++) {
        await page.keyboard.press('Tab');
        if (await page.evaluate(() => document.activeElement?.id === 'search')) {
            foundSearchByTab = true;
            break;
        }
    }
    expect(foundSearchByTab).toBe(true);
    await expect(search).toBeFocused();

    await page.keyboard.type('  pHp  ');
    await expect(tiles).toHaveCount(1);
    await expect(tiles.locator('h3')).toHaveText('Websites');
    await expect(count).toHaveText('1 module planned');

    await search.fill('  rEdIrEcTs  ');
    await expect(tiles).toHaveCount(1);
    await expect(tiles.locator('h3')).toHaveText('Domains');
    await expect(count).toHaveText('1 module planned');

    await search.fill('feature with no matching module');
    await expect(tiles).toHaveCount(0);
    await expect(count).toHaveText('0 modules planned');

    await search.fill('');
    await expect(tiles).toHaveCount(12);
    await expect(count).toHaveText('12 modules planned');
    await expect(page.locator('#tiles .tile a, #tiles .tile button, #tiles .tile input')).toHaveCount(0);
    await expect(page.locator('#tiles .planned')).toHaveCount(12);
});

test('home launcher grid fits representative desktop, tablet, and phone widths', async ({ page }) => {
    await open(page);
    for (const [width, columns] of [[1440, 4], [1024, 3], [768, 2], [390, 2], [320, 2]]) {
        await page.setViewportSize({ width, height: 900 });
        const layout = await page.evaluate(() => ({
            viewport: window.innerWidth,
            document: document.documentElement.scrollWidth,
            columns: getComputedStyle(document.querySelector('#tiles')).gridTemplateColumns.split(' ').length,
            searchRight: document.querySelector('#search').getBoundingClientRect().right,
        }));
        expect(layout.document, `document overflow at ${width}px`).toBeLessThanOrEqual(layout.viewport);
        expect(layout.columns, `grid columns at ${width}px`).toBe(columns);
        expect(layout.searchRight, `search field overflow at ${width}px`).toBeLessThanOrEqual(width);
    }
});
