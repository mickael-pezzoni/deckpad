import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import App from './App'
import './styles.css'
import { installClickSound } from './feedback/clickSound'
import { installKeepAwake } from './feedback/keepAwake'

installClickSound()
installKeepAwake()

// Le service worker n'est accepté qu'en contexte sécurisé (HTTPS ou localhost) :
// en HTTP simple sur le réseau local, navigator.serviceWorker n'existe pas et on s'en passe.
if (import.meta.env.PROD && 'serviceWorker' in navigator) {
  navigator.serviceWorker.register('/sw.js').catch(() => {})
}

createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <App />
  </StrictMode>,
)
