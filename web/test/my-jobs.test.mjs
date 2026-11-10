import assert from 'node:assert/strict';
import test from 'node:test';
import { loadMyJob, loadMyJobs, loadMyJobsPage, MyJobsError } from '../../cmd/tongd/ui/my-jobs.mjs';

const fixture = () => ({ id: 'a'.repeat(32), tenant_id: 'tenant-a', requested_by: 'user-a', kind: 'validation.noop',
    state: 'queued', idempotency_key: 'request-fixture-001', created_at: '2026-11-10T01:02:03.123456789Z', updated_at: '2026-11-10T01:02:03Z' });
const response = (data, status = 200) => new Response(JSON.stringify(data), { status, headers: { 'Content-Type': 'application/json; charset=utf-8' } });
const rejects = (promise, code) => assert.rejects(promise, error => error instanceof MyJobsError && error.code === code && !error.message.includes('private-canary'));

test('requests bounded, ordered pages and validates continuation boundaries', async () => {
    const items = Array.from({ length: 50 }, (_, i) => ({ ...fixture(), id: (i + 1).toString(16).padStart(32, '0') }));
    const first = await loadMyJobsPage('', async (path, options) => {
        assert.equal(path, '/api/v1/jobs?limit=50');
        assert.equal(options.credentials, 'same-origin');
        return response({ jobs: items, next_after: items[49].id });
    });
    assert.equal(first.jobs.length, 50);
    const next = { ...fixture(), id: (51).toString(16).padStart(32, '0') };
    assert.deepEqual(await loadMyJobsPage(first.next_after, async path => {
        assert.equal(path, `/api/v1/jobs?limit=50&after=${items[49].id}`);
        return response({ jobs: [next], next_after: '' });
    }), { jobs: [next], next_after: '' });
    for (const data of [{ jobs: null, next_after: '' }, { jobs: items, next_after: 'bad' },
        { jobs: items.slice(0, 1), next_after: items[0].id }, { jobs: [items[1], items[0]], next_after: '' },
        { jobs: [...items, next], next_after: '' }, { jobs: items }]) {
        await rejects(loadMyJobsPage('', async () => response(data)), 'invalid_response');
    }
    await rejects(loadMyJobsPage(items[0].id, async () => response({ jobs: [items[0]], next_after: '' })), 'invalid_response');
    await rejects(loadMyJobsPage('../bad', () => assert.fail('unsafe cursor issued')), 'invalid_response');
    await rejects(loadMyJobsPage('', async () => response({ error: { code: 'private-canary' } }, 404)), 'unavailable');
});

test('requests only fixed same-origin read-only paths and projects validated metadata', async () => {
    const job = fixture();
    const fetcher = async (path, options) => {
        assert.equal(options.method, 'GET');
        assert.equal(options.credentials, 'same-origin');
        assert.equal(options.cache, 'no-store');
        assert.equal(options.redirect, 'error');
        assert.deepEqual(options.headers, { Accept: 'application/json' });
        assert.ok(options.signal instanceof AbortSignal);
        return response(path === '/api/v1/jobs' ? { jobs: [{ ...job, unknown: '<script>private-canary</script>' }] } : job);
    };
    assert.deepEqual(await loadMyJobs(fetcher), [job]);
    assert.deepEqual(await loadMyJob(job.id, fetcher), job);
    for (const jobs of [null, []]) assert.deepEqual(await loadMyJobs(async () => response({ jobs })), []);
    const journal = fixture(); delete journal.requested_by;
    assert.deepEqual(await loadMyJobs(async () => response({ jobs: [journal] })), [journal]);
});

test('rejects unvalidated path input without making any request', async () => {
    for (const id of ['../admin', 'a'.repeat(32) + '?tenant=other', '<img src=x>', 'A'.repeat(32), 123, null]) {
        await rejects(loadMyJob(id, () => { assert.fail('unsafe request issued'); }), 'invalid_response');
    }
});

test('preserves valid multibyte journal keys using Go UTF-8 byte bounds', async () => {
    for (const key of ['éééé', '😀😀', 'é'.repeat(64)]) {
        const job = fixture();
        delete job.requested_by;
        job.idempotency_key = key;
        assert.deepEqual(await loadMyJobs(async () => response({ jobs: [job] })), [job]);
    }
    for (const key of ['ééé', 'é'.repeat(65)]) {
        await rejects(loadMyJobs(async () => response({ jobs: [{ ...fixture(), idempotency_key: key }] })), 'invalid_response');
    }
});

