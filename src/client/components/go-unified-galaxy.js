import { useState, useRef, useEffect } from 'react'
import { useRouter } from 'next/router'
import { Projection, zoomFrame, panFrame, panelLayout, mapCameraFrame, projectionUnits } from './go-galaxy-projection'
import { atlasExtents } from './go-atlas'
import { spatialSegments } from './go-spatial'
import { distance } from '../lib/go-display-controller'
import { coordinates, remainingRoute, automaticFrame, localScale, adjustAxis } from '../lib/go-galaxy-policy'

export async function searchSystem (sendEvent, query, role) {
  const name = query.trim()
  if (!name) return null
  const cached = await sendEvent('getGoSystem', { name })
  if (cached.point?.name || (cached.availability && cached.availability !== 'unavailable') || !cached.community?.enabled) return cached
  return sendEvent('lookupGoSystem', { name })
}
// XYZ controls adjust the inspected data volume, never the camera.
export function adjustSlice (frame, camera, axis, direction) {
  return { selection: adjustAxis(frame, axis, direction), camera: camera || mapCameraFrame(frame) }
}
export default function GoGalaxy ({ context, route, sendEvent, role, policy = {}, compact = false, inspection, onInspection, onViewSystem }) {
  const router = useRouter()
  const [manualFrame, setManualFrame] = useState(inspection?.frame || null); const [manualCamera, setManualCamera] = useState(inspection?.camera || null); const [forcedMode, setForcedMode] = useState(inspection?.mode || null)
  const [query, setQuery] = useState(inspection?.query || ''); const [searched, setSearched] = useState(inspection?.searched || null); const [notice, setNotice] = useState(''); const [busy, setBusy] = useState(false)
  const searchPending = useRef(false)
  const [atlas, setAtlas] = useState(null); const [atlasNotice, setAtlasNotice] = useState(''); const [spatial, setSpatial] = useState(null); const [refresh, setRefresh] = useState(0)
  const area = useRef(null); const [size, setSize] = useState([320, 320]); const previousMode = useRef('galaxy')
  const galaxyEnabled = policy.galaxyMap !== false; const localEnabled = policy.localAtlas !== false
  const current = context?.current; const remaining = remainingRoute(context, route)
  const destination = (context?.points || []).find(p => p.name === route?.destinationSystem)
  const frame = manualFrame || automaticFrame(current, remaining, atlasExtents)
  const frameKey = JSON.stringify(frame)
  const manualKey = JSON.stringify(manualFrame); const cameraKey = JSON.stringify(manualCamera)
  useEffect(() => { onInspection?.({ frame: manualFrame, camera: manualCamera, mode: forcedMode, query, searched }) }, [manualKey, cameraKey, forcedMode, query, JSON.stringify(searched), onInspection])
  const knownCenter = !!coordinates(current) || !!manualFrame || remaining.points.length > 0
  const wantsLocal = localEnabled && knownCenter && (forcedMode === 'local' || (!forcedMode && remaining.complete && localScale(frame, previousMode.current)))
  const q = { low: 'xyz'.split('').map(a => frame[a][0]), high: 'xyz'.split('').map(a => frame[a][1]), route: remaining.points.slice(0, 5).map(coordinates) }
  const spatialKey = JSON.stringify(q)
  const local = wantsLocal && spatial?.key === spatialKey && spatial.complete
  // Task 3 / F04: the XYZ controls follow the selected Local inspection, not
  // data readiness, so a pending, incomplete or failed slice can be adjusted.
  // Only a complete response for the current query renders as Local data.
  const localStatus = !wantsLocal || local ? '' : spatial?.key !== spatialKey || spatial.pending ? 'Loading local systems…' : spatial.failed ? 'Local data request failed. Adjust the slice or retry.' : (spatial.notice || 'Local data unavailable for this slice.')
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
    const load = () => sendEvent('getGoSpatialView', JSON.parse(spatialKey)).then(v => { if (!active) return; setSpatial({ ...v, key: spatialKey }); if (v.pending) timer = setTimeout(load, 600) }).catch(() => { if (active) setSpatial({ key: spatialKey, complete: false, pending: false, failed: true }) })
    timer = setTimeout(load, 180)
    return () => { active = false; clearTimeout(timer) }
  }, [sendEvent, spatialKey, wantsLocal, refresh])
  const spans = 'xyz'.split('').map(a => frame[a][1] - frame[a][0])
  const dimensions = panelLayout(size[0], size[1])
  // Keep the camera stable during XYZ slice edits. Y remains a slice axis in
  // the compressed side views; X/Z maintain equal LY scale in the main view.
  const displayFrame = manualCamera ? { ...manualCamera, y: frame.y } : mapCameraFrame(frame)
  const unit = (dimensions.width - 6) / (projectionUnits.x + projectionUnits.y)
  const pixelKey = JSON.stringify(Object.fromEntries(['xz', 'xy', 'yz'].map(axes => [axes, [...axes].map(axis => Math.ceil(unit * projectionUnits[axis]))])))
  const queryPixelsKey = JSON.stringify(Object.fromEntries(Object.entries(JSON.parse(pixelKey)).map(([axes, sizes]) => [axes, sizes.map((n, i) => Math.min(2048, Math.max(1, Math.ceil(n * (frame[axes[i]][1] - frame[axes[i]][0]) / (displayFrame[axes[i]][1] - displayFrame[axes[i]][0])))))])))
  useEffect(() => {
    if (!galaxyEnabled || local) return
    let active = true; let timer
    const load = () => sendEvent('getGoAtlasView', { frame: JSON.parse(frameKey), projectionPixels: JSON.parse(queryPixelsKey), pixels: 512, compact }).then(v => { if (!active) return; setAtlas(v); setAtlasNotice(v.installed ? '' : 'Galaxy atlas not installed'); if (v.pending) timer = setTimeout(load, 600) }).catch(() => { if (active) setAtlasNotice('Galaxy atlas unavailable') })
    timer = setTimeout(load, 180)
    return () => { active = false; clearTimeout(timer) }
  }, [sendEvent, frameKey, queryPixelsKey, galaxyEnabled, local, compact, refresh])
  const search = async () => {
    if (searchPending.current || !query.trim()) return
    searchPending.current = true; setBusy(true)
    try { const result = await searchSystem(sendEvent, query, role); setSearched(result?.point?.name ? result.point : null); setNotice(result?.point?.name ? (result.point.coordinates ? '' : 'No coordinates reported') : (result?.community?.notice || 'System unavailable; external lookup unavailable')) } catch { setNotice('System lookup unavailable') } finally { searchPending.current = false; setBusy(false) }
  }
  const switchMode = next => { setForcedMode(next); setManualCamera(null); setManualFrame(next === 'local' && !localScale(automaticFrame(current, remaining, atlasExtents), 'local') && coordinates(current) ? automaticFrame(current, { points: [] }, atlasExtents) : null); previousMode.current = next }
  const path = [current, ...remaining.rawPoints].filter((p, i) => i !== 0 || coordinates(p))
  const onPan = (axes, dx, dy) => {
    if (Math.abs(dx) + Math.abs(dy) <= 0.01) return
    const ratio = axis => (displayFrame[axis][1] - displayFrame[axis][0]) / (frame[axis][1] - frame[axis][0])
    setManualFrame(panFrame(frame, axes, dx * ratio(axes[0]), dy * ratio(axes[1])))
    setManualCamera(panFrame(displayFrame, axes, dx, dy))
  }
  const zoom = factor => { setManualFrame(zoomFrame(frame, factor)); setManualCamera(zoomFrame(displayFrame, factor)); setForcedMode(null) }
  const projectionProps = { current, destination, searched, route: path.filter(coordinates), onSelect: setSearched, frame: displayFrame, onPan, localGeometry: false, clipSelection: false, selectionFrame: frame, routeSegments: spatialSegments(path, frame), atlas: galaxyEnabled && !local ? atlas : null, spatial: local ? spatial.points || [] : null }
  const selectedDistance = distance(current?.coordinates, searched?.coordinates)
  return (
    <section className={`go-page go-galaxy go-g3d ${compact ? 'is-compact' : ''}`} aria-label='Galaxy map' data-mode={mode}>
      <div className='go-galaxy-sidebar'><h2>Galaxy Map</h2>
        <form onSubmit={e => { e.preventDefault(); search() }}><label>System<input aria-label='Galaxy system search' maxLength={128} value={query} onChange={e => setQuery(e.target.value)} /></label><button disabled={busy || !query.trim()}>Search</button></form>
        <div className='go-galaxy-mode'><button aria-pressed={mode === 'galaxy'} disabled={!galaxyEnabled} onClick={() => switchMode('galaxy')}>Galaxy</button><button aria-pressed={mode === 'local'} disabled={!localEnabled || !knownCenter} onClick={() => switchMode('local')}>Local</button></div>
        <button onClick={() => { setManualFrame(null); setManualCamera(null); setForcedMode(null) }}>Fit Route</button>
        <div className='go-galaxy-mode'><button aria-label='Galaxy zoom out' onClick={() => zoom(2)}>Zoom −</button><button aria-label='Galaxy zoom in' onClick={() => zoom(0.5)}>Zoom +</button></div>
        {wantsLocal && <div className='go-galaxy-axes'>{'xyz'.split('').map((axis, i) => <div key={axis}><span>{axis.toUpperCase()} {Math.round(spans[i])} LY</span><button aria-label={`Decrease ${axis.toUpperCase()} extent`} onClick={() => { const next = adjustSlice(frame, manualCamera, axis, -1); setManualCamera(next.camera); setManualFrame(next.selection) }}>−</button><button aria-label={`Increase ${axis.toUpperCase()} extent`} onClick={() => { const next = adjustSlice(frame, manualCamera, axis, 1); setManualCamera(next.camera); setManualFrame(next.selection) }}>+</button></div>)}</div>}
        {localStatus && <p className='go-galaxy-local-status' role='status'>{localStatus}</p>}
        <p>{current?.name || 'Current galactic position unknown'}</p><p>{coordinates(current) ? current.coordinates.map((v, i) => `${'XYZ'[i]} ${v}`).join(' · ') + ' LY' : 'No coordinates reported'}</p>
        {searched && <><p>{searched.name}</p><p>Distance {selectedDistance === null ? 'Unknown' : `${selectedDistance.toFixed(1)} LY`}</p><button onClick={() => { onViewSystem?.(); router.push({ pathname: '/nav/map', query: { system: searched.systemName || searched.name } }) }}>View System</button></>}
        <div role='status'>{!galaxyEnabled && !localEnabled ? 'Map renderers disabled' : wantsLocal && !local ? (galaxyEnabled ? 'Local detail unavailable; showing available overview' : 'Local detail unavailable; Galaxy Map disabled') : !remaining.complete ? 'Some route coordinates are unknown' : ''}{!local && galaxyEnabled && atlasNotice && <p>{atlasNotice}</p>}{notice && <p>{notice}</p>}</div>
      </div>
      <div ref={area} className='go-galaxy-area'><div className='go-galaxy-projections' style={dimensions}>{['xz', 'xy', 'yz'].map(axes => <Projection key={axes} {...projectionProps} axes={axes} pixels={JSON.parse(pixelKey)[axes][0]} pixelHeight={JSON.parse(pixelKey)[axes][1]} />)}</div></div>
    </section>
  )
}
