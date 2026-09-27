import { useEffect, useState } from 'react'
import { createPortal } from 'react-dom'
import { Lock, Moon, Power, RotateCcw, type LucideIcon } from 'lucide-react'
import { HoldButton } from '../components/HoldButton'

type Action = {
  id: 'lock' | 'sleep' | 'restart' | 'shutdown'
  label: string
  icon: LucideIcon
  done: string
  danger?: boolean
}

const ACTIONS: Action[] = [
  { id: 'lock', label: 'Verrouiller', icon: Lock, done: 'PC verrouillé' },
  { id: 'sleep', label: 'Veille', icon: Moon, done: 'Mise en veille…' },
  { id: 'restart', label: 'Redémarrer', icon: RotateCcw, done: 'Redémarrage…', danger: true },
  { id: 'shutdown', label: 'Éteindre', icon: Power, done: 'Extinction…', danger: true },
]

const HOLD_MS = 1500

export function SystemPage() {
  const [notice, setNotice] = useState<string | null>(null)

  useEffect(() => {
    if (!notice) return
    const id = setTimeout(() => setNotice(null), 3000)
    return () => clearTimeout(id)
  }, [notice])

  async function run(a: Action) {
    const r = await fetch(`/api/system/${a.id}`, { method: 'POST' }).catch(() => null)
    setNotice(r?.ok ? a.done : `Impossible : ${a.label.toLowerCase()}`)
  }

  return (
    <>
      <div className="actions">
        {ACTIONS.map((a) => (
          <HoldButton
            key={a.id}
            className={a.danger ? 'action-danger' : ''}
            holdMs={a.danger ? HOLD_MS : 0}
            onConfirm={() => run(a)}
            onTooShort={() => setNotice('Maintenir appuyé pour confirmer')}
          >
            <a.icon size={48} strokeWidth={1.75} aria-hidden />
            <span className="action-label">{a.label}</span>
            {a.danger && <span className="action-hint">Maintenir appuyé</span>}
          </HoldButton>
        ))}
      </div>
      {notice && createPortal(<div className="toast">{notice}</div>, document.body)}
    </>
  )
}
