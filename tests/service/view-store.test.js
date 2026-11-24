const test = require('node:test')
const assert = require('node:assert/strict')
const fs = require('node:fs')
const path = require('node:path')
const Module = require('node:module')
const { transformSync } = require('next/dist/compiled/babel/core')

// G3K PERF-02/06: shared gameplay view store.

function loadStore () {
  const filename = path.resolve(__dirname, '../../src/client/lib/go-view-store.js')
  const code = transformSync(fs.readFileSync(filename, 'utf8'), { babelrc: false, configFile: false, plugins: [require('next/dist/compiled/babel/plugin-transform-modules-commonjs')] }).code
  const mod = new Module(filename, module); mod.filename = filename; mod.paths = module.paths
  mod._compile(code, filename)
  return mod.exports
}

function fakeDocument () {
  const listeners = new Set()
  const doc = { visibilityState: 'visible', addEventListener: (n, f) => listeners.add(f), removeEventListener: (n, f) => listeners.delete(f) }
  return { doc, hide (v) { doc.visibilityState = v ? 'hidden' : 'visible'; for (const f of listeners) f() } }
}

function transport () {
  const calls = []; const demands = []; const pending = []; const listeners = new Map()
  return {
    calls,
    demands,
    pending,
    sendEvent (name, message) {
      if (name === 'setGoDemand') { demands.push(message); return Promise.resolve({ applied: true }) }
      calls.push(name)
      return new Promise((resolve, reject) => pending.push({ name, resolve, reject }))
    },
    eventListener (name, cb) { if (!listeners.has(name)) listeners.set(name, new Set()); listeners.get(name).add(cb); return () => listeners.get(name).delete(cb) },
    emit (name, v) { for (const cb of listeners.get(name) || []) cb(v) },
    listening (name) { return (listeners.get(name) || new Set()).size }
  }
}
const tick = () => new Promise(resolve => setTimeout(resolve, 5))
const noWindow = () => { global.window = { addEventListener () {}, removeEventListener () {} } }

test('two consumers of one projection share one request and one immutable snapshot', async () => {
  const f = fakeDocument(); global.document = f.doc; noWindow()
  const { createGoViewStore } = loadStore()
  const t = transport(); const store = createGoViewStore(t)
  const a = []; const b = []
  const stopA = store.subscribe({ rpc: 'getGoNavigation', changed: 'goNavigationChanged' }, s => a.push(s))
  const stopB = store.subscribe({ rpc: 'getGoNavigation', changed: 'goNavigationChanged' }, s => b.push(s))
  assert.deepEqual(t.calls, ['getGoNavigation'], 'one shared initial request')
  assert.equal(t.listening('goNavigationChanged'), 1, 'one shared live listener')
  const value = { currentSystem: 'Fixture Alpha' }
  t.pending[0].resolve(value); await tick()
  assert.equal(a.at(-1).value, value); assert.equal(b.at(-1).value, value)
  assert.equal(a.at(-1), b.at(-1), 'both consumers hold the same immutable snapshot')
  assert.ok(Object.isFrozen(a.at(-1)))
  const count = a.length
  t.emit('goNavigationChanged', value)
  assert.equal(a.length, count, 'an identical value does not notify (no re-render)')
  const c = []
  const stopC = store.subscribe({ rpc: 'getGoShip', changed: 'goShipChanged' }, s => c.push(s))
  assert.deepEqual(t.calls, ['getGoNavigation', 'getGoShip'], 'a different projection is independent')
  t.emit('goShipChanged', { shipType: 'asp' })
  assert.equal(a.length, count, 'an unrelated projection does not re-render navigation consumers')
  stopA(); assert.equal(t.listening('goNavigationChanged'), 1)
  stopB(); assert.equal(t.listening('goNavigationChanged'), 0, 'last unsubscribe releases the listener')
  stopC(); assert.equal(store.size, 0)
  store.dispose()
})

test('a live broadcast wins over a slower initial reply; deltas merge; a released entry reply is dropped', async () => {
  const f = fakeDocument(); global.document = f.doc; noWindow()
  const { createGoViewStore } = loadStore()
  const t = transport(); const store = createGoViewStore(t)
  const seen = []
  let stop = store.subscribe({ rpc: 'getGoNavigation', changed: 'goNavigationChanged' }, s => seen.push(s.value))
  t.emit('goNavigationChanged', { currentSystem: 'Newest live' })
  t.pending[0].resolve({ currentSystem: 'Old bootstrap' }); await tick()
  assert.equal(seen.at(-1).currentSystem, 'Newest live')
  stop()
  const delta = { name: 'goFlightChanged', merge: (w, s) => ({ ...w, status: s }) }
  const ws = []
  stop = store.subscribe({ rpc: 'getGoWorkspace', changed: 'goWorkspaceChanged', delta }, s => ws.push(s.value))
  t.emit('goFlightChanged', { fuel: 17 })
  t.pending.at(-1).resolve({ parity: 1, status: { fuel: 16 } }); await tick()
  assert.equal(ws.at(-1).status.fuel, 17, 'a delta received first is merged into the initial reply')
  stop()
  const again = []
  stop = store.subscribe({ rpc: 'getGoWorkspace', changed: 'goWorkspaceChanged', delta }, s => again.push(s.value))
  const old = t.pending.at(-2); const fresh = t.pending.at(-1)
  old.resolve({ parity: 'OLD' }); fresh.resolve({ parity: 'FRESH' }); await tick()
  assert.equal(again.at(-1).parity, 'FRESH')
  assert.ok(!again.some(v => v && v.parity === 'OLD'), 'a reply for a released entry is never shown')
  stop(); store.dispose()
})

