import { useRef, useState, type PointerEvent } from 'react'
import { useTranslation } from 'react-i18next'
import { colorOf, describe, iconOf, type Shortcut } from '../shortcuts/types'

type Props = {
  shortcut: Shortcut
  disabled?: boolean
  onRun: () => void
  onEdit: () => void
}

const LONG_PRESS_MS = 600

// Tuile d'un raccourci : un appui le déclenche, un appui long ouvre sa modification.
// Glisser (pour changer de page) n'ouvre rien.
export function ShortcutTile({ shortcut, disabled, onRun, onEdit }: Props) {
  const { t } = useTranslation()
  const [flash, setFlash] = useState(false)
  const timer = useRef<number | undefined>(undefined)
  const start = useRef<{ x: number; y: number } | null>(null)
  const longPressed = useRef(false)
  const Icon = iconOf(shortcut.icon)

  function stop() {
    clearTimeout(timer.current)
    timer.current = undefined
    start.current = null
  }

  function down(e: PointerEvent) {
    longPressed.current = false
    start.current = { x: e.clientX, y: e.clientY }
    timer.current = window.setTimeout(() => {
      longPressed.current = true
      stop()
      onEdit()
    }, LONG_PRESS_MS)
  }

  function move(e: PointerEvent) {
    const s = start.current
    if (s && Math.hypot(e.clientX - s.x, e.clientY - s.y) > 12) stop()
  }

  function click() {
    if (longPressed.current) return
    if (!disabled) {
      setFlash(true)
      setTimeout(() => setFlash(false), 400)
    }
    onRun()
  }

  return (
    <button
      type="button"
      className={['shortcut', disabled && 'is-disabled', flash && 'is-flash'].filter(Boolean).join(' ')}
      style={{ '--c': colorOf(shortcut.color) } as React.CSSProperties}
      onClick={click}
      onPointerDown={down}
      onPointerMove={move}
      onPointerUp={stop}
      onPointerLeave={stop}
      onPointerCancel={stop}
      onContextMenu={(e) => e.preventDefault()} // appui long sur tablette : pas de menu
    >
      <span className="shortcut-icon">
        <Icon size={36} strokeWidth={1.75} aria-hidden />
      </span>
      <span className="shortcut-label">{shortcut.label}</span>
      <span className="shortcut-detail">{disabled ? t('shortcuts.unavailable') : describe(shortcut)}</span>
    </button>
  )
}
