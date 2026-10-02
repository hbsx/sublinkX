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
