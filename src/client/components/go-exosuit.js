import { Icon, Metric } from './go-terminal-widgets'

const percent = v => (typeof v === 'number' && Number.isFinite(v) ? Math.round(v * 100) : null)
const controlledLabels = { suit: 'On foot', srv: 'In an SRV', fighter: 'Flying a fighter', taxi: 'In a taxi', multicrew: 'In multicrew', ship: 'Aboard the main ship' }

// Known suit equipment is last-known journal history and stays available
// offline. Live values come only from a verified, current-account Status that
// reports the commander on foot; the main ship is never shown as a suit.
export default function GoExosuit ({ workspace }) {
  const exo = workspace?.exosuit || {}
  const status = workspace?.parity?.status || {}
  const live = status.state === 'known' ? status.onFoot : null
  const current = exo.current
  const controlled = exo.controlled || workspace?.parity?.ship?.controlled
  return (
    <section className='go-page go-exosuit' aria-label='Exosuit'>
      <div className='go-page__heading'><h2><Icon name='shield' /> Exosuit</h2>{controlled && <span>{controlledLabels[controlled] || 'Context unknown'}</span>}</div>
      {current
        ? (
          <article className='go-exosuit-loadout'>
            <h3>{current.suitLabel || current.suit || 'Suit'}</h3>
            <p>Loadout {current.name || 'unnamed'} · {current.grade != null ? 'Grade ' + current.grade : 'Grade not reported'}</p>
            {current.mods?.length > 0 && <p>Suit modifications: {current.mods.join(', ')}</p>}
            <div className='go-fact-grid'>{(current.modules || []).map(m => <Metric key={m.slot} label={m.slot} value={(m.label || m.module || 'Unknown') + (m.class != null ? ' · Class ' + m.class : '')} />)}</div>
            <small>Last witnessed {exo.observedAt || 'time unknown'}{exo.state === 'partial' ? ' · some changes could not be applied' : ''}</small>
          </article>
          )
        : <p role='status'>No suit loadout witnessed yet. Equipment appears after the game reports a suit loadout.</p>}
      <h3 className='text-primary'>Live suit telemetry</h3>
      {live
        ? (
          <div className='go-metrics'>
            <Metric label='Health' value={percent(live.health)} unit='%' />
            <Metric label='Oxygen' value={percent(live.oxygen)} unit='%' />
            <Metric label='Temperature' value={live.temperatureK} unit='K' />
            <Metric label='Gravity' value={live.gravityG} unit='g' />
            <Metric label='Selected weapon' value={live.selectedWeaponLabel || live.selectedWeapon || null} />
          </div>
          )
        : <p role='status'>Live suit telemetry unavailable: shown only while this commander is on foot and the game status is current.</p>}
      {(exo.loadouts || []).length > 1 && <details><summary>Known loadouts ({exo.loadouts.length})</summary>{exo.loadouts.map(l => <p key={l.loadoutId}>{l.name || 'Unnamed'} · {l.suitLabel || l.suit || 'Suit'}{l.loadoutId === exo.currentLoadoutId ? ' · current' : ''}</p>)}</details>}
    </section>
  )
}
