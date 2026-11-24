const test = require('node:test')
const assert = require('node:assert/strict')
const fs = require('node:fs')
const path = require('node:path')
const Module = require('node:module')
const React = require('react')
const { renderToStaticMarkup } = require('react-dom/server')
const { transformSync } = require('next/dist/compiled/babel/core')

// Task 18 / F22: BioScan replies are applied only when they are the newest
// answer for the commander connection, body, query and rule revision that are
// currently shown. Deferred promises and mock timers; no real provider calls.

function deferred () { const d = {}; d.promise = new Promise((resolve, reject) => { d.resolve = resolve; d.reject = reject }); return d }

function harness (initialBody = '5') {
  const states = []; const effects = []; const pending = new Map(); const listeners = new Map(); const requests = []
  let cursor = 0
  const react = {
    ...React,
    useState: initial => { const i = cursor++; if (!(i in states)) states[i] = typeof initial === 'function' ? initial() : initial; return [states[i], next => { states[i] = typeof next === 'function' ? next(states[i]) : next }] },
    useRef: initial => { const i = cursor++; if (!(i in states)) states[i] = { current: initial }; return states[i] },
    useEffect: (callback, deps) => { const i = cursor++; if (!effects[i] || deps.some((v, j) => v !== effects[i].deps[j])) pending.set(i, { callback, deps }) }
  }
  const filename = path.resolve(__dirname, '../../src/client/components/go-field-operations.js')
  const code = transformSync(fs.readFileSync(filename, 'utf8'), { babelrc: false, configFile: false, presets: [[require('next/dist/compiled/babel/preset-react'), { runtime: 'automatic' }]], plugins: [require('next/dist/compiled/babel/plugin-transform-modules-commonjs')] }).code
  const mod = new Module(filename, module); mod.filename = filename; mod.paths = module.paths
  mod.require = name => name === 'react' ? react : require(name)
  mod._compile(code, filename)
  const { BioScanPanel } = mod.exports
  const props = {
    connected: true,
    ready: true,
    body: initialBody,
    sendEvent: (name, message) => { const d = deferred(); requests.push({ name, message, ...d }); return d.promise },
    eventListener: (name, callback) => { listeners.set(name, callback); return () => listeners.delete(name) }
  }
  const render = () => { cursor = 0; return renderToStaticMarkup(BioScanPanel(props)) }
  const settle = () => new Promise(resolve => setImmediate(resolve))
  return {
    props,
    requests,
    html: render,
    async flush () { render(); for (const [i, e] of pending) { effects[i]?.cleanup?.(); effects[i] = { deps: e.deps, cleanup: e.callback() } } pending.clear(); await settle(); return render() },
    settle,
    emit (name) { listeners.get(name)?.() },
    unmount () { for (const e of effects) e?.cleanup?.(); pending.clear() }
  }
}

const reply = (body, label, extra = {}) => ({ version: 'bioscan-native-2', body, signals: 1, candidates: [{ symbol: label, name: label, state: 'possible by known criteria', inference: 'possible' }], samples: [], ...extra })

test('BioScan: an older reply never replaces a newer one after a refresh while pending', async t => {
  t.mock.timers.enable({ apis: ['setTimeout', 'setInterval'] })
  const h = harness()
  try {
    await h.flush()
    assert.equal(h.requests.length, 1)
    const first = h.requests[0]
    // A journal change arrives while request one is pending.
    h.emit('journalSummaryChanged'); t.mock.timers.tick(250); await h.settle()
    // Whatever the scheduling, resolve the newest request before the oldest.
    for (let i = h.requests.length - 1; i >= 1; i--) { h.requests[i].resolve(reply('5', 'NEWER')); await h.settle() }
    first.resolve(reply('5', 'OLDER')); await h.settle()
    // A coalesced follow-up may run after the old reply; answer it as newest.
    while (h.requests.some(r => !r.done)) {
      const open = h.requests.filter(r => !r.answered)
      if (!open.length) break
      for (const r of open) { r.answered = true; r.resolve(reply('5', 'NEWER')) }
      await h.settle()
    }
    const html = h.html()
    assert.match(html, /NEWER/, 'the newest reply is shown')
    assert.doesNotMatch(html, /OLDER/, 'an older reply replaced newer information')
    assert.ok(h.requests.filter(r => r.name === 'getGoBioScan').length <= 3, 'refreshes are coalesced, not stacked')
  } finally { h.unmount() }
})

