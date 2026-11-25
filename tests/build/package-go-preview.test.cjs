const test = require('node:test')
const assert = require('node:assert/strict')
const fs = require('node:fs/promises')
const os = require('node:os')
const path = require('node:path')
const crypto = require('node:crypto')
const { packagePreview } = require('../../scripts/build/package-go-preview.cjs')

test('preview contains only executable, prerequisites and matching SHA256 hashes', async t => {
  const root = await fs.mkdtemp(path.join(os.tmpdir(), 'daedalus-preview-test-'))
  t.after(() => fs.rm(root, { recursive: true, force: true }))
  const binary = path.join(root, 'daedalus.exe')
  await fs.writeFile(binary, 'synthetic executable fixture')
  await fs.writeFile(path.join(root, 'tls.json'), 'PRIVATE KEY MUST NEVER SHIP')
  await fs.writeFile(path.join(root, 'remote.json'), 'viewer token MUST NEVER SHIP')
  const readme = path.join(root, 'instructions.md')
  await fs.writeFile(readme, 'EXPERIMENTAL, UNSIGNED, LOCAL ROUTE + SURVEY PREVIEW (NO PROVIDER UPLOADS)\nWindows requires WebView2; Linux requires GTK4/WebKitGTK.\n')
  const directory = await packagePreview({ binaryPath: binary, readmePath: readme, outputParent: root, platform: 'Windows' })
  assert.match(path.basename(directory), /^daedalus-g2b-experimental-unsigned-local-route-survey-no-uploads-/)
  assert.deepEqual((await fs.readdir(directory)).sort(), ['README.md', 'SHA256SUMS', 'daedalus.exe'])
  const hashes = await fs.readFile(path.join(directory, 'SHA256SUMS'), 'utf8')
  for (const name of ['daedalus.exe', 'README.md']) {
    const bytes = await fs.readFile(path.join(directory, name))
    assert.ok(hashes.includes(`${crypto.createHash('sha256').update(bytes).digest('hex')}  ${name}\n`))
  }
  const second = await packagePreview({ binaryPath: binary, readmePath: readme, outputParent: root, platform: 'Windows' })
  assert.notEqual(second, directory, 'existing artifact directory was reused')
})

test('preview refuses missing or linked inputs rather than packaging adjacent files', async t => {
  const root = await fs.mkdtemp(path.join(os.tmpdir(), 'daedalus-preview-invalid-'))
  t.after(() => fs.rm(root, { recursive: true, force: true }))
  const binary = path.join(root, 'daedalus')
  const readme = path.join(root, 'readme.md')
  await fs.writeFile(readme, 'instructions')
  await assert.rejects(packagePreview({ binaryPath: binary, readmePath: readme, outputParent: root, platform: 'Linux' }))
  await fs.writeFile(binary, 'synthetic ELF fixture')
  await assert.rejects(packagePreview({ binaryPath: root, readmePath: readme, outputParent: root, platform: 'Linux' }))
  const link = path.join(root, 'linked-executable')
  try { await fs.symlink(binary, link, 'file') } catch (error) { if (error.code === 'EPERM') return; throw error }
  await assert.rejects(packagePreview({ binaryPath: link, readmePath: readme, outputParent: root, platform: 'Linux' }))
})

// G3L Task 10: identity-bound packaging.
const { packageIdentified, verifyPackage } = require('../../scripts/build/package-go-preview.cjs')
const rev = '0123456789abcdef0123456789abcdef01234567'
function identityFor (os) {
  const { artifactName, executableName } = require('../../scripts/build/go-build-identity.cjs')
  const id = { schema: 'eridion-build-identity/1', product: 'ERIDION', version: `2.2.0-g3l+${rev.slice(0, 12)}`, baseVersion: '2.2.0', milestone: 'g3l', revision: rev, shortRevision: rev.slice(0, 12), sourceDate: '2026-11-25T00:00:00Z', target: { os, arch: 'amd64' }, executable: executableName(os), artifact: '', ci: null }
  id.artifact = artifactName(id)
  return id
}
function reportFor (id) {
  return { schema: id.schema, product: 'ERIDION', version: id.version, milestone: id.milestone, revision: id.revision, shortRevision: id.shortRevision, sourceDate: id.sourceDate, target: { ...id.target }, vcs: { revision: id.revision, modified: false, time: id.sourceDate } }
}
async function identityFixture (t) {
  const root = await fs.mkdtemp(path.join(os.tmpdir(), 'eridion-identity-package-'))
  t.after(() => fs.rm(root, { recursive: true, force: true }))
  const windowsBinary = path.join(root, 'daedalus.exe')
  const linuxBinary = path.join(root, 'daedalus')
  await fs.writeFile(windowsBinary, 'synthetic executable fixture')
  await fs.writeFile(linuxBinary, 'synthetic ELF fixture')
  const readme = path.join(root, 'readme.md')
  await fs.writeFile(readme, 'ERIDION G3L experimental unsigned preview\n')
  return { root, windowsBinary, linuxBinary, readme }
}

