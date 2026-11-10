const siteID = /^[A-Za-z0-9_-]{1,63}$/;

// The cursor is only an ordering boundary. Authorization is server-side.
export async function loadSitesPage(after = '', fetcher = globalThis.fetch, signal) {
    if (typeof after !== 'string' || (after && !siteID.test(after))) throw new Error('Sites could not be loaded.');
    const controller = new AbortController();
    const cancel = () => controller.abort();
    if (signal?.aborted) throw new Error('Sites could not be loaded. Try again.');
    signal?.addEventListener('abort', cancel, { once: true });
    const timeout = setTimeout(cancel, 10000);
    let reader, body, onAbort;
    const aborted = new Promise((_, reject) => {
        onAbort = () => reject(new Error('Sites could not be loaded. Try again.'));
        controller.signal.addEventListener('abort', onAbort, { once: true });
    });
    try {
        return await Promise.race([aborted, (async () => {
        const response = await fetcher(`/api/v1/sites?limit=50${after ? `&after=${encodeURIComponent(after)}` : ''}`, {
            method: 'GET', credentials: 'same-origin', cache: 'no-store', headers: { Accept: 'application/json' }, signal: controller.signal, redirect: 'error',
        });
        if (controller.signal.aborted) { if (response.body) void response.body.cancel().catch(() => {}); throw new Error('Sites could not be loaded.'); }
        body = response.body;
        if (response.status === 401) throw new Error('A secure browser session is required to view sites.');
        if (response.status === 403) throw new Error('Your account cannot view sites.');
        if (response.status !== 200 || !/^application\/json(?:\s*;|$)/i.test(response.headers.get('Content-Type') ?? '') || !response.body?.getReader) throw new Error('Sites could not be loaded.');
        const length = response.headers.get('Content-Length');
        if (length !== null && (!/^\d+$/.test(length) || Number(length) > 65536)) throw new Error('Sites could not be loaded.');
        reader = response.body.getReader();
        const chunks = []; let size = 0;
        while (true) {
            const { value, done } = await reader.read();
            if (done) break;
            size += value.byteLength;
            if (size > 65536) throw new Error('Sites could not be loaded.');
            chunks.push(value);
        }
        const bytes = new Uint8Array(size); let offset = 0;
        for (const chunk of chunks) { bytes.set(chunk, offset); offset += chunk.byteLength; }
        const data = JSON.parse(new TextDecoder('utf-8', { fatal: true }).decode(bytes));
        if (!data || !Array.isArray(data.sites) || data.sites.length > 50 || typeof data.next_after !== 'string' || (data.next_after && !siteID.test(data.next_after))) throw new Error('Sites could not be loaded.');
        let previous = after;
        const sites = data.sites.map(site => {
            if (!site || typeof site.site_id !== 'string' || !siteID.test(site.site_id) || (previous && site.site_id <= previous)
                || typeof site.primary_domain !== 'string' || site.primary_domain.length > 253 || !/^[a-z0-9.-]+$/.test(site.primary_domain)
                || !['pending', 'verified'].includes(site.ownership_state)) throw new Error('Sites could not be loaded.');
            previous = site.site_id;
            return { site_id: site.site_id, primary_domain: site.primary_domain, ownership_state: site.ownership_state };
        });
        if (data.next_after && (sites.length !== 50 || data.next_after !== sites[sites.length - 1].site_id)) throw new Error('Sites could not be loaded.');
        return { sites, next_after: data.next_after };
        })()]);
    } catch (error) {
        if (error instanceof Error && ['A secure browser session is required to view sites.', 'Your account cannot view sites.'].includes(error.message)) throw error;
        throw new Error('Sites could not be loaded. Try again.');
    } finally {
        clearTimeout(timeout); signal?.removeEventListener('abort', cancel); controller.signal.removeEventListener('abort', onAbort);
        if (reader) void reader.cancel().catch(() => {}); else if (body) void body.cancel().catch(() => {});
    }
}
