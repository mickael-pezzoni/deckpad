import { useEffect, useState } from 'react'
import { createPortal } from 'react-dom'
import { ClipboardCopy, Copy, Download, Monitor, X } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { Loader } from '../components/Loader'
import { SideScroll } from '../components/SideScroll'
import { ACTION_ICONS, HistoryChip } from '../components/HistoryChip'
import { useEventStream } from '../hooks/useEventStream'
import { copyToDevice, looksLikeURL } from '../clipboard/copy'
import { loadHistory, pushHistory, removeHistory, type SendAction } from '../clipboard/history'
import { api } from '../api'

type ClipboardState = {
  text?: string
  truncated?: boolean
  image?: string
  unavailable?: string
}

const SENT = {
  copy: 'clipboard.sentCopy',
  open: 'clipboard.sentOpen',
  type: 'clipboard.sentType',
} as const satisfies Record<SendAction, string>

const SEND_BUTTONS = [
  { action: 'copy', label: 'clipboard.copy' },
  { action: 'open', label: 'clipboard.open' },
  { action: 'type', label: 'clipboard.type' },
] as const satisfies readonly { action: SendAction; label: string }[]

export function ClipboardPage() {
  const { t } = useTranslation()
  const [draft, setDraft] = useState('')
  const [history, setHistory] = useState(loadHistory)
  const [pc, setPC] = useState<ClipboardState | null>(null)
  const [notice, setNotice] = useState<string | null>(null)
  const connected = useEventStream<ClipboardState>(api('/clipboard/stream'), setPC)

  useEffect(() => {
    if (!notice) return
    const id = setTimeout(() => setNotice(null), 3000)
    return () => clearTimeout(id)
  }, [notice])

  async function send(action: SendAction, text: string) {
    if (!text.trim()) return setNotice(t('clipboard.needText'))
    if (action === 'open' && !looksLikeURL(text)) return setNotice(t('clipboard.notURL'))
    const r = await fetch(api('/clipboard/send'), {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ action, text }),
    }).catch(() => null)
    if (!r) return setNotice(t('common.unreachable'))
    if (!r.ok) return setNotice((await r.text()).trim())
    setNotice(t(SENT[action]))
    setHistory((h) => pushHistory(h, action, text))
  }

  async function copyFromPC() {
    if (!pc?.text) return
    setNotice((await copyToDevice(pc.text)) ? t('clipboard.copiedHere') : t('clipboard.copyRefused'))
  }

  const isURL = looksLikeURL(draft)

  return (
    <>
      <div className="clip">
        <div className="clip-send">
          <div className="clip-input">
            <textarea
              value={draft}
              onChange={(e) => setDraft(e.target.value)}
              placeholder={t('clipboard.placeholder')}
              spellCheck={false}
            />
            {draft && (
              <button type="button" className="clip-clear" aria-label={t('common.clear')} onClick={() => setDraft('')}>
                <X size={22} strokeWidth={2} aria-hidden />
              </button>
            )}
          </div>
          <div className="clip-actions">
            {SEND_BUTTONS.map(({ action, label }) => {
              const Icon = ACTION_ICONS[action]
              const disabled = !draft.trim() || (action === 'open' && !isURL)
              return (
                <button
                  key={action}
                  type="button"
                  className={`clip-action ${disabled ? 'is-disabled' : ''}`}
                  onClick={() => send(action, draft)}
                >
                  <Icon size={30} strokeWidth={1.75} aria-hidden />
                  <span>{t(label)}</span>
                </button>
              )
            })}
          </div>
        </div>

        <div className="clip-pc">
          <span className="clip-pc-head">
            <Monitor size={20} strokeWidth={2} aria-hidden />
            {t('clipboard.pcClipboard')}
          </span>
          <PCContent state={pc} connected={connected} />
          {pc?.image ? (
            <a className="clip-action clip-pc-btn" href={api(`/clipboard/image?v=${pc.image}&download`)} download>
              <Download size={26} strokeWidth={1.75} aria-hidden />
              <span>{t('clipboard.saveImage')}</span>
            </a>
          ) : (
            <button
              type="button"
              className={`clip-action clip-pc-btn ${pc?.text ? '' : 'is-disabled'}`}
              onClick={copyFromPC}
              disabled={!pc?.text}
            >
              <Copy size={26} strokeWidth={1.75} aria-hidden />
              <span>{t('clipboard.copyHere')}</span>
            </button>
          )}
        </div>
      </div>

      {history.length > 0 && (
        <SideScroll className="clip-history">
          {history.map((h) => (
            <HistoryChip
              key={h.id}
              item={h}
              onSend={() => send(h.action, h.text)}
              onRemove={() => {
                setHistory((list) => removeHistory(list, h.id))
                setNotice(t('clipboard.removed'))
              }}
            />
          ))}
        </SideScroll>
      )}

      {notice && createPortal(<div className="toast">{notice}</div>, document.body)}
    </>
  )
}

// Ce que contient le presse-papiers du PC : texte, image ou rien.
function PCContent({ state, connected }: { state: ClipboardState | null; connected: boolean }) {
  const { t } = useTranslation()
  if (!state) return <Loader label={connected ? undefined : t('common.connecting')} />
  if (state.unavailable) return <p className="clip-pc-empty">{t('clipboard.unavailable', { reason: state.unavailable })}</p>
  if (state.image) {
    return (
      <div className="clip-pc-image">
        <img src={api(`/clipboard/image?v=${state.image}`)} alt={t('clipboard.imageAlt')} />
      </div>
    )
  }
  if (state.text) {
    return (
      <div className="clip-pc-text">
        {state.text}
        {state.truncated && <span className="clip-pc-more">{t('clipboard.truncated')}</span>}
      </div>
    )
  }
  return (
    <div className="clip-pc-empty">
      <ClipboardCopy size={48} strokeWidth={1.5} aria-hidden />
      <p>{t('clipboard.empty')}</p>
    </div>
  )
}
