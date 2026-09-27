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

// Envoi au PC pendant le réglage : pas plus d'une fois par SEND_MS.
const SEND_MS = 80

// La molette couvre 270° : de 0 en bas à gauche à 100 en bas à droite.
const SWEEP = 270
const START = 135 // angle du 0, en degrés depuis « 3 h », sens horaire

// Molette de volume façon table de mixage : on tourne le doigt autour, ou on tape
// directement sur l'anneau. Ne change pas de page pendant le réglage.
export function VolumeKnob({ label, icon, appIcon, volume, muted, onVolume, onMute }: Props) {
  const [shownVolume, latchVolume] = useLatched(volume)
  const [shownMuted, latchMuted] = useLatched(muted)
  const [drag, setDrag] = useState<number | null>(null)
  const knob = useRef<SVGSVGElement>(null)
  const lastSent = useRef(0)
  const trailing = useRef<number | undefined>(undefined)

  const value = drag ?? shownVolume

  // Valeur sous le doigt. Dans le creux du bas, on reste collé au bord le plus proche
  // de la valeur précédente : pas de saut de 100 à 0 en passant par-dessous.
  function valueAt(e: PointerEvent, prev: number) {
    const r = knob.current!.getBoundingClientRect()
    const angle = (Math.atan2(e.clientY - (r.top + r.height / 2), e.clientX - (r.left + r.width / 2)) * 180) / Math.PI
    const rel = (angle - START + 720) % 360
    if (rel > SWEEP) return prev > 50 ? 100 : 0
    const v = Math.round((rel / SWEEP) * 100)
    return Math.abs(v - prev) > 50 && drag !== null ? prev : v
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

  function down(e: PointerEvent<SVGSVGElement>) {
    e.currentTarget.setPointerCapture(e.pointerId)
    const v = valueAt(e, value)
    setDrag(v)
    send(v)
  }

  function move(e: PointerEvent) {
    if (drag === null) return
    const v = valueAt(e, drag)
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

  // Position du repère au bout de l'arc.
  const end = ((START + (value / 100) * SWEEP) * Math.PI) / 180
  const MuteIcon = shownMuted ? VolumeX : Volume2
  return (
    <div className={['knob-card', shownMuted && 'is-muted', drag !== null && 'is-dragging'].filter(Boolean).join(' ')}>
      <span className="tile-head">
        <span className="tile-label">
          <span className={appIcon ? 'tile-icon tile-icon-image' : 'tile-icon'}>{icon}</span>
          <span className="knob-label">{label}</span>
        </span>
        <button
          type="button"
          className="knob-mute"
          aria-label={shownMuted ? `Rétablir le son : ${label}` : `Couper le son : ${label}`}
          aria-pressed={shownMuted}
          onClick={toggleMute}
        >
          <MuteIcon size={24} strokeWidth={2} aria-hidden />
        </button>
      </span>
      <svg
        ref={knob}
        className="knob swiper-no-swiping"
        viewBox="0 0 100 100"
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
        <g transform={`rotate(${START} 50 50)`}>
          <circle className="knob-track" cx="50" cy="50" r="40" pathLength={360} strokeDasharray={`${SWEEP} 360`} />
          <circle
            className="knob-fill"
            cx="50"
            cy="50"
            r="40"
            pathLength={360}
            strokeDasharray={`${(value / 100) * SWEEP} 360`}
          />
        </g>
        <circle className="knob-thumb" cx={50 + 40 * Math.cos(end)} cy={50 + 40 * Math.sin(end)} r="7" />
        <text className="knob-value" x="50" y="50">
          {value}
          <tspan className="knob-unit"> %</tspan>
        </text>
      </svg>
    </div>
  )
}
