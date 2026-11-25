// G3L Task 10: one authoritative build identity record, generated from the
// exact checked-out source. The same record supplies the Go linker values,
// the packaged BUILD-IDENTITY.json, the artifact name and the Actions outputs.
//
// Usage: node scripts/build/go-build-identity.cjs --milestone g3l [--os Windows|Linux] [--arch amd64] [--out FILE]
const { execFileSync } = require('node:child_process')
const fs = require('node:fs')
const path = require('node:path')

const SCHEMA = 'eridion-build-identity/1'
const MILESTONE = /^g[0-9][a-z](-[a-z])?$/
const SHA = /^[0-9a-f]{40}$/
const OS = { Windows: 'windows', Linux: 'linux', windows: 'windows', linux: 'linux' }
const PKG = 'github.com/Jakoczig/eridion/go/internal/buildinfo'

// Version policy (correction plan F13): <package version>-<milestone>+<12-hex revision>.
function deriveVersion (base, milestone, revision) { return `${base}-${milestone}+${revision.slice(0, 12)}` }
function artifactName (id) { return `eridion-${id.baseVersion}-${id.milestone}-${id.shortRevision}-${id.target.os}-${id.target.arch}-experimental-unsigned-preview` }
function executableName (os) { return os === 'windows' ? 'eridion.exe' : 'eridion' }
function ldflags (id) { return [['Version', id.version], ['Revision', id.revision], ['Milestone', id.milestone], ['SourceDate', id.sourceDate]].map(([k, v]) => `-X ${PKG}.${k}=${v}`).join(' ') }

function validateIdentity (id) {
  const fail = message => { throw new Error(`Invalid build identity: ${message}`) }
  if (!id || typeof id !== 'object') fail('not an object')
  const keys = ['schema', 'product', 'version', 'baseVersion', 'milestone', 'revision', 'shortRevision', 'sourceDate', 'target', 'executable', 'artifact', 'ci']
  if (Object.keys(id).sort().join() !== [...keys].sort().join()) fail('unexpected fields')
  if (id.schema !== SCHEMA || id.product !== 'ERIDION') fail('schema')
  if (!/^\d+\.\d+\.\d+$/.test(id.baseVersion)) fail('base version')
  if (!MILESTONE.test(id.milestone)) fail('milestone')
  if (!SHA.test(id.revision) || id.shortRevision !== id.revision.slice(0, 12)) fail('revision')
  if (id.version !== deriveVersion(id.baseVersion, id.milestone, id.revision)) fail('version does not follow the version policy')
  if (Number.isNaN(Date.parse(id.sourceDate)) || new Date(id.sourceDate).toISOString().replace('.000Z', 'Z') !== id.sourceDate) fail('source date')
  if (!id.target || !['windows', 'linux'].includes(id.target.os) || id.target.arch !== 'amd64' || Object.keys(id.target).length !== 2) fail('target')
  if (id.executable !== executableName(id.target.os)) fail('executable name')
  if (id.artifact !== artifactName(id)) fail('artifact name')
  if (id.ci !== null) {
    if (!['pull_request', 'push', 'workflow_dispatch'].includes(id.ci.event)) fail('ci event')
    if (id.ci.builtRevision !== id.revision) fail('ci built revision')
    if (id.ci.event === 'pull_request' && (!SHA.test(id.ci.pullRequestHead) || id.ci.pullRequestHead !== id.revision || !SHA.test(id.ci.syntheticMergeCommit) || id.ci.syntheticMergeBuilt !== false)) fail('pull request provenance')
  }
  return id
}

function git (root, args) { return execFileSync('git', args, { cwd: root, encoding: 'utf8' }).trim() }

function generate ({ root, milestone, os = 'Linux', arch = 'amd64', env = process.env }) {
  if (!MILESTONE.test(milestone || '')) throw new Error('A milestone such as g3l is required')
  const revision = git(root, ['rev-parse', 'HEAD'])
  if (!SHA.test(revision)) throw new Error('Checked-out revision is not a full SHA-1')
  // Never label a modified tree with its parent revision.
  if (git(root, ['status', '--porcelain']) !== '') throw new Error('Source tree has uncommitted changes; identity refused')
  const baseVersion = JSON.parse(fs.readFileSync(path.join(root, 'package.json'), 'utf8')).version
  const sourceDate = new Date(git(root, ['show', '-s', '--format=%cI', 'HEAD'])).toISOString().replace('.000Z', 'Z')
  const target = { os: OS[os], arch: arch === 'x64' ? 'amd64' : arch }
  if (!target.os) throw new Error('Unsupported target OS')
  let ci = null
  if (env.GITHUB_ACTIONS === 'true') {
    ci = { event: env.GITHUB_EVENT_NAME, builtRevision: revision }
    if (env.GITHUB_EVENT_NAME === 'pull_request') {
      const event = JSON.parse(fs.readFileSync(env.GITHUB_EVENT_PATH, 'utf8'))
      // GitHub reports a synthetic merge commit for pull requests; the build
      // uses the PR head. Record both and refuse if the checkout differs.
      ci.pullRequestHead = event.pull_request.head.sha
      ci.syntheticMergeCommit = env.GITHUB_SHA
      ci.syntheticMergeBuilt = false
      if (ci.pullRequestHead !== revision) throw new Error('Checked-out source is not the pull request head')
    } else if (env.GITHUB_SHA !== revision) {
      throw new Error('Checked-out source is not the workflow revision')
    }
  }
  const id = { schema: SCHEMA, product: 'ERIDION', version: deriveVersion(baseVersion, milestone, revision), baseVersion, milestone, revision, shortRevision: revision.slice(0, 12), sourceDate, target, executable: executableName(target.os), artifact: '', ci }
  id.artifact = artifactName(id)
  return validateIdentity(id)
}

function main () {
  const args = process.argv.slice(2)
  const opt = name => { const i = args.indexOf(name); return i >= 0 ? args[i + 1] : undefined }
  const root = path.resolve(__dirname, '../..')
  const id = generate({ root, milestone: opt('--milestone'), os: opt('--os') || process.env.RUNNER_OS || (process.platform === 'win32' ? 'Windows' : 'Linux'), arch: opt('--arch') || 'amd64' })
  const out = path.resolve(root, opt('--out') || 'go/bin/BUILD-IDENTITY.json')
  fs.mkdirSync(path.dirname(out), { recursive: true })
  fs.writeFileSync(out, JSON.stringify(id, null, 2) + '\n')
  if (process.env.GITHUB_OUTPUT) fs.appendFileSync(process.env.GITHUB_OUTPUT, `version=${id.version}\nrevision=${id.revision}\nartifact=${id.artifact}\nldflags=${ldflags(id)}\nidentity=${out}\n`)
  console.log(JSON.stringify(id, null, 2))
}
if (require.main === module) {
  try { main() } catch (error) { console.error(error.message); process.exitCode = 1 }
}
module.exports = { SCHEMA, generate, validateIdentity, deriveVersion, artifactName, executableName, ldflags }
