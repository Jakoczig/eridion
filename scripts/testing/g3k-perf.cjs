#!/usr/bin/env node
// G3K whole-application performance driver (Linux). Starts the real ERIDION
// service (go/cmd/perf-host) on an isolated profile with a synthetic journal,
// drives headless Chromium pages (main and paired displays) over the same
// loopback/remote protocol the native WebView uses, and samples CPU and
// proportional memory (PSS) of the host and of every browser process from
// /proc. Synthetic data only; no provider is configured.
//
// node scripts/testing/g3k-perf.cjs --host BIN --fixture DIR --case NAME --out FILE
//   [--playwright DIR] [--chromium BIN] [--low-latency] [--rep N]
'use strict'
const fs = require('node:fs')
const os = require('node:os')
const path = require('node:path')
const { spawn } = require('node:child_process')

const args = Object.fromEntries(process.argv.slice(2).reduce((a, v, i, all) => { if (v.startsWith('--')) a.push([v.slice(2), all[i + 1] && !all[i + 1].startsWith('--') ? all[i + 1] : true]); return a }, []))
const { chromium } = require(path.join(args.playwright || '.', 'node_modules/playwright-core'))
const delay = ms => new Promise(resolve => setTimeout(resolve, ms))
const TICK = 100 // USER_HZ on Linux

function readStat (pid) {
  try {
    const raw = fs.readFileSync(`/proc/${pid}/stat`, 'utf8')
    const f = raw.slice(raw.lastIndexOf(')') + 2).split(' ')
    return { ppid: Number(f[1]), cpu: (Number(f[11]) + Number(f[12])) / TICK }
  } catch { return null }
}
function pss (pid) {
  try { const m = /Pss:\s+(\d+) kB/.exec(fs.readFileSync(`/proc/${pid}/smaps_rollup`, 'utf8')); return m ? Number(m[1]) * 1024 : 0 } catch { return 0 }
}
function tree (root) {
  const all = fs.readdirSync('/proc').filter(n => /^\d+$/.test(n)).map(Number)
  const parents = new Map(all.map(p => [p, readStat(p)?.ppid]))
  const out = new Set([root])
  let grew = true
  while (grew) { grew = false; for (const [p, pp] of parents) if (!out.has(p) && out.has(pp)) { out.add(p); grew = true } }
  return [...out]
}
function findByMarker (marker) {
  for (const n of fs.readdirSync('/proc').filter(n => /^\d+$/.test(n))) {
    try { if (fs.readFileSync(`/proc/${n}/cmdline`, 'utf8').includes(marker)) { const st = readStat(n); if (st && !fs.readFileSync(`/proc/${st.ppid}/cmdline`, 'utf8').includes(marker)) return Number(n) } } catch {}
  }
  return null
}
function sample (hostPid, browserPid) {
  const s = { host: { cpu: readStat(hostPid)?.cpu || 0, pss: pss(hostPid) }, browser: { cpu: 0, pss: 0, processes: 0 } }
  if (browserPid) for (const p of tree(browserPid)) { const st = readStat(p); if (st) { s.browser.cpu += st.cpu; s.browser.pss += pss(p); s.browser.processes++ } }
  return s
}

