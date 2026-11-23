const test = require('node:test')
const assert = require('node:assert/strict')
const fs = require('node:fs')
const path = require('node:path')
const Module = require('node:module')
const React = require('react')
const { transformSync } = require('next/dist/compiled/babel/core')

function compile (relativePath, params) {
  const filename = path.resolve(__dirname, '../../src/client', relativePath)
  const code = transformSync(fs.readFileSync(filename, 'utf8'), {
    babelrc: false,
    configFile: false,
    presets: [[require('next/dist/compiled/babel/preset-react'), { runtime: 'automatic' }]],
    plugins: [require('next/dist/compiled/babel/plugin-transform-modules-commonjs')]
  }).code
  const mod = new Module(filename, module)
  mod.filename = filename
  mod.paths = module.paths
  return { mod, factory: wrapper => { mod._compile(`module.exports = function (${params}) { const module = { exports: {} }; const exports = module.exports; ${code}\n${wrapper || ''}\nreturn module.exports }`, filename); return mod.exports } }
}

function countParses (fn) {
  const original = JSON.parse
  let count = 0
  JSON.parse = function (...args) { count++; return original.apply(this, args) }
  try { fn() } finally { JSON.parse = original }
  return count
}

function transport () {
  let state = { active: false, connected: false, ready: false }
  let effect
  const timers = new Map()
  const sockets = []
  const browser = {
    window: { location: { protocol: 'http:', host: '127.0.0.1:1', pathname: '/nav/map' }, fetch: async () => ({ ok: true, json: async () => ({ role: 'owner', prototype: { stage: 'G2E' } }) }), localStorage: { getItem: () => null }, sessionStorage: { getItem: () => null }, dispatchEvent: () => {} },
    setTimeout: (callback, delay) => { const id = {}; timers.set(id, { callback, delay }); return id },
    clearTimeout: id => timers.delete(id),
    CustomEvent: class { constructor (name, init) { this.type = name; this.detail = init?.detail } }
  }
  browser.WebSocket = class {
    static OPEN = 1
    constructor (url) { this.url = url; this.readyState = 0; this.sent = []; sockets.push(this) }
    send (raw) { this.sent.push(raw) }
    close () { this.readyState = 3; this.onclose() }
  }
  const { mod, factory } = compile('lib/socket.js', 'window, WebSocket, setTimeout, clearTimeout, CustomEvent')
  mod.require = name => {
    if (name === 'react') return { ...React, useState: initial => { state ??= initial; return [state, update => { state = typeof update === 'function' ? update(state) : update }] }, useEffect: callback => { effect ??= callback }, useContext: () => state }
    if (name === 'next/dynamic') return () => () => null
    if (name === 'next/router') return { __esModule: true, default: {}, useRouter: () => ({ pathname: '/' }) }
    if (name === 'lib/navigation-items') return require('../../src/client/lib/navigation-items')
    if (name.startsWith('components/') || name === 'lib/notification') return () => null
    return require(name)
  }
  const socket = factory()(browser.window, browser.WebSocket, browser.setTimeout, browser.clearTimeout, browser.CustomEvent)
  socket.SocketProvider({ children: null })
  return {
    socket,
    sockets,
    timers,
    get state () { return state },
    async open () {
      effect()
      await new Promise(resolve => setImmediate(resolve))
      const s = sockets.at(-1)
      s.readyState = 1
      const opened = s.onopen()
      const loading = JSON.parse(s.sent.find(raw => raw.includes('getLoadingStatus')))
      s.onmessage({ data: JSON.stringify({ requestId: loading.requestId, message: { loadingComplete: true } }) })
      await opened
      return s
    }
  }
}

test('each incoming socket frame is parsed once and replies resolve or reject as before', async () => {
  const t = transport()
  const ws = await t.open()
  const ok = t.socket.sendEvent('getFixture', { a: 1 }, 0)
  const failed = t.socket.sendEvent('getBroken', null, 0)
  const [okRequest, failedRequest] = ws.sent.slice(-2).map(raw => JSON.parse(raw))
  assert.equal(countParses(() => ws.onmessage({ data: JSON.stringify({ requestId: okRequest.requestId, name: 'getFixture', message: { value: '18446744073709551615' } }) })), 1)
  assert.deepEqual(await ok, { value: '18446744073709551615' })
  assert.equal(countParses(() => ws.onmessage({ data: JSON.stringify({ requestId: failedRequest.requestId, error: 'Synthetic failure' }) })), 1)
  await assert.rejects(failed, /Synthetic failure/)
})

test('malformed frames neither throw nor leak or resolve pending requests', async () => {
  const t = transport()
  const ws = await t.open()
  const pending = t.socket.sendEvent('getFixture', null, 1000)
  const request = JSON.parse(ws.sent.at(-1))
  const previousError = console.error
  console.error = () => {}
  try { ws.onmessage({ data: '{"requestId":' }); ws.onmessage({ data: 'null' }) } finally { console.error = previousError }
  const timer = [...t.timers.entries()].find(([, v]) => v.delay === 1000)
  assert.ok(timer, 'request timeout missing')
  timer[1].callback()
  await assert.rejects(pending, /timed out/)
  // A late reply for the timed-out request is ignored.
  ws.onmessage({ data: JSON.stringify({ requestId: request.requestId, message: 'late' }) })
})

