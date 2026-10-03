const test = require('node:test');
const assert = require('node:assert/strict');
const fs = require('node:fs');
const path = require('node:path');
const vm = require('node:vm');
const ts = require('../webs/node_modules/typescript');
function section(file,start,end) {
  const source=fs.readFileSync(path.join(__dirname,'../webs/src/views/subcription',file),'utf8');
  const begin=source.indexOf(start),finish=source.indexOf(end,begin);
  assert(begin>=0&&finish>begin,'Expected frontend function');
  return ts.transpileModule(source.slice(begin,finish),{compilerOptions:{target:ts.ScriptTarget.ES2022}}).outputText;
}
test('Editing a different subscription resets security flags and selects node IDs',()=>{
  const source=section('subs.vue','const handleEdit =','const handleDel =');
  const context={tableData:{value:[{ID:1,Name:'A',Config:'{"udp":true,"cert":true}',Nodes:[{ID:5,Name:'Same'}]},{ID:2,Name:'B',Config:'{"udp":false,"cert":false}',Nodes:[{ID:6,Name:'Same'}]}]},checkList:{value:[]},editingSubId:{value:null},SubTitle:{value:''},Subname:{value:''},oldSubname:{value:''},Clash:{value:''},Surge:{value:''},dialogVisible:{value:false},value1:{value:[]}};
  vm.createContext(context);vm.runInContext(source+'\nhandleEdit({ID:1});handleEdit({ID:2});',context);
  assert.equal(JSON.stringify(context.checkList.value),'[]');
  assert.equal(context.editingSubId.value,2);
  assert.equal(JSON.stringify(context.value1.value),'[6]');
});
test('Empty node and group lists clear stale rows',async()=>{
  const nodes=section('nodes.vue','async function getnodes()','async function GetGroups()');
  const groups=section('nodes.vue','async function GetGroups()','const handleAddNode');
  const context={getNodes:async()=>({data:[]}),GetGroup:async()=>({data:[]}),tableData:{value:[{ID:1}]},tableDataTemp:{value:[{ID:1}]},allNodes:{value:[1]},allGroupNames:{value:['Old']},RadioGroup:{value:'1'}};
  vm.createContext(context);vm.runInContext(nodes+groups,context);
  await vm.runInContext('getnodes();',context);await vm.runInContext('GetGroups();',context);
  for(const key of ['tableData','tableDataTemp','allNodes','allGroupNames'])assert.equal(context[key].value.length,0);
});
test('Batch input preserves ALPN comma and saves source type',async()=>{
  const source=section('nodes.vue','const SubmitNodeForm =','// const AddNode');
  const valid='anytls://password@example.test:443?alpn=h2,http/1.1#ALPN';
  const posted=[];
  const context={dialogMode:{value:'add'},NodeForm:{value:{Link:valid+'\nss://YWVzLTEyOC1nY206cGFzcw@example.test:443',SourceType:'proxy'}},RadioGroup:{value:'1'},SelectionNodeGroups:{value:[]},NodeGroupInput:{value:''},AddNodes:async fields=>posted.push(fields),UpdateNode:async()=>assert.fail(),ElMessage:{success(){},warning(){},error(){assert.fail('Frontend failed')}},getnodes(){},GetGroups(){},ClearInput(){}};
  vm.createContext(context);vm.runInContext(source,context);await vm.runInContext('SubmitNodeForm();',context);
  assert.equal(posted.length,2);assert.equal(posted[0].link,valid);assert.equal(posted[0].source_type,'proxy');
});


test('Node JSON export preserves metadata but excludes IDs and subscriptions',()=>{
  const source=section('nodes.vue','function exportNodeFile()','async function previewNodeFile');
  let file, downloaded;
  const selected={ID:77,Name:'Custom',Link:'https://example.test/secret',SourceType:'proxy',GroupNodes:[{ID:9,Name:'Group'}],Subscriptions:['private']};
  const context={exportScope:{value:'selected'},exportFormat:{value:'json'},multipleSelection:{value:[selected]},tableDataTemp:{value:[{Name:'Other'}]},exportDialog:{value:true},Blob,URL:{createObjectURL(blob){file=blob;return 'blob:test'},revokeObjectURL(){}},Date,setTimeout(){},document:{createElement(){return {click(){downloaded=this.download},remove(){}}},body:{appendChild(){}}},ElMessage:{warning(){assert.fail('Unexpected export rejection')}}};
  vm.createContext(context);vm.runInContext(source+'\nexportNodeFile();',context);
  return file.text().then(text=>{
    const backup=JSON.parse(text);
    assert.deepEqual(backup,{format:'sublinkx-nodes',version:1,nodes:[{name:'Custom',link:'https://example.test/secret',source_type:'proxy',groups:['Group']}]});
    assert.match(downloaded,/\.json$/);
    assert.equal(context.exportDialog.value,false);
  });
});

test('Import file selection only previews and clears input for retry',async()=>{
  const source=section('nodes.vue','async function previewNodeFile','async function confirmNodeImport');
  const requests=[];
  const context={importBusy:{value:false},importPayload:{value:null},importPreview:{value:null},importDialog:{value:false},importNodes:async payload=>{requests.push(payload);return {data:{valid:1,skipped:0,invalid:0,rows:[]}}},ElMessage:{warning(){assert.fail()},error(){assert.fail()}}};
  vm.createContext(context);vm.runInContext(source,context);
  const input={value:'backup.json',files:[{name:'backup.json',size:20,text:async()=>'{"nodes":[]}'}]};
  context.event={target:input};
  await vm.runInContext('previewNodeFile(event);',context);
  assert.equal(requests.length,1);assert.equal(requests[0].confirm,false);
  assert.equal(input.value,'');assert.equal(context.importDialog.value,true);
});
