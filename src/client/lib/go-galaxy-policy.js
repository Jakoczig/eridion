// Physical LY thresholds match bounded spatial inspection, not zoom-click counts.
export const coordinates = p => p?.coordinates?.length === 3 && p.coordinates.every(v => Number.isFinite(v) && Math.abs(v) <= 1e6) ? p.coordinates : null
export function remainingRoute (context, route) {
  const usable = !route?.truncated && (!route?.routeState || ['verified', 'verified_last_known', 'cached'].includes(route.routeState))
  const hops = usable ? route?.hops || [] : []
  let start = hops.findIndex(h => h.isNext)
  if (start < 0) start = Math.max(0, hops.findIndex(h => h.isCurrent))
  const remaining = hops.slice(start)
  const points = remaining.map(h => (context?.points || []).find(p => p.name === h.name))
  return { rawPoints: points, points: points.filter(coordinates), complete: points.every(coordinates), count: remaining.length }
}
export function automaticFrame (current, remaining, full) {
  const points = [current, ...remaining.points].filter(coordinates)
  if (!points.length) return full
  return Object.fromEntries('xyz'.split('').map((axis, i) => {
    const values = points.map(p => p.coordinates[i]); const low = Math.min(...values); const high = Math.max(...values)
    const span = Math.max(i === 1 ? 100 : 200, (high - low) * 1.3)
    return [axis, [(low + high - span) / 2, (low + high + span) / 2]]
  }))
}
export function localScale (frame, previous = 'galaxy') {
  const span = a => frame[a][1] - frame[a][0]
  const limit = previous === 'local' ? 1200 : 800
  // Conservative 800-LY root intersection bound, before server byte/block/point limits.
  const roots = 'xyz'.split('').reduce((n, a) => n * (Math.ceil(frame[a][1] / 800) - Math.floor(frame[a][0] / 800)), 1)
  return span('x') <= limit && span('z') <= limit && span('y') <= limit / 2 && roots <= 256
}
export function adjustAxis (frame, axis, direction) {
  const [a, b] = frame[axis]; const span = b - a
  const step = span < 100 ? 10 : span < 500 ? 50 : 100
  const next = Math.max(20, Math.min(axis === 'y' ? 600 : 1200, span + direction * step))
  return { ...frame, [axis]: [(a + b - next) / 2, (a + b + next) / 2] }
}