test('activity updates keep socket state identity when activity is unchanged', async () => {
  const t = transport()
  const ws = await t.open()
  const first = t.socket.sendEvent('getA', null, 0)
  const second = t.socket.sendEvent('getB', null, 0)
  const [a, b] = ws.sent.slice(-2).map(raw => JSON.parse(raw))
  ws.onmessage({ data: JSON.stringify({ requestId: a.requestId, message: 1 }) })
  await first
  const before = t.state
  assert.equal(before.active, true)
  // Another reply while a request is still pending would previously allocate
  // a new state object with the same active flag.
  const third = t.socket.sendEvent('getC', null, 0)
  const c = JSON.parse(ws.sent.at(-1))
  ws.onmessage({ data: JSON.stringify({ requestId: b.requestId, message: 2 }) })
  await second
  assert.equal(t.state, before, 'unchanged activity replaced the socket state object')
  ws.onmessage({ data: JSON.stringify({ requestId: c.requestId, message: 3 }) })
  await third
  assert.equal(t.state.active, false)
  assert.notEqual(t.state, before)
})

test('disabled profiling adds no JSON pass to socket traffic', () => {
  class FakeSocket {}
  let handlerCalls = 0
  let stored
  FakeSocket.prototype.send = function () {}
  Object.defineProperty(FakeSocket.prototype, 'onmessage', { set (h) { stored = h }, get () { return stored }, configurable: true })
  const window = {}
  const { mod, factory } = compile('lib/performance-profiler.js', 'window, WebSocket, performance, requestAnimationFrame, cancelAnimationFrame, PerformanceObserver')
  mod.require = name => { if (name === 'next/router') throw new Error('not in Next'); return require(name) }
  const profiler = factory()(window, FakeSocket, { now: () => 1 }, () => 1, () => {}, undefined)
  const ws = new FakeSocket()
  ws.onmessage = () => { handlerCalls++ }
  const frame = JSON.stringify({ requestId: 'r1', name: 'getFixture', message: {} })
  assert.equal(countParses(() => { ws.send(frame); ws.onmessage({ data: frame }) }), 0)
  assert.equal(handlerCalls, 1)
  const log = console.log
  console.log = () => {}
  try {
    profiler.start()
    assert.equal(countParses(() => { ws.send(frame); ws.onmessage({ data: frame }) }), 2)
    assert.equal(handlerCalls, 2)
    profiler.stop()
  } finally { console.log = log }
  assert.ok(window.__perf.entries.some(e => e.type === 'ws-send'))
})

// Task 13 (audit F2/F5): the launcher window connects with the launcher
// scope, every gameplay page without it, and paired (https) displays never.
test('only the launcher page opens a launcher-scoped socket', () => {
  const t = transport()
  const url = t.socket.socketURL
  assert.equal(url({ protocol: 'http:', host: 'h:1', pathname: '/launcher' }), 'ws://h:1/?scope=launcher')
  assert.equal(url({ protocol: 'http:', host: 'h:1', pathname: '/launcher/settings' }), 'ws://h:1/?scope=launcher')
  assert.equal(url({ protocol: 'http:', host: 'h:1', pathname: '/nav/map' }), 'ws://h:1')
  assert.equal(url({ protocol: 'http:', host: 'h:1', pathname: '/launchery' }), 'ws://h:1')
  assert.equal(url({ protocol: 'https:', host: 'h:1', pathname: '/launcher' }), 'wss://h:1')
})

test('requests carry the account epoch and replies from another epoch are rejected', async () => {
  const t = transport()
  const ws = await t.open()
  assert.equal(ws.url, 'ws://127.0.0.1:1')
  ws.onmessage({ data: JSON.stringify({ name: 'accountEpoch', message: { epoch: 5 } }) })
  const stale = t.socket.sendEvent('getGoCommanderOverview', null, 0)
  const request = JSON.parse(ws.sent.at(-1))
  assert.equal(request.epoch, 5)
  ws.onmessage({ data: JSON.stringify({ requestId: request.requestId, epoch: 6, message: { fid: 'SYNTHETIC-A' } }) })
  await assert.rejects(stale, /Commander changed/)
  const fresh = t.socket.sendEvent('getGoCommanderOverview', null, 0)
  const next = JSON.parse(ws.sent.at(-1))
  ws.onmessage({ data: JSON.stringify({ requestId: next.requestId, epoch: 5, message: { fid: 'SYNTHETIC-A' } }) })
  assert.deepEqual(await fresh, { fid: 'SYNTHETIC-A' })
})

test('an epoch change on an open connection cancels requests issued for the previous commander', async () => {
  const t = transport()
  const ws = await t.open()
  ws.onmessage({ data: JSON.stringify({ name: 'accountEpoch', message: { epoch: 1 } }) })
  const queued = t.socket.sendEvent('importGoItinerary', { name: 'Queued' }, 0)
  ws.onmessage({ data: JSON.stringify({ name: 'accountEpoch', message: { epoch: 2 } }) })
  await assert.rejects(queued, /Connection interrupted/)
  const after = t.socket.sendEvent('getGoItinerary', null, 0)
  assert.equal(JSON.parse(ws.sent.at(-1)).epoch, 2)
  ws.onmessage({ data: JSON.stringify({ requestId: JSON.parse(ws.sent.at(-1)).requestId, epoch: 2, message: [] }) })
  assert.deepEqual(await after, [])
})
