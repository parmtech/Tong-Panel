const idPattern=/^[A-Za-z0-9_](?:[A-Za-z0-9_-]{0,61}[A-Za-z0-9_])?$/;

// This endpoint exposes catalog metadata only; no database is provisioned here.
export async function loadSiteDatabases(siteID,fetcher=globalThis.fetch,signal) {
    if(typeof siteID!=='string'||!idPattern.test(siteID)) throw new Error('Invalid site selection.');
    const controller=new AbortController(); const cancel=()=>controller.abort();
    if(signal?.aborted) throw new Error('Database metadata could not be loaded.');
    signal?.addEventListener('abort',cancel,{once:true}); const timeout=setTimeout(cancel,10000);
    let body,reader,onAbort;
    const aborted=new Promise((_,reject)=>{onAbort=()=>reject(new Error('Database metadata could not be loaded.'));controller.signal.addEventListener('abort',onAbort,{once:true})});
    try {
        return await Promise.race([aborted,(async()=>{
            const response=await fetcher(`/api/v1/sites/${encodeURIComponent(siteID)}/databases`,{method:'GET',credentials:'same-origin',cache:'no-store',redirect:'error',headers:{Accept:'application/json'},signal:controller.signal});
            if(controller.signal.aborted){if(response.body)void response.body.cancel().catch(()=>{});throw new Error('Database metadata could not be loaded.');}
            body=response.body;
            if(response.status===401)throw new Error('A secure browser session is required to view sites.');
            if(response.status===403)throw new Error('Your account cannot view sites.');
            if(response.status!==200||!/^application\/json(?:\s*;|$)/i.test(response.headers.get('Content-Type')??'')||!body?.getReader)throw new Error('Database metadata could not be loaded.');
            const length=response.headers.get('Content-Length');
            if(length!==null&&(!/^\d+$/.test(length)||Number(length)>65536))throw new Error('Database metadata could not be loaded.');
            reader=body.getReader();const chunks=[];let size=0;
            while(true){const {value,done}=await reader.read();if(done)break;if(!(value instanceof Uint8Array)||value.byteLength>65536-size)throw new Error('Database metadata could not be loaded.');size+=value.byteLength;chunks.push(value);}
            const bytes=new Uint8Array(size);let offset=0;for(const chunk of chunks){bytes.set(chunk,offset);offset+=chunk.byteLength;}
            const result=JSON.parse(new TextDecoder('utf-8',{fatal:true}).decode(bytes));
            if(!result||!Array.isArray(result.databases)||result.databases.length>200)throw new Error('Database metadata could not be loaded.');
            const seen=new Set();let tenant='';
            return result.databases.map(row=>{
                if(!row||row.SiteID!==siteID||typeof row.TenantID!=='string'||!row.TenantID||row.TenantID.length>128||(tenant&&tenant!==row.TenantID)||typeof row.DatabaseID!=='string'||!idPattern.test(row.DatabaseID)||seen.has(row.DatabaseID)||!/^tp_[a-z2-7]{26}$/.test(row.DatabaseName)||!/^tu_[a-z2-7]{26}$/.test(row.Username)||!Number.isSafeInteger(row.CreatedAt))throw new Error('Database metadata could not be loaded.');
                tenant=row.TenantID;seen.add(row.DatabaseID);
                return {id:row.DatabaseID,name:row.DatabaseName,username:row.Username};
            });
        })()]);
    }catch(error){
        if(error instanceof Error&&['A secure browser session is required to view sites.','Your account cannot view sites.'].includes(error.message))throw error;
        throw new Error('Database metadata could not be loaded. Try again.');
    }finally{clearTimeout(timeout);signal?.removeEventListener('abort',cancel);controller.signal.removeEventListener('abort',onAbort);if(reader)void reader.cancel().catch(()=>{});else if(body)void body.cancel().catch(()=>{});}
}
