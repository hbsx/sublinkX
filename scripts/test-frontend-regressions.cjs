const test = require('node:test');
const assert = require('node:assert/strict');
const fs = require('node:fs');
const path = require('node:path');
const vm = require('node:vm');

function section(file, start, end) {
  const source = fs.readFileSync(path.join(__dirname, '../webs/src/views/subcription', file), 'utf8');
  const from = source.indexOf(start);
  const to = source.indexOf(end, from);
  assert(from >= 0 && to > from, `${file}: expected function found`);
  return source.slice(from, to)
    .replace(/\brow: Node\b/g, 'row')
    .replace(/\bitem: Node\b/g, 'item')
    .replace(/\brow:any\b/g, 'row');
}

function contextWith(source, values) {
  const context = vm.createContext(values);
  vm.runInContext(source, context);
  return context;
}

function flush() {
  return new Promise(resolve => setImmediate(resolve));
}

test('editing after cancelled new-group input keeps the node original group', async () => {
  const source = section('nodes.vue', 'const handleAddNode =', '// const AddNode');
  const posted = [];
  const context = contextWith(source, {
    dialogMode: { value: 'add' }, Nodedialog: { value: false },
    RadioGroup: { value: '2' }, NodeGroupInput: { value: 'stale-new-group' },
    allGroupNames: { value: ['Existing'] }, SelectionNodeGroups: { value: [] },
    NodeForm: { value: {} }, SelectionNode: { value: null },
    AddNodes: async () => assert.fail('Editing should not add a node'),
    UpdateNode: async fields => posted.push(fields),
    ElMessage: { warning: () => assert.fail('Unexpected validation warning'), success() {}, error: () => assert.fail('Update failed') },
    getnodes() {}, GetGroups() {}, ClearInput() {},
  });
  context.row = { ID: 7, Name: 'Original', Link: 'ss://example', GroupNodes: [{ Name: 'Existing' }] };
  vm.runInContext('handleEditNode(row)', context);
  assert.equal(context.RadioGroup.value, '1');
  assert.equal(context.NodeGroupInput.value, '');
  assert.deepEqual(Array.from(context.SelectionNodeGroups.value), ['Existing']);
  await vm.runInContext('SubmitNodeForm()', context);
  assert.equal(posted.length, 1);
  assert.equal(posted[0].group, 'Existing');
});

test('editing a grouped node before the group list loads retains its group', async () => {
  const source = section('nodes.vue', 'async function GetGroups()', 'const SubmitNodeForm');
  const context = contextWith(source, {
    GetGroup: async () => ({ data: [] }),
    allGroupNames: { value: [] }, RadioGroup: { value: '2' },
    Nodedialog: { value: false }, Groupdialog: { value: false },
    dialogMode: { value: 'add' }, NodeGroupInput: { value: 'stale' },
    NodeForm: { value: {} }, SelectionNodeGroups: { value: [] },
    SelectionNode: { value: null },
  });
  context.row = { ID: 7, Name: 'Original', Link: 'ss://example', GroupNodes: [{ Name: 'Existing' }] };
  vm.runInContext('handleEditNode(row)', context);
  assert.equal(context.RadioGroup.value, '1');
  assert.deepEqual(Array.from(context.SelectionNodeGroups.value), ['Existing']);
  await vm.runInContext('GetGroups()', context);
  assert.equal(context.RadioGroup.value, '1');
  assert.deepEqual(Array.from(context.SelectionNodeGroups.value), ['Existing']);
});

