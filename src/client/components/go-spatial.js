import { useEffect, useRef, useState } from 'react'
import { project } from '../lib/go-display-controller'
// Exact XYZ semantics: frozen G3A spatial.py/local_view.py, GPL-3.0,
// 5d1cbf1aee7a707259000e57ea963175f7cd4f3b. Positions are not spectral classes.
export function spatialFrame (center, span, thickness, depth = span) { return Object.fromEntries('xyz'.split('').map((a, i) => [a, [Math.max(-1e6, center[i] - [span, thickness, depth][i] / 2), Math.min(1e6, center[i] + [span, thickness, depth][i] / 2)]])) }
export function spatialPixel (point, axes, frame, w, h) {
  if (!point?.every((p, i) => Number.isFinite(p) && p >= frame['xyz'[i]][0] && p < frame['xyz'[i]][1])) return null
  const p = project(point, axes, frame)
  return [Math.min(w - 1, Math.floor(p.x * w)), Math.min(h - 1, Math.floor(p.y * h))]
}
export function spatialSegments (route, frame) {
  const result = []
  for (let i = 1; i < route.length; i++) {
    const a = route[i - 1]?.coordinates; const b = route[i]?.coordinates; if (!a || !b) continue; let enter = 0; let leave = 1
    for (let j = 0; j < 3; j++) { const [lo, hi] = frame['xyz'[j]]; const delta = b[j] - a[j]; if (!delta) { if (a[j] < lo || a[j] >= hi) { enter = 1; leave = 0; break } } else { const bounds = [(lo - a[j]) / delta, (hi - a[j]) / delta].sort((x, y) => x - y); enter = Math.max(enter, bounds[0]); leave = Math.min(leave, bounds[1]) } }
    if (enter <= leave) result.push([enter, leave].map(t => a.map((v, j) => v + (b[j] - v) * t)))
  }
  return result
}
// paintPoints aggregates points into image (RGBA, w x h) exactly as
// spatialPixel places them (same bounds test, projection and clamping), with
// no per-point allocation (G3K PERF-09). Returns the number painted.
export function paintPoints (data, w, h, points, axes, frame) {
  const a = 'xyz'.indexOf(axes[0]); const b = 'xyz'.indexOf(axes[1])
  const [x0, x1] = frame.x; const [y0, y1] = frame.y; const [z0, z1] = frame.z
  const [a0, a1] = frame[axes[0]]; const [b0, b1] = frame[axes[1]]
  const aSpan = a1 - a0; const bSpan = b1 - b0
  let painted = 0
  for (let k = 0; k < points.length; k++) {
    const p = points[k]
    if (!p) continue
    const px = p[0]; const py = p[1]; const pz = p[2]
    if (!Number.isFinite(px) || !Number.isFinite(py) || !Number.isFinite(pz) || px < x0 || px >= x1 || py < y0 || py >= y1 || pz < z0 || pz >= z1) continue
    const u = (p[a] - a0) / aSpan; const v = 1 - (p[b] - b0) / bSpan
    const cx = Math.min(w - 1, Math.floor(u * w)); const cy = Math.min(h - 1, Math.floor(v * h))
    const i = (cy * w + cx) * 4
    data[i] = 168; data[i + 1] = 231; data[i + 2] = 253; data[i + 3] = Math.min(255, data[i + 3] + 130)
    painted++
  }
  return painted
}

