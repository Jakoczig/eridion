const test = require('node:test')
const assert = require('node:assert/strict')
const { harness } = require('../helpers/go-panel-harness.cjs')
test('fixed panel layout and padded common frame preserve all axis scales', () => {
  const h = harness('go-galaxy-projection', null, 'Projection')
  try {
    const { panelLayout, fittedFrame } = h.exports
    const layout = panelLayout(1000, 700)
    assert.equal(layout.gridTemplateColumns, '100fr 8fr')
    assert.equal(layout.gridTemplateRows, '100fr 8fr')
    const f = fittedFrame({ x: [-100, 100], y: [-50, 50], z: [-200, 200] })
    assert.equal((f.x[1] - f.x[0]) / 100, (f.y[1] - f.y[0]) / 8)
    assert.equal(f.x[1] - f.x[0], f.z[1] - f.z[0])
    Object.assign(h.props, { axes: 'yz', frame: f, localGeometry: true })
    assert.match(h.html(), /preserveAspectRatio="xMidYMid meet"/)
  } finally { h.unmount() }
})
test('search is local first and never requests external data for viewers or disabled consent', async () => {
  const h = harness('go-unified-galaxy', {})
  try {
    const calls = []
    const send = async name => { calls.push(name); return name === 'getGoSystem' ? { point: {}, community: { enabled: true } } : { point: { name: 'Found', coordinates: [1, 2, 3] } } }
    await h.exports.searchSystem(send, 'Target', 'owner')
    assert.deepEqual(calls, ['getGoSystem', 'lookupGoSystem'])
    calls.length = 0; await h.exports.searchSystem(send, 'Target', 'viewer'); assert.deepEqual(calls, ['getGoSystem'])
    calls.length = 0; await h.exports.searchSystem(send, '   ', 'owner'); assert.deepEqual(calls, [])
    calls.length = 0; await h.exports.searchSystem(async name => { calls.push(name); return { point: {}, availability: 'unavailable', community: { enabled: false } } }, 'Unknown', 'owner'); assert.deepEqual(calls, ['getGoSystem'])
    calls.length = 0; await h.exports.searchSystem(async name => { calls.push(name); return { point: { name: 'Known' }, community: { enabled: true } } }, 'Known', 'owner'); assert.deepEqual(calls, ['getGoSystem'])
    assert.match(h.html(), />Search<\/button>/)
    assert.doesNotMatch(h.html(), /Lookup with enabled EDSM|Search local \/ cached/)
  } finally { h.unmount() }
})

test('search form rejects whitespace and synchronously blocks duplicate submissions', async () => {
  const h = harness('go-unified-galaxy', {})
  let finish; const calls = []
  h.props.policy = { galaxyMap: false, localAtlas: false }
  h.props.sendEvent = name => { calls.push(name); return new Promise(resolve => { finish = resolve }) }
  try {
    h.html()
    assert.equal(h.elements('button').find(b => b.props.children === 'Search').props.disabled, true)
    h.elements('input')[0].props.onChange({ target: { value: '   ' } }); h.html()
    assert.equal(h.elements('button').find(b => b.props.children === 'Search').props.disabled, true)
    h.elements('input')[0].props.onChange({ target: { value: ' Known ' } }); h.html()
    const form = h.elements('form')[0]
    form.props.onSubmit({ preventDefault () {} }); form.props.onSubmit({ preventDefault () {} })
    assert.deepEqual(calls, ['getGoSystem'])
    finish({ point: { name: 'Known', coordinates: [1, 2, 3] } }); await new Promise(resolve => setImmediate(resolve)); h.html()
    assert.equal(h.elements('button').find(b => b.props.children === 'Search').props.disabled, false)
  } finally { h.unmount() }
})

test('padding preserves projected Euclidean distances and canonical marker circles', () => {
  const h = harness('go-galaxy-projection', null, 'Projection')
  try {
    const f = h.exports.fittedFrame({ x: [-100, 100], y: [-50, 50], z: [-200, 200] })
    Object.assign(h.props, { axes: 'xy', localGeometry: true, frame: f, current: { name: 'A', coordinates: [3, 4, 0] } })
    const html = h.html()
    const match = /data-marker="current"[^>]+data-position="([^,]+),([^"]+)"/.exec(html)
    const unit = (f.x[1] - f.x[0]) / 100
    assert.ok(Math.abs(Math.hypot((Number(match[1]) - 0.5) * 100, (Number(match[2]) - 0.5) * 8) * unit - 5) < 1e-10)
    assert.match(html, /<circle/)
    assert.doesNotMatch(html, /<ellipse/)
  } finally { h.unmount() }
})

test('selection clipping changes independently and out-of-volume markers stay synchronized', () => {
  const h = harness('go-galaxy-projection', null, 'Projection')
  try {
    const selection = { x: [-100, 100], y: [-50, 50], z: [-100, 100] }
    Object.assign(h.props, { axes: 'xz', localGeometry: true, frame: h.exports.fittedFrame(selection), selectionFrame: selection, current: { name: 'Outside Y', coordinates: [0, 60, 0] } })
    const before = h.html()
    assert.doesNotMatch(before, /data-marker="current"/)
    h.props.selectionFrame = { ...selection, x: [-125, 125] }
    const after = h.html()
    assert.notEqual(/<clipPath[^>]*>(.*?)<\/clipPath>/.exec(before)[1], /<clipPath[^>]*>(.*?)<\/clipPath>/.exec(after)[1])
  } finally { h.unmount() }
})
