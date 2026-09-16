import { readFile } from 'node:fs/promises'
import vm from 'node:vm'
import assert from 'node:assert/strict'
import test from 'node:test'

const source = (await readFile(new URL('../src/composables/useWorkspace.js', import.meta.url), 'utf8'))
  .replace(/^import .*\n/gm, '')
  .replace('export function useWorkspace()', 'function useWorkspace()')

function workspace(api) {
  const context = vm.createContext({
    reactive: (value) => value, onMounted() {}, onUnmounted() {}, nextTick: async () => {},
    api, FormData, sitePath: (path) => path, formatDecimal: String,
    setTimeout: () => 0, clearTimeout() {},
  })
  vm.runInContext(source + '\nthis.workspace = useWorkspace()', context)
  const actions = context.workspace
  actions.state.file = new Blob(['xlsx'])
  actions.state.file.name = 'test.xlsx'
  return actions
}
const result = (sheet, valid = true) => ({ valid, row_count: 1, result: { sheet, sheets: ['A', 'B'], rows: [], errors: valid ? [] : ['错误'] } })

test('multi-sheet preview requires every selected sheet to pass; empty selection is rejected', async () => {
  const calls = []
  const actions = workspace(async (_, options) => {
    const sheet = options.body.get('sheet')
    calls.push(sheet)
    return result(sheet, sheet !== 'B')
  })
  actions.state.selectedSheets = ['A', 'B']
  await actions.preview()
  assert.deepEqual(calls, ['A', 'B'])
  assert.equal(actions.state.previews.length, 2)
  assert.equal(actions.state.valid, false)
  actions.state.selectedSheets = []
  await actions.preview()
  assert.match(actions.state.uploadError, /至少选择/)
  assert.equal(calls.length, 2)
})

test('partial upload failure retries only unfinished sheets', async () => {
  const calls = []
  let fail = true
  const actions = workspace(async (path, options) => {
    if (path.startsWith('/api/uploads?')) return { items: [] }
    const sheet = options.body.get('sheet')
    calls.push(sheet)
    if (sheet === 'B' && fail) throw new Error('模拟失败')
    return { inserted_rows: 1, updated_rows: 0 }
  })
  let closed = false
  actions.dialogRef('upload', { close() { closed = true } })
  Object.assign(actions.state, { selectedSheets: ['A', 'B'], valid: true, previews: [result('A'), result('B')] })
  await actions.upload()
  assert.deepEqual([...actions.state.selectedSheets], ['B'])
  assert.match(actions.state.uploadError, /已完成 1 个工作表/)
  assert.equal(closed, false)
  fail = false
  await actions.upload()
  assert.deepEqual(calls, ['A', 'B', 'B'])
  assert.equal(closed, true)
})

test('stale previews cannot overwrite a newer sheet selection', async () => {
  let resolveOld
  const actions = workspace(async (_, options) => {
    const sheet = options.body.get('sheet')
    if (sheet === 'A') return new Promise((resolve) => { resolveOld = resolve })
    return result(sheet)
  })
  actions.state.selectedSheets = ['A']
  const old = actions.preview()
  actions.state.selectedSheets = ['B']
  await actions.preview()
  resolveOld(result('A'))
  await old
  assert.deepEqual([...actions.state.selectedSheets], ['B'])
  assert.equal(actions.state.previews[0].result.sheet, 'B')
})
