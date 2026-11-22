#!/usr/bin/env node
// Deterministic synthetic journal for native performance measurement.
// Usage: node scripts/testing/perf-journal-fixture.cjs <outDir> <events> [eventsPerFile]
// Fabricated commander/FID only; never derived from a real profile.
'use strict'
const fs = require('node:fs')
const path = require('node:path')
const crypto = require('node:crypto')

const [outDir, total = '100000', perFile = '10000'] = process.argv.slice(2)
if (!outDir) {
  console.error('usage: perf-journal-fixture.cjs <outDir> <events> [eventsPerFile]')
  process.exit(2)
}
const count = Number(total); const chunk = Number(perFile)
fs.mkdirSync(outDir, { recursive: true })
let seed = 42
const rand = () => { seed = (seed * 1103515245 + 12345) & 0x7fffffff; return seed / 0x7fffffff }
let clock = Date.UTC(2026, 9, 1, 0, 0, 0)
const stamp = () => { clock += 1000 + Math.floor(rand() * 4000); return new Date(clock).toISOString().replace('.000Z', 'Z') }
const hash = crypto.createHash('sha256')
let written = 0; let file = 0; let system = 1
const systemName = n => `Perf Fixture ${n}`
const kinds = ['FSDJump', 'FSSDiscoveryScan', 'Scan', 'Scan', 'Scan', 'Scan', 'Music', 'ReceiveText', 'Docked', 'MarketBuy', 'MarketSell', 'Undocked', 'MaterialCollected', 'Scan']
while (written < count) {
  file++
  const lines = []
  const day = String(file).padStart(2, '0')
  lines.push(JSON.stringify({ timestamp: stamp(), event: 'Fileheader', part: 1, language: 'English/UK', Odyssey: true, gameversion: '4.1.0.100', build: 'r300000/r0 ' }))
  lines.push(JSON.stringify({ timestamp: stamp(), event: 'LoadGame', FID: 'F9999001', Commander: 'Perf Fixture', Horizons: true, Odyssey: true, Ship: 'anaconda', ShipID: 7, ShipName: 'PERF ONE', ShipIdent: 'PF-01', FuelLevel: 32, FuelCapacity: 32, GameMode: 'Solo', Credits: 1000000000, Loan: 0 }))
  lines.push(JSON.stringify({ timestamp: stamp(), event: 'Location', StarSystem: systemName(system), SystemAddress: 1000000 + system, StarPos: [system * 1.5, 0, -system * 0.5], Docked: false }))
  written += 3
  while (written < count && lines.length < chunk) {
    const kind = kinds[Math.floor(rand() * kinds.length)]
    let e
    switch (kind) {
      case 'FSDJump': system++; e = { event: kind, StarSystem: systemName(system), SystemAddress: 1000000 + system, StarPos: [system * 1.5, 0, -system * 0.5], JumpDist: 12.3, FuelUsed: 2.1, FuelLevel: 29.9 }; break
      case 'FSSDiscoveryScan': e = { event: kind, Progress: 0.4, BodyCount: 12, NonBodyCount: 3, SystemName: systemName(system), SystemAddress: 1000000 + system }; break
      case 'Scan': e = { event: kind, ScanType: 'Detailed', BodyName: `${systemName(system)} ${Math.floor(rand() * 12)}`, BodyID: Math.floor(rand() * 40), StarSystem: systemName(system), SystemAddress: 1000000 + system, DistanceFromArrivalLS: Math.round(rand() * 5000), PlanetClass: 'Rocky body', Landable: rand() > 0.5, MassEM: 0.2, Radius: 2000000, SurfaceGravity: 3.1, SurfaceTemperature: 180, WasDiscovered: true, WasMapped: false }; break
      case 'Docked': e = { event: kind, StationName: 'Perf Port', StationType: 'Coriolis', StarSystem: systemName(system), SystemAddress: 1000000 + system, MarketID: 3700000000 + system }; break
      case 'MarketBuy': e = { event: kind, MarketID: 3700000000 + system, Type: 'gold', Count: 4, BuyPrice: 9000, TotalCost: 36000 }; break
      case 'MarketSell': e = { event: kind, MarketID: 3700000000 + system, Type: 'gold', Count: 4, SellPrice: 9500, TotalSale: 38000, AvgPricePaid: 9000 }; break
      case 'Undocked': e = { event: kind, StationName: 'Perf Port', MarketID: 3700000000 + system }; break
      case 'MaterialCollected': e = { event: kind, Category: 'Raw', Name: 'iron', Count: 3 }; break
      case 'ReceiveText': e = { event: kind, From: 'Perf NPC', Message: 'fixture', Channel: 'npc' }; break
      default: e = { event: 'Music', MusicTrack: 'Exploration' }
    }
    lines.push(JSON.stringify({ timestamp: stamp(), ...e }))
    written++
  }
  const body = lines.join('\n') + '\n'
  hash.update(body)
  fs.writeFileSync(path.join(outDir, `Journal.2026-10-${day}T000000.01.log`), body)
}
console.log(JSON.stringify({ events: written, files: file, sha256: hash.digest('hex') }))
