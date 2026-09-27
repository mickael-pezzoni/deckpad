import { useRef, useState, type PointerEvent, type ReactNode } from 'react'

type Props = {
  children: ReactNode
  className?: string
  holdMs?: number // 0 : un simple appui suffit
  onConfirm: () => void
  onTooShort?: () => void
}

// Gros bouton d'action. Avec holdMs, il faut rester appuyé : une jauge se remplit
// et l'action part quand elle est pleine. Glisser (pour changer de page) annule.
export function HoldButton({ children, className = '', holdMs = 0, onConfirm, onTooShort }: Props) {
  const [holding, setHolding] = useState(false)
  const timer = useRef<number | undefined>(undefined)
  const start = useRef<{ x: number; y: number } | null>(null)

  function cancel(tooShort: boolean) {
    if (timer.current === undefined) return
    clearTimeout(timer.current)
    timer.current = undefined
    setHolding(false)
    if (tooShort) onTooShort?.()
  }

  function down(e: PointerEvent) {
    if (!holdMs) return
    start.current = { x: e.clientX, y: e.clientY }
    setHolding(true)
    timer.current = window.setTimeout(() => {
      timer.current = undefined
      setHolding(false)
      onConfirm()
    }, holdMs)
  }

  function move(e: PointerEvent) {
    const s = start.current
    if (s && Math.hypot(e.clientX - s.x, e.clientY - s.y) > 12) cancel(false)
  }

  return (
    <button
      type="button"
      className={`action ${holding ? 'holding' : ''} ${className}`}
      style={{ '--hold-ms': `${holdMs}ms` } as React.CSSProperties}
      onClick={holdMs ? undefined : onConfirm}
      onPointerDown={down}
      onPointerMove={move}
      onPointerUp={() => cancel(true)}
      onPointerLeave={() => cancel(false)}
      onPointerCancel={() => cancel(false)}
      onContextMenu={(e) => e.preventDefault()} // appui long sur tablette : pas de menu
    >
      {holdMs > 0 && <span className="action-progress" aria-hidden />}
      {children}
    </button>
  )
}
