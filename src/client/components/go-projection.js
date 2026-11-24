import { useEffect, useState } from 'react'
import { viewStore, EMPTY } from '../lib/go-view-store'

// useGoProjection reads one gameplay projection through the shared view
// store (G3K PERF-02): equivalent consumers share the initial request and
// the immutable snapshot, and a newer live broadcast is never replaced by a
// delayed initial reply (see lib/go-view-store.js).
export default function useGoProjection ({ connected, ready, sendEvent, eventListener }, rpc, changed, delta = null) {
  const [snapshot, setSnapshot] = useState(EMPTY)
  useEffect(() => {
    if (!connected || !ready) { setSnapshot(EMPTY); return }
    return viewStore({ sendEvent, eventListener }).subscribe({ rpc, changed, delta }, setSnapshot)
  }, [connected, ready, sendEvent, eventListener, rpc, changed, delta])
  return { value: snapshot.value, error: snapshot.error }
}
