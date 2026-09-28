import { useState } from 'react'
import { Download, Lock, Settings, ShieldCheck } from 'lucide-react'
import { goSecure, later, secureReachable } from './secure'

const isApple = /iPad|iPhone|Macintosh/.test(navigator.userAgent) && navigator.maxTouchPoints > 1

// Étapes illustrées pour installer le certificat, selon la tablette.
const steps = [
  {
    Icon: Download,
    title: 'Télécharger',
    text: 'Touche «\u00a0Télécharger le certificat\u00a0» ci-dessous.',
  },
  isApple
    ? {
        Icon: Settings,
        title: 'Installer',
        text: 'Réglages › Profil téléchargé › Installer.',
      }
    : {
        Icon: Settings,
        title: 'Installer',
        text: 'Dans Paramètres, cherche «\u00a0Certificat CA\u00a0» et choisis deckpad-ca.crt.',
      },
  isApple
    ? {
        Icon: ShieldCheck,
        title: 'Autoriser',
        text: 'Réglages › Général › Informations › Réglages des certificats : active deckpad.',
      }
    : {
        Icon: ShieldCheck,
        title: 'Revenir',
        text: 'Reviens ici et touche «\u00a0Continuer en sécurisé\u00a0».',
      },
]

type Status = { kind: 'idle' } | { kind: 'checking' } | { kind: 'failed' }

// Après l'appairage : installer le certificat de deckpad pour passer en HTTPS,
// ce qui permet d'installer l'appli en plein écran.
export function SecureSetup({ port, onSkip }: { port: string; onSkip: () => void }) {
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

  let message = "Une étape, une seule fois : elle permet d'installer deckpad en plein écran."
  if (status.kind === 'checking') message = 'Vérification…'
  if (status.kind === 'failed')
    message = isApple
      ? "Le certificat n'est pas encore reconnu. Vérifie qu'il est installé et activé dans Réglages des certificats."
      : "Le certificat n'est pas encore reconnu. Vérifie qu'il est installé comme «\u00a0Certificat CA\u00a0», puis réessaie."

  return (
    <main className="secure">
      <div className="pairing-intro">
        <Lock size={40} strokeWidth={1.75} aria-hidden className="pairing-icon" />
        <h1>Passer en connexion sécurisée</h1>
        <p className={`pairing-message${status.kind === 'failed' ? ' is-error' : ''}`} role="status">
          {message}
        </p>
      </div>
      <ol className="secure-steps">
        {steps.map(({ Icon, title, text }, i) => (
          <li key={title} className="secure-step">
            <span className="secure-step-num">{i + 1}</span>
            <Icon size={32} strokeWidth={1.75} aria-hidden className="secure-step-icon" />
            <strong>{title}</strong>
            <p>{text}</p>
          </li>
        ))}
      </ol>
      <div className="secure-actions">
        <a className="btn secure-btn" href="/ca" download="deckpad-ca.crt">
          <Download size={24} strokeWidth={1.75} aria-hidden />
          Télécharger le certificat
        </a>
        <button type="button" className="btn btn-primary secure-btn" onClick={proceed} disabled={status.kind === 'checking'}>
          <Lock size={24} strokeWidth={1.75} aria-hidden />
          Continuer en sécurisé
        </button>
      </div>
      <button type="button" className="secure-later" onClick={skip}>
        Plus tard
      </button>
    </main>
  )
}
