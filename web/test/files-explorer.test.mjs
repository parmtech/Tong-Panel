import assert from 'node:assert/strict';
import { readFile } from 'node:fs/promises';
import test from 'node:test';
import { downloadTenantFile, FilesExplorerError, listTenantFiles, validateFilesPath } from '../../cmd/tongd/ui/files-explorer.mjs';

const jsonResponse = (body, ok = true, status = 200) => ({ ok, status, json: async () => body });

test('validates relative paths before making a same-origin read-only listing request', async () => {
    const calls = [];
    const entries = await listTenantFiles('htdocs/site one', async (url, options) => {
        calls.push({ url, options });
        return jsonResponse({ entries: [{ name: 'index.html', kind: 'file', size: 5 }] });
    });
    assert.deepEqual(entries, [{ name: 'index.html', kind: 'file', size: 5 }]);
    assert.equal(calls.length, 1);
    assert.equal(calls[0].url, '/api/v1/files?path=htdocs%2Fsite+one');
    assert.equal(calls[0].options.method, 'GET');
    assert.equal(calls[0].options.credentials, 'same-origin');
    assert.equal(calls[0].options.cache, 'no-store');
    assert.equal(calls[0].options.headers.Accept, 'application/json');
    assert.equal('Authorization' in calls[0].options.headers, false);
    assert.ok(calls[0].options.signal instanceof AbortSignal);
});

test('rejects traversal, absolute, empty-segment and backslash paths without fetching', async () => {
    for (const path of ['../secret', 'a/../../secret', '/etc/passwd', 'a//b', 'a\\b', 'a/./b', 'a/'])
        assert.throws(() => validateFilesPath(path), FilesExplorerError, path);
    let calls = 0;
    await assert.rejects(listTenantFiles('../secret', async () => { calls++; }), FilesExplorerError);
    assert.equal(calls, 0);
});

test('maps authorization, not-found and disabled responses to fixed guidance', async () => {
    for (const [status, expected] of [
        [401, /secure browser session/],
        [403, /cannot access/],
        [404, /not available/],
        [503, /not enabled/],
    ]) {
        await assert.rejects(listTenantFiles('', async () => jsonResponse({ error: { code: 'secret-detail', message: 'raw internal path' } }, false, status)), expected);
    }
});

test('rejects malformed and unsafe listing records without reflecting names', async () => {
    for (const entries of [
        [{ name: '../secret', kind: 'file', size: 2 }],
        [{ name: 'link', kind: 'symlink', size: -1 }],
        [{ name: 'file', kind: 'html', size: 2 }],
        Array.from({ length: 1001 }, () => ({ name: 'x', kind: 'file', size: 0 })),
    ]) {
        await assert.rejects(listTenantFiles('', async () => jsonResponse({ entries })), /unexpected file listing/);
    }
    await assert.rejects(listTenantFiles('', async () => ({ ok: true, json: async () => { throw new Error('sensitive'); } })), /unexpected file listing/);
});

test('downloads only selected regular files through same-origin GET and returns an inert binary blob', async () => {
    let call;
    const result = await downloadTenantFile('public/hello world.txt', async (url, options) => {
        call = { url, options };
        return { ok: true, status: 200, headers: new Headers({ 'Content-Type': 'application/octet-stream' }), arrayBuffer: async () => new TextEncoder().encode('hello').buffer };
    });
    assert.equal(call.url, '/api/v1/files/content?path=public%2Fhello+world.txt');
    assert.equal(call.options.method, 'GET');
    assert.equal(call.options.credentials, 'same-origin');
    assert.equal(call.options.cache, 'no-store');
    assert.equal(call.options.headers.Accept, 'application/octet-stream');
    assert.equal('Authorization' in call.options.headers, false);
    assert.equal(result.name, 'hello world.txt');
    assert.equal(result.blob.type, 'application/octet-stream');
    assert.equal(await result.blob.text(), 'hello');
});

test('rejects an oversized or non-binary download response', async () => {
    const tooLarge = new ArrayBuffer(16 * 1024 * 1024 + 1);
    const binaryHeaders = new Headers({ 'Content-Type': 'application/octet-stream' });
    await assert.rejects(downloadTenantFile('large.bin', async () => ({ ok: true, headers: binaryHeaders, arrayBuffer: async () => tooLarge })), /supported download limit/);
    await assert.rejects(downloadTenantFile('bad.bin', async () => ({ ok: true, headers: binaryHeaders, arrayBuffer: async () => 'not bytes' })), /supported download limit/);
    await assert.rejects(downloadTenantFile('redirect.html', async () => ({ ok: true, headers: new Headers({ 'Content-Type': 'text/html' }), arrayBuffer: async () => new ArrayBuffer(0) })), /unexpected file format/);
});

