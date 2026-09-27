import { useCallback, useEffect, useRef, useState } from 'react'
import { createPortal } from 'react-dom'
import { ArrowLeft, File, Folder, HardDrive, Usb } from 'lucide-react'
import { Tile } from '../components/Tile'
import { Loader } from '../components/Loader'
import { UsageBar } from '../components/UsageBar'
import { usePageActive } from '../layout/pageActive'
import { formatBytes } from '../format'

type Drive = { path: string; name: string; total: number; used: number; removable: boolean }
type Entry = { name: string; dir: boolean; size: number }
type Listing = { path: string; parent: string; entries: Entry[]; truncated: number }

// Au-delà, le disque est signalé « presque plein ».
const FULL = 90

export function FilesPage() {
  const active = usePageActive()
  const [drives, setDrives] = useState<Drive[] | null>(null)
  const [drivesError, setDrivesError] = useState(false)
  const [drive, setDrive] = useState<Drive | null>(null)
  const [listing, setListing] = useState<Listing | null>(null)
  const [loading, setLoading] = useState(false)
  const [notice, setNotice] = useState<string | null>(null)
  const request = useRef(0)

  useEffect(() => {
    if (!notice) return
    const id = setTimeout(() => setNotice(null), 3000)
    return () => clearTimeout(id)
  }, [notice])

  // Ouvre un dossier. En cas d'échec, on reste sur le dossier affiché.
  const open = useCallback(async (path: string, onFail?: () => void) => {
    const id = ++request.current
    setLoading(true)
    const r = await fetch(`/api/files/list?path=${encodeURIComponent(path)}`).catch(() => null)
    const data: Listing | null = r?.ok ? await r.json().catch(() => null) : null
    if (id !== request.current) return // une autre navigation a pris le relais
    setLoading(false)
    if (data) {
      setListing(data)
      return
    }
    setNotice(r?.status === 403 ? 'Accès refusé à ce dossier' : r?.status === 404 ? 'Dossier introuvable' : 'PC injoignable')
    onFail?.()
  }, [])

  // Liste des disques rechargée à chaque retour sur la page (clé USB branchée entre-temps).
  useEffect(() => {
    if (!active || drive) return
    fetch('/api/files/drives')
      .then((r) => (r.ok ? r.json() : Promise.reject()))
      .then((d: Drive[]) => {
        setDrives(d)
        setDrivesError(false)
      })
      .catch(() => setDrivesError(true))
  }, [active, drive])

  // Retour sur la page : le dossier affiché peut avoir changé.
  const shownPath = listing?.path
  useEffect(() => {
    if (active && drive && shownPath) open(shownPath, () => closeDrive())
  }, [active])

  function openDrive(d: Drive) {
    setDrive(d)
    setListing(null)
    open(d.path, () => setDrive(null))
  }

  function closeDrive() {
    request.current++
    setDrive(null)
    setListing(null)
    setLoading(false)
  }

  function back() {
    if (listing?.parent) open(listing.parent)
    else closeDrive()
  }

  const toast = notice && createPortal(<div className="toast">{notice}</div>, document.body)

  if (!drive) {
    if (!drives) return drivesError ? <p className="coming-soon">PC injoignable</p> : <Loader />
    if (drives.length === 0) return <p className="coming-soon">Aucun disque trouvé</p>
    return (
      <>
        <div className="grid grid-drives">
          {drives.map((d) => {
            const pct = d.total > 0 ? (d.used / d.total) * 100 : 0
            const full = pct >= FULL
            return (
              <Tile
                key={d.path}
                label={driveLabel(d.path)}
                icon={d.removable ? Usb : HardDrive}
                detail={`${formatBytes(d.total - d.used)} libres sur ${formatBytes(d.total)}`}
                chart={<UsageBar percent={pct} critical={full} />}
                badge={full ? { text: 'Presque plein', tone: 'critical' } : undefined}
                onClick={() => openDrive(d)}
              >
                {d.name}
              </Tile>
            )
          })}
        </div>
        {toast}
      </>
    )
  }

  return (
    <>
      <div className="files-bar">
        <button type="button" className="files-back" onClick={back} aria-label="Retour">
          <ArrowLeft size={28} aria-hidden />
        </button>
        <span className="files-path">{breadcrumb(drive, listing?.path ?? drive.path)}</span>
      </div>
      {loading && !listing ? (
        <Loader />
      ) : listing && listing.entries.length === 0 ? (
        <p className="coming-soon files-empty">Dossier vide</p>
      ) : (
        <div className={loading ? 'grid grid-files is-loading' : 'grid grid-files'}>
          {listing?.entries.map((e) =>
            e.dir ? (
              <Tile key={e.name} label="Dossier" icon={Folder} onClick={() => open(join(listing.path, e.name))}>
                {e.name}
              </Tile>
            ) : (
              <Tile key={e.name} label="Fichier" icon={File} detail={formatBytes(e.size)}>
                {e.name}
              </Tile>
            ),
          )}
          {listing && listing.truncated > 0 && (
            <p className="files-more">+ {listing.truncated} éléments non affichés</p>
          )}
        </div>
      )}
      {toast}
    </>
  )
}

// « C:\ » → « C: » ; sous Linux, le point de montage tel quel.
function driveLabel(path: string) {
  return /^[A-Z]:\\$/i.test(path) ? path.slice(0, 2) : path
}

function separator(path: string) {
  return path.includes('\\') ? '\\' : '/'
}

function join(dir: string, name: string) {
  const sep = separator(dir)
  return dir.endsWith(sep) ? dir + name : dir + sep + name
}

// Nom du disque puis les dossiers traversés : « Jeux › Steam › common ».
function breadcrumb(drive: Drive, path: string) {
  const rest = path.slice(drive.path.length).split(/[\\/]/).filter(Boolean)
  const parts = [drive.name, ...rest]
  const shown = parts.length > 4 ? [parts[0], '…', ...parts.slice(-2)] : parts
  return shown.join('  ›  ')
}