// SpatialCanvas paints verified XYZ positions. Paint requests (new data,
// frame, size or DPR) coalesce into one animation frame; the ImageData is
// reused while the size is unchanged; hidden documents are not painted and
// are repainted once when visible; pending work is cancelled on unmount.
export function SpatialCanvas ({ points, axes, frame, width, height }) {
  const ref = useRef(null)
  const frameKey = JSON.stringify(frame)
  useEffect(() => {
    const canvas = ref.current; if (!canvas) return
    let pending = 0; let image = null; let alive = true
    const paint = () => {
      pending = 0
      if (!alive || (typeof document !== 'undefined' && document.visibilityState === 'hidden')) return
      const r = canvas.getBoundingClientRect(); const dpr = Math.min(2, window.devicePixelRatio || 1)
      const w = Math.max(1, Math.min(2048, Math.ceil(r.width * dpr))); const h = Math.max(1, Math.min(2048, Math.ceil(r.height * dpr)))
      if (canvas.width !== w || canvas.height !== h) { canvas.width = w; canvas.height = h; image = null }
      const ctx = canvas.getContext('2d')
      if (!image || image.width !== w || image.height !== h) image = ctx.createImageData(w, h)
      else image.data.fill(0)
      paintPoints(image.data, w, h, points, axes, frame)
      ctx.putImageData(image, 0, 0); canvas.dataset.positions = String(points.length); canvas.dataset.mode = 'verified-xyz-pixel-aggregation'
      canvas.dataset.paints = String((Number(canvas.dataset.paints) || 0) + 1)
    }
    const request = () => {
      if (pending || !alive) return
      pending = typeof window.requestAnimationFrame === 'function' ? window.requestAnimationFrame(paint) : setTimeout(paint, 16)
    }
    request()
    const observer = typeof window.ResizeObserver === 'function' ? new window.ResizeObserver(request) : null
    if (observer) observer.observe(canvas)
    const onVisibility = () => { if (document.visibilityState !== 'hidden') request() }
    if (typeof document !== 'undefined') document.addEventListener('visibilitychange', onVisibility)
    return () => {
      alive = false
      if (pending) { if (typeof window.cancelAnimationFrame === 'function') window.cancelAnimationFrame(pending); else clearTimeout(pending) }
      if (observer) observer.disconnect()
      if (typeof document !== 'undefined') document.removeEventListener('visibilitychange', onVisibility)
    }
  }, [points, axes, frameKey])
  return <foreignObject x='0' y='0' width={width} height={height} pointerEvents='none' data-spatial-projection={axes}><canvas ref={ref} style={{ width: '100%', height: '100%', display: 'block' }} aria-label='Verified EDSM XYZ positions; overlapping pixels aggregated' /></foreignObject>
}
const mib = n => `${((n || 0) / 1048576).toFixed(2)} MiB`
export function SpatialSettings ({ role, sendEvent, onChange }) {
  const [value, setValue] = useState(null); const [error, setError] = useState(''); const [refresh, setRefresh] = useState(0)
  useEffect(() => { if (role !== 'owner') return; let active = true; let timer; const load = () => sendEvent('getGoSpatialStatus').then(v => { if (!active) return; setValue(v); if (v.busy) timer = setTimeout(load, 800); else { onChange?.(); window.dispatchEvent(new Event('eridion-spatial-changed')) } }).catch(() => { if (active) setError('Spatial status unavailable') }); load(); return () => { active = false; clearTimeout(timer) } }, [role, sendEvent, refresh])
  if (role !== 'owner') return null
  const action = async (name, message = {}) => { setError(''); try { setValue(await sendEvent(name, message)); setRefresh(v => v + 1) } catch (e) { setError(e.message || 'Spatial operation failed; verified cache retained') } }
  const configure = (key, v) => action('configureGoSpatial', { downloads: !!value?.downloads, automatic: !!value?.automatic, metered: !!value?.metered, offline: !!value?.offline, [key]: v })
  const enabled = value?.downloads && !value?.offline && !value?.metered
  return <section className='go-spatial-settings' aria-label='Spatial Atlas settings'><h3>Spatial Atlas · actual XYZ</h3><p>Requests reveal approximate requested regions to eridion.fitoserver.org. No commander identity, journal, credentials or tokens are sent. Downloads are disabled by default. Two downloads maximum; regional disk cache {mib(value?.cacheBytes)} / 64 MiB. Density retains its independent 128 MiB policy.</p>{[['downloads', 'Permit selective spatial downloads'], ['automatic', 'Update signed spatial index daily at startup'], ['offline', 'Offline mode'], ['metered', 'Pause on metered connection']].map(([key, label]) => <label key={key}><input type='checkbox' checked={!!value?.[key]} disabled={!value} onChange={e => configure(key, e.target.checked)} />{label}</label>)}<div className='go-toolbar'><button disabled={!enabled || value?.busy} onClick={() => action('installGoSpatial')}>Check and install signed spatial index</button><button disabled={!enabled || value?.busy || !value?.bubbleBytes} onClick={() => action('installGoBubble', { sha256: value.bubbleHash, bytes: value.bubbleBytes })}>Install permanent 350 LY Bubble ({mib(value?.bubbleBytes)})</button><button disabled={value?.busy || !value?.previous} onClick={() => action('rollbackGoSpatial')}>Restore previous spatial index</button><button disabled={value?.busy || !value?.bubbleInstalled} onClick={() => { if (window.confirm('Remove the permanent offline Bubble? Density and regional cache remain.')) action('removeGoBubble', { confirm: true }) }}>Remove offline Bubble…</button></div><p role='status'>{value?.busy ? 'Downloading and verifying; startup remains available.' : value?.installed ? `Verified release ${value.installed}` : 'Not installed; first install the signed spatial index to inspect exact Bubble size.'} · Bubble {value?.bubbleInstalled ? `available offline · ${value.bubbleSystems} positions · ${value.bubbleSource}` : 'not installed'} · {value?.bubbleUpdate ? 'New Bubble available; owner confirmation required.' : ''} · received {mib(value?.downloaded)}</p>{(error || value?.error) && <p role='alert'>{error || value.error}</p>}</section>
}
