const siteIDPattern = /^[A-Za-z0-9_](?:[A-Za-z0-9_-]{0,61}[A-Za-z0-9_])?$/;
const domainPattern = /^(?=.{1,253}$)(?:[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?\.)+[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?$/;

// Read-only tenant-scoped catalog details; the server authorizes each request.
export async function loadSiteDetails(siteID, fetcher = globalThis.fetch, signal) {
    if (typeof siteID !== 'string' || !siteIDPattern.test(siteID)) throw new Error('Invalid site selection.');
    const controller = new AbortController();
    const cancel = () => controller.abort();
    if (signal?.aborted) throw new Error('Site details could not be loaded.');
    signal?.addEventListener('abort', cancel, {once:true});
    const timeout = setTimeout(cancel, 10000);
    let body, reader, onAbort;
    const aborted = new Promise((_, reject) => { onAbort = () => reject(new Error('Site details could not be loaded.')); controller.signal.addEventListener('abort', onAbort, {once:true}); });
    try {
        return await Promise.race([aborted, (async () => {
            const response = await fetcher(`/api/v1/sites/${encodeURIComponent(siteID)}`, {method:'GET',credentials:'same-origin',cache:'no-store',redirect:'error',headers:{Accept:'application/json'},signal:controller.signal});
            if (controller.signal.aborted) { if (response.body) void response.body.cancel().catch(() => {}); throw new Error('Site details could not be loaded.'); }
            body = response.body;
            if (response.status === 401) throw new Error('A secure browser session is required to view sites.');
            if (response.status === 403) throw new Error('Your account cannot view sites.');
            if (response.status !== 200 || !/^application\/json(?:\s*;|$)/i.test(response.headers.get('Content-Type') ?? '') || !body?.getReader) throw new Error('Site details could not be loaded.');
            const length = response.headers.get('Content-Length');
            if (length !== null && (!/^\d+$/.test(length) || Number(length)>16384)) throw new Error('Site details could not be loaded.');
            reader = body.getReader(); const chunks=[]; let size=0;
            while (true) { const {value,done}=await reader.read(); if (done) break; if (!(value instanceof Uint8Array) || value.byteLength > 16384-size) throw new Error('Site details could not be loaded.'); size+=value.byteLength; chunks.push(value); }
            const bytes=new Uint8Array(size); let offset=0; for (const chunk of chunks) { bytes.set(chunk,offset); offset+=chunk.byteLength; }
            const record=JSON.parse(new TextDecoder('utf-8',{fatal:true}).decode(bytes));
            if (!record || record.SiteID!==siteID || typeof record.TenantID!=='string' || !record.TenantID || record.TenantID.length>128 || record.DocumentRoot!==`sites/${siteID}/public` || !Number.isSafeInteger(record.CreatedAt) || !Array.isArray(record.Domains) || record.Domains.length<1 || record.Domains.length>33) throw new Error('Site details could not be loaded.');
            const seen=new Set(); let primary=0;
            const domains=record.Domains.map(claim => {
                if (!claim || typeof claim.Name!=='string' || !domainPattern.test(claim.Name) || !['primary','alias'].includes(claim.Kind) || !['pending','verified'].includes(claim.OwnershipState) || seen.has(claim.Name)) throw new Error('Site details could not be loaded.');
                seen.add(claim.Name); if (claim.Kind==='primary') primary++;
                return {name:claim.Name,kind:claim.Kind,state:claim.OwnershipState};
            });
            if (primary!==1) throw new Error('Site details could not be loaded.');
            return {siteID, documentRoot:record.DocumentRoot, domains};
        })()]);
    } catch(error) {
        if (error instanceof Error && ['A secure browser session is required to view sites.','Your account cannot view sites.'].includes(error.message)) throw error;
        throw new Error('Site details could not be loaded. Try again.');
    } finally {
        clearTimeout(timeout); signal?.removeEventListener('abort',cancel); controller.signal.removeEventListener('abort',onAbort);
        if (reader) void reader.cancel().catch(()=>{}); else if (body) void body.cancel().catch(()=>{});
    }
}
