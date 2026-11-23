const test = require('node:test')
const assert = require('node:assert/strict')
const { harness } = require('../helpers/go-panel-harness.cjs')

// Confirmed D0: fresh validated Frontier, then journal/Status, then Inara and
// every other provider at one equal tier. The component renders the resolved
// order from the host; it does not choose authority itself.
test('Commander renders the effective Frontier rank and never attaches journal progress from another level', async () => {
  global.window = { addEventListener () {}, removeEventListener () {} }
  const overview = {
    name: 'Fixture',
    shipName: 'FIXTURE ONE',
    shipType: 'Anaconda',
    identitySource: 'frontier',
    facts: [],
    ranks: [
      { category: 'combat', rank: '8', source: 'frontier', authority: 'frontier', fetchedAt: '2026-11-15T12:00:00Z', stale: false, progress: null },
      { category: 'trade', rank: '3', source: 'journal', authority: 'journal', observedAt: '2026-11-15T11:00:00Z', stale: false, progress: '40', progressSource: 'journal' },
      { category: 'explore', rank: '4', source: 'inara', authority: 'third-party', fetchedAt: '2026-11-15T12:00:00Z', stale: false, progress: null }
    ]
  }
  const h = harness('go-commander-overview', () => overview)
  Object.assign(h.props, { role: 'owner', workspace: { activity: { commander: { ranks: [{ category: 'Combat', rank: 5, progress: 55 }, { category: 'Trade', rank: 3, progress: 40 }] } } } })
  try {
    await h.flush()
    const html = await h.flush()
    assert.match(html, /Combat<\/h3><strong>Rank 8<\/strong>/)
    assert.doesNotMatch(html, /Rank 5/)
    assert.doesNotMatch(html, /55%/)
    assert.match(html, /Trade<\/h3><strong>Rank 3<\/strong>/)
    assert.match(html, /40%/)
    assert.match(html, /Frontier/)
    assert.match(html, /Journal/)
    assert.match(html, /inara/i)
    assert.doesNotMatch(html, /observation time unknown/)
  } finally { h.unmount(); delete global.window }
})

test('Commander falls back to journal ranks only while the host overview is unavailable', () => {
  const h = harness('go-commander-overview')
  const ranks = h.exports.commanderRanks(null, { ranks: [{ category: 'Combat', rank: 5, progress: 55 }] })
  assert.equal(ranks.length, 1); assert.equal(ranks[0].rank, 5); assert.equal(ranks[0].progress, 55)
  const resolved = h.exports.commanderRanks({ ranks: [{ category: 'combat', rank: '8', source: 'frontier', progress: null }] }, { ranks: [{ category: 'Combat', rank: 5, progress: 55 }] })
  assert.equal(resolved.length, 1); assert.equal(resolved[0].rank, '8'); assert.equal(resolved[0].progress, null)
  h.unmount()
})

test('goal totals follow fresh Frontier, then journal after expiry; global-only data never invents participation', () => {
  const h = harness('go-community-goals')
  try {
    const page = { goalMembership: { known: true, ids: [] }, current: [{ entity: '77', title: 'progress_snapshot', at: '2026-01-01', details: { fields: { CurrentTotal: '200' } } }], reference: [{ field: 'communityGoals', source: 'frontier', stale: false, eligible: true, fetchedAt: '2026-01-02', value: { global: [{ id: '77', title: 'Goal', globalTotal: '100' }], personalParticipationVerified: false } }] }
    let card = h.exports.goalCards(page)[0]
    assert.equal(card.total, '100'); assert.equal(card.subscribed, false); assert.equal(card.contribution, null); assert.equal(card.reward, undefined)
    page.reference[0].stale = true; page.reference[0].eligible = false
    card = h.exports.goalCards(page)[0]
    assert.equal(card.total, '200')
    // A verified personal Frontier contribution wins under the same order.
    page.reference[0].stale = false; page.reference[0].eligible = true
    page.reference[0].value.personalParticipationVerified = true
    page.reference[0].value.global[0].playerContribution = '5'
    page.current[0].details.fields.PlayerContribution = '3'
    assert.equal(h.exports.goalCards(page)[0].contribution, '5')
    page.reference[0].value.personalParticipationVerified = false
    assert.equal(h.exports.goalCards(page)[0].contribution, '3')
  } finally { h.unmount() }
})
