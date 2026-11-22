import { useRef } from 'react'
import { AtlasTiles, atlasExtents } from './go-atlas'
import { project } from '../lib/go-display-controller'
import { SpatialCanvas } from './go-spatial'
// These maps mount after the client bootstrap (React 17); IDs remain unique across overlays.
let projectionSerial = 0
const validPoint = p => p?.coordinates?.length === 3 && p.coordinates.every(v => Number.isFinite(v) && Math.abs(v) <= 1e6)
export function routeFrame (current, destination, full = atlasExtents) {
  if (!validPoint(current) || !validPoint(destination)) return full
  return Object.fromEntries(['x', 'y', 'z'].map((axis, i) => { const a = current.coordinates[i]; const b = destination.coordinates[i]; const span = Math.max(100, Math.abs(a - b) * 1.3); const center = (a + b) / 2; return [axis, [center - span / 2, center + span / 2]] }))
}
export function zoomFrame (frame, factor) { return Object.fromEntries(Object.entries(frame).map(([axis, [a, b]]) => { const half = Math.max(5, Math.min(1e6, (b - a) * factor / 2)); const center = (a + b) / 2; return [axis, [center - half, center + half]] })) }
export function panFrame (frame, axes, dx, dy) { const out = { ...frame }; for (const [axis, fraction] of [[axes[0], -dx], [axes[1], dy]]) { const [a, b] = frame[axis]; out[axis] = [a + (b - a) * fraction, b + (b - a) * fraction] } return out }
// Canonical density layout: fixed 100:8 edge panels, independent of selection.
export const projectionUnits = Object.freeze({ x: 100, y: 8, z: 100 })
export function panelLayout (width, height) {
  const { x, y, z } = projectionUnits
  const unit = Math.max(0.01, Math.min((width - 6) / (x + y), (height - 6) / (z + y)))
  return { width: unit * (x + y) + 6, height: unit * (z + y) + 6, gridTemplateColumns: `${x}fr ${y}fr`, gridTemplateRows: `${z}fr ${y}fr` }
}
export function fittedFrame (frame) {
  const unit = Math.max(...Object.entries(projectionUnits).map(([axis, units]) => (frame[axis][1] - frame[axis][0]) / units))
  return Object.fromEntries(Object.entries(projectionUnits).map(([axis, units]) => {
    const center = (frame[axis][0] + frame[axis][1]) / 2; const half = unit * units / 2
    return [axis, [center - half, center + half]]
  }))
}
export function Projection ({ axes, current, destination, searched, route = [], onSelect, frame, onPan, layers = [], atlas, showRoute = true, pixels = 320, pixelHeight, spatial = null, routeSegments = null, localGeometry = false, selectionFrame = null }) {
  const drag = useRef(null); const clipID = useRef(null)
  if (!clipID.current) clipID.current = `galaxy-clip-${++projectionSerial}`
  const height = localGeometry ? 100 * (frame[axes === 'xy' ? 'y' : 'z'][1] - frame[axes === 'xy' ? 'y' : 'z'][0]) / (frame.x[1] - frame.x[0]) : axes === 'xy' ? 8 : 100
  const width = axes === 'yz' ? localGeometry ? 100 * (frame.y[1] - frame.y[0]) / (frame.x[1] - frame.x[0]) : 8 : 100
  const low = selectionFrame && project(['x', 'y', 'z'].map(a => selectionFrame[a][0]), axes, frame)
  const high = selectionFrame && project(['x', 'y', 'z'].map(a => selectionFrame[a][1]), axes, frame)
  const position = p => project(p?.coordinates, axes, frame)
  const markerPosition = p => !selectionFrame || (validPoint(p) && p.coordinates.every((v, i) => v >= selectionFrame['xyz'[i]][0] && v <= selectionFrame['xyz'[i]][1])) ? position(p) : null
  const points = route.map(position).filter(Boolean).map(p => `${p.x * width},${p.y * height}`).join(' ')
  return <figure className={`go-galaxy-plane go-galaxy-${axes}`}><figcaption>{axes[0].toUpperCase()} / {axes[1].toUpperCase()}</figcaption><svg data-projection={axes} viewBox={`0 0 ${width} ${height}`} preserveAspectRatio='xMidYMid meet' role='img' aria-label={`Galaxy ${axes.toUpperCase()} projection`} onPointerDown={e => { drag.current = { x: e.clientX, y: e.clientY }; e.currentTarget.setPointerCapture?.(e.pointerId) }} onPointerUp={e => { if (drag.current) { const r = e.currentTarget.getBoundingClientRect(); onPan(axes, (e.clientX - drag.current.x) / r.width, (e.clientY - drag.current.y) / r.height); drag.current = null } }}>{selectionFrame && <defs><clipPath id={clipID.current}><rect x={low.x * width} y={high.y * height} width={(high.x - low.x) * width} height={(low.y - high.y) * height} /></clipPath></defs>}<g clipPath={selectionFrame ? `url(#${clipID.current})` : undefined}>{spatial ? <SpatialCanvas points={spatial} axes={axes} frame={frame} width={width} height={height} /> : <AtlasTiles atlas={atlas} axes={axes} frame={frame} width={width} height={height} pixels={pixels} pixelHeight={pixelHeight} />}{showRoute && <g className='go-galaxy-route' pointerEvents='none'>{(routeSegments ? routeSegments.map(pair => pair.map(coordinates => position({ coordinates })).map(p => `${p.x * width},${p.y * height}`).join(' ')) : [points]).map((segment, i) => <g key={i}><polyline points={segment} fill='none' stroke='#000' strokeOpacity='.45' strokeWidth='2.8' vectorEffect='non-scaling-stroke' /><polyline points={segment} fill='none' stroke='#9fb8c9' strokeOpacity='.65' strokeWidth='1.1' vectorEffect='non-scaling-stroke' /></g>)}{route.slice(0, 256).map((point, i) => { const p = markerPosition(point); return p && <circle key={i} data-marker='waypoint' data-axes={axes} data-position={`${p.x},${p.y}`} cx={p.x * width} cy={p.y * height} r='.22' fill='none' stroke='#b8cbd5' strokeWidth='.7' vectorEffect='non-scaling-stroke' /> })}</g>}{layers.map((point, i) => { const p = markerPosition(point); return p && <g key={i} role='button' tabIndex={0} aria-label={point.name} onClick={() => onSelect(point)} onKeyDown={e => { if (e.key === 'Enter') onSelect(point) }}><circle cx={p.x * width} cy={p.y * height} r={axes === 'xz' ? 0.45 : 0.4} fill={point.category === 'nebula' ? '#b988ff' : '#55dfff'} /><title>{point.name} · {point.source}</title></g> })}{[['destination', destination, '#ffc631'], ['searched', searched, '#55e5ff'], ['current', current, '#39ff69']].map(([kind, point, color]) => { const p = markerPosition(point); return p && <g key={kind} data-marker={kind} data-axes={axes} data-position={`${p.x},${p.y}`} role='button' tabIndex={0} aria-label={`${kind}: ${point.name}`} onClick={() => onSelect?.(point)} onKeyDown={e => { if (e.key === 'Enter') onSelect?.(point) }}><circle cx={p.x * width} cy={p.y * height} r={kind === 'current' ? 1.1 : 0.8} fill={color} stroke='#000' strokeWidth='.2' /><title>{point.name} · {point.source}</title></g> })}</g></svg></figure>
}
