import { useCallback, useEffect, useRef, useState } from 'react'
import { createPortal } from 'react-dom'
import { ArrowLeft, Download, Folder, FolderSearch, HardDrive, House, MonitorUp, Star, Usb } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { Tile } from '../components/Tile'
import { useRadialMenu, type RadialItem } from '../components/RadialMenu'
import { IconButton } from '../components/IconButton'
import { Loader } from '../components/Loader'
import { UsageBar } from '../components/UsageBar'
import { usePageActive } from '../layout/pageActive'
import { formatBytes } from '../format'
import { FILE_ICONS, fileKind } from '../files/fileKind'
import { api } from '../api'

type Drive = { path: string; name: string; total: number; used: number; removable: boolean }
type Entry = { name: string; dir: boolean; size: number }
type Recent = { name: string; path: string; folder: string; size: number; used: number }
type Favorite = { name: string; path: string; folder: string; size: number }
type Listing = { path: string; parent: string; entries: Entry[]; truncated: number }

// Au-delà, le disque est signalé « presque plein ».
const FULL = 90

export function FilesPage() {
  const { t } = useTranslation()
  const active = usePageActive()
  const [drives, setDrives] = useState<Drive[] | null>(null)
  const [drivesError, setDrivesError] = useState(false)
  const [recents, setRecents] = useState<Recent[]>([])
  const [favorites, setFavorites] = useState<Favorite[]>([])
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
    const r = await fetch(api(`/files/list?path=${encodeURIComponent(path)}`)).catch(() => null)
    const data: Listing | null = r?.ok ? await r.json().catch(() => null) : null
    if (id !== request.current) return // une autre navigation a pris le relais
    setLoading(false)
    if (data) {
      setListing(data)
      return
    }
    setNotice(t(r?.status === 403 ? 'files.denied' : r?.status === 404 ? 'files.notFound' : 'common.unreachable'))
    onFail?.()
  }, [t])

  // Liste des disques rechargée à chaque retour sur la page (clé USB branchée entre-temps).
  useEffect(() => {
    if (!active || drive) return
    fetch(api('/files/drives'))
      .then((r) => (r.ok ? r.json() : Promise.reject()))
      .then((d: Drive[]) => {
        setDrives(d)
        setDrivesError(false)
      })
      .catch(() => setDrivesError(true))
    fetch(api('/files/recent'))
      .then((r) => (r.ok ? r.json() : Promise.reject()))
      .then(setRecents)
      .catch(() => setRecents([]))
  }, [active, drive])

  // Favoris : sur l'accueil, et pour savoir si un fichier d'un dossier en est un.
  useEffect(() => {
    if (!active) return
    fetch(api('/files/favorites'))
      .then((r) => (r.ok ? r.json() : Promise.reject()))
      .then(setFavorites)
      .catch(() => {})
  }, [active])

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

  // Fichier récent ou favori : on ouvre le dossier qui le contient, sur son disque.
  function openRecent(f: { folder: string }) {
    const d = driveOf(drives ?? [], f.folder)
    if (!d) return
    setDrive(d)
    setListing(null)
    open(f.folder, () => setDrive(null))
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

  // Actions du menu circulaire d'un fichier.
  const fileActions: FileActions = {
    isFavorite: (path) => favorites.some((f) => samePath(f.path, path)),
    async run(action, path) {
      const name = baseName(path)
      if (action === 'download') {
        const a = document.createElement('a')
        a.href = api(`/files/download?path=${encodeURIComponent(path)}`)
        a.download = name
        a.click()
        setNotice(t('files.downloading', { name }))
        return
      }
      if (action === 'favorite' || action === 'unfavorite') {
        const r = await post('/files/favorites', { path, favorite: action === 'favorite' })
        const list: Favorite[] | null = r?.ok ? await r.json().catch(() => null) : null
        if (list) setFavorites(list)
        setNotice(list ? t(action === 'favorite' ? 'files.favorited' : 'files.unfavorited') : fileError(r, t))
        return
      }
      const r = await post(`/files/${action}`, { path })
      setNotice(r?.ok ? (action === 'open' ? t('files.opened', { name }) : t('files.revealed')) : fileError(r, t))
    },
  }

  const toast = notice && createPortal(<div className="toast">{notice}</div>, document.body)

  if (!drive) {
    if (!drives) return drivesError ? <p className="coming-soon">{t('common.unreachable')}</p> : <Loader />
    if (drives.length === 0) return <p className="coming-soon">{t('files.noDrive')}</p>
    return (
      <div className={recents.length > 0 || favorites.length > 0 ? 'files-home has-recent' : 'files-home'}>
        <div className="grid grid-drives">
          {drives.map((d) => {
            const pct = d.total > 0 ? (d.used / d.total) * 100 : 0
            const full = pct >= FULL
            return (
              <Tile
                key={d.path}
                label={driveLabel(d.path)}
                icon={d.removable ? Usb : HardDrive}
                detail={t('files.free', { free: formatBytes(d.total - d.used), total: formatBytes(d.total) })}
                chart={<UsageBar percent={pct} critical={full} />}
                badge={full ? { text: t('files.almostFull'), tone: 'critical' } : undefined}
                onClick={() => openDrive(d)}
              >
                {d.name}
              </Tile>
            )
          })}
        </div>
        {(recents.length > 0 || favorites.length > 0) && (
          <div className="files-side">
            {favorites.length > 0 && (
              <section className="files-recent files-favorites">
                <h2 className="files-recent-title">{t('files.favorites')}</h2>
                <div className="grid files-recent-grid">
                  {favorites.map((f) => (
                    <FileTile key={f.path} name={f.name} path={f.path} size={f.size} actions={fileActions} onClick={() => openRecent(f)} />
                  ))}
                </div>
              </section>
            )}
            {recents.length > 0 && (
              <section className="files-recent">
                <h2 className="files-recent-title">{t('files.recent')}</h2>
                <div className="grid files-recent-grid">
                  {recents.map((f) => (
                    <FileTile key={f.path} name={f.name} path={f.path} size={f.size} actions={fileActions} onClick={() => openRecent(f)} />
                  ))}
                </div>
              </section>
            )}
          </div>
        )}
        {toast}
      </div>
    )
  }

  return (
    <>
      <div className="files-bar">
        <IconButton icon={ArrowLeft} label={t('common.back')} onClick={back} />
        <IconButton icon={House} label={t('files.drives')} onClick={closeDrive} />
        <span className="files-path">{breadcrumb(drive, listing?.path ?? drive.path)}</span>
      </div>
      {loading && !listing ? (
        <Loader />
      ) : listing && listing.entries.length === 0 ? (
        <p className="coming-soon files-empty">{t('files.empty')}</p>
      ) : (
        <div className={loading ? 'grid grid-files is-loading' : 'grid grid-files'}>
          {listing?.entries.map((e) =>
            e.dir ? (
              <Tile key={e.name} label={t('files.folder')} icon={Folder} onClick={() => open(join(listing.path, e.name))}>
                {e.name}
              </Tile>
            ) : (
              <FileTile key={e.name} name={e.name} path={join(listing.path, e.name)} size={e.size} actions={fileActions} />
            ),
          )}
          {listing && listing.truncated > 0 && (
            <p className="files-more">{t('files.more', { n: listing.truncated })}</p>
          )}
        </div>
      )}
      {toast}
    </>
  )
}

