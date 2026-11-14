import assert from 'node:assert/strict';
import test from 'node:test';
import {loadSiteDatabases} from '../../cmd/tongd/ui/site-databases.mjs';
const row={TenantID:'tenant-a',SiteID:'site-a',DatabaseID:'main',DatabaseName:'tp_aaaaaaaaaaaaaaaaaaaaaaaaaa',Username:'tu_bbbbbbbbbbbbbbbbbbbbbbbbbb',CreatedAt:1};
const response=(body,status=200)=>new Response(JSON.stringify(body),{status,headers:{'Content-Type':'application/json'}});
test('loads tenant-scoped database metadata, without claiming provisioning',async()=>{
 let call;const data=await loadSiteDatabases('site-a',async(url,options)=>{call={url,options};return response({databases:[row]})});
 assert.equal(call.url,'/api/v1/sites/site-a/databases');assert.equal(call.options.credentials,'same-origin');assert.equal(call.options.cache,'no-store');
 assert.deepEqual(data,[{id:'main',name:row.DatabaseName,username:row.Username}]);
});
test('rejects malformed, excessive, mismatched and denied responses',async()=>{
 await assert.rejects(loadSiteDatabases('../bad',()=>assert.fail('unsafe')));
 for(const bad of [{...row,SiteID:'site-b'},{...row,DatabaseID:'<bad>'},{...row,DatabaseName:'../../evil'},{...row,Username:'secret'}])await assert.rejects(loadSiteDatabases('site-a',async()=>response({databases:[bad]})));
 await assert.rejects(loadSiteDatabases('site-a',async()=>response({databases:[row,{...row,DatabaseID:'other',TenantID:'tenant-b'}]})));
 await assert.rejects(loadSiteDatabases('site-a',async()=>response({databases:Array.from({length:201},()=>row)})));
 await assert.rejects(loadSiteDatabases('site-a',async()=>response({},401)),/secure browser session/);
 await assert.rejects(loadSiteDatabases('site-a',async()=>response({},403)),/cannot view/);
 await assert.rejects(loadSiteDatabases('site-a',async()=>new Response('x'.repeat(65537),{headers:{'Content-Type':'application/json'}})));
});
test('cancellation settles if transport ignores abort',async()=>{const controller=new AbortController();const pending=loadSiteDatabases('site-a',()=>new Promise(()=>{}),controller.signal);controller.abort();await assert.rejects(pending);});

test('database metadata pages are bounded, ordered and use explicit cursors',async()=>{
 const {loadSiteDatabasePage}=await import('../../cmd/tongd/ui/site-databases.mjs');
 const entries=Array.from({length:50},(_,i)=>({...row,DatabaseID:`db-${String(i).padStart(2,'0')}`}));let requested;
 const page=await loadSiteDatabasePage('site-a','',async url=>{requested=url;return response({databases:entries,next_after:'db-49'})});
 assert.equal(requested,'/api/v1/sites/site-a/databases?page=1&limit=50');assert.equal(page.next_after,'db-49');
 const next=await loadSiteDatabasePage('site-a',page.next_after,async url=>{assert.equal(url,'/api/v1/sites/site-a/databases?page=1&limit=50&after=db-49');return response({databases:[{...row,DatabaseID:'db-50'}],next_after:''})});assert.equal(next.databases[0].id,'db-50');
 await assert.rejects(loadSiteDatabasePage('site-a','db-49',async()=>response({databases:[{...row,DatabaseID:'db-01'}],next_after:''})));
 await assert.rejects(loadSiteDatabasePage('site-a','',async()=>response({databases:[row],next_after:'main'})));
});
