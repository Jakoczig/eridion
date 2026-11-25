const fs = require('node:fs/promises')
const path = require('node:path')
const os = require('node:os')
const crypto = require('node:crypto')
const { execFileSync } = require('node:child_process')
const { validateIdentity } = require('./go-build-identity.cjs')

async function regularBytes (filename) {
  const info = await fs.lstat(filename)
  if (!info.isFile() || info.isSymbolicLink() || info.size === 0) throw new Error('Preview inputs must be nonempty regular files')
  return fs.readFile(filename)
}

async function packagePreview ({ binaryPath, readmePath, outputParent, platform, milestone = 'g2b' }) {
  if (!['g2b', 'g2c', 'g2d', 'g2e', 'g2f', 'g2g', 'g2h', 'g2j', 'g2k', 'g2l', 'g2m-a', 'g2m-b', 'g2m-c', 'g2n', 'g2o', 'g2p', 'g2q', 'g3c', 'g3d', 'g3e'].includes(milestone)) throw new Error('Unsupported preview milestone')
  if (!['Windows', 'Linux'].includes(platform)) throw new Error('Unsupported preview platform')
  // Read explicit inputs only. Never enumerate/copy a build directory or profile.
  const binary = await regularBytes(binaryPath)
  const readme = await regularBytes(readmePath)
  const descriptor = ['g2h', 'g2j', 'g2k', 'g2l', 'g2m-a', 'g2m-b', 'g2m-c', 'g2n', 'g2o', 'g2p', 'g2q', 'g3c', 'g3d', 'g3e'].includes(milestone) ? 'preview' : 'local-route-survey-no-uploads'
  const directory = await fs.mkdtemp(path.join(outputParent, `${['g2m-a', 'g2m-b', 'g2m-c', 'g2n', 'g2o', 'g2p', 'g2q', 'g3c', 'g3d', 'g3e'].includes(milestone) ? 'eridion' : 'daedalus'}-${milestone}-experimental-unsigned-${descriptor}-`))
  const executable = ['g2m-a', 'g2m-b', 'g2m-c', 'g2n', 'g2o', 'g2p', 'g2q', 'g3c', 'g3d', 'g3e'].includes(milestone) ? 'eridion' : 'daedalus'
  const files = { [executable + (platform === 'Windows' ? '.exe' : '')]: binary, 'README.md': readme }
  const hashes = []
  for (const [name, bytes] of Object.entries(files)) {
    await fs.writeFile(path.join(directory, name), bytes)
    hashes.push(`${crypto.createHash('sha256').update(bytes).digest('hex')}  ${name}\n`)
  }
  await fs.writeFile(path.join(directory, 'SHA256SUMS'), hashes.join(''))
  return directory
}

// G3L Task 10: package a preview bound to one generated build identity. The
// executable's own --build-info report, the packaged BUILD-IDENTITY.json, the
// file and artifact names and SHA256SUMS must all agree, or packaging fails.
const IDENTITY_FIELDS = ['schema', 'version', 'milestone', 'revision', 'shortRevision', 'sourceDate']
async function packageIdentified ({ binaryPath, readmePath, outputParent, platform, identity, reported, artifact }) {
  validateIdentity(identity)
  if (!['Windows', 'Linux'].includes(platform) || identity.target.os !== platform.toLowerCase()) throw new Error('Build identity target does not match the packaging platform')
  if (artifact !== undefined && artifact !== identity.artifact) throw new Error('Artifact name does not match the build identity')
  if ((path.extname(binaryPath).toLowerCase() === '.exe') !== (identity.target.os === 'windows')) throw new Error('Executable filename does not match the build identity target')
  if (!reported || typeof reported !== 'object') throw new Error('Executable did not report a build identity')
  for (const field of IDENTITY_FIELDS) {
    if (reported[field] !== identity[field]) throw new Error(`Executable ${field} does not match the build identity`)
  }
  if (!reported.target || reported.target.os !== identity.target.os || reported.target.arch !== identity.target.arch) throw new Error('Executable target does not match the build identity')
  // The Go toolchain's VCS stamp is independent of the linker values.
  if (!reported.vcs || reported.vcs.revision !== identity.revision || reported.vcs.modified !== false) throw new Error('Executable VCS revision does not match the build identity (or the tree was modified)')
  const binary = await regularBytes(binaryPath)
  const readme = await regularBytes(readmePath)
  if (!readme.toString('utf8').includes(identity.milestone.toUpperCase())) throw new Error('README does not describe this milestone')
  const manifest = Buffer.from(JSON.stringify(identity, null, 2) + '\n')
  const directory = await fs.mkdtemp(path.join(outputParent, identity.artifact + '-'))
  const files = { [identity.executable]: binary, 'README.md': readme, 'BUILD-IDENTITY.json': manifest }
  const hashes = []
  for (const [name, bytes] of Object.entries(files)) {
    await fs.writeFile(path.join(directory, name), bytes)
    hashes.push(`${crypto.createHash('sha256').update(bytes).digest('hex')}  ${name}\n`)
  }
  await fs.writeFile(path.join(directory, 'SHA256SUMS'), hashes.join(''))
  await verifyPackage(directory)
  return { directory, artifact: identity.artifact }
}

