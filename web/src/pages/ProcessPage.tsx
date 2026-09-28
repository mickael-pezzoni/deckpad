import { useEffect, useState } from 'react'
import { createPortal } from 'react-dom'
import { useTranslation } from 'react-i18next'
import i18n from '../i18n'
import { decimal, gb, mb, percent } from '../format'
import type { Tone } from '../components/Tile'
import { HoldButton } from '../components/HoldButton'
import { AppIcon } from '../components/AppIcon'
import { useEventStream } from '../hooks/useEventStream'
import { Loader } from '../components/Loader'
import { SearchField, matches } from '../components/SearchField'
import type { Stats } from '../hooks/useStatsStream'

type App = { name: string; cpu: number; ram: number; count: number }
type SortKey = 'cpu' | 'ram'

const SHOWN = 12
const HOLD_MS = 1500

export function ProcessPage() {
  const { t } = useTranslation()
  const [apps, setApps] = useState<App[] | null>(null)
  const [sortBy, setSortBy] = useState<SortKey>('cpu')
  const [notice, setNotice] = useState<string | null>(null)
  const [stats, setStats] = useState<Stats | null>(null)
  const [query, setQuery] = useState('')
  const connected = useEventStream<App[]>('/api/processes/stream', setApps)
  useEventStream<Stats>('/api/stats/stream', setStats)

  useEffect(() => {
    if (!notice) return
    const id = setTimeout(() => setNotice(null), 3000)
    return () => clearTimeout(id)
  }, [notice])

  async function kill(app: App) {
    const r = await fetch('/api/processes/kill', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ name: app.name }),
    }).catch(() => null)
    if (r?.ok) setNotice(t('process.closed', { name: displayName(app.name) }))
    else if (r?.status === 403) setNotice(t('process.adminRequired', { name: displayName(app.name) }))
    else setNotice(t('process.cannotClose', { name: displayName(app.name) }))
  }

  const tooShort = () => setNotice(t('process.holdToClose'))

  if (!apps) return <Loader label={connected ? undefined : t('common.connecting')} />

  const sorted = [...apps].sort((a, b) => b[sortBy] - a[sortBy])
  const searching = query.trim() !== ''
  const found = searching ? sorted.filter((app) => matches(displayName(app.name), query)) : []
  const [top, ...rest] = sorted.slice(0, SHOWN)
  const medium = rest.slice(0, 3)
  const small = rest.slice(3)

  return (
    <>
      <div className={searching ? 'process-bar process-searching' : 'process-bar'}>
        <div className="segmented">
          <button type="button" className={sortBy === 'cpu' ? 'active' : ''} onClick={() => setSortBy('cpu')}>
            {t('process.sortCpu')}
          </button>
          <button type="button" className={sortBy === 'ram' ? 'active' : ''} onClick={() => setSortBy('ram')}>
            {t('process.sortRam')}
          </button>
        </div>
        <SearchField value={query} onChange={setQuery} placeholder={t('process.search')} />
        {stats && (
          <span className="process-total">
            CPU {percent(stats.cpu)} · RAM {gb(stats.ramUsed)} / {gb(stats.ramTotal)} {t('units.GB')}
          </span>
        )}
      </div>
      {searching ? (
        // Recherche : tous les processus qui correspondent, en petites tuiles ; seule la liste défile.
        found.length > 0 ? (
          <div className="process-results">
            {found.map((app) => (
              <ProcessTile key={app.name} app={app} size="small" sortBy={sortBy} onHold={() => kill(app)} onTooShort={tooShort} />
            ))}
          </div>
        ) : (
          <p className="process-none">{t('process.none', { query: query.trim() })}</p>
        )
      ) : (
        /* Mosaïque : la plus gourmande en grand, les 3 suivantes en moyen, le reste en petit. */
        <div className="mosaic">
          {top && (
            <div className="mosaic-top">
              <ProcessTile app={top} size="large" sortBy={sortBy} onHold={() => kill(top)} onTooShort={tooShort} />
              {medium.length > 0 && (
                <div className="mosaic-medium">
                  {medium.map((app) => (
                    <ProcessTile key={app.name} app={app} size="medium" sortBy={sortBy} onHold={() => kill(app)} onTooShort={tooShort} />
                  ))}
                </div>
              )}
            </div>
          )}
          {small.length > 0 && (
            <div className="mosaic-small">
              {small.map((app) => (
                <ProcessTile key={app.name} app={app} size="small" sortBy={sortBy} onHold={() => kill(app)} onTooShort={tooShort} />
              ))}
            </div>
          )}
        </div>
      )}
      {/* Hors du carrousel : sinon « fixed » se place par rapport aux slides. */}
      {notice && createPortal(<div className="toast">{notice}</div>, document.body)}
    </>
  )
}

type TileProps = {
  app: App
  size: 'large' | 'medium' | 'small'
  sortBy: SortKey
  onHold: () => void
  onTooShort: () => void
}

function ProcessTile({ app, size, sortBy, onHold, onTooShort }: TileProps) {
  const { t } = useTranslation()
  const badge = usageBadge(app)
  const count = t('process.count', { count: app.count })
  const className = ['process-tile', `process-${size}`, badge && `tile-${badge.tone}`]
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
    // Appui long pour fermer (comme Arrêter/Redémarrer) : pas de fenêtre de confirmation.
    <HoldButton className={className} holdMs={HOLD_MS} onConfirm={onHold} onTooShort={onTooShort}>
      {head}
      {size === 'large' && (
        <>
          <span className="process-figures">
            <Figure value={decimal(app.cpu)} unit="%" label="CPU" />
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
    </HoldButton>
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

// Seuils de consommation : CPU en % de la machine, RAM en octets.
const GB = 1024 ** 3

function usageBadge(app: App): { text: string; tone: Tone } | undefined {
  if (app.cpu >= 50 || app.ram >= 4 * GB) return { text: i18n.t('common.veryHigh'), tone: 'critical' }
  if (app.cpu >= 20 || app.ram >= 2 * GB) return { text: i18n.t('common.high'), tone: 'warning' }
  return undefined
}

function displayName(name: string) {
  return name.replace(/\.exe$/i, '')
}
