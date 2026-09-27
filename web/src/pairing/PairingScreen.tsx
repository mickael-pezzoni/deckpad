import { useCallback, useEffect, useState } from 'react'
import { Delete, MonitorSmartphone } from 'lucide-react'

const LENGTH = 6
const KEYS = ['1', '2', '3', '4', '5', '6', '7', '8', '9', 'clear', '0', 'back'] as const

type Status =
  | { kind: 'starting' }
  | { kind: 'ready' }
  | { kind: 'checking' }
  | { kind: 'wrong'; remaining: number }
  | { kind: 'blocked'; message: string } // code expiré ou trop d'essais : il faut un nouveau code
  | { kind: 'offline' }

// Nom affiché dans la liste des appareils appairés.
function deviceName() {
  const ua = navigator.userAgent
  if (/iPad|Macintosh/.test(ua) && navigator.maxTouchPoints > 1) return 'iPad'
  if (/Android/.test(ua)) return 'Tablette Android'
  if (/Windows/.test(ua)) return 'Navigateur Windows'
  if (/Linux/.test(ua)) return 'Navigateur Linux'
  return 'Tablette'
}

// Premier lancement : le PC affiche un code, on le tape ici.
export function PairingScreen({ onPaired }: { onPaired: () => void }) {
  const [digits, setDigits] = useState('')
  const [status, setStatus] = useState<Status>({ kind: 'starting' })

  const start = useCallback(async () => {
    setDigits('')
    setStatus({ kind: 'starting' })
    const r = await fetch('/api/pair/start', { method: 'POST' }).catch(() => null)
    if (!r) setStatus({ kind: 'offline' })
    else if (r.status === 429) setStatus({ kind: 'blocked', message: "Trop d'essais. Réessaie dans quelques minutes." })
    else setStatus(r.ok ? { kind: 'ready' } : { kind: 'offline' })
  }, [])

  useEffect(() => {
    start()
  }, [start])

  const submit = useCallback(
    async (code: string) => {
      setStatus({ kind: 'checking' })
      const r = await fetch('/api/pair/confirm', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ code, name: deviceName() }),
      }).catch(() => null)
      if (r?.ok) return onPaired()
      const err = await r?.json().catch(() => null)
      setDigits('')
      if (err?.reason === 'bad-code') setStatus({ kind: 'wrong', remaining: err.remaining })
      else if (err?.reason === 'locked') setStatus({ kind: 'blocked', message: "Trop d'essais. Réessaie dans quelques minutes." })
      else if (err?.reason === 'expired') setStatus({ kind: 'blocked', message: 'Code expiré.' })
      else setStatus({ kind: 'offline' })
    },
    [onPaired],
  )

  const locked = status.kind !== 'ready' && status.kind !== 'wrong'

  const press = useCallback(
    (key: string) => {
      if (locked) return
      if (key === 'clear') return setDigits('')
      if (key === 'back') return setDigits((d) => d.slice(0, -1))
      setDigits((d) => {
        if (d.length >= LENGTH) return d
        const next = d + key
        if (next.length === LENGTH) submit(next)
        return next
      })
    },
    [locked, submit],
  )

  // Clavier physique (utile pour tester depuis un PC).
  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      if (/^\d$/.test(e.key)) press(e.key)
      else if (e.key === 'Backspace') press('back')
      else if (e.key === 'Escape') press('clear')
    }
    window.addEventListener('keydown', onKey)
    return () => window.removeEventListener('keydown', onKey)
  }, [press])

  let message: string
  switch (status.kind) {
    case 'starting':
      message = 'Demande du code…'
      break
    case 'checking':
      message = 'Vérification…'
      break
    case 'wrong':
      message = `Code incorrect. ${status.remaining} essai${status.remaining > 1 ? 's' : ''} restant${status.remaining > 1 ? 's' : ''}.`
      break
    case 'blocked':
      message = status.message
      break
    case 'offline':
      message = 'Le PC ne répond pas. Vérifie que deckpad est lancé.'
      break
    default:
      message = "Tape le code affiché sur l'écran du PC."
  }

  return (
    <main className="pairing">
      <div className="pairing-intro">
        <MonitorSmartphone size={40} strokeWidth={1.75} aria-hidden className="pairing-icon" />
        <h1>Associer cette tablette</h1>
        <p className={`pairing-message${status.kind === 'wrong' || status.kind === 'blocked' ? ' is-error' : ''}`} role="status">
          {message}
        </p>
        <div className={`pin${status.kind === 'wrong' ? ' shake' : ''}`} key={status.kind === 'wrong' ? status.remaining : 'pin'}>
          {Array.from({ length: LENGTH }, (_, i) => (
            <span key={i} className={i < digits.length ? 'filled' : ''} />
          ))}
        </div>
        {(status.kind === 'blocked' || status.kind === 'offline') && (
          <button type="button" className="btn" onClick={start}>
            Nouveau code
          </button>
        )}
      </div>
      <div className="keypad" aria-disabled={locked}>
        {KEYS.map((k) => (
          <button
            key={k}
            type="button"
            className={`key${k.length > 1 ? ' key-muted' : ''}`}
            onClick={() => press(k)}
            disabled={locked || (k.length > 1 && digits === '')}
            aria-label={k === 'back' ? 'Effacer le dernier chiffre' : k === 'clear' ? 'Tout effacer' : k}
          >
            {k === 'back' ? <Delete size={32} strokeWidth={1.75} /> : k === 'clear' ? 'Effacer' : k}
          </button>
        ))}
      </div>
    </main>
  )
}
