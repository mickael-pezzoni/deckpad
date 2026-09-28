import { useEffect, useState } from 'react'
import { createPortal } from 'react-dom'
import { ClipboardCopy, Copy, Download, Monitor, X } from 'lucide-react'
import { Loader } from '../components/Loader'
import { ACTION_ICONS, HistoryChip } from '../components/HistoryChip'
import { useEventStream } from '../hooks/useEventStream'
import { copyToDevice, looksLikeURL } from '../clipboard/copy'
import { loadHistory, pushHistory, removeHistory, type SendAction } from '../clipboard/history'

type ClipboardState = {
  text?: string
  truncated?: boolean
  image?: string
  unavailable?: string
}

const SENT: Record<SendAction, string> = {
  copy: 'Copié sur le PC',
  open: 'Ouvert sur le PC',
  type: 'Tapé sur le PC',
}

const SEND_BUTTONS: { action: SendAction; label: string }[] = [
  { action: 'copy', label: 'Copier sur le PC' },
  { action: 'open', label: 'Ouvrir sur le PC' },
  { action: 'type', label: 'Taper sur le PC' },
]

export function ClipboardPage() {
  const [draft, setDraft] = useState('')
  const [history, setHistory] = useState(loadHistory)
  const [pc, setPC] = useState<ClipboardState | null>(null)
  const [notice, setNotice] = useState<string | null>(null)
  const connected = useEventStream<ClipboardState>('/api/clipboard/stream', setPC)

  useEffect(() => {
    if (!notice) return
    const id = setTimeout(() => setNotice(null), 3000)
    return () => clearTimeout(id)
  }, [notice])

  async function send(action: SendAction, text: string) {
    if (!text.trim()) return setNotice('Écris ou colle un texte d’abord')
    if (action === 'open' && !looksLikeURL(text)) return setNotice('Ce n’est pas une adresse web')
    const r = await fetch('/api/clipboard/send', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ action, text }),
    }).catch(() => null)
    if (!r) return setNotice('PC injoignable')
    if (!r.ok) return setNotice((await r.text()).trim())
    setNotice(SENT[action])
    setHistory((h) => pushHistory(h, action, text))
  }

  async function copyFromPC() {
    if (!pc?.text) return
    setNotice((await copyToDevice(pc.text)) ? 'Copié sur la tablette' : 'Copie refusée par le navigateur')
  }

  const isURL = looksLikeURL(draft)

  return (
    <>
      <div className="clip">
        <div className="clip-send">
          <div className="clip-input">
            <textarea
              className="swiper-no-swiping"
              value={draft}
              onChange={(e) => setDraft(e.target.value)}
              placeholder="Écris ou colle un texte, un lien…"
              spellCheck={false}
            />
            {draft && (
              <button type="button" className="clip-clear" aria-label="Effacer" onClick={() => setDraft('')}>
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
                  <span>{label}</span>
                </button>
              )
            })}
          </div>
        </div>

        <div className="clip-pc">
          <span className="clip-pc-head">
            <Monitor size={20} strokeWidth={2} aria-hidden />
            Presse-papiers du PC
          </span>
          <PCContent state={pc} connected={connected} />
          {pc?.image ? (
            <a className="clip-action clip-pc-btn" href={`/api/clipboard/image?v=${pc.image}&download`} download>
              <Download size={26} strokeWidth={1.75} aria-hidden />
              <span>Enregistrer l’image</span>
            </a>
          ) : (
            <button
              type="button"
              className={`clip-action clip-pc-btn ${pc?.text ? '' : 'is-disabled'}`}
              onClick={copyFromPC}
              disabled={!pc?.text}
            >
              <Copy size={26} strokeWidth={1.75} aria-hidden />
              <span>Copier sur la tablette</span>
            </button>
          )}
        </div>
      </div>

      {history.length > 0 && (
        <div className="clip-history swiper-no-swiping">
          {history.map((h) => (
            <HistoryChip
              key={h.id}
              item={h}
              onSend={() => send(h.action, h.text)}
              onRemove={() => {
                setHistory((list) => removeHistory(list, h.id))
                setNotice('Retiré de l’historique')
              }}
            />
          ))}
        </div>
      )}

      {notice && createPortal(<div className="toast">{notice}</div>, document.body)}
    </>
  )
}

// Ce que contient le presse-papiers du PC : texte, image ou rien.
function PCContent({ state, connected }: { state: ClipboardState | null; connected: boolean }) {
  if (!state) return <Loader label={connected ? 'Chargement…' : 'Connexion au PC…'} />
  if (state.unavailable) return <p className="clip-pc-empty">Indisponible : {state.unavailable}</p>
  if (state.image) {
    return (
      <div className="clip-pc-image">
        <img src={`/api/clipboard/image?v=${state.image}`} alt="Image copiée sur le PC" />
      </div>
    )
  }
  if (state.text) {
    return (
      <div className="clip-pc-text swiper-no-swiping">
        {state.text}
        {state.truncated && <span className="clip-pc-more">… (texte trop long, début seulement)</span>}
      </div>
    )
  }
  return (
    <div className="clip-pc-empty">
      <ClipboardCopy size={48} strokeWidth={1.5} aria-hidden />
      <p>Rien de copié sur le PC</p>
    </div>
  )
}
