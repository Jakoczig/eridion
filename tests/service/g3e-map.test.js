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
// D1: an explicit lookup is the same on paired and main displays; disabled
// consent still prevents external requests.
test('search is local first and never requests external data with disabled consent', async () => {
  const h = harness('go-unified-galaxy', {})
  try {
    const calls = []
    const send = async name => { calls.push(name); return name === 'getGoSystem' ? { point: {}, community: { enabled: true } } : { point: { name: 'Found', coordinates: [1, 2, 3] } } }
    await h.exports.searchSystem(send, 'Target', 'owner')
    assert.deepEqual(calls, ['getGoSystem', 'lookupGoSystem'])
    calls.length = 0; await h.exports.searchSystem(send, 'Target', 'viewer'); assert.deepEqual(calls, ['getGoSystem', 'lookupGoSystem'])
    calls.length = 0; await h.exports.searchSystem(send, '   ', 'owner'); assert.deepEqual(calls, [])
    calls.length = 0; await h.exports.searchSystem(async name => { calls.push(name); return { point: {}, availability: 'unavailable', community: { enabled: false } } }, 'Unknown', 'owner'); assert.deepEqual(calls, ['getGoSystem'])
    // Task 4 / F05: a named local hit without coordinates is not a usable
    // result; with consent it continues to the authorized lookup.
    calls.length = 0; await h.exports.searchSystem(async name => { calls.push(name); return { point: { name: 'Known' }, community: { enabled: true } } }, 'Known', 'owner'); assert.deepEqual(calls, ['getGoSystem', 'lookupGoSystem'])
    calls.length = 0; await h.exports.searchSystem(async name => { calls.push(name); return { point: { name: 'Known' }, community: { enabled: false } } }, 'Known', 'viewer'); assert.deepEqual(calls, ['getGoSystem'])
    calls.length = 0; await h.exports.searchSystem(async name => { calls.push(name); return { point: { name: 'Known', coordinates: [1, 2, 3] }, community: { enabled: true } } }, 'Known', 'owner'); assert.deepEqual(calls, ['getGoSystem'])
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

test('G3F camera ignores slice thickness while fixed panels use their full SVG area', () => {
  const h = harness('go-galaxy-projection', null, 'Projection')
  try {
    const selection = { x: [-254, 254], y: [-58.5, 58.5], z: [-254, 254] }
    const camera = h.exports.mapCameraFrame(selection)
    assert.deepEqual(camera.x, [-254, 254])
    assert.deepEqual(camera.z, [-254, 254])
    assert.deepEqual(camera.y, selection.y)
    const thicker = h.exports.mapCameraFrame({ ...selection, y: [-300, 300] })
    assert.deepEqual(thicker.x, camera.x)
    assert.deepEqual(thicker.z, camera.z)
    const uneven = h.exports.mapCameraFrame({ ...selection, x: [-600, 600] })
    assert.equal(uneven.x[1] - uneven.x[0], uneven.z[1] - uneven.z[0])
    Object.assign(h.props, { axes: 'xz', frame: camera, selectionFrame: selection, localGeometry: false, clipSelection: false })
    assert.match(h.html(), /viewBox="0 0 100 100"/)
    assert.doesNotMatch(h.html(), /<clipPath/)
    h.props.axes = 'xy'
    assert.match(h.html(), /viewBox="0 0 100 8"/)
    h.props.axes = 'yz'
    assert.match(h.html(), /viewBox="0 0 8 100"/)
  } finally { h.unmount() }
})

test('G3F XYZ controls change the selected volume without changing the camera or moving markers', () => {
  const h = harness('go-unified-galaxy', {})
  try {
    const selection = { x: [-254, 254], y: [-58.5, 58.5], z: [-254, 254] }
    const first = h.exports.adjustSlice(selection, null, 'x', 1)
    const camera = first.camera
    assert.equal(first.selection.x[1] - first.selection.x[0], 608)
    assert.deepEqual(camera.x, selection.x)
    assert.deepEqual(camera.z, selection.z)
    const second = h.exports.adjustSlice(first.selection, camera, 'y', 1)
    const third = h.exports.adjustSlice(second.selection, camera, 'z', -1)
    assert.strictEqual(second.camera, camera)
    assert.strictEqual(third.camera, camera)
    assert.deepEqual(third.camera.x, selection.x)
    assert.deepEqual(third.camera.z, selection.z)
    assert.notDeepEqual(third.selection, selection)
    assert.equal(third.selection.y[1] - third.selection.y[0], 167)
    // Local requests still use the exact selected bounds; no camera extent
    // substitution is needed in the signed spatial backend.
  } finally { h.unmount() }
})

// Task 3 / F04: the six XYZ controls stay while Local data are pending or
// incomplete, the backend reason is shown, one inverse edit recovers, and a
// late reply for a superseded slice never renders. The camera never moves.
test('XYZ controls survive pending and over-budget Local data; stale slices never render', async () => {
  const h = harness('go-unified-galaxy', {})
  const wait = ms => new Promise(resolve => setTimeout(resolve, ms))
  const requests = []
  h.props.sendEvent = async (name, message) => {
    if (name === 'getGoSpatialView') return new Promise(resolve => requests.push({ message, resolve }))
    if (name === 'getGoAtlasView') return { installed: false, pending: false }
    return {}
  }
  Object.assign(h.props, { role: 'owner', context: { current: { name: 'Fixture Alpha', coordinates: [3, 4, 5] }, points: [] }, route: { hops: [] } })
  const button = label => h.elements('button').find(b => b.props['aria-label'] === label || b.props.children === label)
  const axes = () => h.elements('button').filter(b => /^(Increase|Decrease) [XYZ] extent$/.test(b.props['aria-label'] || '')).length
  // One flush renders the new state, the next runs its effects (the 180 ms
  // request timer); then the scheduled request is issued.
  const settle = async () => { await h.flush(); await h.flush(); await wait(220); return h.flush() }
  try {
    await h.flush()
    button('Local').props.onClick(); let html = await settle()
    assert.equal(requests.length, 1, 'Local slice requested')
    assert.equal(axes(), 6, 'controls missing while pending'); assert.match(html, /Loading local systems/)
    const camera = () => h.elements('figure').length && JSON.stringify(h.elements('svg').map(s => s.props.viewBox))
    const before = camera()
    requests[0].resolve({ complete: false, pending: false, notice: 'Point budget exceeded', points: [] }); html = await h.flush()
    assert.equal(axes(), 6, 'controls removed by an incomplete reply'); assert.match(html, /Point budget exceeded/); assert.match(html, /data-mode="galaxy"/)
    // Two edits in a row: the first reply arrives late, after the second.
    button('Increase X extent').props.onClick(); await settle()
    button('Decrease X extent').props.onClick(); html = await settle()
    assert.equal(axes(), 6); assert.equal(camera(), before, 'XYZ edit moved the camera')
    const late = requests[1]; const current = requests.at(-1)
    assert.notEqual(late, current)
    current.resolve({ complete: true, pending: false, points: [[3, 4, 5, 'Fixture Alpha']] }); html = await h.flush()
    assert.match(html, /data-mode="local"/, 'one inverse edit did not recover Local data')
    assert.doesNotMatch(html, /Point budget exceeded|Loading local systems/)
    late.resolve({ complete: true, pending: false, points: [[90, 90, 90, 'Superseded'], [91, 91, 91, 'Superseded'], [92, 92, 92, 'Superseded']] }); html = await h.flush()
    assert.doesNotMatch(html, /Superseded/); assert.match(html, /data-mode="local"/)
    assert.equal(camera(), before, 'camera changed')
  } finally { h.unmount() }
})

// Task 4 / F05: results belong to the query that produced them.
test('search results are bound to their query; stale, failed and unmounted replies never act', async () => {
  const h = harness('go-unified-galaxy', {})
  const pending = []
  h.props.policy = { galaxyMap: false, localAtlas: false }
  h.props.sendEvent = (name, message) => new Promise(resolve => pending.push({ name, message, resolve }))
  const type = value => { h.elements('input')[0].props.onChange({ target: { value } }); h.html() }
  const submit = () => h.elements('form')[0].props.onSubmit({ preventDefault () {} })
  const tick = () => new Promise(resolve => setImmediate(resolve))
  const viewSystem = () => h.elements('button').some(b => b.props.children === 'View System')
  try {
    h.html()
    type('Alpha'); submit(); await tick()
    pending.shift().resolve({ point: { name: 'Alpha', address: '1', coordinates: [1, 2, 3], source: 'journal' }, community: { enabled: false } }); await tick()
    let html = h.html()
    assert.ok(viewSystem(), 'successful search not actionable'); assert.match(html, /Alpha/)
    // Editing the query invalidates the previous search result.
    type('Beta'); html = h.html()
    assert.ok(!viewSystem(), 'previous result stayed actionable after the query changed')
    // A failed search for Beta cannot leave Alpha actionable.
    submit(); await tick()
    pending.shift().resolve({ point: {}, availability: 'unavailable', community: { enabled: false } }); await tick(); html = h.html()
    assert.ok(!viewSystem()); assert.match(html, /Beta/); assert.match(html, /unavailable/i)
    // A reply for a query the user has since changed is discarded.
    type('Gamma'); submit(); await tick()
    const late = pending.shift(); type('Delta'); late.resolve({ point: { name: 'Gamma', address: '3', coordinates: [3, 3, 3] }, community: { enabled: false } }); await tick(); html = h.html()
    assert.ok(!viewSystem(), 'reply for a superseded query became actionable'); assert.doesNotMatch(html, /Distance/)
    // Stale cached community data stay labelled as stale with their date.
    type('Epsilon'); submit(); await tick()
    pending.shift().resolve({ point: { name: 'Epsilon', address: '5', coordinates: [5, 5, 5], source: 'EDSM' }, community: { enabled: true, status: 'stale', fetchedAt: '2026-01-02T03:04:05Z' } }); await tick(); html = h.html()
    assert.ok(viewSystem()); assert.match(html, /stale/i); assert.match(html, /2026-01-02/)
    // A reply arriving after unmount changes nothing and throws nothing.
    type('Zeta'); submit(); await tick()
    const orphan = pending.shift(); h.unmount()
    orphan.resolve({ point: { name: 'Zeta', coordinates: [6, 6, 6] } }); await tick()
  } finally { h.unmount() }
})

// G3H review finding 3: a different query submitted while a search is pending
// supersedes it (the earlier reply is discarded); an identical duplicate
// submission is still blocked.
test('a new query submitted while a search is pending supersedes it', async () => {
  const h = harness('go-unified-galaxy', {})
  const pending = []
  h.props.policy = { galaxyMap: false, localAtlas: false }
  h.props.sendEvent = (name, message) => new Promise(resolve => pending.push({ name, message, resolve }))
  const type = value => { h.elements('input')[0].props.onChange({ target: { value } }); h.html() }
  const submit = () => h.elements('form')[0].props.onSubmit({ preventDefault () {} })
  const tick = () => new Promise(resolve => setImmediate(resolve))
  try {
    h.html()
    type('Alpha'); submit(); submit(); await tick()
    assert.equal(pending.length, 1, 'duplicate submission not blocked')
    type('Beta'); h.html()
    assert.equal(h.elements('button').find(b => b.props.children === 'Search').props.disabled, false, 'new query cannot be submitted while the old one is pending')
    submit(); await tick()
    assert.equal(pending.length, 2, 'new query was dropped')
    assert.equal(pending[1].message.name, 'Beta')
    pending[1].resolve({ point: { name: 'Beta', coordinates: [2, 2, 2] }, community: { enabled: false } }); await tick()
    pending[0].resolve({ point: { name: 'Alpha', coordinates: [1, 1, 1] }, community: { enabled: false } }); await tick()
    const html = h.html()
    assert.match(html, /Beta/); assert.doesNotMatch(html, /<p>Alpha<\/p>/)
    assert.ok(h.elements('button').some(b => b.props.children === 'View System'))
  } finally { h.unmount() }
})
