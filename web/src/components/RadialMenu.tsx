import { useEffect, useRef, useState, type CSSProperties, type PointerEvent as ReactPointerEvent } from 'react'
import { createPortal } from 'react-dom'
import type { LucideIcon } from 'lucide-react'
import { lockSwipe } from '../layout/swipeLock'
import { playClick } from '../feedback/clickSound'
import { useMediaQuery } from '../hooks/useMediaQuery'

export type RadialItem = {
  id: string
  label: string
  icon: LucideIcon
  active?: boolean // option « allumée », ex. déjà en favori
  onSelect: () => void
}

const HOLD_MS = 450
const SLOP = 10 // au-delà, c'est un glissement (page, défilement) : pas de menu
const DEAD_ZONE = 30 // relâcher aussi près du point de départ annule
const MARGIN = 16
const LABEL = 48 // hauteur réservée au nom de l'option visée

type Open = { x: number; y: number; cx: number; cy: number; radius: number; size: number }

// Menu circulaire : un appui long ouvre les options en cercle autour du doigt.
// On glisse vers une option (elle grossit), on relâche pour la choisir ; relâcher
// au centre annule. Pendant le geste, les pages ne glissent plus et rien ne défile.
// Renvoie les gestionnaires à poser sur l'élément et le menu à afficher.
export function useRadialMenu(items: RadialItem[], onPress?: () => void) {
  const phone = useMediaQuery('(max-width: 560px)')
  const [open, setOpen] = useState<Open | null>(null)
  const [selected, setSelected] = useState<number | null>(null)
  const timer = useRef<number | undefined>(undefined)
  const start = useRef<{ x: number; y: number } | null>(null)
  const fired = useRef(false)
  const itemsRef = useRef(items)
  itemsRef.current = items
  const teardown = useRef<(() => void) | null>(null)

  useEffect(() => () => teardown.current?.(), [])

  function cancelHold() {
    clearTimeout(timer.current)
    timer.current = undefined
    start.current = null
  }

  function show(x: number, y: number) {
    const radius = phone ? 96 : 120
    const size = phone ? 72 : 84
    const reach = radius + size / 2 + MARGIN
    // Le cercle reste entier à l'écran, même depuis une tuile au bord.
    const clamp = (v: number, min: number, max: number) => Math.min(Math.max(v, min), Math.max(min, max))
    const cx = clamp(x, reach, window.innerWidth - reach)
    const cy = clamp(y, reach + LABEL, window.innerHeight - reach) // le nom de l'option s'affiche au-dessus
    const menu = { x, y, cx, cy, radius, size }
    fired.current = true
    setOpen(menu)
    setSelected(null)
    lockSwipe(true)
    navigator.vibrate?.(10)

    let current: number | null = null
    const track = (px: number, py: number) => {
      const next = pick(menu, itemsRef.current.length, px, py)
      if (next === current) return
      current = next
      setSelected(next)
      if (next !== null) navigator.vibrate?.(5)
    }
    const release = () => {
      close()
      const item = current === null ? undefined : itemsRef.current[current]
      if (item) {
        playClick()
        item.onSelect()
      }
    }
    // Au doigt, on suit les événements tactiles : empêcher leur action par défaut
    // bloque le défilement, et ils continuent même si le navigateur annule le pointeur.
    const onTouchMove = (e: TouchEvent) => {
      if (e.cancelable) e.preventDefault()
      const t = e.touches[0]
      if (t) track(t.clientX, t.clientY)
    }
    const onTouchEnd = (e: TouchEvent) => {
      if (e.touches.length > 0) return
      if (e.cancelable) e.preventDefault() // pas de clic sur la tuile au relâcher
      release()
    }
    const onPointerMove = (e: PointerEvent) => {
      if (e.pointerType !== 'touch') track(e.clientX, e.clientY)
    }
    const onPointerUp = (e: PointerEvent) => {
      if (e.pointerType !== 'touch') release()
    }
    const onKey = (e: KeyboardEvent) => {
      if (e.key === 'Escape') close()
    }
    const opts = { capture: true, passive: false } as const
    document.addEventListener('touchmove', onTouchMove, opts)
    document.addEventListener('touchend', onTouchEnd, opts)
    document.addEventListener('touchcancel', onTouchEnd, opts)
    document.addEventListener('pointermove', onPointerMove, true)
    document.addEventListener('pointerup', onPointerUp, true)
    document.addEventListener('keydown', onKey)
    teardown.current = () => {
      document.removeEventListener('touchmove', onTouchMove, opts)
      document.removeEventListener('touchend', onTouchEnd, opts)
      document.removeEventListener('touchcancel', onTouchEnd, opts)
      document.removeEventListener('pointermove', onPointerMove, true)
      document.removeEventListener('pointerup', onPointerUp, true)
      document.removeEventListener('keydown', onKey)
    }
  }

  function close() {
    teardown.current?.()
    teardown.current = null
    setOpen(null)
    setSelected(null)
    lockSwipe(false)
  }

  const bind = {
    onPointerDown(e: ReactPointerEvent) {
      if (e.button !== 0) return
      fired.current = false
      const { clientX: x, clientY: y } = e
      start.current = { x, y }
      clearTimeout(timer.current)
      timer.current = window.setTimeout(() => {
        cancelHold()
        show(x, y)
      }, HOLD_MS)
    },
    onPointerMove(e: ReactPointerEvent) {
      const s = start.current
      if (s && Math.hypot(e.clientX - s.x, e.clientY - s.y) > SLOP) cancelHold()
    },
    onPointerUp: cancelHold,
    onPointerCancel: cancelHold,
    onPointerLeave() {
      if (!open) cancelHold()
    },
    onClick() {
      // Le clic qui suit un appui long ne compte pas comme un appui court.
      if (fired.current) fired.current = false
      else onPress?.()
    },
    onContextMenu(e: { preventDefault: () => void }) {
      e.preventDefault() // appui long sur tablette : pas de menu du navigateur
    },
  }

  const menu =
    open &&
    createPortal(
      <div className="radial-backdrop" aria-hidden>
        <div
          className="radial"
          style={{ left: open.cx, top: open.cy, '--radial-r': `${open.radius}px`, '--radial-size': `${open.size}px` } as CSSProperties}
        >
          <span className="radial-ring" />
          {items.map((item, i) => {
            const [dx, dy] = offset(i, items.length, open.radius)
            const Icon = item.icon
            const cls = ['radial-item', item.active && 'is-active', selected === i && 'is-selected'].filter(Boolean).join(' ')
            return (
              <span key={item.id} className={cls} style={{ '--dx': `${dx}px`, '--dy': `${dy}px` } as CSSProperties}>
                <Icon size={phone ? 26 : 30} strokeWidth={2} fill={item.active ? 'currentColor' : 'none'} />
              </span>
            )
          })}
          <span className="radial-label">{selected === null ? '' : items[selected]?.label}</span>
          <span className="radial-origin" style={{ '--ox': `${open.x - open.cx}px`, '--oy': `${open.y - open.cy}px` } as CSSProperties} />
        </div>
      </div>,
      document.body,
    )

  return { bind, menu }
}

// Position d'une option : la première en haut, puis dans le sens des aiguilles d'une montre.
function offset(i: number, n: number, radius: number): [number, number] {
  const angle = -Math.PI / 2 + (i * 2 * Math.PI) / n
  return [Math.cos(angle) * radius, Math.sin(angle) * radius]
}

// Option visée : celle sous le doigt, sinon celle dans la direction du geste
// (mesurée depuis le point de départ, qui peut être décalé du centre près d'un bord).
function pick(menu: Open, n: number, x: number, y: number): number | null {
  for (let i = 0; i < n; i++) {
    const [dx, dy] = offset(i, n, menu.radius)
    if (Math.hypot(x - (menu.cx + dx), y - (menu.cy + dy)) <= menu.size / 2 + 8) return i
  }
  const dx = x - menu.x
  const dy = y - menu.y
  if (Math.hypot(dx, dy) < DEAD_ZONE) return null
  const angle = Math.atan2(dy, dx) + Math.PI / 2 // 0 = en haut
  const step = (2 * Math.PI) / n
  return (((Math.round(angle / step) % n) + n) % n) as number
}
