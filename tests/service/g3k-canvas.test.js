const test = require('node:test')
const assert = require('node:assert/strict')
const fs = require('node:fs')
const path = require('node:path')
const Module = require('node:module')
const React = require('react')
const { transformSync } = require('next/dist/compiled/babel/core')

// G3K PERF-09: coalesced spatial paints, buffer reuse, exact pixel placement.

function load (file, react = React) {
  const filename = path.resolve(__dirname, '../../src/client', file)
  const code = transformSync(fs.readFileSync(filename, 'utf8'), { babelrc: false, configFile: false, presets: [[require('next/dist/compiled/babel/preset-react'), { runtime: 'automatic' }]], plugins: [require('next/dist/compiled/babel/plugin-transform-modules-commonjs')] }).code
  const mod = new Module(filename, module); mod.filename = filename; mod.paths = module.paths
  mod.require = name => name === 'react' ? react : name.startsWith('.') ? load(path.relative(path.resolve(__dirname, '../../src/client'), path.resolve(path.dirname(filename), name + '.js')), react) : require(name)
  mod._compile(code, filename)
  return mod.exports
}

test('the allocation-free painter places every point exactly where spatialPixel does', () => {
  const { paintPoints, spatialPixel } = load('components/go-spatial.js')
  const frame = { x: [-50, 50], y: [-5, 5], z: [-30, 70] }
  let seed = 7; const rand = () => { seed = (seed * 1103515245 + 12345) & 0x7fffffff; return seed / 0x7fffffff }
  const points = []
  for (let i = 0; i < 5000; i++) points.push([rand() * 140 - 70, rand() * 14 - 7, rand() * 140 - 50])
  points.push([NaN, 0, 0], [Infinity, 0, 0], null, [50, 0, 0], [-50, -5, -30])
  for (const axes of ['xz', 'xy', 'yz']) {
    for (const [w, h] of [[640, 512], [1, 1], [2048, 37]]) {
      const expected = new Uint8ClampedArray(w * h * 4); let n = 0
      for (const p of points) { const at = spatialPixel(p, axes, frame, w, h); if (!at) continue; n++; const i = (at[1] * w + at[0]) * 4; expected[i] = 168; expected[i + 1] = 231; expected[i + 2] = 253; expected[i + 3] = Math.min(255, expected[i + 3] + 130) }
      const actual = new Uint8ClampedArray(w * h * 4)
      assert.equal(paintPoints(actual, w, h, points, axes, frame), n)
      assert.deepEqual(actual, expected, `${axes} ${w}x${h}`)
    }
  }
})

function canvasHarness () {
  const frames = []; let nextId = 1; const cancelled = []
  const visibility = new Set()
  global.window = { devicePixelRatio: 1, requestAnimationFrame: f => { const id = nextId++; frames.push({ id, f }); return id }, cancelAnimationFrame: id => cancelled.push(id), ResizeObserver: class { constructor (cb) { this.cb = cb; observers.push(this) } observe () {} disconnect () { this.off = true } } }
  const observers = []
  global.document = { visibilityState: 'visible', addEventListener: (n, f) => visibility.add(f), removeEventListener: (n, f) => visibility.delete(f) }
  const created = []; let puts = 0
  const canvas = { width: 0, height: 0, dataset: {}, getBoundingClientRect: () => ({ width: 200, height: 100 }), getContext: () => ({ createImageData: (w, h) => { const img = { width: w, height: h, data: new Uint8ClampedArray(w * h * 4) }; created.push(img); return img }, putImageData: () => { puts++ } }) }
  const effects = []
  const react = { ...React, useRef: () => ({ current: canvas }), useEffect: (cb, deps) => effects.push({ cb, deps }) }
  const { SpatialCanvas } = load('components/go-spatial.js', react)
  return {
    canvas,
    frames,
    cancelled,
    created,
    observers,
    get puts () { return puts },
    mount (props) { effects.length = 0; SpatialCanvas(props); return effects[0].cb() },
    run () { const due = frames.splice(0); for (const f of due) if (!this.cancelled.includes(f.id)) f.f() },
    hide (v) { global.document.visibilityState = v ? 'hidden' : 'visible'; for (const f of visibility) f() }
  }
}

test('paint requests coalesce into one frame, reuse the buffer, skip hidden documents and cancel on unmount', () => {
  const h = canvasHarness()
  const frame = { x: [0, 10], y: [0, 10], z: [0, 10] }
  const cleanup = h.mount({ points: [[1, 1, 1], [2, 2, 2]], axes: 'xz', frame, width: 10, height: 10 })
  h.observers[0].cb(); h.observers[0].cb() // resize callbacks before the frame
  assert.equal(h.frames.length, 1, 'one pending frame for several triggers')
  h.run()
  assert.equal(h.puts, 1); assert.equal(h.created.length, 1)
  assert.equal(h.canvas.dataset.positions, '2')
  h.observers[0].cb(); h.run()
  assert.equal(h.puts, 2); assert.equal(h.created.length, 1, 'the ImageData is reused while the size is unchanged')
  h.hide(true); h.observers[0].cb(); h.run()
  assert.equal(h.puts, 2, 'a hidden document is not painted')
  h.hide(false); h.run()
  assert.equal(h.puts, 3, 'becoming visible repaints once')
  h.observers[0].cb()
  cleanup()
  h.run()
  assert.equal(h.puts, 3, 'a pending paint is cancelled on unmount')
  assert.ok(h.observers[0].off)
})

test('route lookup memoizes by snapshot identity and keeps first-match names', () => {
  const { remainingRoute } = load('lib/go-galaxy-policy.js')
  const context = { points: [{ name: 'A', coordinates: [1, 2, 3] }, { name: 'B', coordinates: [4, 5, 6] }, { name: 'A', coordinates: [9, 9, 9] }] }
  const route = { routeState: 'verified', hops: [{ name: 'A', isCurrent: true }, { name: 'B', isNext: true }, { name: 'A' }, { name: 'Unknown' }] }
  const r = remainingRoute(context, route)
  assert.deepEqual(r.rawPoints.map(p => p?.coordinates), [[4, 5, 6], [1, 2, 3], undefined], 'first point per name, unknown stays unknown')
  assert.equal(r.complete, false); assert.equal(r.count, 3)
  assert.equal(remainingRoute(context, route), r, 'same snapshots reuse the result')
  assert.notEqual(remainingRoute({ ...context }, route), r, 'a new context snapshot recomputes')
  assert.deepEqual(remainingRoute(null, route).rawPoints, [undefined, undefined, undefined])
})
