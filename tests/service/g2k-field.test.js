const test = require('node:test')
const assert = require('node:assert/strict')
const { harness } = require('../helpers/go-panel-harness.cjs')
// D1: every trusted display has the same itinerary import, activation and
// host-copy actions as the main display.
test('G2K itinerary is explicitly separate and paired displays match the main display', async () => {
  const h = harness('go-field-operations', { state: 'off_route', completed: 1, total: 3, next: { name: 'Fixture next', neutron: true }, gameTarget: 'Different' }, 'ImportedItinerary')
  h.html(); const html = await h.flush()
  assert.match(html, /IMPORTED ITINERARY/); assert.match(html, /GAME ROUTE/); assert.match(html, /Fixture next/); assert.match(html, /Progress paused/)
  assert.match(html, /Copy next to owner PC|Import selected CSV/)
  h.props.role = 'owner'; assert.equal(h.html(), html)
  h.unmount()
})
test('G2K BGS export needs the same acknowledgement on paired and main displays', () => {
  const h = harness('go-field-operations', null, 'BGSTally')
  const paired = h.html(); assert.doesNotMatch(paired, /only to the local owner/)
  h.props.role = 'owner'; assert.equal(h.html(), paired)
  h.unmount()
})
