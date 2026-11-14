import assert from 'node:assert/strict';
import test from 'node:test';
import {loadSiteDetails} from '../../cmd/tongd/ui/site-details.mjs';
const record = {TenantID:'tenant-a',SiteID:'site-a',DocumentRoot:'sites/site-a/public',CreatedAt:1,Domains:[{Name:'example.test',Kind:'primary',OwnershipState:'verified'},{Name:'alias.example.test',Kind:'alias',OwnershipState:'pending'}]};
const response = (value,status=200) => new Response(JSON.stringify(value),{status,headers:{'Content-Type':'application/json'}});
test('reads bounded authorized site details and maps domain claims',async()=>{
    let call;const got=await loadSiteDetails('site-a',async(url,options)=>{call={url,options};return response(record)});
    assert.equal(call.url,'/api/v1/sites/site-a');assert.equal(call.options.credentials,'same-origin');assert.equal(call.options.cache,'no-store');
    assert.deepEqual(got,{siteID:'site-a',documentRoot:'sites/site-a/public',domains:[{name:'example.test',kind:'primary',state:'verified'},{name:'alias.example.test',kind:'alias',state:'pending'}]});
});
test('rejects malformed, oversized, unauthorized and cross-site data',async()=>{
    await assert.rejects(loadSiteDetails('../site',()=>assert.fail('unsafe request')));
    for(const bad of [{...record,SiteID:'site-b'},{...record,DocumentRoot:'sites/site-b/public'},{...record,Domains:[{Name:'<script>',Kind:'primary',OwnershipState:'verified'}]},{...record,Domains:[{Name:'example.test',Kind:'alias',OwnershipState:'verified'}]},{...record,Domains:Array.from({length:34},()=>record.Domains[0])}]) await assert.rejects(loadSiteDetails('site-a',async()=>response(bad)));
    await assert.rejects(loadSiteDetails('site-a',async()=>response({},401)),/secure browser session/);
    await assert.rejects(loadSiteDetails('site-a',async()=>response({},403)),/cannot view/);
    await assert.rejects(loadSiteDetails('site-a',async()=>new Response('x'.repeat(16385),{headers:{'Content-Type':'application/json'}})));
});
test('cancellation rejects even if transport ignores abort',async()=>{
    const controller=new AbortController();const pending=loadSiteDetails('site-a',()=>new Promise(()=>{}),controller.signal);controller.abort();await assert.rejects(pending);
});
