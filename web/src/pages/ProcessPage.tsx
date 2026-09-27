import { useEffect, useState } from 'react'
import { createPortal } from 'react-dom'
import { Tile } from '../components/Tile'
import { Confirm } from '../components/Confirm'
import { useEventStream } from '../hooks/useEventStream'

type App = { name: string; cpu: number; ram: number; count: number }
type SortKey = 'cpu' | 'ram'

const SHOWN = 12

export function ProcessPage() {
  const [apps, setApps] = useState<App[] | null>(null)
  const [sortBy, setSortBy] = useState<SortKey>('cpu')
  const [target, setTarget] = useState<App | null>(null)
  const [notice, setNotice] = useState<string | null>(null)
  const connected = useEventStream<App[]>('/api/processes/stream', setApps)

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

  if (!apps) return <p className="coming-soon">{connected ? 'Chargement…' : 'Connexion au PC…'}</p>

  const sorted = [...apps].sort((a, b) => b[sortBy] - a[sortBy]).slice(0, SHOWN)

  return (
    <>
      <div className="segmented">
        <button type="button" className={sortBy === 'cpu' ? 'active' : ''} onClick={() => setSortBy('cpu')}>
          Tri CPU
        </button>
        <button type="button" className={sortBy === 'ram' ? 'active' : ''} onClick={() => setSortBy('ram')}>
          Tri RAM
        </button>
      </div>
      <div className="grid">
        {sorted.map((app) => (
          <Tile
            key={app.name}
            label={app.count > 1 ? `${app.count} processus` : '1 processus'}
            detail={`CPU ${app.cpu.toFixed(1).replace('.', ',')} % · ${mb(app.ram)}`}
            onClick={() => setTarget(app)}
          >
            {displayName(app.name)}
          </Tile>
        ))}
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

function displayName(name: string) {
  return name.replace(/\.exe$/i, '')
}

function mb(bytes: number) {
  return bytes >= 1024 ** 3
    ? `${(bytes / 1024 ** 3).toFixed(1).replace('.', ',')} Go`
    : `${Math.round(bytes / 1024 ** 2)} Mo`
}
