import { useCallback, useEffect, useRef, useState } from 'react'
import { createPortal } from 'react-dom'
import { Plus } from 'lucide-react'
import { Loader } from '../components/Loader'
import { ShortcutTile } from '../components/ShortcutTile'
import { ShortcutEditor } from '../components/ShortcutEditor'
import { usePageActive } from '../layout/pageActive'
import type { Shortcut, ShortcutsState } from '../shortcuts/types'

// Taille minimale d'une tuile (plus petite sur téléphone) : sert à calculer
// combien en tiennent sur l'écran.
const MIN_W = 220
const MIN_W_PHONE = 140
const MIN_H = 150
const GAP = 24

export function ShortcutsPage() {
  const active = usePageActive()
  const [state, setState] = useState<ShortcutsState | null>(null)
  const [failed, setFailed] = useState(false)
  const [editing, setEditing] = useState<Shortcut | 'new' | null>(null)
  const [notice, setNotice] = useState<string | null>(null)
  const [page, setPage] = useState(0)
  const [fit, gridRef] = useFit()
  const perPage = fit.cols * fit.rows

  useEffect(() => {
    if (!notice) return
    const id = setTimeout(() => setNotice(null), 3000)
    return () => clearTimeout(id)
  }, [notice])

  // Rechargés à chaque retour sur la page : un autre appareil a pu les modifier.
  useEffect(() => {
    if (!active) return
    fetch('/api/shortcuts')
      .then((r) => (r.ok ? r.json() : Promise.reject()))
      .then((s: ShortcutsState) => {
        setState(s)
        setFailed(false)
      })
      .catch(() => setFailed(true))
  }, [active])

  async function run(s: Shortcut) {
    if (s.kind === 'keys' && !state?.keys.ok) return setNotice(state?.keys.reason ?? 'Touches indisponibles')
    const r = await fetch(`/api/shortcuts/${encodeURIComponent(s.id)}/run`, { method: 'POST' }).catch(() => null)
    if (!r?.ok) setNotice(r ? `${s.label} : ${(await r.text()).trim()}` : 'PC injoignable')
  }

  // Enregistre toute la liste sur le PC ; renvoie un message d'erreur.
  const store = useCallback(async (list: Shortcut[]) => {
    const r = await fetch('/api/shortcuts', {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(list),
    }).catch(() => null)
    if (!r) return 'PC injoignable'
    if (!r.ok) return (await r.text()).trim()
    setState(await r.json())
    setEditing(null)
    return null
  }, [])

  if (!state) return failed ? <p className="coming-soon">PC injoignable</p> : <Loader />

  const list = state.shortcuts
  const saveOne = (s: Shortcut) =>
    store(editing === 'new' ? [...list, s] : list.map((x) => (x.id === s.id ? s : x)))
  const remove = (id: string) =>
    store(list.filter((x) => x.id !== id)).then((err) => err && setNotice(err))

  // La tuile « Ajouter » est toujours la dernière.
  const count = list.length + 1
  const pages = Math.max(1, Math.ceil(count / perPage))
  const current = Math.min(page, pages - 1)
  const from = current * perPage
  const shown = list.slice(from, from + perPage)
  const showAdd = from + perPage >= count
  // Une seule page : les tuiles s'agrandissent pour remplir l'écran. Sinon toutes
  // les pages gardent la même grille.
  const rows = pages > 1 ? fit.rows : Math.ceil(count / fit.cols)

  return (
    <>
      <div
        ref={gridRef}
        className="shortcuts-grid"
        style={{ gridTemplateColumns: `repeat(${fit.cols}, 1fr)`, gridTemplateRows: `repeat(${rows}, 1fr)` }}
      >
        {shown.map((s) => (
          <ShortcutTile
            key={s.id}
            shortcut={s}
            disabled={s.kind === 'keys' && !state.keys.ok}
            onRun={() => run(s)}
            onEdit={() => setEditing(s)}
          />
        ))}
        {showAdd && (
          <button type="button" className="shortcut shortcut-add" onClick={() => setEditing('new')}>
            <span className="shortcut-icon">
              <Plus size={36} strokeWidth={1.75} aria-hidden />
            </span>
            <span className="shortcut-label">Ajouter</span>
            <span className="shortcut-detail">Appui long sur une tuile pour la modifier</span>
          </button>
        )}
      </div>
      {pages > 1 && (
        <div className="pager">
          {Array.from({ length: pages }, (_, i) => (
            <button
              key={i}
              type="button"
              className={i === current ? 'active' : ''}
              onClick={() => setPage(i)}
              aria-label={`Page ${i + 1}`}
            >
              <span />
            </button>
          ))}
        </div>
      )}
      {notice && createPortal(<div className="toast">{notice}</div>, document.body)}
      {editing && (
        <ShortcutEditor
          shortcut={editing === 'new' ? null : editing}
          keysReason={state.keys.ok ? undefined : state.keys.reason}
          onSave={saveOne}
          onDelete={() => editing !== 'new' && remove(editing.id)}
          onCancel={() => setEditing(null)}
        />
      )}
    </>
  )
}

// useFit calcule combien de colonnes et de lignes de tuiles tiennent dans la zone
// sans défiler, et suit ses changements de taille (rotation de la tablette).
function useFit() {
  const [fit, setFit] = useState({ cols: 3, rows: 2 })
  const observer = useRef<ResizeObserver | null>(null)

  const ref = useCallback((el: HTMLDivElement | null) => {
    observer.current?.disconnect()
    if (!el) return
    observer.current = new ResizeObserver(([e]) => {
      const { width, height } = e.contentRect
      if (!width || !height) return // page cachée
      const minW = width < 560 ? MIN_W_PHONE : MIN_W
      const cols = Math.max(1, Math.floor((width + GAP) / (minW + GAP)))
      const rows = Math.max(1, Math.floor((height + GAP) / (MIN_H + GAP)))
      setFit((f) => (f.cols === cols && f.rows === rows ? f : { cols, rows }))
    })
    observer.current.observe(el)
  }, [])

  return [fit, ref] as const
}
