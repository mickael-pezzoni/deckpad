import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Languages, Power, Rocket } from 'lucide-react'
import { LANGS, getLang, setLang } from '../i18n'
import { api } from '../api'
import { getStartPage, setStartPage } from '../settings/startPage'
import { pages } from '.'

// Réglages : langue et page de lancement (propres à la tablette), démarrage de
// l'agent avec la session (propre au PC choisi).
export function SettingsPage() {
  const { t } = useTranslation() // re-rend la page quand la langue change
  const [start, setStart] = useState(getStartPage)
  const choices = pages.filter((p) => p.id !== 'settings')

  return (
    <div className="settings">
      <div className="settings-row">
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
        <AutostartSection />
      </div>

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

type Autostart = {
  supported: boolean
  enabled: boolean
  reason?: 'dev' | 'os'
}

// Démarrage de l'agent à l'ouverture de session du PC (clé Run sous Windows,
// ~/.config/autostart sous Linux). Grisé si l'agent ne peut pas l'enregistrer.
function AutostartSection() {
  const { t } = useTranslation()
  const [state, setState] = useState<Autostart | null>(null)
  const [busy, setBusy] = useState(false)
  const [failed, setFailed] = useState(false)

  useEffect(() => {
    fetch(api('/autostart'))
      .then((r) => (r.ok ? r.json() : Promise.reject()))
      .then(setState)
      .catch(() => setState({ supported: false, enabled: false }))
  }, [])

  const choose = async (enabled: boolean) => {
    if (!state?.supported || busy || state.enabled === enabled) return
    setBusy(true)
    const r = await fetch(api('/autostart'), {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ enabled }),
    }).catch(() => null)
    const next: Autostart | null = r?.ok ? await r.json().catch(() => null) : null
    setFailed(!next)
    if (next) setState(next)
    setBusy(false)
  }

  const disabled = !state?.supported || busy
  const hint = failed
    ? t('settings.autostartFailed')
    : state && !state.supported
      ? t(state.reason === 'dev' ? 'settings.autostartDev' : 'settings.autostartUnavailable')
      : null
  const options = [
    { on: true, label: t('settings.on') },
    { on: false, label: t('settings.off') },
  ]

  return (
    <section className="tile settings-section">
      <Heading icon={<Power size={20} strokeWidth={2} aria-hidden />} label={t('settings.autostart')} />
      <div className="settings-options settings-langs" role="radiogroup" aria-label={t('settings.autostart')}>
        {options.map(({ on, label }) => {
          const active = state?.supported === true && state.enabled === on
          return (
            <button
              key={label}
              type="button"
              role="radio"
              aria-checked={active}
              disabled={disabled}
              className={active ? 'settings-option active' : 'settings-option'}
              onClick={() => choose(on)}
            >
              {label}
            </button>
          )
        })}
      </div>
      {hint && <p className="settings-hint">{hint}</p>}
    </section>
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