test('uses bounded requests and maps network failures without surfacing internals', async () => {
    await assert.rejects(listTenantFiles('', async (_url, { signal }) => new Promise((_resolve, reject) => {
        signal.addEventListener('abort', () => reject(Object.assign(new Error('secret'), { name: 'AbortError' })), { once: true });
    }), 10), /timed out/);
    await assert.rejects(listTenantFiles('', async (_url, { signal }) => ({
        ok: true,
        json: () => new Promise((_resolve, reject) => signal.addEventListener('abort', () => reject(Object.assign(new Error('private'), { name: 'AbortError' })), { once: true })),
    }), 10), /timed out/);
    await assert.rejects(listTenantFiles('', async () => { throw new Error('private host path'); }), /could not be reached/);
});

test('wires an accessible read-only explorer without upload, edit or delete controls', async () => {
    const root = new URL('../../cmd/tongd/ui/', import.meta.url);
    const [html, app, css] = await Promise.all([
        readFile(new URL('index.html', root), 'utf8'),
        readFile(new URL('app.js', root), 'utf8'),
        readFile(new URL('style.css', root), 'utf8'),
    ]);
    assert.match(html, /id="files-explorer-title"/);
    assert.match(html, /id="files-refresh" type="button"/);
    assert.match(html, /id="files-status" class="files-status" role="status" aria-live="polite"/);
    assert.match(app, /import \{ downloadTenantFile, listTenantFilesPage, previewTenantText, FilesExplorerError \} from '\.\/files-explorer\.mjs'/);
    assert.match(app, /name\.textContent =/);
    assert.match(html, /Files are not loaded/);
    assert.match(app, /filesRefresh\?\.addEventListener\('click'/);
    assert.match(css, /\.files-list\{/);
    assert.doesNotMatch(html, /type="file"|>Upload<|>Delete<|>Edit</);
});

test('loads a single paged directory response and rejects a stalled cursor', async () => {
    const { listTenantFilesPage } = await import('../../cmd/tongd/ui/files-explorer.mjs');
    let request;
    const page = await listTenantFilesPage('site', 'old name', async url => { request=url; return jsonResponse({entries:[{name:'new',kind:'file',size:1}],next_after:'new'}); });
    assert.equal(request, '/api/v1/files?path=site&page=1&after=old%20name');
    assert.equal(page.next_after, 'new');
    await assert.rejects(listTenantFilesPage('', 'same', async () => jsonResponse({entries:[],next_after:'same'})), /unexpected file listing/);
});

 test('page navigation retains position on failure and resets on refresh', async () => {
    const { runInNewContext } = await import('node:vm');
    const app = await readFile(new URL('../../cmd/tongd/ui/app.js', import.meta.url), 'utf8');
    const source = app.slice(app.indexOf("const filesFilter ="), app.indexOf("async function downloadFile("));
    class Element {
        constructor() { this.dataset = {}; this.disabled = false; this.children = []; this.listeners = new Map(); this.attributes = new Map(); this.textContent = ''; }
        addEventListener(type, fn) { this.listeners.set(type, fn); }
        setAttribute(key, value) { this.attributes.set(key, value); }
        removeAttribute(key) { this.attributes.delete(key); }
        replaceChildren(...children) { this.children = children; }
        append(...children) { this.children.push(...children); }
        click() { if (!this.disabled) this.listeners.get('click')?.(); }
    }
    const ids = ['#files-filter','#files-prefix','#files-refresh','#files-previous','#files-next','#files-status','#files-list','#files-breadcrumbs'];
    const elements = new Map(ids.map(id => [id, new Element()]));
    const calls = []; let fail = false; let denied = false;
    class Denied extends Error { constructor(){super('denied'); this.status=403;} }
    const context = {document: {querySelector: id => elements.get(id), createElement: () => new Element()},
        FilesExplorerError: Denied, fetch: () => {}, clearFilePreview: () => {}, listTenantFilesPage: async (path, after, _fetch, _timeout, prefix) => {
            calls.push({path,after,prefix});
            if (denied) throw new Denied();
            if (fail) throw new Error('network failure');
            return after === '' ? {entries:[{name:'a',kind:'file',size:1}],next_after:'a'} : {entries:[{name:'b',kind:'file',size:1}],next_after:''};
        }, downloadFile: () => {}};
    runInNewContext(source + "\nfilesRefresh.addEventListener('click', () => void refreshFiles());", context);
    const flush = () => new Promise(resolve => setTimeout(resolve,0));
    const refresh=elements.get('#files-refresh'), next=elements.get('#files-next'), previous=elements.get('#files-previous'), status=elements.get('#files-status');
    refresh.click(); await flush(); assert.equal(next.disabled,false); assert.equal(previous.disabled,true); assert.match(status.textContent,/Page 1:/);
    next.click(); await flush(); assert.equal(previous.disabled,false); assert.equal(next.disabled,true); assert.match(status.textContent,/Page 2:/);
    fail=true; previous.click(); await flush(); assert.equal(previous.disabled,false); assert.match(status.textContent,/Files could not be loaded/);
    fail=false; previous.click(); await flush(); assert.equal(previous.disabled,true); assert.match(status.textContent,/Page 1:/);
    next.click(); await flush(); refresh.click(); await flush(); assert.equal(previous.disabled,true); assert.match(status.textContent,/Page 1:/);
    assert.deepEqual(calls.map(c => c.after),['','a','','','a','']);
    elements.get('#files-prefix').value='Ab'; elements.get('#files-filter').listeners.get('submit')({preventDefault(){}}); await flush();
    assert.equal(calls.at(-1).prefix,'Ab'); assert.equal(calls.at(-1).after,''); assert.match(status.textContent,/Page 1:/);
    assert.ok(elements.get('#files-list').children.length); denied=true; refresh.click(); await flush();
    assert.equal(elements.get('#files-list').children.length,0); assert.equal(next.disabled,true); assert.equal(previous.disabled,true);
});

test('text preview streams with hard size, UTF-8 and binary checks', async () => {
    const { previewTenantText } = await import('../../cmd/tongd/ui/files-explorer.mjs');
    const make = (chunks, length = null) => ({ok:true,status:200,headers:new Headers({...{'Content-Type':'application/octet-stream'}, ...(length===null ? {} : {'Content-Length':length})}), body: new ReadableStream({start(controller){ for(const chunk of chunks) controller.enqueue(chunk); controller.close(); }})});
    let url;
    const text = await previewTenantText('a/file', async u => {url=u; return make([new TextEncoder().encode('<script>safe text</script>')]);});
    assert.equal(url, '/api/v1/files/content?path=a%2Ffile'); assert.equal(text,'<script>safe text</script>');
    await assert.rejects(previewTenantText('a',async () => make([], '262145')), /too large/);
    await assert.rejects(previewTenantText('a',async () => make([new Uint8Array(262145)])), /too large/);
    await assert.rejects(previewTenantText('a',async () => make([new Uint8Array(262144),new Uint8Array(1)],'1')), /too large/);
    await assert.rejects(previewTenantText('a',async () => make([new Uint8Array([0xff])])), /UTF-8/);
    await assert.rejects(previewTenantText('a',async () => make([new Uint8Array([0])])), /UTF-8/);
    await assert.rejects(previewTenantText('a',async () => ({ok:false,status:403})), /cannot access/);
});

test('preview renders inert text and ignores results after closing', async () => {
    const { runInNewContext } = await import('node:vm');
    const app = await readFile(new URL('../../cmd/tongd/ui/app.js', import.meta.url),'utf8');
    const setup = app.slice(app.indexOf('const filesPreview ='),app.indexOf('const filesFilter ='));
    const preview = app.slice(app.indexOf('async function showFilePreview('),app.indexOf('async function downloadFile('));
    const elements = new Map(['#files-preview','#files-preview-title','#files-preview-content','#files-preview-close'].map(id => [id,{hidden:true,textContent:'',addEventListener(){}}]));
    const document = {querySelector:id => elements.get(id), addEventListener(){}, hidden:false, hasFocus:()=>true};
    let resolve; const context={document,window:{addEventListener(){}},filesPath:['site'],filesStatus:{textContent:'',dataset:{}},previewTenantText:()=>new Promise(r=>{resolve=r;})};
    const api=runInNewContext(setup+preview+'\n({showFilePreview,clearFilePreview})',context);
    const first=api.showFilePreview('file', 'site'); resolve('<script>inert</script>'); await first;
    assert.equal(elements.get('#files-preview-content').textContent,'<script>inert</script>'); assert.equal(elements.get('#files-preview').hidden,false);
    const second=api.showFilePreview('file', 'site'); api.clearFilePreview(); resolve('stale secret'); await second;
    assert.equal(elements.get('#files-preview-content').textContent,''); assert.equal(elements.get('#files-preview').hidden,true);
});

test('stale file rows cannot act during folder navigation', async () => {
    const { runInNewContext } = await import('node:vm');
    const app = await readFile(new URL('../../cmd/tongd/ui/app.js', import.meta.url),'utf8');
    const source = app.slice(app.indexOf('const filesFilter ='),app.indexOf('async function showFilePreview('));
    class Element {
        constructor(){this.dataset={};this.disabled=false;this.children=[];this.listeners=new Map();this.attributes=new Map();}
        addEventListener(k,v){this.listeners.set(k,v)} setAttribute(k,v){this.attributes.set(k,v)} removeAttribute(k){this.attributes.delete(k)}
        replaceChildren(...c){this.children=c} append(...c){this.children.push(...c)} click(){if(!this.disabled)this.listeners.get('click')?.()}
    }
    const elements=new Map(['#files-refresh','#files-previous','#files-next','#files-status','#files-list','#files-breadcrumbs'].map(k=>[k,new Element()]));
    let resolveFolder; let previews=0,downloads=0; const calls=[];
    const context={document:{querySelector:k=>elements.get(k),createElement:()=>new Element()},clearFilePreview(){}, FilesExplorerError: class extends Error {}, fetch: () => {},
        listTenantFilesPage:(path,after)=>{calls.push(path); if(path==='folder') return new Promise(r=>{resolveFolder=r}); return Promise.resolve({entries:[{name:'folder',kind:'directory',size:0},{name:'shared.txt',kind:'file',size:1}],next_after:''});},
        showFilePreview:()=>{previews++},downloadFile:()=>{downloads++}};
    runInNewContext(source+"\nfilesRefresh.addEventListener('click', () => void refreshFiles());",context);
    const tick=()=>new Promise(r=>setTimeout(r,0)); elements.get('#files-refresh').click(); await tick();
    const rows=elements.get('#files-list').children; const open=rows[0].children[2],download=rows[1].children[2],preview=rows[1].children[3];
    open.click(); assert.equal(elements.get('#files-list').children.length,0);
    download.click(); preview.click(); open.click(); assert.equal(downloads,0); assert.equal(previews,0); assert.deepEqual(calls,['','folder']);
    resolveFolder({entries:[],next_after:''}); await tick();
});

test('in-flight download does not act after navigation', async () => {
    const {runInNewContext}=await import('node:vm');
    const app=await readFile(new URL('../../cmd/tongd/ui/app.js',import.meta.url),'utf8');
    const code=app.slice(app.indexOf('async function downloadFile('),app.indexOf("filesRefresh?.addEventListener('click'"));
    let resolve; let created=0, clicked=0; const requested=[];
    const context={downloadTenantFile:path=>{requested.push(path);return new Promise(r=>{resolve=r})},
        URL:{createObjectURL:()=>{created++;return 'blob:test'}},window:{setTimeout(){}},
        document:{body:{append(){}},createElement:()=>({click(){clicked++},remove(){}})},clearFilePreview(){}};
    const api=runInNewContext("let filesRequestID=1; let filesPath=['old']; const filesStatus={dataset:{}};\n"+code+"\n({downloadFile,navigate:()=>{filesRequestID++;filesPath=['new'];}})",context);
    const promise=api.downloadFile('same.txt',{disabled:false},'old'); api.navigate(); resolve({name:'same.txt',blob:{}}); await promise;
    assert.deepEqual(requested,['old/same.txt']); assert.equal(created,0); assert.equal(clicked,0);
});

test('prefix filtering is encoded and limited before request', async () => {
    const { listTenantFilesPage } = await import('../../cmd/tongd/ui/files-explorer.mjs');
    let url; await listTenantFilesPage('dir','',async u => {url=u; return jsonResponse({entries:[],next_after:''})},undefined,'Ab é');
    assert.equal(url,'/api/v1/files?path=dir&page=1&after=&prefix=Ab%20%C3%A9');
    await assert.rejects(listTenantFilesPage('','',async () => {throw Error('should not fetch')},undefined,'bad/name'), /prefix is invalid/);
    await assert.rejects(listTenantFilesPage('','',async () => {throw Error('should not fetch')},undefined,'é'.repeat(65)), /prefix is invalid/);
});
