// Shared gameplay view store (G3K PERF-02/06).
//
// Components that show the same projection share one store entry per
// transport: one initial request (single flight), one live listener and one
// immutable snapshot, so N consumers cost one request and an unchanged value
// never re-renders them. Different projections stay independent.
//
// Ordering: a live broadcast always wins over a slower initial reply, deltas
// received before the initial reply are merged into it, and a reply that
// belongs to a previous connection, account or entry is discarded.
//
// Demand: the store declares to the host which live topics its visible
// entries need (setGoDemand, replace semantics with an increasing
// generation). A hidden or minimized document releases them; becoming
// visible declares them again and the host sends what changed meanwhile.
// Keyboard focus is irrelevant: an unfocused visible screen stays live.

const TOPICS = { getGoNavigation: 'navigation', getGoExploration: 'exploration', getGoShip: 'ship', getGoWorkspace: 'workspace', getGoNavigationContext: 'context' }
// Pushes some components listen to directly; declared whenever visible so a
// declared demand never silences them (the host still only pushes them to
// connections that requested them).
const ALWAYS = ['providers', 'spatial', 'community']
const EMPTY = Object.freeze({ value: null, error: '' })
const stores = new WeakMap()

function documentHidden () {
  return typeof document !== 'undefined' && document.visibilityState === 'hidden'
}

export function createGoViewStore (transport) {
  const entries = new Map()
  let epoch = 0
  let generation = 0
  let declared = null
  let declareTimer = null
  let failures = 0
  let hidden = documentHidden()

  const notify = entry => { for (const l of entry.listeners) l(entry.snapshot) }
  const set = (entry, value, error = '') => {
    if (entry.snapshot.value === value && entry.snapshot.error === error) return
    entry.snapshot = Object.freeze({ value, error })
    notify(entry)
  }

  const load = entry => {
    const mine = { epoch, entry }
    entry.pending = transport.sendEvent(entry.rpc).then(next => {
      if (mine.epoch !== epoch || entries.get(entry.key) !== entry || entry.liveSeen) return
      set(entry, entry.latestDelta && entry.delta ? entry.delta.merge(next, entry.latestDelta) : next)
    }).catch(() => {
      if (mine.epoch !== epoch || entries.get(entry.key) !== entry || entry.liveSeen) return
      set(entry, entry.snapshot.value, 'Local journal view unavailable. Reconnect or check journal status.')
    }).finally(() => { if (entries.get(entry.key) === entry) entry.pending = null })
  }

  const topics = () => {
    if (hidden) return []
    const out = new Set(ALWAYS)
    for (const e of entries.values()) if (TOPICS[e.rpc]) out.add(TOPICS[e.rpc])
    return [...out].sort()
  }
  // Coalesce subscribe/unsubscribe bursts into one replace.
  const declare = () => {
    if (declareTimer) return
    declareTimer = setTimeout(() => {
      declareTimer = null
      const next = JSON.stringify({ visible: !hidden, topics: topics() })
      if (next === declared) return
      declared = next
      generation++
      transport.sendEvent('setGoDemand', { generation: String(generation), ...JSON.parse(next) }).then(() => { failures = 0 }).catch(() => {
        // A failed replace (account switch, transient error) is retried
        // with bounded backoff so the host never keeps an outdated demand
        // (review R5).
        if (declared !== next) return
        declared = null
        failures = Math.min(failures + 1, 5)
        setTimeout(declare, 250 * 2 ** failures)
      })
    }, 0)
  }

  const onVisibility = () => {
    const now = documentHidden()
    if (now === hidden) return
    hidden = now
    declare()
  }
  const onAccount = () => {
    // The host switched commander: every entry belongs to the old account.
    epoch++
    for (const e of entries.values()) { e.liveSeen = false; e.latestDelta = null; set(e, null); load(e) }
  }
  if (typeof document !== 'undefined') document.addEventListener('visibilitychange', onVisibility)
  if (typeof window !== 'undefined') window.addEventListener('eridion-account-changed', onAccount)

  return {
    subscribe ({ rpc, changed, delta = null }, listener) {
      const key = rpc + '|' + changed + '|' + (delta ? delta.name : '')
      let entry = entries.get(key)
      // First entry after the store emptied (for example after a reconnect,
      // which unmounts every view): declare to the new connection even if
      // the topics equal the last declaration (review R9).
      if (entries.size === 0) declared = null
      if (!entry) {
        entry = { key, rpc, delta, listeners: new Set(), snapshot: EMPTY, liveSeen: false, latestDelta: null, pending: null }
        entries.set(key, entry)
        const owner = entry
        const stopLive = transport.eventListener(changed, next => {
          if (entries.get(key) !== owner) return
          owner.liveSeen = true; owner.latestDelta = null
          set(owner, next)
        })
        const stopDelta = delta
          ? transport.eventListener(delta.name, next => {
            if (entries.get(key) !== owner) return
            owner.latestDelta = next
            if (owner.snapshot.value) set(owner, delta.merge(owner.snapshot.value, next))
          })
          : () => {}
        entry.stop = () => { stopLive(); stopDelta() }
        load(entry)
        declare()
      }
      entry.listeners.add(listener)
      listener(entry.snapshot)
      return () => {
        entry.listeners.delete(listener)
        if (entry.listeners.size === 0 && entries.get(key) === entry) {
          entries.delete(key)
          entry.stop()
          declare()
        }
      }
    },
    // reset forgets every entry (the connection was lost); listeners that
    // remain subscribe again when the connection is back.
    reset () { epoch++; declared = null; for (const e of entries.values()) e.stop(); entries.clear() },
    get size () { return entries.size },
    dispose () {
      if (typeof document !== 'undefined') document.removeEventListener('visibilitychange', onVisibility)
      if (typeof window !== 'undefined') window.removeEventListener('eridion-account-changed', onAccount)
    }
  }
}

// viewStore returns the store bound to a transport (one per sendEvent).
export function viewStore (transport) {
  let store = stores.get(transport.sendEvent)
  if (!store) {
    store = createGoViewStore(transport)
    stores.set(transport.sendEvent, store)
  }
  return store
}

export { EMPTY }
