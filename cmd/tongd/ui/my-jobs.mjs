const maximumBytes = 256 * 1024;
const jobID = /^[a-f0-9]{32}$/;
const states = new Set(['queued', 'leased', 'running', 'completed', 'failed', 'cancelled']);
const messages = Object.freeze({
    session_required: 'A secure browser session is required to view your jobs.',
    forbidden: 'Your account cannot view these jobs.',
    auth_unconfigured: 'Browser authentication is not configured on this server.',
    jobs_unconfigured: 'The jobs backend is not configured on this server.',
    jobs_unavailable: 'The jobs backend is temporarily unavailable. Try again.',
    not_found: 'This job is not available in your workspace.',
    invalid_response: 'The server returned an unexpected jobs response.',
    timeout: 'The jobs request timed out. Try again.',
    aborted: 'The jobs request was canceled.',
    unavailable: 'Your jobs could not be loaded. Try again.',
});

export class MyJobsError extends Error {
    constructor(code) {
        super(messages[code] ?? messages.unavailable);
        this.name = 'MyJobsError';
        this.code = Object.hasOwn(messages, code) ? code : 'unavailable';
    }
}

function validText(value, minimum, maximum) {
    if (typeof value !== 'string' || /[\u0000-\u001f\u007f]/u.test(value)) return false;
    // The journal and Go request guards bound strings by UTF-8 bytes.
    const length = new TextEncoder().encode(value).byteLength;
    return length >= minimum && length <= maximum;
}

function validTimestamp(value) {
    if (typeof value !== 'string') return false;
    const match = /^(\d{4})-(\d{2})-(\d{2})T(\d{2}):(\d{2}):(\d{2})(?:\.\d{1,9})?(?:Z|([+-])(\d{2}):(\d{2}))$/.exec(value);
    if (!match) return false;
    const [, year, month, day, hour, minute, second, , zoneHour, zoneMinute] = match;
    const leap = Number(year) % 4 === 0 && (Number(year) % 100 !== 0 || Number(year) % 400 === 0);
    const days = [31, leap ? 29 : 28, 31, 30, 31, 30, 31, 31, 30, 31, 30, 31];
    return Number(month) >= 1 && Number(month) <= 12 && Number(day) >= 1 && Number(day) <= days[Number(month) - 1]
        && Number(hour) <= 23 && Number(minute) <= 59 && Number(second) <= 59
        && (zoneHour === undefined || (Number(zoneHour) <= 23 && Number(zoneMinute) <= 59)) && Number.isFinite(Date.parse(value));
}

function validateJob(value) {
    if (!value || typeof value !== 'object' || Array.isArray(value) || typeof value.id !== 'string' || !jobID.test(value.id)
        || value.kind !== 'validation.noop' || !states.has(value.state)
        || !validText(value.tenant_id, 1, 128) || !validText(value.idempotency_key, 8, 128)
        || !validTimestamp(value.created_at) || !validTimestamp(value.updated_at)
        || (value.requested_by !== undefined && !validText(value.requested_by, 1, 128))) {
        throw new MyJobsError('invalid_response');
    }
    // Project only current wire fields. Additional server data cannot become
    // DOM properties, credentials or links through this loader.
    const job = { id: value.id, tenant_id: value.tenant_id, kind: value.kind, state: value.state,
        idempotency_key: value.idempotency_key, created_at: value.created_at, updated_at: value.updated_at };
    if (value.requested_by !== undefined) job.requested_by = value.requested_by;
    return job;
}

