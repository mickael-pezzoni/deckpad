import { useEffect, useState } from 'react'
import { createPortal } from 'react-dom'
import type { Tone } from '../components/Tile'
import { Confirm } from '../components/Confirm'
import { AppIcon } from '../components/AppIcon'
import { useEventStream } from '../hooks/useEventStream'
import { Loader } from '../components/Loader'
import type { Stats } from '../hooks/useStatsStream'

type App = { name: string; cpu: number; ram: number; count: number }
type SortKey = 'cpu' | 'ram'

const SHOWN = 12

export function ProcessPage() {
  const [apps, setApps] = useState<App[] | null>(null)
  const [sortBy, setSortBy] = useState<SortKey>('cpu')
  const [target, setTarget] = useState<App | null>(null)
  const [notice, setNotice] = useState<string | null>(null)
  const [stats, setStats] = useState<Stats | null>(null)
  const connected = useEventStream<App[]>('/api/processes/stream', setApps)
  useEventStream<Stats>('/api/stats/stream', setStats)

  useEffect(() => {
    if (!notice) return
    const id = setTimeout(() => setNotice(null), 3000)
    return () => clearTimeout(id)
  }, [notice])

  async function kill(app: App) {
    setTarget(null)
    const r = await fetch('/api/processes/kill', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ name: app.name }),
    }).catch(() => null)
    if (r?.ok) setNotice(`${displayName(app.name)} fermé`)
    else if (r?.status === 403) setNotice(`Impossible de fermer ${displayName(app.name)} : droits administrateur requis`)
    else setNotice(`Impossible de fermer ${displayName(app.name)}`)
  }

  if (!apps) return <Loader label={connected ? 'Chargement…' : 'Connexion au PC…'} />

  const [top, ...rest] = [...apps].sort((a, b) => b[sortBy] - a[sortBy]).slice(0, SHOWN)
  const medium = rest.slice(0, 3)
  const small = rest.slice(3)

  return (
    <>
      <div className="process-bar">
        <div className="segmented">
          <button type="button" className={sortBy === 'cpu' ? 'active' : ''} onClick={() => setSortBy('cpu')}>
            Tri CPU
          </button>
          <button type="button" className={sortBy === 'ram' ? 'active' : ''} onClick={() => setSortBy('ram')}>
            Tri RAM
          </button>
        </div>
        {stats && (
          <span className="process-total">
            CPU {percent(stats.cpu)} · RAM {gb(stats.ramUsed)} / {gb(stats.ramTotal)}
          </span>
        )}
      </div>
      {/* Mosaïque : la plus gourmande en grand, les 3 suivantes en moyen, le reste en petit. */}
      <div className="mosaic">
        {top && (
          <div className="mosaic-top">
            <ProcessTile app={top} size="large" sortBy={sortBy} onClick={() => setTarget(top)} />
            {medium.length > 0 && (
              <div className="mosaic-medium">
                {medium.map((app) => (
                  <ProcessTile key={app.name} app={app} size="medium" sortBy={sortBy} onClick={() => setTarget(app)} />
                ))}
              </div>
            )}
          </div>
        )}
        {small.length > 0 && (
          <div className="mosaic-small">
            {small.map((app) => (
              <ProcessTile key={app.name} app={app} size="small" sortBy={sortBy} onClick={() => setTarget(app)} />
            ))}
          </div>
        )}
      </div>
      {/* Hors du carrousel : sinon « fixed » se place par rapport aux slides. */}
      {notice && createPortal(<div className="toast">{notice}</div>, document.body)}
      {target && (
        <Confirm
          title={`Fermer ${displayName(target.name)} ?`}
          message={
            target.count > 1
              ? `Les ${target.count} processus seront fermés. Les données non enregistrées seront perdues.`
              : 'Les données non enregistrées seront perdues.'
          }
          confirmLabel="Fermer"
          onConfirm={() => kill(target)}
          onCancel={() => setTarget(null)}
        />
      )}
    </>
  )
}

type TileProps = { app: App; size: 'large' | 'medium' | 'small'; sortBy: SortKey; onClick: () => void }

function ProcessTile({ app, size, sortBy, onClick }: TileProps) {
  const badge = usageBadge(app)
  const count = app.count > 1 ? `${app.count} processus` : '1 processus'
  const className = ['tile', 'tile-button', 'process-tile', `process-${size}`, badge && `tile-${badge.tone}`]
    .filter(Boolean)
    .join(' ')
  const badgeEl = badge && <span className={`badge badge-${badge.tone}`}>{badge.text}</span>
  const head = (
    <span className="process-head">
      <span className="process-icon">
        <AppIcon name={app.name} />
      </span>
      <span className="process-name">
        <span className="process-title">{displayName(app.name)}</span>
        <span className="process-sub">
          {size !== 'large' && badgeEl}
          <span>{size === 'small' ? `${percent(app.cpu)} · ${mb(app.ram)}` : count}</span>
        </span>
      </span>
    </span>
  )

  return (
    <button type="button" className={className} onClick={onClick}>
      {head}
      {size === 'large' && (
        <>
          <span className="process-figures">
            <Figure value={app.cpu.toFixed(1).replace('.', ',')} unit="%" label="CPU" />
            <Figure {...splitUnit(mb(app.ram))} label="RAM" />
          </span>
          {badgeEl}
        </>
      )}
      {size === 'medium' && (
        <span className="process-value">
          <strong>{sortBy === 'cpu' ? percent(app.cpu) : mb(app.ram)}</strong>
          <span>{sortBy === 'cpu' ? mb(app.ram) : percent(app.cpu)}</span>
        </span>
      )}
    </button>
  )
}

function Figure({ value, unit, label }: { value: string; unit: string; label: string }) {
  return (
    <span className="process-figure">
      <strong>
        {value}
        <small> {unit}</small>
      </strong>
      <span>{label}</span>
    </span>
  )
}

function splitUnit(text: string) {
  const i = text.lastIndexOf(' ')
  return { value: text.slice(0, i), unit: text.slice(i + 1) }
}

function percent(value: number) {
  return `${value.toFixed(1).replace('.', ',')} %`
}

function gb(bytes: number) {
  return `${(bytes / GB).toFixed(1).replace('.', ',')} Go`
}

// Seuils de consommation : CPU en % de la machine, RAM en octets.
const GB = 1024 ** 3

function usageBadge(app: App): { text: string; tone: Tone } | undefined {
  if (app.cpu >= 50 || app.ram >= 4 * GB) return { text: 'Très élevé', tone: 'critical' }
  if (app.cpu >= 20 || app.ram >= 2 * GB) return { text: 'Élevé', tone: 'warning' }
  return undefined
}

function displayName(name: string) {
  return name.replace(/\.exe$/i, '')
}

function mb(bytes: number) {
  return bytes >= 1024 ** 3
    ? `${(bytes / 1024 ** 3).toFixed(1).replace('.', ',')} Go`
    : `${Math.round(bytes / 1024 ** 2)} Mo`
}
