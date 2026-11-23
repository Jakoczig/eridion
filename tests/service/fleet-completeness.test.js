const test = require('node:test')
const assert = require('node:assert/strict')
const { harness } = require('../helpers/go-panel-harness.cjs')

// Task 15: one resolved fleet drives rows, total, global search and paging.
test('Fleet searches the whole resolved inventory on the host and shows its total and coverage', () => {
  const h = harness('go-fleet')
  const searches = []
  Object.assign(h.props, {
    page: { fleetTotal: 75, fleetMatched: 1, fleetCoverage: 'partial', fleetNext: 0, rows: [{ id: 'ship:74', entity: '74', state: 'last_known', details: { shipName: 'Ship 074', modelLabel: 'Cobra Mk III', current: false } }] },
    search: 'ship 074',
    onSearch: value => searches.push(value)
  })
  try {
    const html = h.html()
    assert.match(html, /Ship 074/)
    assert.match(html, /Search whole fleet/)
    assert.doesNotMatch(html, /Find in this page/)
    assert.match(html, /1 of 75 owned ships/)
    assert.match(html, /partial/i)
    const input = h.elements('input').find(e => e.props['aria-label'] === 'Find fleet ship')
    input.props.onChange({ target: { value: 'krait' } })
    assert.deepEqual(searches, ['krait'])
  } finally { h.unmount() }
})

test('Fleet labels Frontier-established, not-reported and current ships without inventing sales', () => {
  const h = harness('go-fleet')
  Object.assign(h.props, {
    page: {
      fleetTotal: 2,
      fleetMatched: 3,
      fleetCoverage: 'complete',
      rows: [
        { id: 'ship:1', entity: '1', state: 'known_active', details: { shipName: 'Flagship', current: true } },
        { id: 'ship:9', entity: '9', state: 'capi_last_known', details: { shipName: 'Remote', capiReference: { stale: false, fetchedAt: '2026-11-15T12:00:00Z' } } },
        { id: 'ship:3', entity: '3', state: 'not_currently_reported', details: { shipName: 'Absent' } }
      ]
    },
    fleetState: ''
  })
  try {
    const html = h.html()
    assert.match(html, /Current ship/)
    assert.match(html, /Frontier owned-fleet reference/)
    assert.match(html, /Not in the latest complete Frontier fleet/)
    assert.doesNotMatch(html, /<small>Sold<\/small>/)
  } finally { h.unmount() }
})