async function startHost (fixture) {
  const profile = fs.mkdtempSync(path.join(os.tmpdir(), 'eridion-perf-profile-'))
  const journal = fs.mkdtempSync(path.join(os.tmpdir(), 'eridion-perf-journal-'))
  for (const f of fs.readdirSync(fixture)) fs.copyFileSync(path.join(fixture, f), path.join(journal, f))
  const child = spawn(args.host, ['-journal', journal, '-profile', profile], { stdio: ['pipe', 'pipe', 'inherit'] })
  const lines = []; const waiters = []
  let buffer = ''
  child.stdout.on('data', d => { buffer += d; let i; while ((i = buffer.indexOf('\n')) >= 0) { const l = buffer.slice(0, i); buffer = buffer.slice(i + 1); lines.push(l); for (const w of [...waiters]) if (w.test(l)) { waiters.splice(waiters.indexOf(w), 1); w.resolve(l) } } })
  const expect = (re, ms = 120000) => new Promise((resolve, reject) => { const hit = lines.find(l => re.test(l)); if (hit) { lines.splice(lines.indexOf(hit), 1); return resolve(hit) } const w = { test: l => re.test(l), resolve }; waiters.push(w); setTimeout(() => reject(new Error('timeout ' + re)), ms) })
  const url = (await expect(/^PERF_HOST_URL=/)).split('=')[1]
  const pid = Number((await expect(/^PERF_HOST_PID=/)).split('=')[1])
  const command = async (cmd, re) => { child.stdin.write(cmd + '\n'); return expect(re) }
  return {
    url, pid, journal, profile,
    async stats () { return JSON.parse((await command('stats', /^PERF_STATS /)).slice('PERF_STATS '.length)) },
    async reset () { await command('reset', /^PERF_RESET/) },
    async stop () { child.stdin.write('quit\n'); await new Promise(resolve => { child.once('exit', resolve); setTimeout(() => { child.kill(); resolve() }, 15000) }); for (const d of [profile, journal]) fs.rmSync(d, { recursive: true, force: true }) }
  }
}

async function ownerSession (host, request) {
  const r = await request.post(host.url + '/auth/session', { headers: { Origin: host.url }, data: { role: 'viewer' } })
  if (!r.ok()) throw new Error('session ' + r.status())
}
async function remoteAction (host, request, body) {
  const r = await request.post(host.url + '/owner/remote', { headers: { Origin: host.url, 'X-Eridion-Scope': 'launcher' }, data: body })
  if (!r.ok()) throw new Error('remote ' + r.status() + ' ' + await r.text())
  return r.json()
}

// waitText resolves when the page shows the given text (checked each frame).
const waitText = (page, text, ms = 15000) => page.waitForFunction(t => document.body && document.body.textContent.includes(t), text, { timeout: ms, polling: 'raf' })

function frameCounter (page) {
  const c = { frames: 0, bytes: 0, last: 0 }
  page.on('websocket', ws => ws.on('framereceived', f => { c.frames++; c.bytes += Buffer.byteLength(typeof f.payload === 'string' ? f.payload : f.payload || ''); c.last = Date.now() }))
  return c
}
async function quiet (counter, ms = 400, limit = 15000) { const start = Date.now(); while (Date.now() - start < limit) { if (Date.now() - counter.last >= ms) return; await delay(50) } }

function appendJournal (host, events) {
  const file = fs.readdirSync(host.journal).filter(f => f.startsWith('Journal.')).sort().pop()
  fs.appendFileSync(path.join(host.journal, file), events.map(e => JSON.stringify({ timestamp: new Date().toISOString(), ...e }) + '\n').join(''))
}
function writeStatus (host, status) {
  fs.writeFileSync(path.join(host.journal, 'Status.json'), JSON.stringify({ timestamp: new Date().toISOString(), event: 'Status', ...status }))
}

async function measureWindow (host, browserPid, seconds, step = 5) {
  const series = []
  const first = sample(host.pid, browserPid); series.push(first)
  for (let t = step; t <= seconds; t += step) { await delay(step * 1000); series.push(sample(host.pid, browserPid)) }
  const last = series[series.length - 1]
  const mean = k => series.reduce((a, s) => a + s[k].pss, 0) / series.length
  return {
    seconds,
    hostCPUSeconds: +(last.host.cpu - first.host.cpu).toFixed(3),
    browserCPUSeconds: +(last.browser.cpu - first.browser.cpu).toFixed(3),
    hostPSSMB: { start: +(first.host.pss / 2 ** 20).toFixed(1), end: +(last.host.pss / 2 ** 20).toFixed(1), mean: +(mean('host') / 2 ** 20).toFixed(1) },
    browserPSSMB: { start: +(first.browser.pss / 2 ** 20).toFixed(1), end: +(last.browser.pss / 2 ** 20).toFixed(1), mean: +(mean('browser') / 2 ** 20).toFixed(1) },
    browserProcesses: last.browser.processes
  }
}
const pct = (v, q) => { const s = [...v].sort((a, b) => a - b); return s[Math.min(s.length - 1, Math.floor(s.length * q))] }

