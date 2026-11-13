import assert from 'node:assert/strict';
import test from 'node:test';
import { loadSitesPage } from '../../cmd/tongd/ui/sites-list.mjs';

const response = (data, status = 200) => new Response(JSON.stringify(data), { status, headers: { 'Content-Type': 'application/json' } });
const item = n => ({ site_id: `site${String(n).padStart(3, '0')}`, primary_domain: `site${n}.example.com`, ownership_state: 'pending' });

test('loads bounded ordered pages through same-origin requests', async () => {
    const items = Array.from({ length: 50 }, (_, n) => item(n));
    const first = await loadSitesPage('', async (path, options) => {
        assert.equal(path, '/api/v1/sites?limit=50');
        assert.equal(options.credentials, 'same-origin'); assert.equal(options.cache, 'no-store');
        return response({ sites: items, next_after: items[49].site_id });
    });
    assert.equal(first.sites.length, 50);
    const second = await loadSitesPage(first.next_after, async path => {
        assert.equal(path, '/api/v1/sites?limit=50&after=site049');
        return response({ sites: [item(50)], next_after: '' });
    });
    assert.equal(second.sites[0].site_id, 'site050');
});

test('rejects malformed pages and unsafe cursors without reflecting server data', async () => {
    await assert.rejects(loadSitesPage('../bad', () => assert.fail('unsafe request')));
    for (const body of [
        { sites: [item(1), item(0)], next_after: '' },
        { sites: [item(0)], next_after: item(0).site_id },
        { sites: [{ ...item(0), primary_domain: '<script>' }], next_after: '' },
        { sites: [{ ...item(0), ownership_state: 'private-canary' }], next_after: '' },
    ]) await assert.rejects(loadSitesPage('', async () => response(body)), error => !error.message.includes('private-canary'));
    await assert.rejects(loadSitesPage('', async () => response({}, 401)), /secure browser session/);
    await assert.rejects(loadSitesPage('', async () => response({}, 403)), /cannot view/);
    await assert.rejects(loadSitesPage('', async () => response({ sites: [], next_after: '' }, 206)));
    await assert.rejects(loadSitesPage('', async () => new Response('x'.repeat(65537), { headers: { 'Content-Type': 'application/json' } })));
});

test('cancellation settles even when a fetcher does not honor the signal', async () => {
    const controller = new AbortController();
    const result = loadSitesPage('', () => new Promise(() => {}), controller.signal);
    controller.abort();
    await assert.rejects(result);
});

test('site prefix is literal and constrained on every page',async()=>{
    let requested;
    const page=await loadSitesPage('',async path=>{requested=path;return response({sites:[{site_id:'app_01',primary_domain:'app.example.test',ownership_state:'pending'}],next_after:''})},undefined,'app_');
    assert.equal(requested,'/api/v1/sites?limit=50&prefix=app_');assert.equal(page.sites[0].site_id,'app_01');
    await assert.rejects(loadSitesPage('',async()=>response({sites:[{site_id:'appX01',primary_domain:'app.example.test',ownership_state:'pending'}],next_after:''}),undefined,'app_'));
    await assert.rejects(loadSitesPage('',()=>assert.fail('unsafe prefix'),undefined,'app%'));
    await assert.rejects(loadSitesPage('',()=>assert.fail('oversized prefix'),undefined,'x'.repeat(64)));
});
