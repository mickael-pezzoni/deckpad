import { useState } from 'react'
import { createPortal } from 'react-dom'
import { Confirm } from './Confirm'
import { COLORS, ICONS, KEYS, MODIFIERS, comboLabel, type Kind, type Shortcut } from '../shortcuts/types'

type Props = {
  shortcut: Shortcut | null // null : nouveau raccourci
  keysReason?: string // renseigné si le PC ne peut pas recevoir de touches
  captureReason?: string // renseigné si le PC ne sait pas faire de capture
  onSave: (s: Shortcut) => Promise<string | null> // renvoie un message d'erreur
  onDelete: () => void
  onCancel: () => void
}

const KINDS: { id: Kind; label: string }[] = [
  { id: 'keys', label: 'Touches' },
  { id: 'launch', label: 'Programme' },
  { id: 'open', label: 'Ouvrir' },
  { id: 'capture', label: 'Capture' },
]

const EMPTY: Shortcut = { id: '', label: '', icon: 'zap', color: 'blue', kind: 'keys', keys: ['ctrl', 'c'] }

// Fenêtre de création / modification d'un raccourci.
export function ShortcutEditor({ shortcut, keysReason, captureReason, onSave, onDelete, onCancel }: Props) {
  const [draft, setDraft] = useState<Shortcut>(shortcut ?? EMPTY)
  const [error, setError] = useState<string | null>(null)
  const [saving, setSaving] = useState(false)
  const [confirmDelete, setConfirmDelete] = useState(false)

  const keys = draft.keys ?? []
  const mods = keys.filter((k) => MODIFIERS.some((m) => m.id === k))
  const main = keys.find((k) => !MODIFIERS.some((m) => m.id === k)) ?? 'a'

  const set = (patch: Partial<Shortcut>) => setDraft((d) => ({ ...d, ...patch }))
  const setKeys = (nextMods: string[], nextMain: string) =>
    set({ keys: [...MODIFIERS.map((m) => m.id).filter((id) => nextMods.includes(id)), nextMain] })

  function toggleMod(id: string) {
    setKeys(mods.includes(id) ? mods.filter((m) => m !== id) : [...mods, id], main)
  }

  async function save() {
    const s: Shortcut = { ...draft, label: draft.label.trim() }
    if (!s.label) return setError('Donne un nom au raccourci')
    if (s.kind === 'keys') s.keys = [...mods, main]
    if (s.kind === 'launch' && !s.command?.trim()) return setError('Indique la commande à lancer')
    if (s.kind === 'open' && !s.target?.trim()) return setError('Indique le dossier ou l’adresse à ouvrir')
    setSaving(true)
    const err = await onSave(s)
    setSaving(false)
    setError(err)
  }

  return createPortal(
    <div className="overlay">
      <div className="dialog editor" role="dialog" aria-modal>
        <h2>{shortcut ? 'Modifier le raccourci' : 'Nouveau raccourci'}</h2>

        <div className="editor-body">
          <label className="field">
            <span>Nom</span>
            <input
              value={draft.label}
              maxLength={40}
              placeholder="Ex : Capture d'écran"
              onChange={(e) => set({ label: e.target.value })}
            />
          </label>

          <div className="field">
            <span>Icône</span>
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
            <span>Couleur</span>
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
            <span>Action</span>
            <div className="segmented editor-kinds">
              {KINDS.map((k) => (
                <button
                  key={k.id}
                  type="button"
                  className={draft.kind === k.id ? 'active' : ''}
                  onClick={() => set({ kind: k.id, keys: k.id === 'keys' ? [...mods, main] : draft.keys })}
                >
                  {k.label}
                </button>
              ))}
            </div>
          </div>

          {draft.kind === 'keys' && (
            <div className="field">
              <span>Combinaison : {comboLabel([...mods, main])}</span>
              <div className="editor-keys">
                {MODIFIERS.map((m) => (
                  <button
                    key={m.id}
                    type="button"
                    className={`chip ${mods.includes(m.id) ? 'active' : ''}`}
                    onClick={() => toggleMod(m.id)}
                  >
                    {m.label}
                  </button>
                ))}
                <select value={main} onChange={(e) => setKeys(mods, e.target.value)} aria-label="Touche">
                  {KEYS.map(([id, label]) => (
                    <option key={id} value={id}>
                      {label}
                    </option>
                  ))}
                </select>
              </div>
              {keysReason && <p className="editor-warning">{keysReason}</p>}
            </div>
          )}
          {draft.kind === 'capture' && (
            <div className="field">
              <span>Capture de tout l'écran, copiée dans le presse-papiers (à coller avec Ctrl + V).</span>
              {captureReason && <p className="editor-warning">{captureReason}</p>}
            </div>
          )}
          {draft.kind === 'launch' && (
            <label className="field">
              <span>Commande</span>
              <input
                value={draft.command ?? ''}
                placeholder="Ex : notepad, obs64.exe, steam"
                autoCapitalize="off"
                autoCorrect="off"
                spellCheck={false}
                onChange={(e) => set({ command: e.target.value })}
              />
            </label>
          )}
          {draft.kind === 'open' && (
            <label className="field">
              <span>Dossier, fichier ou adresse web</span>
              <input
                value={draft.target ?? ''}
                placeholder="Ex : ~/Images, D:\Jeux, https://…"
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
              Supprimer
            </button>
          )}
          <button type="button" className="btn" onClick={onCancel}>
            Annuler
          </button>
          <button type="button" className="btn btn-primary" onClick={save} disabled={saving}>
            Enregistrer
          </button>
        </div>
      </div>
      {confirmDelete && (
        <Confirm
          title={`Supprimer « ${draft.label || shortcut?.label} » ?`}
          message="Le raccourci disparaîtra de tous les appareils."
          confirmLabel="Supprimer"
          onConfirm={onDelete}
          onCancel={() => setConfirmDelete(false)}
        />
      )}
    </div>,
    document.body,
  )
}