test('validates states, exact types, dates and detail identity', async () => {
    for (const patch of [{ id: '<script>private-canary</script>' }, { id: 123 }, { kind: '<img src=x>' }, { state: 'provisioned' },
        { created_at: '2026-02-30T01:02:03Z' }, { updated_at: '2026-11-10T25:02:03Z' }, { created_at: 'yesterday' },
        { tenant_id: '' }, { requested_by: '\nprivate-canary' }, { idempotency_key: null }]) {
        await rejects(loadMyJobs(async () => response({ jobs: [{ ...fixture(), ...patch }] })), 'invalid_response');
    }
    for (const state of ['queued', 'leased', 'running', 'completed', 'failed', 'cancelled']) {
        assert.equal((await loadMyJobs(async () => response({ jobs: [{ ...fixture(), state }] })))[0].state, state);
    }
    await rejects(loadMyJob('b'.repeat(32), async () => response(fixture())), 'invalid_response');
    await rejects(loadMyJobs(async () => response({ jobs: [fixture(), fixture()] })), 'invalid_response');
    for (const data of [null, [], {}, { jobs: {} }]) await rejects(loadMyJobs(async () => response(data)), 'invalid_response');
});

test('uses fixed error messages and allowed status-specific codes only', async () => {
    for (const [status, code, expected] of [[401, 'private-canary', 'session_required'], [403, 'private-canary', 'forbidden'],
        [404, 'private-canary', 'unavailable'], [503, 'auth_unconfigured', 'auth_unconfigured'], [503, 'jobs_unconfigured', 'jobs_unconfigured'],
        [503, 'jobs_unavailable', 'jobs_unavailable'], [503, 'private-canary', 'unavailable'], [500, 'auth_unconfigured', 'unavailable']]) {
        await rejects(loadMyJobs(async () => response({ error: { code, message: 'private-canary' } }, status)), expected);
    }
    await rejects(loadMyJobs(async () => { throw new Error('private-canary'); }), 'unavailable');
    await rejects(loadMyJob(fixture().id, async () => response({ error: { code: 'not_found', message: 'private-canary' } }, 404)), 'not_found');
    await rejects(loadMyJobs(async () => response({ jobs: [] }, 201)), 'invalid_response');
});

test('requires JSON and valid UTF-8 and bounds declared and streamed bytes', async () => {
    await rejects(loadMyJobs(async () => new Response('private-canary', { headers: { 'Content-Type': 'text/html' } })), 'invalid_response');
    await rejects(loadMyJobs(async () => new Response('{private-canary', { headers: { 'Content-Type': 'application/json' } })), 'invalid_response');
    await rejects(loadMyJobs(async () => new Response(new Uint8Array([0xff]), { headers: { 'Content-Type': 'application/json' } })), 'invalid_response');
    await rejects(loadMyJobs(async () => new Response('{}', { headers: { 'Content-Type': 'application/json', 'Content-Length': '262145' } })), 'invalid_response');
    let canceled = false;
    const stream = new ReadableStream({ start(controller) { controller.enqueue(new Uint8Array(262144)); controller.enqueue(new Uint8Array(1)); }, cancel() { canceled = true; } });
    await rejects(loadMyJobs(async () => new Response(stream, { headers: { 'Content-Type': 'application/json' } })), 'invalid_response');
    assert.equal(canceled, true);
    const exact = JSON.stringify({ jobs: [] }).padEnd(262144, ' ');
    assert.deepEqual(await loadMyJobs(async () => new Response(exact, { headers: { 'Content-Type': 'application/json' } })), []);
});

test('access failure status overrides malformed bodies and cancels their streams', async () => {
    for (const [status, expected, content] of [[401, 'session_required', '<html>private-canary</html>'], [403, 'forbidden', '']]) {
        let canceled = false;
        const body = new ReadableStream({ start(controller) { controller.enqueue(new TextEncoder().encode(content)); }, cancel() { canceled = true; } });
        await rejects(loadMyJobs(async () => new Response(body, { status, headers: { 'Content-Type': 'text/html' } })), expected);
        assert.equal(canceled, true);
    }
});

test('bounds fetch and stream waits and supports caller cancellation', async () => {
    await rejects(loadMyJobs(() => new Promise(() => {}), { timeoutMs: 5 }), 'timeout');
    let canceled = false;
    const stalled = new ReadableStream({ pull() { return new Promise(() => {}); }, cancel() { canceled = true; } });
    await rejects(loadMyJobs(async () => new Response(stalled, { headers: { 'Content-Type': 'application/json' } }), { timeoutMs: 5 }), 'timeout');
    assert.equal(canceled, true);
    const controller = new AbortController();
    const pending = loadMyJobs(() => new Promise(() => {}), { signal: controller.signal });
    controller.abort();
    await rejects(pending, 'aborted');
    await rejects(loadMyJobs(() => { assert.fail('already canceled request issued'); }, { signal: controller.signal }), 'aborted');
    for (const timeoutMs of [0, -1, NaN, Infinity, 30001]) await rejects(loadMyJobs(() => { assert.fail('invalid timeout request issued'); }, { timeoutMs }), 'unavailable');
});