type FileAction = 'open' | 'download' | 'favorite' | 'unfavorite' | 'reveal'
type FileActions = { isFavorite: (path: string) => boolean; run: (action: FileAction, path: string) => void }

// Tuile de fichier. Appui long : menu circulaire (ouvrir sur le PC, télécharger,
// favori, afficher dans le dossier).
function FileTile({ name, path, size, actions, onClick }: { name: string; path: string; size: number; actions: FileActions; onClick?: () => void }) {
  const { t } = useTranslation()
  const kind = fileKind(name)
  const favorite = actions.isFavorite(path)
  const items: RadialItem[] = [
    { id: 'open', label: t('files.actions.open'), icon: MonitorUp, onSelect: () => actions.run('open', path) },
    { id: 'download', label: t('files.actions.download'), icon: Download, onSelect: () => actions.run('download', path) },
    {
      id: 'favorite',
      label: t(favorite ? 'files.actions.unfavorite' : 'files.actions.favorite'),
      icon: Star,
      active: favorite,
      onSelect: () => actions.run(favorite ? 'unfavorite' : 'favorite', path),
    },
    { id: 'reveal', label: t('files.actions.reveal'), icon: FolderSearch, onSelect: () => actions.run('reveal', path) },
  ]
  const { bind, menu } = useRadialMenu(items, onClick)
  return (
    <>
      <Tile
        className={`file-${kind}`}
        label={t(`files.kinds.${kind}`)}
        icon={FILE_ICONS[kind]}
        detail={formatBytes(size)}
        press={bind}
        onClick={bind.onClick}
      >
        {name}
      </Tile>
      {menu}
    </>
  )
}

function post(path: string, body: unknown) {
  return fetch(api(path), { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(body) }).catch(() => null)
}

function fileError(r: Response | null, t: ReturnType<typeof useTranslation>['t']) {
  if (r?.status === 403) return t('files.denied')
  if (r?.status === 404) return t('files.notFound')
  if (r?.status === 400) return t('files.tooMany')
  return t('common.unreachable')
}

function baseName(path: string) {
  return path.split(/[\\/]/).pop() ?? path
}

// Chemins Windows (avec « \ ») : la casse ne compte pas.
function samePath(a: string, b: string) {
  return a.includes('\\') ? a.toLowerCase() === b.toLowerCase() : a === b
}

// « C:\ » → « C: » ; sous Linux, le point de montage tel quel.
function driveLabel(path: string) {
  return /^[A-Z]:\\$/i.test(path) ? path.slice(0, 2) : path
}

// Disque qui contient path : le plus précis si des montages sont imbriqués (/ et /home).
function driveOf(drives: Drive[], path: string) {
  const lower = path.toLowerCase()
  return drives
    .filter((d) => {
      const root = d.path.toLowerCase()
      const sep = separator(root)
      return lower === root || lower.startsWith(root.endsWith(sep) ? root : root + sep)
    })
    .sort((a, b) => b.path.length - a.path.length)[0]
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