async function request(path, fetcher, { signal, timeoutMs = 10000 } = {}) {
    if (typeof fetcher !== 'function' || !Number.isFinite(timeoutMs) || timeoutMs < 1 || timeoutMs > 30000) throw new MyJobsError('unavailable');
    const controller = new AbortController();
    let timedOut = false;
    let reader;
    let body;
    const forwardAbort = () => controller.abort();
    if (signal?.aborted) throw new MyJobsError('aborted');
    signal?.addEventListener('abort', forwardAbort, { once: true });
    const timer = setTimeout(() => { timedOut = true; controller.abort(); }, timeoutMs);
    let onAbort;
    const aborted = new Promise((_, reject) => {
        onAbort = () => reject(new MyJobsError(timedOut ? 'timeout' : 'aborted'));
        controller.signal.addEventListener('abort', onAbort, { once: true });
    });
    try {
        return await Promise.race([aborted, (async () => {
            const response = await fetcher(path, { method: 'GET', credentials: 'same-origin', cache: 'no-store',
                headers: { Accept: 'application/json' }, signal: controller.signal, redirect: 'error' });
            if (controller.signal.aborted) {
                if (response.body) void response.body.cancel().catch(() => {});
                throw new MyJobsError(timedOut ? 'timeout' : 'aborted');
            }
            body = response.body;
            // Access withdrawal is authoritative even when an intermediary
            // replaces the JSON error with an empty or non-JSON response.
            if (response.status === 401) throw new MyJobsError('session_required');
            if (response.status === 403) throw new MyJobsError('forbidden');
            if (!/^application\/json(?:\s*;|$)/i.test(response.headers.get('Content-Type') ?? '') || !response.body?.getReader) throw new MyJobsError('invalid_response');
            const declaredLength = response.headers.get('Content-Length');
            if (declaredLength !== null && (!/^\d+$/.test(declaredLength) || Number(declaredLength) > maximumBytes)) throw new MyJobsError('invalid_response');
            reader = response.body.getReader();
            const bytes = new Uint8Array(maximumBytes);
            let length = 0;
            while (true) {
                const { value, done } = await reader.read();
                if (done) break;
                if (length + value.byteLength > maximumBytes) throw new MyJobsError('invalid_response');
                bytes.set(value, length);
                length += value.byteLength;
            }
            let data;
            try { data = JSON.parse(new TextDecoder('utf-8', { fatal: true }).decode(bytes.subarray(0, length))); }
            catch { throw new MyJobsError('invalid_response'); }
            if (!response.ok) {
                if (response.status === 404 && path.split('?')[0] !== '/api/v1/jobs') throw new MyJobsError('not_found');
                const code = data?.error?.code;
                if (response.status === 503 && ['auth_unconfigured', 'jobs_unconfigured', 'jobs_unavailable'].includes(code)) throw new MyJobsError(code);
                throw new MyJobsError('unavailable');
            }
            if (response.status !== 200) throw new MyJobsError('invalid_response');
            return data;
        })()]);
    } catch (error) {
        if (error instanceof MyJobsError) throw error;
        throw new MyJobsError(controller.signal.aborted ? (timedOut ? 'timeout' : 'aborted') : 'unavailable');
    } finally {
        clearTimeout(timer);
        signal?.removeEventListener('abort', forwardAbort);
        controller.signal.removeEventListener('abort', onAbort);
        if (reader) void reader.cancel().catch(() => {});
        else if (body) void body.cancel().catch(() => {});
    }
}

export async function loadMyJobs(fetcher = globalThis.fetch, options) {
    const data = await request('/api/v1/jobs', fetcher, options);
    if (!data || typeof data !== 'object' || Array.isArray(data) || !Object.hasOwn(data, 'jobs') || (data.jobs !== null && !Array.isArray(data.jobs))) throw new MyJobsError('invalid_response');
    const result = (data.jobs ?? []).map(validateJob);
    if (new Set(result.map(job => job.id)).size !== result.length) throw new MyJobsError('invalid_response');
    return result;
}

// Explicit pagination bounds each response while preserving the legacy API for
// other development clients. The cursor is an ordering boundary, not authority.
export async function loadMyJobsPage(after = '', fetcher = globalThis.fetch, options) {
    if (typeof after !== 'string' || (after !== '' && !jobID.test(after))) throw new MyJobsError('invalid_response');
    const path = `/api/v1/jobs?limit=50${after ? `&after=${after}` : ''}`;
    const data = await request(path, fetcher, options);
    if (!data || typeof data !== 'object' || Array.isArray(data) || !Array.isArray(data.jobs)
        || data.jobs.length > 50 || typeof data.next_after !== 'string'
        || (data.next_after !== '' && !jobID.test(data.next_after))) throw new MyJobsError('invalid_response');
    const jobs = data.jobs.map(validateJob);
    let previous = after;
    for (const job of jobs) {
        if (previous && job.id <= previous) throw new MyJobsError('invalid_response');
        previous = job.id;
    }
    if (data.next_after && (jobs.length !== 50 || data.next_after !== jobs[jobs.length - 1].id)) throw new MyJobsError('invalid_response');
    return { jobs, next_after: data.next_after };
}

export async function loadMyJob(id, fetcher = globalThis.fetch, options) {
    if (typeof id !== 'string' || !jobID.test(id)) throw new MyJobsError('invalid_response');
    const job = validateJob(await request(`/api/v1/jobs/${id}`, fetcher, options));
    if (job.id !== id) throw new MyJobsError('invalid_response');
    return job;
}
