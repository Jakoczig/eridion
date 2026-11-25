const test = require('node:test')
const assert = require('node:assert/strict')
const { waitFor } = require('../../scripts/testing/g2d-visual.cjs')

test('gate polling retries only navigation races and still times out', async () => {
  let calls = 0
  const value = await waitFor(async () => {
    calls++
    if (calls === 1) throw new Error('{"code":-32000,"message":"Inspected target navigated or closed"}')
    if (calls === 2) throw new Error('Execution context was destroyed.')
    return calls >= 3 && 'ready'
  }, 2000)
  assert.equal(value, 'ready')
  await assert.rejects(waitFor(async () => { throw new Error('ReferenceError: broken is not defined') }, 2000), /ReferenceError/)
  await assert.rejects(waitFor(async () => false, 300), /timed out/)
})
