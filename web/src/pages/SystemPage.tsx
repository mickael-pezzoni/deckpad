import { useEffect, useState } from 'react'
import { createPortal } from 'react-dom'
import { Lock, Moon, Power, RotateCcw, type LucideIcon } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { HoldButton } from '../components/HoldButton'
import { api } from '../api'

type Action = {
  id: 'lock' | 'sleep' | 'restart' | 'shutdown'
  icon: LucideIcon
  danger?: boolean
}

const ACTIONS: Action[] = [
  { id: 'lock', icon: Lock },
  { id: 'sleep', icon: Moon },
  { id: 'restart', icon: RotateCcw, danger: true },
  { id: 'shutdown', icon: Power, danger: true },
]

const HOLD_MS = 1500

export function SystemPage() {
  const { t } = useTranslation()
  const [notice, setNotice] = useState<string | null>(null)

  useEffect(() => {
    if (!notice) return
    const id = setTimeout(() => setNotice(null), 3000)
    return () => clearTimeout(id)
  }, [notice])

  async function run(a: Action) {
    const r = await fetch(api(`/system/${a.id}`), { method: 'POST' }).catch(() => null)
    setNotice(r?.ok ? t(`system.${a.id}Done`) : t('system.failed', { action: t(`system.${a.id}`).toLowerCase() }))
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
            onTooShort={() => setNotice(t('common.holdToConfirm'))}
          >
            <a.icon size={48} strokeWidth={1.75} aria-hidden />
            <span className="action-label">{t(`system.${a.id}`)}</span>
            {a.danger && <span className="action-hint">{t('common.hold')}</span>}
          </HoldButton>
        ))}
      </div>
      {notice && createPortal(<div className="toast">{notice}</div>, document.body)}
    </>
  )
}
