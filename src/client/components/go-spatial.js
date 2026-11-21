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
export function SpatialCanvas ({ points, axes, frame, width, height }) {
  const ref = useRef(null)
  useEffect(() => {
    const canvas = ref.current; if (!canvas) return
    const render = () => {
      const r = canvas.getBoundingClientRect(); const dpr = Math.min(2, window.devicePixelRatio || 1)
      const w = Math.max(1, Math.min(2048, Math.ceil(r.width * dpr))); const h = Math.max(1, Math.min(2048, Math.ceil(r.height * dpr)))
      canvas.width = w; canvas.height = h; const ctx = canvas.getContext('2d'); const image = ctx.createImageData(w, h)
      for (const p of points) { const at = spatialPixel(p, axes, frame, w, h); if (!at) continue; const i = (at[1] * w + at[0]) * 4; image.data[i] = 168; image.data[i + 1] = 231; image.data[i + 2] = 253; image.data[i + 3] = Math.min(255, image.data[i + 3] + 130) }
      ctx.putImageData(image, 0, 0); canvas.dataset.positions = String(points.length); canvas.dataset.mode = 'verified-xyz-pixel-aggregation'
    }
    render(); const observer = new window.ResizeObserver(render); observer.observe(canvas); return () => observer.disconnect()
  }, [points, axes, JSON.stringify(frame)])
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
