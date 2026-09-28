import { useRef, type PointerEvent } from 'react'

const LONG_PRESS_MS = 600

// Appui court → onPress, appui long → onLongPress. Glisser (pour changer de page
// ou faire défiler) n'en déclenche aucun.
export function useLongPress(onPress: () => void, onLongPress: () => void) {
  const timer = useRef<number | undefined>(undefined)
  const start = useRef<{ x: number; y: number } | null>(null)
  const longPressed = useRef(false)
  const moved = useRef(false)

  function stop() {
    clearTimeout(timer.current)
    timer.current = undefined
    start.current = null
  }

  return {
    onPointerDown(e: PointerEvent) {
      longPressed.current = false
      moved.current = false
      start.current = { x: e.clientX, y: e.clientY }
      timer.current = window.setTimeout(() => {
        longPressed.current = true
        stop()
        onLongPress()
      }, LONG_PRESS_MS)
    },
    onPointerMove(e: PointerEvent) {
      const s = start.current
      if (s && Math.hypot(e.clientX - s.x, e.clientY - s.y) > 12) {
        moved.current = true
        stop()
      }
    },
    onPointerUp: stop,
    onPointerLeave: stop,
    onPointerCancel: stop,
    onClick() {
      if (!longPressed.current && !moved.current) onPress()
    },
    onContextMenu(e: { preventDefault: () => void }) {
      e.preventDefault() // appui long sur tablette : pas de menu
    },
  }
}
