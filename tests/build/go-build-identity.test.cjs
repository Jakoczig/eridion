const test = require('node:test')
const assert = require('node:assert/strict')
const fs = require('node:fs')
const os = require('node:os')
const path = require('node:path')
const { execFileSync } = require('node:child_process')
const { generate, validateIdentity, ldflags } = require('../../scripts/build/go-build-identity.cjs')

function repo (t) {
  const root = fs.mkdtempSync(path.join(os.tmpdir(), 'eridion-identity-'))
  t.after(() => fs.rmSync(root, { recursive: true, force: true }))
  const git = (...a) => execFileSync('git', a, { cwd: root, encoding: 'utf8' }).trim()
  git('init', '-q')
  git('config', 'user.email', 'fixture@example.invalid')
  git('config', 'user.name', 'Fixture')
  git('config', 'core.autocrlf', 'false')
  fs.writeFileSync(path.join(root, 'package.json'), JSON.stringify({ version: '2.2.0' }))
  git('add', '-A')
  git('commit', '-q', '-m', 'fixture')
  return { root, git, head: git('rev-parse', 'HEAD') }
}

test('identity is generated from the exact checked-out revision and version policy', t => {
  const { root, head } = repo(t)
  const id = generate({ root, milestone: 'g3l', os: 'Windows', env: {} })
  assert.equal(id.revision, head)
  assert.equal(id.version, `2.2.0-g3l+${head.slice(0, 12)}`)
  assert.equal(id.executable, 'eridion.exe')
  assert.equal(id.artifact, `eridion-2.2.0-g3l-${head.slice(0, 12)}-windows-amd64-experimental-unsigned-preview`)
  assert.equal(id.ci, null)
  assert.match(ldflags(id), new RegExp(`buildinfo\\.Revision=${head} `))
  assert.ok(!id.version.includes('g3g'), 'old G3G suffix')
})

test('a modified tree is never labelled with its parent revision', t => {
  const { root } = repo(t)
  fs.writeFileSync(path.join(root, 'package.json'), JSON.stringify({ version: '2.2.1' }))
  assert.throws(() => generate({ root, milestone: 'g3l', env: {} }), /uncommitted/)
  fs.writeFileSync(path.join(root, 'untracked.go'), 'package x')
  assert.throws(() => generate({ root, milestone: 'g3l', env: {} }), /uncommitted/)
})

test('pull request builds record the PR head and the synthetic merge commit separately', t => {
  const { root, head } = repo(t)
  const merge = 'f'.repeat(40)
  const eventPath = path.join(root, '..', path.basename(root) + '-event.json')
  t.after(() => fs.rmSync(eventPath, { force: true }))
  fs.writeFileSync(eventPath, JSON.stringify({ pull_request: { head: { sha: head } } }))
  const env = { GITHUB_ACTIONS: 'true', GITHUB_EVENT_NAME: 'pull_request', GITHUB_EVENT_PATH: eventPath, GITHUB_SHA: merge }
  const id = generate({ root, milestone: 'g3l', env })
  assert.deepEqual(id.ci, { event: 'pull_request', builtRevision: head, pullRequestHead: head, syntheticMergeCommit: merge, syntheticMergeBuilt: false })
  fs.writeFileSync(eventPath, JSON.stringify({ pull_request: { head: { sha: 'a'.repeat(40) } } }))
  assert.throws(() => generate({ root, milestone: 'g3l', env }), /not the pull request head/)
  assert.throws(() => generate({ root, milestone: 'g3l', env: { GITHUB_ACTIONS: 'true', GITHUB_EVENT_NAME: 'push', GITHUB_SHA: merge } }), /not the workflow revision/)
})

test('tampered identity records are rejected', t => {
  const { root } = repo(t)
  const id = generate({ root, milestone: 'g3l', os: 'Linux', env: {} })
  for (const mutate of [
    x => { x.version = x.version.replace('g3l', 'g3g') },
    x => { x.shortRevision = 'a'.repeat(12) },
    x => { x.artifact = 'eridion-g3e-Linux-x64-experimental-unsigned-preview' },
    x => { x.executable = 'eridion.exe' },
    x => { x.target.os = 'windows' },
    x => { x.extra = true },
    x => { x.ci = { event: 'pull_request', builtRevision: x.revision, pullRequestHead: 'b'.repeat(40), syntheticMergeCommit: 'c'.repeat(40), syntheticMergeBuilt: false } }
  ]) {
    const bad = JSON.parse(JSON.stringify(id))
    mutate(bad)
    assert.throws(() => validateIdentity(bad), /Invalid build identity/)
  }
})