async function main () {
  const host = await startHost(args.fixture)
  const marker = 'eridion-perf-' + process.pid + '-' + Date.now()
  const browser = await chromium.launch({ executablePath: args.chromium, args: ['--no-sandbox', '--marker=' + marker] })
  const browserPid = findByMarker(marker)
  const result = { case: args.case, rep: Number(args.rep || 1), lowLatency: !!args['low-latency'], startedAt: new Date().toISOString() }
  try {
    const main = await browser.newContext({ viewport: { width: 1280, height: 800 } })
    await ownerSession(host, main.request)
    // Wait for the import to finish (the summary reaches the dashboard).
    const warm = await main.newPage()
    await warm.goto(host.url + '/nav/route')
    await waitText(warm, 'Perf Fixture', 120000)
    await warm.close()
    if (args['low-latency']) {
      const r = await main.request.post(host.url + '/owner/companion', { headers: { Origin: host.url, 'X-Eridion-Scope': 'launcher' }, data: { lowLatency: true } })
      result.lowLatencyApplied = r.ok()
    }
    const pair = async (page) => {
      const ctx = await browser.newContext({ ignoreHTTPSErrors: true, viewport: { width: 1024, height: 768 } })
      const issued = await remoteAction(host, main.request, { action: 'pair' })
      const r = await ctx.request.post(remoteURL + '/auth/pair', { headers: { Origin: remoteURL }, data: { code: issued.code, name: 'Perf display' } })
      if (!r.ok()) throw new Error('pair ' + r.status())
      const p = await ctx.newPage(); const counter = frameCounter(p)
      await p.goto(remoteURL + page)
      return { ctx, page: p, counter }
    }
    let remoteURL = ''
    if (/paired/.test(args.case)) remoteURL = (await remoteAction(host, main.request, { action: 'enable', address: '127.0.0.1', port: 0 })).url

    await main.addInitScript(() => { const Native = window.WebSocket; window.__sockets = []; window.WebSocket = function (...a) { const ws = new Native(...a); window.__sockets.push(ws); return ws }; window.WebSocket.prototype = Native.prototype; Object.assign(window.WebSocket, { CONNECTING: 0, OPEN: 1, CLOSING: 2, CLOSED: 3 }) })
    const open = async (route) => { const p = await main.newPage(); const counter = frameCounter(p); await p.goto(host.url + route); await quiet(counter, 1000, 30000); return { page: p, counter } }
    await host.reset()
    switch (args.case) {
      case 'launcher-idle': case 'dashboard-idle': case 'dashboard-hidden': {
        const v = await open(args.case === 'launcher-idle' ? '/launcher' : '/nav/route')
        if (args.case === 'dashboard-hidden') await v.page.evaluate(() => { Object.defineProperty(document, 'visibilityState', { configurable: true, get: () => 'hidden' }); Object.defineProperty(document, 'hidden', { configurable: true, get: () => true }); document.dispatchEvent(new Event('visibilitychange')) })
        await delay(30000)
        const f0 = v.counter.frames; const b0 = v.counter.bytes
        await host.reset()
        result.window = await measureWindow(host, browserPid, Number(args.seconds || 300))
        result.socket = { frames: v.counter.frames - f0, bytes: v.counter.bytes - b0 }
        break
      }
      case 'live-latency': {
        const v = await open('/nav/route')
        const lat = []
        const before = sample(host.pid, browserPid)
        for (let i = 0; i < 30; i++) {
          const name = `Perf Live ${result.rep}-${i}`
          const t = Date.now()
          appendJournal(host, [{ event: 'FSDJump', StarSystem: name, SystemAddress: 8000000 + i, StarPos: [i, 2, 0], JumpDist: 10, FuelUsed: 1, FuelLevel: 20 }])
          await waitText(v.page, name); lat.push(Date.now() - t)
          await delay(700)
        }
        const after = sample(host.pid, browserPid)
        result.latencyMs = { p50: pct(lat, 0.5), p95: pct(lat, 0.95), max: Math.max(...lat), samples: lat }
        result.cpu = { host: +(after.host.cpu - before.host.cpu).toFixed(3), browser: +(after.browser.cpu - before.browser.cpu).toFixed(3) }
        result.socket = { frames: v.counter.frames, bytes: v.counter.bytes }
        break
      }
      case 'fsd-transitions': {
        const v = await open('/ship/status')
        const before = sample(host.pid, browserPid); const f0 = v.counter.frames
        // 20 cycles: Status updates every 250 ms through charge, jump, arrival.
        for (let i = 0; i < 20; i++) {
          for (let k = 0; k < 8; k++) { writeStatus(host, { Flags: 0x01000008 | (k < 4 ? 1 << 17 : 0), Flags2: 0, Fuel: { FuelMain: 20 - k * 0.1, FuelReservoir: 0.5 }, Cargo: 0 }); await delay(250) }
          appendJournal(host, [{ event: 'StartJump', JumpType: 'Hyperspace', StarSystem: `Perf Jump ${i}`, SystemAddress: 8100000 + i, StarClass: 'K' }])
          await delay(500)
          appendJournal(host, [{ event: 'FSDJump', StarSystem: `Perf Jump ${i}`, SystemAddress: 8100000 + i, StarPos: [i, 3, 0], JumpDist: 12, FuelUsed: 1, FuelLevel: 19 }])
          await delay(500)
        }
        const after = sample(host.pid, browserPid)
        result.cpu = { host: +(after.host.cpu - before.host.cpu).toFixed(3), browser: +(after.browser.cpu - before.browser.cpu).toFixed(3) }
        result.socket = { frames: v.counter.frames - f0, bytes: v.counter.bytes }
        break
      }
      case 'tab-switch': {
        const v = await open('/nav/route')
        const routes = ['/cmdr/overview', '/ship/fleet', '/exploration/biologicals', '/ship/status', '/nav/route']
        const times = {}; const before = sample(host.pid, browserPid)
        for (let round = 0; round < 3; round++) {
          for (const r of routes) {
            const t = Date.now(); const f = v.counter.frames
            await v.page.evaluate(r => window.next.router.push(r), r)
            await v.page.waitForFunction(r => location.pathname.replace(/\/$/, '') === r, r)
            await quiet(v.counter, 300, 10000)
            ;(times[r] = times[r] || []).push(Math.max(0, (v.counter.frames > f ? v.counter.last : Date.now()) - t))
          }
        }
        const after = sample(host.pid, browserPid)
        result.tabSwitchMs = Object.fromEntries(Object.entries(times).map(([k, v]) => [k, { p50: pct(v, 0.5), max: Math.max(...v), samples: v }]))
        result.cpu = { host: +(after.host.cpu - before.host.cpu).toFixed(3), browser: +(after.browser.cpu - before.browser.cpu).toFixed(3) }
        result.socket = { frames: v.counter.frames, bytes: v.counter.bytes }
        break
      }
      case 'paired-1-same': case 'paired-2-same': case 'paired-2-different': {
        const mainView = await open('/nav/route')
        const n = args.case.startsWith('paired-1') ? 1 : 2
        const pages = args.case.endsWith('different') ? ['/nav/route', '/ship/status'] : ['/nav/route', '/nav/route']
        const displays = []
        for (let i = 0; i < n; i++) displays.push(await pair(pages[i]))
        await delay(15000)
        await host.reset()
        const counters = [mainView.counter, ...displays.map(d => d.counter)]
        const f0 = counters.map(c => c.frames); const b0 = counters.map(c => c.bytes)
        // Two minutes with live activity: one jump every 10 s, Status every 1 s.
        const before = sample(host.pid, browserPid)
        for (let t = 0; t < 120; t++) {
          writeStatus(host, { Flags: 0x01000008, Flags2: 0, Fuel: { FuelMain: 20, FuelReservoir: 0.5 }, Cargo: 0 })
          if (t % 10 === 0) appendJournal(host, [{ event: 'FSDJump', StarSystem: `Perf Paired ${t}`, SystemAddress: 8200000 + t, StarPos: [t, 4, 0], JumpDist: 8, FuelUsed: 1, FuelLevel: 19 }])
          await delay(1000)
        }
        const after = sample(host.pid, browserPid)
        result.cpu = { host: +(after.host.cpu - before.host.cpu).toFixed(3), browser: +(after.browser.cpu - before.browser.cpu).toFixed(3) }
        result.socket = counters.map((c, i) => ({ frames: c.frames - f0[i], bytes: c.bytes - b0[i] }))
        result.pss = { hostMB: +(after.host.pss / 2 ** 20).toFixed(1), browserMB: +(after.browser.pss / 2 ** 20).toFixed(1) }
        break
      }
      case 'resume-reconnect': {
        const v = await open('/nav/route')
        // Hidden for 60 s while events arrive, then visible: time to fresh content.
        await v.page.evaluate(() => { Object.defineProperty(document, 'visibilityState', { configurable: true, get: () => 'hidden' }); Object.defineProperty(document, 'hidden', { configurable: true, get: () => true }); document.dispatchEvent(new Event('visibilitychange')) })
        const f0 = v.counter.frames; const b0 = v.counter.bytes
        for (let t = 0; t < 6; t++) { appendJournal(host, [{ event: 'FSDJump', StarSystem: `Perf Hidden ${t}`, SystemAddress: 8300000 + t, StarPos: [t, 5, 0], JumpDist: 8, FuelUsed: 1, FuelLevel: 19 }]); await delay(10000) }
        result.hiddenSocket = { frames: v.counter.frames - f0, bytes: v.counter.bytes - b0 }
        let t = Date.now()
        await v.page.evaluate(() => { Object.defineProperty(document, 'visibilityState', { configurable: true, get: () => 'visible' }); Object.defineProperty(document, 'hidden', { configurable: true, get: () => false }); document.dispatchEvent(new Event('visibilitychange')) })
        await waitText(v.page, 'Perf Hidden 5')
        result.resumeToFreshMs = Date.now() - t
        // Reconnect: close the page's sockets (the client reconnects on its
        // own), with one event written while disconnected.
        await v.page.evaluate(() => window.__sockets.forEach(ws => ws.close()))
        appendJournal(host, [{ event: 'FSDJump', StarSystem: 'Perf Reconnected', SystemAddress: 8300100, StarPos: [9, 5, 0], JumpDist: 8, FuelUsed: 1, FuelLevel: 19 }])
        t = Date.now()
        await waitText(v.page, 'Perf Reconnected', 30000)
        result.reconnectToFreshMs = Date.now() - t
        break
      }
      default: throw new Error('unknown case ' + args.case)
    }
    result.host = await host.stats()
  } finally {
    await browser.close().catch(() => {})
    await host.stop()
  }
  fs.mkdirSync(path.dirname(args.out), { recursive: true })
  fs.appendFileSync(args.out, JSON.stringify(result) + '\n')
  console.log(JSON.stringify({ case: result.case, rep: result.rep, ok: true }))
}
main().catch(e => { console.error(e); process.exit(1) })
