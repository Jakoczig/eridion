const test = require('node:test')
const assert = require('node:assert/strict')
const fs = require('node:fs')
const Module = require('node:module')
const { transformSync } = require('next/dist/compiled/babel/core')
const mod = new Module(__filename, module)
mod._compile(transformSync(fs.readFileSync('src/client/lib/go-galaxy-policy.js', 'utf8'), { babelrc: false, configFile: false, plugins: [require('next/dist/compiled/babel/plugin-transform-modules-commonjs')] }).code, __filename)
const { remainingRoute, automaticFrame, localScale, adjustAxis } = mod.exports
const point = (name, x) => ({ name, coordinates: [x, 20, 30] })
test('remaining route excludes completed legs and preserves unknown coordinates', () => {
  const context = { points: [point('old', -20000), point('now', 500), point('next', 550)] }
  const route = { hops: [{ name: 'old' }, { name: 'now', isCurrent: true }, { name: 'next', isNext: true }] }
  const remaining = remainingRoute(context, route)
  assert.equal(remaining.points.length, 1)
  assert.equal(localScale(automaticFrame(context.points[1], remaining)), true)
  assert.equal(remainingRoute(context, { hops: [{ name: 'missing' }] }).complete, false)
})
test('physical hysteresis and asymmetric axis adjustments', () => {
  const frame = { x: [0, 900], y: [0, 300], z: [0, 700] }
  assert.equal(localScale(frame, 'galaxy'), false)
  assert.equal(localScale(frame, 'local'), true)
  const next = adjustAxis(frame, 'y', 1)
  assert.deepEqual(next.x, frame.x); assert.deepEqual(next.z, frame.z)
  assert.equal(next.y[1] - next.y[0], 350)
})

test('unknown positions never create a local player or route', () => {
  const full = { x: [-65000, 65000], y: [-15000, 15000], z: [-35000, 95000] }
  assert.equal(automaticFrame(null, { points: [] }, full), full)
  assert.equal(localScale(full), false)
  for (const routeState of ['unverified', 'stale', 'cleared']) assert.equal(remainingRoute({ points: [point('A', 0)] }, { routeState, hops: [{ name: 'A' }] }).count, 0)
})
test('axis ranges remain bounded and do not mutate the original frame', () => {
  const frame = { x: [-10, 10], y: [-300, 300], z: [-100, 100] }
  assert.deepEqual(adjustAxis(frame, 'x', -1), frame)
  assert.deepEqual(adjustAxis(frame, 'y', 1), frame)
  assert.equal(adjustAxis(frame, 'z', 1).z[1] - adjustAxis(frame, 'z', 1).z[0], 250)
})

test('a disabled density renderer is never selected or described as fallback', () => {
  const { harness } = require('../helpers/go-panel-harness.cjs')
  const h = harness('go-unified-galaxy', {})
  try {
    h.props.context = { current: point('Known', 10) }
    h.props.policy = { galaxyMap: false, localAtlas: true }
    const html = h.html()
    assert.doesNotMatch(html, /data-mode="galaxy"/)
    assert.doesNotMatch(html, /showing available overview/)
    h.props.policy.localAtlas = false
    assert.match(h.html(), /Map renderers disabled/)
  } finally { h.unmount() }
})
