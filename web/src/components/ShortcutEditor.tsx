import { useState } from 'react'
import { createPortal } from 'react-dom'
import { useTranslation } from 'react-i18next'
import { Confirm } from './Confirm'
import { COLORS, ICONS, KEYS, MODIFIERS, comboLabel, keyLabel, type Kind, type Shortcut } from '../shortcuts/types'

type Props = {
  shortcut: Shortcut | null // null : nouveau raccourci
  keysReason?: string // renseigné si le PC ne peut pas recevoir de touches
  captureReason?: string // renseigné si le PC ne sait pas faire de capture
  onSave: (s: Shortcut) => Promise<string | null> // renvoie un message d'erreur
  onDelete: () => void
  onCancel: () => void
}

const KINDS = [
  { id: 'keys', label: 'shortcuts.kindKeys' },
  { id: 'launch', label: 'shortcuts.kindLaunch' },
  { id: 'open', label: 'shortcuts.kindOpen' },
  { id: 'capture', label: 'shortcuts.kindCapture' },
] as const satisfies readonly { id: Kind; label: string }[]

const EMPTY: Shortcut = { id: '', label: '', icon: 'zap', color: 'blue', kind: 'keys', keys: ['ctrl', 'c'] }

// Fenêtre de création / modification d'un raccourci.
export function ShortcutEditor({ shortcut, keysReason, captureReason, onSave, onDelete, onCancel }: Props) {
  const { t } = useTranslation()
  const [draft, setDraft] = useState<Shortcut>(shortcut ?? EMPTY)
  const [error, setError] = useState<string | null>(null)
  const [saving, setSaving] = useState(false)
  const [confirmDelete, setConfirmDelete] = useState(false)

  const keys = draft.keys ?? []
  const mods = keys.filter((k) => MODIFIERS.includes(k))
  const main = keys.find((k) => !MODIFIERS.includes(k)) ?? 'a'

  const set = (patch: Partial<Shortcut>) => setDraft((d) => ({ ...d, ...patch }))
  const setKeys = (nextMods: string[], nextMain: string) =>
    set({ keys: [...MODIFIERS.filter((id) => nextMods.includes(id)), nextMain] })

  function toggleMod(id: string) {
    setKeys(mods.includes(id) ? mods.filter((m) => m !== id) : [...mods, id], main)
  }

  async function save() {
    const s: Shortcut = { ...draft, label: draft.label.trim() }
    if (!s.label) return setError(t('shortcuts.needName'))
    if (s.kind === 'keys') s.keys = [...mods, main]
    if (s.kind === 'launch' && !s.command?.trim()) return setError(t('shortcuts.needCommand'))
    if (s.kind === 'open' && !s.target?.trim()) return setError(t('shortcuts.needTarget'))
    setSaving(true)
    const err = await onSave(s)
    setSaving(false)
    setError(err)
  }

  return createPortal(
    <div className="overlay">
      <div className="dialog editor" role="dialog" aria-modal>
        <h2>{shortcut ? t('shortcuts.edit') : t('shortcuts.new')}</h2>

        <div className="editor-body">
          <label className="field">
            <span>{t('shortcuts.name')}</span>
            <input
              value={draft.label}
              maxLength={40}
              placeholder={t('shortcuts.namePlaceholder')}
              onChange={(e) => set({ label: e.target.value })}
            />
          </label>

          <div className="field">
            <span>{t('shortcuts.icon')}</span>
            <div className="picker">
              {Object.entries(ICONS).map(([id, Icon]) => (
                <button
                  key={id}
                  type="button"
                  className={draft.icon === id ? 'active' : ''}
                  onClick={() => set({ icon: id })}
                  aria-label={id}
                >
                  <Icon size={24} aria-hidden />
                </button>
              ))}
            </div>
          </div>

          <div className="field">
            <span>{t('shortcuts.color')}</span>
            <div className="picker">
              {Object.entries(COLORS).map(([id, c]) => (
                <button
                  key={id}
                  type="button"
                  className={`swatch ${draft.color === id ? 'active' : ''}`}
                  style={{ '--c': c } as React.CSSProperties}
                  onClick={() => set({ color: id })}
                  aria-label={id}
                />
              ))}
            </div>
          </div>

          <div className="field">
            <span>{t('shortcuts.action')}</span>
            <div className="segmented editor-kinds">
              {KINDS.map((k) => (
                <button
                  key={k.id}
                  type="button"
                  className={draft.kind === k.id ? 'active' : ''}
                  onClick={() => set({ kind: k.id, keys: k.id === 'keys' ? [...mods, main] : draft.keys })}
                >
                  {t(k.label)}
                </button>
              ))}
            </div>
          </div>

          {draft.kind === 'keys' && (
            <div className="field">
              <span>{t('shortcuts.combo', { combo: comboLabel([...mods, main]) })}</span>
              <div className="editor-keys">
                {MODIFIERS.map((m) => (
                  <button
                    key={m}
                    type="button"
                    className={`chip ${mods.includes(m) ? 'active' : ''}`}
                    onClick={() => toggleMod(m)}
                  >
                    {keyLabel(m)}
                  </button>
                ))}
                <select value={main} onChange={(e) => setKeys(mods, e.target.value)} aria-label={t('shortcuts.key')}>
                  {KEYS.map((id) => (
                    <option key={id} value={id}>
                      {keyLabel(id)}
                    </option>
                  ))}
                </select>
              </div>
              {keysReason && <p className="editor-warning">{keysReason}</p>}
            </div>
          )}
          {draft.kind === 'capture' && (
            <div className="field">
              <span>{t('shortcuts.captureHelp')}</span>
              {captureReason && <p className="editor-warning">{captureReason}</p>}
            </div>
          )}
          {draft.kind === 'launch' && (
            <label className="field">
              <span>{t('shortcuts.command')}</span>
              <input
                value={draft.command ?? ''}
                placeholder={t('shortcuts.commandPlaceholder')}
                autoCapitalize="off"
                autoCorrect="off"
                spellCheck={false}
                onChange={(e) => set({ command: e.target.value })}
              />
            </label>
          )}
          {draft.kind === 'open' && (
            <label className="field">
              <span>{t('shortcuts.target')}</span>
              <input
                value={draft.target ?? ''}
                placeholder={t('shortcuts.targetPlaceholder')}
                autoCapitalize="off"
                autoCorrect="off"
                spellCheck={false}
                onChange={(e) => set({ target: e.target.value })}
              />
            </label>
          )}
        </div>

        {error && <p className="editor-error">{error}</p>}
        <div className={`dialog-actions ${shortcut ? 'dialog-actions-3' : ''}`}>
          {shortcut && (
            <button type="button" className="btn btn-danger" onClick={() => setConfirmDelete(true)}>
              {t('common.delete')}
            </button>
          )}
          <button type="button" className="btn" onClick={onCancel}>
            {t('common.cancel')}
          </button>
          <button type="button" className="btn btn-primary" onClick={save} disabled={saving}>
            {t('common.save')}
          </button>
        </div>
      </div>
      {confirmDelete && (
        <Confirm
          title={t('shortcuts.confirmDelete', { name: draft.label || shortcut?.label })}
          message={t('shortcuts.deleteMessage')}
          confirmLabel={t('common.delete')}
          onConfirm={onDelete}
          onCancel={() => setConfirmDelete(false)}
        />
      )}
    </div>,
    document.body,
  )
}