test('node and group refresh preserve the current filter and open-dialog mode', async () => {
  const source = section('nodes.vue', 'async function getnodes()', 'const handleAddNode');
  const first = [
    { ID: 1, GroupNodes: [{ Name: 'A' }] },
    { ID: 2, GroupNodes: [{ Name: 'B' }] },
  ];
  const second = [
    { ID: 3, GroupNodes: [{ Name: 'A' }] },
    { ID: 4, GroupNodes: [{ Name: 'B' }] },
  ];
  let serverNodes = first;
  const context = contextWith(source, {
    getNodes: async () => ({ data: serverNodes }),
    GetGroup: async () => ({ data: ['A', 'B'] }),
    tableData: { value: [] }, tableDataTemp: { value: [] },
    activeName: { value: 'A' }, allNodes: { value: [] },
    allGroupNames: { value: [] }, RadioGroup: { value: '2' },
    Nodedialog: { value: true }, Groupdialog: { value: false },
  });
  await vm.runInContext('getnodes()', context);
  assert.deepEqual(Array.from(context.tableData.value, row => row.ID), [1]);
  serverNodes = second;
  await vm.runInContext('getnodes()', context);
  assert.deepEqual(Array.from(context.tableData.value, row => row.ID), [3]);
  assert.deepEqual(Array.from(context.allNodes.value), [3, 4]);
  await vm.runInContext('GetGroups()', context);
  assert.equal(context.RadioGroup.value, '2', 'async group refresh must not overwrite an open dialog choice');
  context.Nodedialog.value = false;
  await vm.runInContext('GetGroups()', context);
  assert.equal(context.RadioGroup.value, '1');
  serverNodes = null;
  await vm.runInContext('getnodes()', context);
  assert.equal(context.tableData.value.length, 0, 'non-array API response is treated as empty');
});

for (const scenario of [
  { file: 'template.vue', getter: 'gettemps', api: 'DelTemp', key: 'file', rows: [{ file: 'unused.yaml' }, { file: 'used.yaml' }] },
  { file: 'subs.vue', getter: 'getsubs', api: 'DelSub', key: 'ID', rows: [{ ID: 1 }, { ID: 2 }] },
]) {
  test(`${scenario.file} batch deletion waits for responses and keeps failed rows`, async () => {
    const getSource = section(scenario.file, `async function ${scenario.getter}()`,
      scenario.file === 'template.vue' ? 'onMounted(async()' : 'async function gettemps()');
    const deleteSource = section(scenario.file, 'const selectDel =', '// 分页显示');
    let serverRows = [...scenario.rows];
    const calls = [];
    const messages = [];
    let resolveFirst;
    let rejectSecond;
    const context = contextWith(`${getSource}\n${deleteSource}`, {
      tableData: { value: [...serverRows] },
      multipleSelection: { value: [...serverRows] },
      ElMessageBox: { confirm: async () => {} },
      ElMessage: {
        success: message => messages.push({ type: 'success', message }),
        warning: message => messages.push({ type: 'warning', message }),
        error: message => messages.push({ type: 'error', message }),
      },
      [scenario.file === 'template.vue' ? 'getTemp' : 'getSubs']: async () => ({ data: [...serverRows] }),
      [scenario.api]: payload => new Promise((resolve, reject) => {
        calls.push(payload);
        if (calls.length === 1) {
          resolveFirst = () => {
            serverRows = serverRows.filter(row => row[scenario.key] !== scenario.rows[0][scenario.key]);
            resolve({ code: '00000' });
          };
        } else {
          rejectSecond = () => reject(new Error('HTTP failure'));
        }
      }),
    });
    const pending = vm.runInContext('selectDel()', context);
    await flush();
    assert.equal(calls.length, 1);
    assert.equal(context.tableData.value.length, 2, 'rows stay visible while deletion is pending');
    assert.equal(messages.length, 0, 'success is not shown before a response');
    resolveFirst();
    await flush();
    assert.equal(calls.length, 2);
    assert.equal(context.tableData.value.length, 2, 'refresh waits until all requests settle');
    rejectSecond();
    await pending;
    assert.deepEqual(Array.from(context.tableData.value, row => row[scenario.key]), [scenario.rows[1][scenario.key]]);
    assert.equal(messages.length, 1);
    assert.equal(messages[0].type, 'warning');
    assert.match(messages[0].message, /成功 1 项，失败 1 项/);
  });
}
