import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Languages, Rocket } from 'lucide-react'
import { LANGS, getLang, setLang } from '../i18n'
import { getStartPage, setStartPage } from '../settings/startPage'
import { pages } from '.'

// Réglages de la tablette : langue de l'appli et page affichée au lancement.
export function SettingsPage() {
  const { t } = useTranslation() // re-rend la page quand la langue change
  const [start, setStart] = useState(getStartPage)
  const choices = pages.filter((p) => p.id !== 'settings')

  return (
    <div className="settings">
      <section className="tile settings-section">
        <Heading icon={<Languages size={20} strokeWidth={2} aria-hidden />} label={t('settings.language')} />
        <div className="settings-options settings-langs" role="radiogroup" aria-label={t('settings.language')}>
          {LANGS.map(({ id, label }) => (
            <button
              key={id}
              type="button"
              role="radio"
              lang={id}
              aria-checked={getLang() === id}
              className={getLang() === id ? 'settings-option active' : 'settings-option'}
              onClick={() => setLang(id)}
            >
              {label}
            </button>
          ))}
        </div>
      </section>

      <section className="tile settings-section settings-start">
        <Heading icon={<Rocket size={20} strokeWidth={2} aria-hidden />} label={t('settings.startPage')} />
        <div className="settings-options settings-pages" role="radiogroup" aria-label={t('settings.startPage')}>
          {choices.map(({ id, icon: Icon }) => (
            <button
              key={id}
              type="button"
              role="radio"
              aria-checked={start === id}
              className={start === id ? 'settings-option active' : 'settings-option'}
              onClick={() => {
                setStartPage(id)
                setStart(id)
              }}
            >
              <Icon size={26} strokeWidth={2} aria-hidden />
              <span>{t(`pages.${id}`)}</span>
            </button>
          ))}
        </div>
      </section>
    </div>
  )
}

function Heading({ icon, label }: { icon: React.ReactNode; label: string }) {
  return (
    <h2 className="tile-label settings-heading">
      <span className="tile-icon">{icon}</span>
      {label}
    </h2>
  )
}
