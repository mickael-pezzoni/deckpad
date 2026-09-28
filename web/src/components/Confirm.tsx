import { createPortal } from 'react-dom'
import { useTranslation } from 'react-i18next'

type Props = {
  title: string
  message: string
  confirmLabel: string
  onConfirm: () => void
  onCancel: () => void
}

// Fenêtre de confirmation plein écran, avec deux gros boutons.
export function Confirm({ title, message, confirmLabel, onConfirm, onCancel }: Props) {
  const { t } = useTranslation()
  return createPortal(
    <div className="overlay" onClick={onCancel}>
      <div className="dialog" onClick={(e) => e.stopPropagation()} role="dialog" aria-modal>
        <h2>{title}</h2>
        <p>{message}</p>
        <div className="dialog-actions">
          <button type="button" className="btn" onClick={onCancel}>
            {t('common.cancel')}
          </button>
          <button type="button" className="btn btn-danger" onClick={onConfirm}>
            {confirmLabel}
          </button>
        </div>
      </div>
    </div>,
    document.body,
  )
}
