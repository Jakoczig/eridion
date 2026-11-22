const test = require('node:test')
const assert = require('node:assert/strict')
const { harness } = require('../helpers/go-panel-harness.cjs')

const exosuit = {
  state: 'known',
  controlled: 'suit',
  observedAt: '2026-11-14T10:00:02Z',
  currentLoadoutId: '4293000005',
  current: { loadoutId: '4293000005', name: 'Surveyor', suitId: '1700217809818876', suit: 'explorationsuit_class3', suitLabel: 'Artemis Suit', grade: null, mods: ['suit_increasedsprintduration'], modules: [{ slot: 'PrimaryWeapon1', moduleId: '1700217863661544', module: 'wpn_m_assaultrifle_kinetic_fauto', label: 'Karma AR-50', class: 2, mods: [] }] },
  loadouts: [],
  suits: []
}

test('Exosuit shows known equipment offline and live suit values only from a known on-foot Status', () => {
  const h = harness('go-exosuit')
  try {
    h.props.workspace = { exosuit, parity: { status: { state: 'known', onFoot: { health: 0.75, oxygen: 0.5, temperatureK: 293.5, gravityG: 0.17, selectedWeaponLabel: 'Unarmed' } }, ship: { shipType: 'anaconda', controlled: 'suit' } } }
    let html = h.html()
    assert.match(html, /Artemis Suit/); assert.match(html, /Surveyor/); assert.match(html, /Karma AR-50/)
    assert.match(html, />75 %</); assert.match(html, />50 %</); assert.match(html, /Unarmed/)
    assert.match(html, /Grade not reported/)
    // Offline / stale source: equipment stays, live claim disappears.
    h.props.workspace = { exosuit, parity: { status: { state: 'stale_source' } } }
    html = h.html()
    assert.match(html, /Artemis Suit/)
    assert.doesNotMatch(html, />75 %</)
    assert.match(html, /Live suit telemetry unavailable/)
    // Absent values stay unknown rather than 0%.
    h.props.workspace = { exosuit, parity: { status: { state: 'known', onFoot: { health: null, oxygen: null } } } }
    html = h.html()
    assert.doesNotMatch(html, />0 %</)
  } finally { h.unmount() }
})

test('Exosuit without evidence says unknown and never shows the ship as a suit', () => {
  const h = harness('go-exosuit')
  try {
    h.props.workspace = { exosuit: { state: 'unknown', loadouts: [], suits: [] }, parity: { status: { state: 'known', onFoot: null }, ship: { shipType: 'anaconda' } } }
    const html = h.html()
    assert.match(html, /No suit loadout witnessed yet/)
    assert.doesNotMatch(html, /anaconda/i)
  } finally { h.unmount() }
})

test('Commander navigation lists Exosuit as its own tab', () => {
  const navigation = require('../../src/client/lib/navigation-items.js')
  const items = (navigation.CmdrPanelNavItems || navigation.default?.CmdrPanelNavItems)('Exosuit')
  const exo = items.find(i => i.name === 'Exosuit')
  assert.ok(exo); assert.equal(exo.url, '/cmdr/exosuit'); assert.equal(exo.active, true)
})
