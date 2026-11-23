const test = require('node:test')
const assert = require('node:assert/strict')
const fs = require('node:fs')
const path = require('node:path')
const { harness } = require('../helpers/go-panel-harness.cjs')

const components = path.resolve(__dirname, '../../src/client/components')
const source = name => fs.readFileSync(path.join(components, name + '.js'), 'utf8')

// Confirmed D1 (Task 13 / F16): a paired display renders the same gameplay
// content and actions as the main display for the same host data.
const rpc = name => ({
  getGoCommanderOverview: { name: 'Fixture', shipName: 'FIXTURE ONE', shipType: 'Anaconda', facts: [], ranks: [{ category: 'combat', rank: '8', source: 'frontier', progress: null }], fleetCount: 3 },
  getGoConstruction: [{ marketId: '3700000001', systemAddress: '1', progress: 0.5, complete: false, failed: false, resources: [] }],
  getGoBGSTally: [{ category: 'bounty_awarded', faction: 'F', system: '1', period: 'day', unit: 'CR', amount: '5' }],
  getGoProviderGameplay: [{ id: 'frontier', state: 'last_request_succeeded', readReady: true, snapshot: { activeCommander: true, source: 'frontier/profile', fetchedAt: 'f', expiresAt: 'e', facts: [{ name: 'credits', value: '100' }] } }],
  getGoOperations: { rows: [{ id: 'ship:1', entity: '1', state: 'known_active', details: { shipName: 'Flagship', current: true } }], fleetTotal: 1, fleetMatched: 1, fleetCoverage: 'partial' }
})[name] ?? {}

for (const [panel, extra] of [['go-commander-overview', {}], ['go-colonisation', {}], ['go-provider-reference', {}], ['go-operations', { path: '/ship/fleet' }], ['go-community-goals', { page: { goalMembership: { known: true, ids: [] }, rows: [] } }]]) {
  test(`${panel}: paired display matches the main display`, async () => {
    global.window = { addEventListener () {}, removeEventListener () {} }
    const render = async role => {
      const h = harness(panel, rpc)
      Object.assign(h.props, { role, connected: true, ready: true, workspace: { activity: { commander: { ranks: [] } } }, ...extra })
      try { await h.flush(); await h.flush(); return { html: await h.flush(), calls: [...h.calls] } } finally { h.unmount() }
    }
    try {
      const main = await render('owner'); const paired = await render('viewer')
      assert.equal(paired.html, main.html)
      assert.deepEqual(paired.calls, main.calls)
      assert.doesNotMatch(paired.html, /owner desktop|local owner only|remain read-only|LOCAL OWNER ONLY/i)
      assert.ok(!paired.calls.includes('getGoProviders'), 'gameplay used the launcher provider RPC')
    } finally { delete global.window }
  })
}

test('terminal has no settings gear, sidebar or launcher management for either role', () => {
  const terminal = source('go-terminal')
  for (const forbidden of ['CompanionSettings', 'go-data-history', 'BuiltinModules', 'ProviderDiagnostics', 'remote-displays', "aria-label='Display settings'", 'Owner settings', 'READ-ONLY DISPLAY']) {
    assert.ok(!terminal.includes(forbidden), `terminal still contains ${forbidden}`)
  }
})

test('launcher owns the relocated advanced tools', () => {
  const launcher = source('go-companion')
  for (const required of ["'Advanced'", 'DataHistory', 'BuiltinModules', 'ProviderDiagnostics', 'RemoteDisplays']) assert.ok(launcher.includes(required), `launcher lacks ${required}`)
})

// Audit follow-up (Task 13 / contract 4): the main gameplay display learns a
// pending launcher commander choice through the bounded gameplay gate, never
// through the launcher's account RPC (found by the G2N browser gate).
test('terminal asks only the gameplay account gate, not the launcher account RPC', async () => {
  const h = harness('go-terminal', name => name === 'getGoAccountGate' ? { choiceRequired: false } : {})
  Object.assign(h.props, { role: 'owner', stage: 'G2E', connected: true, ready: true, path: '/nav/map' })
  try {
    h.html(); await h.flush()
    assert.ok(h.calls.includes('getGoAccountGate'), 'gate not requested: ' + h.calls.join(','))
    assert.ok(!h.calls.includes('getGoAccounts'), 'gameplay display called the launcher account RPC')
  } finally { h.unmount() }
})