test('BioScan: at most one request is in flight; repeated refreshes coalesce', async t => {
  t.mock.timers.enable({ apis: ['setTimeout', 'setInterval'] })
  const h = harness()
  try {
    await h.flush()
    for (let i = 0; i < 5; i++) { h.emit('journalSummaryChanged'); t.mock.timers.tick(250); t.mock.timers.tick(1500); await h.settle() }
    assert.equal(h.requests.length, 1, 'no overlapping BioScan requests while one is pending')
    h.requests[0].resolve(reply('5', 'FIRST')); await h.settle()
    assert.equal(h.requests.length, 2, 'one coalesced follow-up for the invalidations received meanwhile')
    h.requests[1].resolve(reply('5', 'LATEST')); await h.settle()
    assert.match(h.html(), /LATEST/)
  } finally { h.unmount() }
})

test('BioScan: a reply for a previous body, connection or another body is discarded', async t => {
  t.mock.timers.enable({ apis: ['setTimeout', 'setInterval'] })
  const h = harness('5')
  try {
    await h.flush()
    const forFive = h.requests[0]
    h.props.body = '6'; await h.flush()
    forFive.resolve(reply('5', 'BODY_FIVE')); await h.settle()
    assert.doesNotMatch(h.html(), /BODY_FIVE/, 'rapid body change: old body data shown')
    const forSix = h.requests.at(-1)
    assert.deepEqual(forSix.message, { body: '6' })
    forSix.resolve(reply('5', 'MISMATCHED')); await h.settle()
    assert.doesNotMatch(h.html(), /MISMATCHED/, 'a reply naming another body is not applied')
    // Disconnect and reconnect: the reply to the old connection is dropped.
    const next = h.requests.at(-1); if (next !== forSix) next.resolve(reply('6', 'SIX'))
    await h.settle()
    h.emit('journalSummaryChanged'); t.mock.timers.tick(250); await h.settle()
    const beforeDrop = h.requests.at(-1)
    h.props.connected = false; await h.flush()
    h.props.connected = true; await h.flush()
    beforeDrop.resolve(reply('6', 'OLD_CONNECTION')); await h.settle()
    assert.doesNotMatch(h.html(), /OLD_CONNECTION/, 'reply from before the reconnect applied')
    h.requests.at(-1).resolve(reply('6', 'RECONNECTED')); await h.settle()
    assert.match(h.html(), /RECONNECTED/)
  } finally { h.unmount() }
})

test('BioScan: a reply after unmount or from another rule revision is not applied', async t => {
  t.mock.timers.enable({ apis: ['setTimeout', 'setInterval'] })
  const h = harness()
  await h.flush()
  h.requests[0].resolve(reply('5', 'CURRENT')); await h.settle()
  assert.match(h.html(), /CURRENT/)
  h.emit('journalSummaryChanged'); t.mock.timers.tick(250); await h.settle()
  // The rule revision is fixed per connection: a reply carrying another
  // revision answers a different context and is not applied.
  h.requests.at(-1).resolve(reply('5', 'OTHER_RULES', { version: 'bioscan-native-10' })); await h.settle()
  assert.match(h.html(), /CURRENT/); assert.doesNotMatch(h.html(), /OTHER_RULES/)
  h.emit('journalSummaryChanged'); t.mock.timers.tick(250); await h.settle()
  const late = h.requests.at(-1)
  h.unmount()
  late.resolve(reply('5', 'AFTER_UNMOUNT')); await h.settle()
  t.mock.timers.tick(5000); await h.settle()
  const count = h.requests.length
  t.mock.timers.tick(5000); await h.settle()
  assert.equal(h.requests.length, count, 'no polling after unmount')
})

test('BioScan: a stale reply keeps the shown view and triggers one fresh read', async t => {
  t.mock.timers.enable({ apis: ['setTimeout', 'setInterval'] })
  const h = harness()
  try {
    await h.flush()
    h.requests[0].resolve(reply('5', 'SHOWN')); await h.settle()
    h.emit('journalSummaryChanged'); t.mock.timers.tick(250); await h.settle()
    const count = h.requests.length
    h.requests.at(-1).resolve({ version: 'bioscan-native-2', body: '5', candidates: [], samples: [], stale: true }); await h.settle()
    assert.match(h.html(), /SHOWN/, 'a stale reply emptied the view')
    assert.equal(h.requests.length, count + 1, 'a stale reply is followed by one fresh read')
    h.requests.at(-1).resolve(reply('5', 'FRESH')); await h.settle()
    assert.match(h.html(), /FRESH/)
  } finally { h.unmount() }
})