test('identity-bound preview: executable, filename, manifest and checksums agree', async t => {
  const f = await identityFixture(t)
  const id = identityFor('windows')
  const { directory, artifact } = await packageIdentified({ binaryPath: f.windowsBinary, readmePath: f.readme, outputParent: f.root, platform: 'Windows', identity: id, reported: reportFor(id), artifact: id.artifact })
  assert.equal(artifact, `eridion-2.2.0-g3l-${rev.slice(0, 12)}-windows-amd64-experimental-unsigned-preview`)
  assert.ok(path.basename(directory).startsWith(artifact + '-'))
  assert.deepEqual((await fs.readdir(directory)).sort(), ['BUILD-IDENTITY.json', 'README.md', 'SHA256SUMS', 'eridion.exe'])
  const manifest = JSON.parse(await fs.readFile(path.join(directory, 'BUILD-IDENTITY.json'), 'utf8'))
  assert.deepEqual(manifest, id)
  const sums = await fs.readFile(path.join(directory, 'SHA256SUMS'), 'utf8')
  assert.match(sums, / {2}BUILD-IDENTITY\.json\n/)
  assert.equal((await verifyPackage(directory)).revision, rev)
})

test('mismatched filenames, revisions or manifests make packaging fail', async t => {
  const f = await identityFixture(t)
  const win = identityFor('windows')
  const linux = identityFor('linux')
  const ok = { binaryPath: f.windowsBinary, readmePath: f.readme, outputParent: f.root, platform: 'Windows', identity: win, reported: reportFor(win) }
  const cases = {
    'artifact name differs from identity': { artifact: 'eridion-g3e-Windows-x64-experimental-unsigned-preview' },
    'executable filename does not match the target': { binaryPath: f.linuxBinary },
    'identity target differs from platform': { platform: 'Linux', binaryPath: f.linuxBinary },
    'executable reports another revision': { reported: { ...reportFor(win), revision: 'f'.repeat(40) } },
    'executable reports another version': { reported: { ...reportFor(win), version: `2.2.0-g3g+${rev.slice(0, 12)}` } },
    'Go VCS revision differs from linker revision': { reported: { ...reportFor(win), vcs: { revision: 'e'.repeat(40), modified: false } } },
    'tree was modified at build time': { reported: { ...reportFor(win), vcs: { revision: rev, modified: true } } },
    'executable built for another OS': { reported: reportFor(linux) },
    'manifest version off policy': { identity: { ...win, version: `2.2.0-g3e+${rev.slice(0, 12)}` } },
    'manifest revision inconsistent': { identity: { ...win, shortRevision: 'a'.repeat(12) } },
    'no executable report': { reported: undefined },
    'README for another milestone': { readmePath: f.windowsBinary }
  }
  for (const [name, change] of Object.entries(cases)) {
    await assert.rejects(packageIdentified({ ...ok, ...change }), Error, name)
  }
  assert.ok(linux.executable === 'eridion')
})

test('a packaged preview altered afterwards fails verification', async t => {
  const f = await identityFixture(t)
  const id = identityFor('linux')
  const pack = () => packageIdentified({ binaryPath: f.linuxBinary, readmePath: f.readme, outputParent: f.root, platform: 'Linux', identity: id, reported: reportFor(id) })
  let { directory } = await pack()
  await fs.appendFile(path.join(directory, 'eridion'), 'x')
  await assert.rejects(verifyPackage(directory), /Checksum mismatch/)
  ;({ directory } = await pack())
  const manifest = JSON.parse(await fs.readFile(path.join(directory, 'BUILD-IDENTITY.json'), 'utf8'))
  manifest.revision = 'f'.repeat(40)
  await fs.writeFile(path.join(directory, 'BUILD-IDENTITY.json'), JSON.stringify(manifest))
  await assert.rejects(verifyPackage(directory), /Invalid build identity/)
  ;({ directory } = await pack())
  await fs.writeFile(path.join(directory, 'tls.json'), 'PRIVATE KEY MUST NEVER SHIP')
  await assert.rejects(verifyPackage(directory), /unexpected or missing/)
})