// verifyPackage re-reads a packaged preview: exactly the expected files, every
// checksum matching, and a valid manifest naming the packaged executable.
async function verifyPackage (directory) {
  const identity = validateIdentity(JSON.parse(await fs.readFile(path.join(directory, 'BUILD-IDENTITY.json'), 'utf8')))
  const expected = [identity.executable, 'BUILD-IDENTITY.json', 'README.md', 'SHA256SUMS'].sort()
  if ((await fs.readdir(directory)).sort().join() !== expected.join()) throw new Error('Preview contains unexpected or missing files')
  const lines = (await fs.readFile(path.join(directory, 'SHA256SUMS'), 'utf8')).trim().split('\n')
  if (lines.length !== 3) throw new Error('SHA256SUMS does not cover exactly the packaged files')
  for (const line of lines) {
    const m = /^([0-9a-f]{64}) {2}(.+)$/.exec(line)
    if (!m || !expected.includes(m[2]) || m[2] === 'SHA256SUMS') throw new Error('Malformed SHA256SUMS')
    const actual = crypto.createHash('sha256').update(await fs.readFile(path.join(directory, m[2]))).digest('hex')
    if (actual !== m[1]) throw new Error(`Checksum mismatch for ${m[2]}`)
  }
  return identity
}

async function main () {
  const root = path.resolve(__dirname, '../..')
  const platform = process.env.RUNNER_OS || (process.platform === 'win32' ? 'Windows' : 'Linux')
  const identityPath = process.env.ERIDION_BUILD_IDENTITY || path.join(root, 'go/bin/BUILD-IDENTITY.json')
  const identity = JSON.parse(await fs.readFile(identityPath, 'utf8'))
  const binaryPath = path.join(root, 'go/bin', platform === 'Windows' ? 'daedalus.exe' : 'daedalus')
  // Ask the executable itself; never infer its revision from the workflow.
  const reported = JSON.parse(execFileSync(binaryPath, ['--build-info'], { encoding: 'utf8', timeout: 30000 }))
  const { directory, artifact } = await packageIdentified({
    binaryPath,
    readmePath: path.join(root, 'docs/testing/g3l-reliability-build-preview.md'),
    outputParent: process.env.RUNNER_TEMP || os.tmpdir(),
    platform,
    identity,
    reported,
    artifact: process.env.ERIDION_EXPECTED_ARTIFACT
  })
  if (process.env.GITHUB_OUTPUT) await fs.appendFile(process.env.GITHUB_OUTPUT, `directory=${directory}\nartifact=${artifact}\n`)
  console.log(`Packaged experimental, unsigned preview ${identity.version} (${platform}); executable, README, BUILD-IDENTITY.json and hashes only`)
}
if (require.main === module) main().catch(error => { console.error('Experimental preview packaging failed: ' + error.message); process.exitCode = 1 })
module.exports = { packagePreview, packageIdentified, verifyPackage }
