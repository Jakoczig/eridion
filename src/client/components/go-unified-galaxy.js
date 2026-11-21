import { useState, useRef, useEffect } from 'react'
import { useRouter } from 'next/router'
import { Projection, zoomFrame, panFrame } from './go-galaxy-projection'
import { atlasExtents } from './go-atlas'
import { spatialSegments } from './go-spatial'
import { distance } from '../lib/go-display-controller'
import { coordinates, remainingRoute, automaticFrame, localScale, adjustAxis } from '../lib/go-galaxy-policy'

export default function GoGalaxy ({ context, route, sendEvent, role, policy = {}, compact = false, inspection, onInspection }) {
  const router = useRouter()
  const [manualFrame, setManualFrame] = useState(inspection?.frame || null); const [forcedMode, setForcedMode] = useState(inspection?.mode || null)
  const [query, setQuery] = useState(inspection?.query || ''); const [searched, setSearched] = useState(inspection?.searched || null); const [notice, setNotice] = useState(''); const [busy, setBusy] = useState(false)
  const [atlas, setAtlas] = useState(null); const [atlasNotice, setAtlasNotice] = useState(''); const [spatial, setSpatial] = useState(null); const [refresh, setRefresh] = useState(0)
  const area = useRef(null); const [size, setSize] = useState([320, 320]); const previousMode = useRef('galaxy')
  const galaxyEnabled = policy.galaxyMap !== false; const localEnabled = policy.localAtlas !== false
  const current = context?.current; const remaining = remainingRoute(context, route)
  const destination = (context?.points || []).find(p => p.name === route?.destinationSystem)
  const frame = manualFrame || automaticFrame(current, remaining, atlasExtents)
  const frameKey = JSON.stringify(frame)
  const manualKey = JSON.stringify(manualFrame)
  useEffect(() => { onInspection?.({ frame: manualFrame, mode: forcedMode, query, searched }) }, [manualKey, forcedMode, query, JSON.stringify(searched), onInspection])
  const knownCenter = !!coordinates(current) || !!manualFrame || remaining.points.length > 0
  const wantsLocal = localEnabled && knownCenter && (forcedMode === 'local' || (!forcedMode && remaining.complete && localScale(frame, previousMode.current)))
  const q = { low: 'xyz'.split('').map(a => frame[a][0]), high: 'xyz'.split('').map(a => frame[a][1]), route: remaining.points.slice(0, 5).map(coordinates) }
  const spatialKey = JSON.stringify(q)
  const local = wantsLocal && spatial?.key === spatialKey && spatial.complete
  const mode = local ? 'local' : galaxyEnabled ? 'galaxy' : 'unavailable'
  useEffect(() => { if (local) previousMode.current = 'local'; else if (!wantsLocal) previousMode.current = 'galaxy' }, [local, wantsLocal])
  useEffect(() => {
    if (typeof window === 'undefined') return
    const refresh = () => setRefresh(v => v + 1)
    const events = ['eridion-atlas-changed', 'eridion-spatial-changed', 'socketEvent_goSpatialChanged']
    events.forEach(name => window.addEventListener(name, refresh))
    return () => events.forEach(name => window.removeEventListener(name, refresh))
  }, [])
  useEffect(() => {
    if (!area.current || typeof window.ResizeObserver === 'undefined') return
    const measure = () => { if (!area.current) return; const next = [area.current.clientWidth, area.current.clientHeight]; setSize(old => old[0] === next[0] && old[1] === next[1] ? old : next) }
    const observer = new window.ResizeObserver(measure); observer.observe(area.current); measure()
    return () => observer.disconnect()
  }, [])
  useEffect(() => {
    if (!wantsLocal) return
    let active = true; let timer
    const load = () => sendEvent('getGoSpatialView', JSON.parse(spatialKey)).then(v => { if (!active) return; setSpatial({ ...v, key: spatialKey }); if (v.pending) timer = setTimeout(load, 600) }).catch(() => { if (active) setSpatial({ key: spatialKey, complete: false }) })
    timer = setTimeout(load, 180)
    return () => { active = false; clearTimeout(timer) }
  }, [sendEvent, spatialKey, wantsLocal, refresh])
  const spans = 'xyz'.split('').map(a => frame[a][1] - frame[a][0])
  const scale = Math.max(0.01, Math.min((size[0] - 6) / (spans[0] + spans[1]), (size[1] - 6) / (spans[2] + spans[1])))
  const dimensions = { width: scale * (spans[0] + spans[1]) + 6, height: scale * (spans[2] + spans[1]) + 6, gridTemplateColumns: `${spans[0]}fr ${spans[1]}fr`, gridTemplateRows: `${spans[2]}fr ${spans[1]}fr` }
  const pixelKey = JSON.stringify({ xz: [Math.ceil(scale * spans[0]), Math.ceil(scale * spans[2])], xy: [Math.ceil(scale * spans[0]), Math.ceil(scale * spans[1])], yz: [Math.ceil(scale * spans[1]), Math.ceil(scale * spans[2])] })
  useEffect(() => {
    if (!galaxyEnabled || local) return
    let active = true; let timer
    const load = () => sendEvent('getGoAtlasView', { frame: JSON.parse(frameKey), projectionPixels: JSON.parse(pixelKey), pixels: 512, compact }).then(v => { if (!active) return; setAtlas(v); setAtlasNotice(v.installed ? '' : 'Galaxy atlas not installed'); if (v.pending) timer = setTimeout(load, 600) }).catch(() => { if (active) setAtlasNotice('Galaxy atlas unavailable') })
    timer = setTimeout(load, 180)
    return () => { active = false; clearTimeout(timer) }
  }, [sendEvent, frameKey, pixelKey, galaxyEnabled, local, compact, refresh])
  const search = async external => {
    setBusy(true)
    try { const result = await sendEvent(external ? 'lookupGoSystem' : 'getGoSystem', { name: query.trim() }); setSearched(result.point?.name ? result.point : null); setNotice(result.point?.coordinates ? '' : 'No coordinates reported') } catch { setNotice('Lookup unavailable; enabled EDSM lookup requires the local owner') } finally { setBusy(false) }
  }
  const switchMode = next => { setForcedMode(next); setManualFrame(next === 'local' && !localScale(automaticFrame(current, remaining, atlasExtents), 'local') && coordinates(current) ? automaticFrame(current, { points: [] }, atlasExtents) : null); previousMode.current = next }
  const path = [current, ...remaining.rawPoints].filter((p, i) => i !== 0 || coordinates(p))
  const onPan = (axes, dx, dy) => { if (Math.abs(dx) + Math.abs(dy) > 0.01) setManualFrame(panFrame(frame, axes, dx, dy)) }
  const zoom = factor => { setManualFrame(zoomFrame(frame, factor)); setForcedMode(null) }
  const projectionProps = { current, destination, searched, route: path.filter(coordinates), onSelect: setSearched, frame, onPan, localGeometry: true, routeSegments: spatialSegments(path, frame), atlas: galaxyEnabled && !local ? atlas : null, spatial: local ? spatial.points || [] : null }
  const selectedDistance = distance(current?.coordinates, searched?.coordinates)
  return (
    <section className={`go-page go-galaxy go-g3d ${compact ? 'is-compact' : ''}`} aria-label='Galaxy map' data-mode={mode}>
      <div className='go-galaxy-sidebar'><h2>Galaxy Map</h2>
        <form onSubmit={e => { e.preventDefault(); search(false) }}><label>System<input aria-label='Galaxy system search' maxLength={128} value={query} onChange={e => setQuery(e.target.value)} /></label><button disabled={busy || !query.trim()}>Search local / cached</button>{role === 'owner' && <button type='button' disabled={busy || !query.trim()} onClick={() => search(true)}>Lookup with enabled EDSM</button>}</form>
        <div className='go-galaxy-mode'><button aria-pressed={mode === 'galaxy'} disabled={!galaxyEnabled} onClick={() => switchMode('galaxy')}>Galaxy</button><button aria-pressed={mode === 'local'} disabled={!localEnabled || !knownCenter} onClick={() => switchMode('local')}>Local</button></div>
        <button onClick={() => { setManualFrame(null); setForcedMode(null) }}>Fit Route</button>
        <div className='go-galaxy-mode'><button aria-label='Galaxy zoom out' onClick={() => zoom(2)}>Zoom −</button><button aria-label='Galaxy zoom in' onClick={() => zoom(0.5)}>Zoom +</button></div>
        {local && <div className='go-galaxy-axes'>{'xyz'.split('').map((axis, i) => <div key={axis}><span>{axis.toUpperCase()} {Math.round(spans[i])} LY</span><button aria-label={`Decrease ${axis.toUpperCase()} extent`} onClick={() => setManualFrame(adjustAxis(frame, axis, -1))}>−</button><button aria-label={`Increase ${axis.toUpperCase()} extent`} onClick={() => setManualFrame(adjustAxis(frame, axis, 1))}>+</button></div>)}</div>}
        <p>{current?.name || 'Current galactic position unknown'}</p><p>{coordinates(current) ? current.coordinates.map((v, i) => `${'XYZ'[i]} ${v}`).join(' · ') + ' LY' : 'No coordinates reported'}</p>
        {searched && <><p>{searched.name}</p><p>Distance {selectedDistance === null ? 'Unknown' : `${selectedDistance.toFixed(1)} LY`}</p><button onClick={() => router.push({ pathname: '/nav/map', query: { system: searched.systemName || searched.name } })}>View System</button></>}
        <div role='status'>{!galaxyEnabled && !localEnabled ? 'Map renderers disabled' : wantsLocal && !local ? (galaxyEnabled ? 'Local detail unavailable; showing available overview' : 'Local detail unavailable; Galaxy Map disabled') : !remaining.complete ? 'Some route coordinates are unknown' : ''}{!local && galaxyEnabled && atlasNotice && <p>{atlasNotice}</p>}{notice && <p>{notice}</p>}</div>
      </div>
      <div ref={area} className='go-galaxy-area'><div className='go-galaxy-projections' style={dimensions}>{['xz', 'xy', 'yz'].map(axes => <Projection key={axes} {...projectionProps} axes={axes} pixels={JSON.parse(pixelKey)[axes][0]} pixelHeight={JSON.parse(pixelKey)[axes][1]} />)}</div></div>
    </section>
  )
}
