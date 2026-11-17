const sitePattern = /^[A-Za-z0-9_-]{1,63}$/;
const generic = 'The certificate domain snapshot could not be loaded. Try again.';
const messages = new Map([
    [401, 'A secure browser session is required to view this snapshot.'],
    [403, 'Your account cannot view this snapshot.'],
    [404, 'This site is not available in your workspace.'],
    [409, 'Verify the site’s primary domain before loading its snapshot.'],
    [503, 'The site catalog is not enabled on this server.'],
]);
function domainIsValid(name) {
    return typeof name === 'string' && name.length <= 253 && name.includes('.')
        && !/^\d+\.\d+\.\d+\.\d+$/.test(name)
        && name.split('.').every(label => /^[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?$/.test(label));
}

export async function loadACMEPreflight(siteID, fetcher = globalThis.fetch, signal) {
    if (typeof siteID !== 'string' || !sitePattern.test(siteID)) throw new Error('Enter a valid site ID.');
    if (signal?.aborted) throw new Error(generic);
    const controller = new AbortController();
    const cancel = () => controller.abort();
    signal?.addEventListener('abort', cancel, { once: true });
    const timer = setTimeout(cancel, 10000);
    let body, reader, rejectAbort;
    const aborted = new Promise((_, reject) => {
        rejectAbort = () => reject(new Error(generic));
        controller.signal.addEventListener('abort', rejectAbort, { once: true });
    });
    try {
        return await Promise.race([aborted, (async () => {
            const response = await fetcher('/api/v1/sites/' + encodeURIComponent(siteID) + '/preflight/acme-http01', {
                method: 'GET', credentials: 'same-origin', cache: 'no-store', redirect: 'error',
                headers: { Accept: 'application/json' }, signal: controller.signal,
            });
            body = response.body;
            if (controller.signal.aborted) { if (body) void body.cancel().catch(() => {}); throw new Error(generic); }
            if (messages.has(response.status)) throw new Error(messages.get(response.status));
            if (response.status !== 200 || !/^application\/json(?:\s*;|$)/i.test(response.headers.get('Content-Type') ?? '') || !body?.getReader) throw new Error(generic);
            const length = response.headers.get('Content-Length');
            if (length !== null && (!/^\d+$/.test(length) || Number(length) > 16384)) throw new Error(generic);
            reader = body.getReader();
            const chunks = []; let size = 0;
            while (true) {
                const { value, done } = await reader.read();
                if (done) break;
                size += value.byteLength;
                if (size > 16384) throw new Error(generic);
                chunks.push(value);
            }
            const bytes = new Uint8Array(size); let offset = 0;
            for (const chunk of chunks) { bytes.set(chunk, offset); offset += chunk.byteLength; }
            const data = JSON.parse(new TextDecoder('utf-8', { fatal: true }).decode(bytes));
            if (!data || Object.keys(data).sort().join(',') !== 'challenge,domains,execution_enabled,site_id,snapshot_only'
                || data.site_id !== siteID || data.challenge !== 'http-01' || data.snapshot_only !== true || data.execution_enabled !== false
                || !Array.isArray(data.domains) || data.domains.length < 1 || data.domains.length > 33
                || !data.domains.every(domainIsValid) || new Set(data.domains).size !== data.domains.length) throw new Error(generic);
            return { siteID: data.site_id, domains: [...data.domains] };
        })()]);
    } catch (error) {
        if (error instanceof Error && [...messages.values()].includes(error.message)) throw error;
        throw new Error(generic);
    } finally {
        clearTimeout(timer);
        signal?.removeEventListener('abort', cancel);
        controller.signal.removeEventListener('abort', rejectAbort);
        if (reader) void reader.cancel().catch(() => {});
        else if (body) void body.cancel().catch(() => {});
    }
}

export function mountACMEPreflight(root = document) {
    const form = root.querySelector('#acme-preflight-form');
    const input = root.querySelector('#acme-preflight-id');
    const status = root.querySelector('#acme-preflight-status');
    const output = root.querySelector('#acme-preflight-domains');
    const button = root.querySelector('#acme-preflight-submit');
    if (!form || !input || !status || !output || !button) return;
    let sequence = 0, controller;
    const reset = () => {
        ++sequence;
        controller?.abort();
        output.replaceChildren(); output.hidden = true;
        form.removeAttribute('aria-busy'); button.disabled = false;
        status.dataset.state = ''; status.textContent = 'No certificate domain snapshot loaded.';
    };
    input.addEventListener('input', reset);
    form.addEventListener('submit', async event => {
        event.preventDefault();
        reset();
        const current = sequence, selected = input.value;
        controller = new AbortController();
        button.disabled = true; form.setAttribute('aria-busy', 'true');
        status.dataset.state = 'loading'; status.textContent = 'Loading certificate domain snapshot…';
        try {
            const snapshot = await loadACMEPreflight(selected, undefined, controller.signal);
            if (sequence !== current || input.value !== selected) return;
            for (const name of snapshot.domains) {
                const item = root.createElement('li'); item.textContent = name; output.append(item);
            }
            output.hidden = false;
            status.dataset.state = 'available';
            status.textContent = 'Verified domains recorded for this site. Snapshot only; certificate issuance and renewal are unavailable.';
        } catch (error) {
            if (sequence !== current || input.value !== selected) return;
            status.dataset.state = 'unavailable'; status.textContent = error instanceof Error ? error.message : generic;
        } finally {
            if (sequence === current) { button.disabled = false; form.removeAttribute('aria-busy'); }
        }
    });
    globalThis.addEventListener?.('pagehide', reset);
}
