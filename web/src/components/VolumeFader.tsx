import { useRef, useState, type PointerEvent, type ReactNode } from 'react'
import { Volume2, VolumeX } from 'lucide-react'
import { useLatched } from '../hooks/useLatched'

type Props = {
  label: string
  icon: ReactNode
  appIcon?: boolean // icône de programme (pastille neutre) plutôt qu'une icône de l'app
  volume: number // 0 à 100
  muted: boolean
  onVolume: (volume: number) => void
  onMute: (muted: boolean) => void
}

// Envoi au PC pendant le glissement : pas plus d'une fois par SEND_MS.
const SEND_MS = 80

// Curseur de volume vertical façon table de mixage : on glisse ou on tape sur la
// piste. Glisser verticalement ne change pas de page (swiper-no-swiping).
export function VolumeFader({ label, icon, appIcon, volume, muted, onVolume, onMute }: Props) {
  const [shownVolume, latchVolume] = useLatched(volume)
  const [shownMuted, latchMuted] = useLatched(muted)
  const [drag, setDrag] = useState<number | null>(null)
  const track = useRef<HTMLDivElement>(null)
  const lastSent = useRef(0)
  const trailing = useRef<number | undefined>(undefined)

  const value = drag ?? shownVolume

  function valueAt(e: PointerEvent) {
    const r = track.current!.getBoundingClientRect()
    return Math.min(100, Math.max(0, Math.round(((r.bottom - e.clientY) / r.height) * 100)))
  }

  function send(v: number, now = false) {
    clearTimeout(trailing.current)
    const wait = SEND_MS - (Date.now() - lastSent.current)
    if (now || wait <= 0) {
      lastSent.current = Date.now()
      onVolume(v)
    } else {
      trailing.current = window.setTimeout(() => send(v, true), wait)
    }
  }

  function down(e: PointerEvent) {
    e.currentTarget.setPointerCapture(e.pointerId)
    const v = valueAt(e)
    setDrag(v)
    send(v)
  }

  function move(e: PointerEvent) {
    if (drag === null) return
    const v = valueAt(e)
    if (v === drag) return
    setDrag(v)
    send(v)
  }

  function up() {
    if (drag === null) return
    send(drag, true)
    latchVolume(drag)
    setDrag(null)
  }

  function toggleMute() {
    latchMuted(!shownMuted)
    onMute(!shownMuted)
  }

  const MuteIcon = shownMuted ? VolumeX : Volume2
  return (
    <div className={shownMuted ? 'fader is-muted' : 'fader'}>
      <span className="fader-head">
        <span className={appIcon ? 'tile-icon tile-icon-image' : 'tile-icon'}>{icon}</span>
        <span className="fader-label">{label}</span>
      </span>
      <span className="fader-value">{value} %</span>
      <div
        ref={track}
        className="fader-track swiper-no-swiping"
        role="slider"
        aria-label={`Volume ${label}`}
        aria-valuemin={0}
        aria-valuemax={100}
        aria-valuenow={value}
        onPointerDown={down}
        onPointerMove={move}
        onPointerUp={up}
        onPointerCancel={up}
      >
        <span className="fader-fill" style={{ height: `${value}%` }} />
      </div>
      <button
        type="button"
        className="fader-mute"
        aria-label={shownMuted ? `Rétablir le son : ${label}` : `Couper le son : ${label}`}
        aria-pressed={shownMuted}
        onClick={toggleMute}
      >
        <MuteIcon size={26} strokeWidth={2} aria-hidden />
      </button>
    </div>
  )
}