test('an account change refetches every entry for the new commander', async () => {
  const f = fakeDocument(); global.document = f.doc
  let accountListener = null
  global.window = { addEventListener: (n, cb) => { if (n === 'eridion-account-changed') accountListener = cb }, removeEventListener () {} }
  const { createGoViewStore } = loadStore()
  const t = transport(); const store = createGoViewStore(t)
  const seen = []
  const stop = store.subscribe({ rpc: 'getGoShip', changed: 'goShipChanged' }, s => seen.push(s.value))
  const first = t.pending[0]
  accountListener()
  assert.equal(seen.at(-1), null, 'old commander data cleared at once')
  first.resolve({ shipType: 'OLD_ACCOUNT' }); await tick()
  assert.equal(seen.at(-1), null, 'a reply for the previous commander is not applied')
  t.pending.at(-1).resolve({ shipType: 'NEW_ACCOUNT' }); await tick()
  assert.equal(seen.at(-1).shipType, 'NEW_ACCOUNT')
  stop(); store.dispose()
})

test('demand follows mounted views and visibility, not focus', async () => {
  const f = fakeDocument(); global.document = f.doc; noWindow()
  const { createGoViewStore } = loadStore()
  const t = transport(); const store = createGoViewStore(t)
  const s1 = store.subscribe({ rpc: 'getGoNavigation', changed: 'goNavigationChanged' }, () => {})
  const s2 = store.subscribe({ rpc: 'getGoWorkspace', changed: 'goWorkspaceChanged' }, () => {})
  await tick()
  assert.equal(t.demands.length, 1, 'subscription bursts coalesce into one replace')
  assert.deepEqual(t.demands[0].topics, ['community', 'navigation', 'providers', 'spatial', 'workspace'])
  assert.equal(t.demands[0].visible, true)
  f.hide(true); await tick()
  assert.deepEqual(t.demands.at(-1), { generation: '2', visible: false, topics: [] }, 'hidden releases view demand')
  f.hide(false); await tick()
  assert.equal(t.demands.at(-1).visible, true); assert.equal(t.demands.at(-1).generation, '3')
  assert.ok(t.demands.at(-1).topics.includes('workspace'))
  const n = t.demands.length; await tick()
  assert.equal(t.demands.length, n, 'no declaration without a visibility or view change (focus is irrelevant)')
  s2(); await tick()
  assert.ok(!t.demands.at(-1).topics.includes('workspace'), 'last consumer release drops the topic')
  s1(); await tick()
  assert.deepEqual(t.demands.at(-1).topics, ['community', 'providers', 'spatial'])
  store.dispose()
})

test('a failed demand declaration is retried; a new connection is always declared (review R5/R9)', async () => {
  const f = fakeDocument(); global.document = f.doc; noWindow()
  const { createGoViewStore } = loadStore()
  const t = transport()
  let fail = 1
  const send = t.sendEvent
  t.sendEvent = (name, message) => { if (name === 'setGoDemand' && fail-- > 0) { t.demands.push(message); return Promise.reject(new Error('Commander changed')) } return send(name, message) }
  const store = createGoViewStore(t)
  const stop = store.subscribe({ rpc: 'getGoNavigation', changed: 'goNavigationChanged' }, () => {})
  await tick()
  assert.equal(t.demands.length, 1)
  await new Promise(resolve => setTimeout(resolve, 600))
  assert.equal(t.demands.length, 2, 'the failed declaration was retried')
  assert.deepEqual(t.demands[1].topics, t.demands[0].topics)
  // Reconnect: every view unmounts, then mounts again with the same topics.
  stop(); await tick()
  const n = t.demands.length
  const again = store.subscribe({ rpc: 'getGoNavigation', changed: 'goNavigationChanged' }, () => {})
  await tick()
  assert.ok(t.demands.length > n && t.demands.at(-1).topics.includes('navigation'), 'the new connection is declared')
  again(); store.dispose()
})
