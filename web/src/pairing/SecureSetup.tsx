import { useState } from 'react'
import { Download, Lock, Settings, ShieldCheck, type LucideIcon } from 'lucide-react'
import type { ParseKeys } from 'i18next'
import { useTranslation } from 'react-i18next'
import { goSecure, later, secureReachable } from './secure'

const isApple = /iPad|iPhone|Macintosh/.test(navigator.userAgent) && navigator.maxTouchPoints > 1

// Étapes illustrées pour installer le certificat, selon la tablette.
const steps: { Icon: LucideIcon; title: ParseKeys; text: ParseKeys }[] = [
  { Icon: Download, title: 'secure.download', text: 'secure.downloadText' },
  { Icon: Settings, title: 'secure.install', text: isApple ? 'secure.installApple' : 'secure.installAndroid' },
  isApple
    ? { Icon: ShieldCheck, title: 'secure.allow', text: 'secure.allowApple' }
    : { Icon: ShieldCheck, title: 'secure.comeBack', text: 'secure.comeBackText' },
]

type Status = { kind: 'idle' } | { kind: 'checking' } | { kind: 'failed' }

// Après l'appairage : installer le certificat de deckpad pour passer en HTTPS,
// ce qui permet d'installer l'appli en plein écran.
export function SecureSetup({ port, onSkip }: { port: string; onSkip: () => void }) {
  const { t } = useTranslation()
  const [status, setStatus] = useState<Status>({ kind: 'idle' })

  const proceed = async () => {
    setStatus({ kind: 'checking' })
    if ((await secureReachable(port)) && (await goSecure(port))) return
    setStatus({ kind: 'failed' })
  }

  const skip = () => {
    later()
    onSkip()
  }

  let message = t('secure.intro')
  if (status.kind === 'checking') message = t('pairing.checking')
  if (status.kind === 'failed') message = t(isApple ? 'secure.failedApple' : 'secure.failedAndroid')

  return (
    <main className="secure">
      <div className="pairing-intro">
        <Lock size={40} strokeWidth={1.75} aria-hidden className="pairing-icon" />
        <h1>{t('secure.title')}</h1>
        <p className={`pairing-message${status.kind === 'failed' ? ' is-error' : ''}`} role="status">
          {message}
        </p>
      </div>
      <ol className="secure-steps">
        {steps.map(({ Icon, title, text }, i) => (
          <li key={title} className="secure-step">
            <span className="secure-step-num">{i + 1}</span>
            <Icon size={32} strokeWidth={1.75} aria-hidden className="secure-step-icon" />
            <strong>{t(title)}</strong>
            <p>{t(text)}</p>
          </li>
        ))}
      </ol>
      <div className="secure-actions">
        <a className="btn secure-btn" href="/ca" download="deckpad-ca.crt">
          <Download size={24} strokeWidth={1.75} aria-hidden />
          {t('secure.downloadCert')}
        </a>
        <button type="button" className="btn btn-primary secure-btn" onClick={proceed} disabled={status.kind === 'checking'}>
          <Lock size={24} strokeWidth={1.75} aria-hidden />
          {t('secure.proceed')}
        </button>
      </div>
      <button type="button" className="secure-later" onClick={skip}>
        {t('secure.later')}
      </button>
    </main>
  )
}
