import { useEffect, useState } from 'react'
// Cached snapshots of the selected commander (D1: same on every trusted
// display); rendering never initiates an external request.
export default function GoProviderReference ({ role, sendEvent, publicGoals = false }) {
  const [snapshots, setSnapshots] = useState([])
  useEffect(() => { let active = true; sendEvent('getGoProviderGameplay').then(values => { if (active) setSnapshots(values.filter(v => (v.snapshot?.activeCommander || (publicGoals && v.snapshot?.source?.includes('communitygoals')))).map(v => v.snapshot)) }).catch(() => {}); return () => { active = false } }, [role, sendEvent, publicGoals])
  if (!snapshots.length) return null
  return <details className='go-provider-reference'><summary>Provider snapshots · resolved by source priority (Frontier, journal, then other providers)</summary>{snapshots.map((s, i) => <section key={i}><h3>{s.source}</h3><small>Fetched {s.fetchedAt} · expires {s.expiresAt}</small><dl>{s.facts.map((f, i) => <div key={i}><dt>{f.name}</dt><dd>{f.value}</dd></div>)}</dl></section>)}</details>
}
