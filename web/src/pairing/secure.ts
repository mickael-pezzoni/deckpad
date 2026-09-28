// Passage du HTTP au HTTPS local : l'agent sert les deux, le HTTPS sur son propre port.
// Une fois le certificat de deckpad installé sur la tablette, l'appli s'installe en vraie PWA.

const DONE = 'deckpad.secure' // la tablette a déjà basculé : on y retourne tout seul
const LATER = 'deckpad.secureLater' // « Plus tard » : on redemande après une semaine
const LATER_MS = 7 * 24 * 3600 * 1000

function get(key: string) {
  try {
    return localStorage.getItem(key)
  } catch {
    return null
  }
}

function set(key: string, value: string) {
  try {
    localStorage.setItem(key, value)
  } catch {
    // stockage indisponible : la question reviendra au prochain lancement
  }
}

// Page ouverte en HTTP depuis la tablette (pas sur le PC lui-même, déjà sécurisé).
export function onInsecureLan() {
  return location.protocol === 'http:' && !['localhost', '127.0.0.1', '[::1]'].includes(location.hostname)
}

// Port HTTPS de l'agent, ou null s'il n'a pas pu démarrer.
export async function securePort(): Promise<string | null> {
  const r = await fetch('/api/pair/secure').catch(() => null)
  const info = await r?.json().catch(() => null)
  return info?.port || null
}

function secureOrigin(port: string) {
  return `https://${location.hostname}:${port}`
}

// Le certificat est-il reconnu ? Une requête vers le HTTPS échoue si la tablette
// ne fait pas confiance au certificat (no-cors : on n'a pas besoin de lire la réponse).
export async function secureReachable(port: string) {
  try {
    await fetch(`${secureOrigin(port)}/icon.svg?t=${Date.now()}`, {
      mode: 'no-cors',
      cache: 'no-store',
      signal: AbortSignal.timeout(4000),
    })
    return true
  } catch {
    return false
  }
}

// Bascule vers le HTTPS en emportant l'appairage (code à usage unique, valable une minute).
export async function goSecure(port: string) {
  const r = await fetch('/api/pair/handoff', { method: 'POST' }).catch(() => null)
  const res = await r?.json().catch(() => null)
  if (!res?.code) return false
  set(DONE, '1')
  location.replace(`${secureOrigin(port)}/?handoff=${encodeURIComponent(res.code)}`)
  return true
}

// Arrivée sur le HTTPS : on récupère l'appairage puis on nettoie l'adresse.
export async function claimHandoff() {
  const code = new URLSearchParams(location.search).get('handoff')
  if (!code) return
  history.replaceState(null, '', '/')
  await fetch('/api/pair/claim', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ code }),
  }).catch(() => null)
}

export const alreadySecured = () => get(DONE) === '1'

export function askedRecently() {
  const at = Number(get(LATER))
  return at > 0 && Date.now() - at < LATER_MS
}

export const later = () => set(LATER, String(Date.now()))
