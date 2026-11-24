// visibleInterval runs fn every ms while the document is visible (G3K
// PERF-06). Hidden or minimized windows skip ticks (nothing is painted
// there); becoming visible runs fn at once. Keyboard focus is irrelevant: an
// unfocused visible screen keeps refreshing. Returns a stop function.
export function visibleInterval (fn, ms) {
  const hidden = () => typeof document !== 'undefined' && document.visibilityState === 'hidden'
  const timer = setInterval(() => { if (!hidden()) fn() }, ms)
  const onVisibility = () => { if (!hidden()) fn() }
  if (typeof document !== 'undefined') document.addEventListener('visibilitychange', onVisibility)
  return () => {
    clearInterval(timer)
    if (typeof document !== 'undefined') document.removeEventListener('visibilitychange', onVisibility)
  }
}
