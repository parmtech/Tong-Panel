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
