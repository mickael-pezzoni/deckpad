import { useTranslation } from 'react-i18next'
import { Languages } from 'lucide-react'
import { LANGS, getLang, setLang } from '../i18n'

// Choix compact de la langue (écran d'appairage, au premier lancement).
export function LanguageSwitch({ className = '' }: { className?: string }) {
  const { t } = useTranslation() // re-rend quand la langue change
  return (
    <div className={`segmented lang-switch ${className}`} role="radiogroup" aria-label={t('settings.language')}>
      <Languages size={22} strokeWidth={2} aria-hidden className="lang-switch-icon" />
      {LANGS.map(({ id, label }) => (
        <button
          key={id}
          type="button"
          role="radio"
          lang={id}
          aria-checked={getLang() === id}
          className={getLang() === id ? 'active' : ''}
          onClick={() => setLang(id)}
        >
          {label}
        </button>
      ))}
    </div>
  )
}
