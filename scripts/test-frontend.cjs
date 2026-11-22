const test = require('node:test');
const assert = require('node:assert/strict');
const fs = require('node:fs');
const path = require('node:path');
const vm = require('node:vm');
const ts = require('../webs/node_modules/typescript');
require('./test-frontend-regressions.cjs');
function section(file,start,end) {
  const source=fs.readFileSync(path.join(__dirname,'../webs/src/views/subcription',file),'utf8');
  const begin=source.indexOf(start),finish=source.indexOf(end,begin);
  assert(begin>=0&&finish>begin,'Expected frontend function');
  return ts.transpileModule(source.slice(begin,finish),{compilerOptions:{target:ts.ScriptTarget.ES2022}}).outputText;
}

function svgPluginModule() {
  const source=fs.readFileSync(path.join(__dirname,'../webs/build/svg-icons.ts'),'utf8');
  const compiled=ts.transpileModule(source,{compilerOptions:{target:ts.ScriptTarget.ES2022,module:ts.ModuleKind.CommonJS}}).outputText;
  const webRequire=require('node:module').createRequire(path.join(__dirname,'../webs/package.json'));
  const context=vm.createContext({exports:{},require:webRequire});
  vm.runInContext(compiled,context);
  return context.exports;
}

test('SVG sprites preserve viewBox and namespace IDs while stripping scripts',()=>{
  const {symbolFromSVG}=svgPluginModule();
  const output=symbolFromSVG('<?xml version="1.0"?><svg xmlns="http://www.w3.org/2000/svg" width="24" height="24"><defs><clipPath id="clip"><path d="M0 0h24v24H0z"/></clipPath></defs><path clip-path="url(#clip)" onload="bad()" d="M0 0h24"/><script>bad()</script></svg>','icon-example');
  assert.match(output,/^<symbol[^>]*id="icon-example"/);
  assert.match(output,/viewBox="0 0 24 24"/);
  assert.match(output,/id="icon-example__clip"/);
  assert.match(output,/url\(#icon-example__clip\)/);
  assert.doesNotMatch(output,/<script|onload=|<\?xml|<!DOCTYPE/);
});

test('Every repository SVG is available under its existing icon ID',()=>{
  const {symbolFromSVG}=svgPluginModule();
  const directory=path.join(__dirname,'../webs/src/assets/icons');
  const names=fs.readdirSync(directory).filter(name=>name.endsWith('.svg'));
  assert(names.length>40);
  for(const name of names){
    const id='icon-'+name.slice(0,-4);
    const output=symbolFromSVG(fs.readFileSync(path.join(directory,name),'utf8'),id);
    assert(output.startsWith('<symbol'),name);
    assert(output.includes(`id="${id}"`),name);
    assert(output.includes('viewBox='),name);
  }
});

test('Virtual SVG registration mounts a complete sprite only once',()=>{
  const {createSvgIconsPlugin}=svgPluginModule();
  const plugin=createSvgIconsPlugin(path.join(__dirname,'../webs/src/assets/icons'));
  const context={addWatchFile(){}};
  const names=JSON.parse(plugin.load.call(context,'\0virtual:svg-icons-names').replace('export default ',''));
  const script=plugin.load.call(context,'\0virtual:svg-icons-register');
  const mounted=[];
  const document={readyState:'complete',getElementById:id=>mounted.find(item=>item.id===id),createElementNS:()=>({style:{},setAttribute(){}}),body:{prepend:item=>mounted.push(item)}};
  vm.runInNewContext(script,{document});vm.runInNewContext(script,{document});
  assert.equal(mounted.length,1);
  assert.equal((mounted[0].innerHTML.match(/<symbol\b/g)||[]).length,names.length);
  assert(names.includes('icon-security')&&names.includes('icon-user'));
});

test('Subscription links explicitly select each client format',()=>{
  const source=section('subs.vue','const handleClient =','const Qrdialog');
  const context={location:{protocol:'https:',hostname:'sub.example.test',port:''},ClientDiaLog:{value:false},ClientUrl:{value:''},ClientList:['v2ray','clash','surge'],ClientUrls:{value:{}}};
  vm.createContext(context);vm.runInContext(source+'\nhandleClient({Token:"a&b"});',context);
  for(const client of context.ClientList){
    const url=new URL(context.ClientUrls.value[client]);
    assert.equal(url.searchParams.get('client'),client);
    assert.equal(url.searchParams.get('token'),'a&b');
  }
});

test('Failed node save preserves dialog and user input for retry',async()=>{
  const source=section('nodes.vue','const SubmitNodeForm =','// const AddNode');
  let cleared=0,errors=0;
  const form={Link:'ss://retry.test:443',SourceType:'proxy'};
  const context={dialogMode:{value:'add'},NodeForm:{value:form},RadioGroup:{value:'1'},SelectionNodeGroups:{value:[]},NodeGroupInput:{value:''},AddNodes:async()=>{throw Error('network failed')},UpdateNode:async()=>assert.fail(),ElMessage:{success(){assert.fail()},warning(){assert.fail()},error(){errors++}},getnodes(){},GetGroups(){},ClearInput(){cleared++}};
  vm.createContext(context);vm.runInContext(source,context);await vm.runInContext('SubmitNodeForm()',context);
  assert.equal(cleared,0);assert.equal(errors,1);assert.equal(context.NodeForm.value,form);
});

test('Group selection reads all nodes when another group is filtered',()=>{
  const source=section('nodes.vue','const  handleShownodeGroupList =','// 表格所属分组格式化');
  const context={allNodes:{value:[1,2]},SelectionNode:{value:2},SelectionNodeGroups:{value:[]},tableData:{value:[{ID:1,GroupNodes:[{Name:'Visible'}]}]},tableDataTemp:{value:[{ID:1,GroupNodes:[{Name:'Visible'}]},{ID:2,GroupNodes:[{Name:'Existing'}]}]}};
  vm.createContext(context);vm.runInContext(source+'\nhandleShownodeGroupList()',context);
  assert.deepEqual(Array.from(context.SelectionNodeGroups.value),['Existing']);
});

test('Password update preserves leading and trailing spaces',async()=>{
  const source=section('../system/user/set.vue','function resetPassword','</script>');
  const posted=[];
  const context={username:{value:' admin '},password:{value:'  a long password  '},t:x=>x,ElMessage:{error(){assert.fail()},success(){}},ElMessageBox:{confirm:async()=>{}},updateUserPassword:async fields=>posted.push(fields),userStore:{resetToken:async()=>{}},window:{location:{reload(){}}}};
  vm.createContext(context);vm.runInContext(source+'\nresetPassword({})',context);
  await new Promise(resolve=>setImmediate(resolve));
  assert.equal(posted.length,1);assert.equal(posted[0].password,'  a long password  ');assert.equal(posted[0].username,'admin');
});
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
  const context={getNodes:async()=>({data:[]}),GetGroup:async()=>({data:[]}),tableData:{value:[{ID:1}]},tableDataTemp:{value:[{ID:1}]},allNodes:{value:[1]},allGroupNames:{value:['Old']},activeName:{value:'全部'},RadioGroup:{value:'1'},Nodedialog:{value:false},Groupdialog:{value:false}};
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
