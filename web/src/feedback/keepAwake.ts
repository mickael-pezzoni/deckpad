// Garde l'écran de la tablette allumé tant que l'appli est affichée (API Wake Lock).
// Le navigateur relâche le verrou dès que l'appli passe en arrière-plan : on le
// redemande quand elle revient. L'API n'existe qu'en contexte sécurisé (HTTPS) :
// en HTTP simple, rien ne se passe et la tablette suit sa mise en veille normale.
// Actif par défaut ; un réglage viendra dans la page Paramètres.

let lock: WakeLockSentinel | null = null
let pending = false

async function acquire() {
  if (lock || pending || document.visibilityState !== 'visible') return
  pending = true
  try {
    lock = await navigator.wakeLock.request('screen')
    lock.addEventListener('release', () => {
      lock = null
    })
  } catch {
    // refusé (batterie faible, pas de geste encore…) : on réessaiera au prochain appui ou retour
  } finally {
    pending = false
  }
}

export function installKeepAwake() {
  if (!('wakeLock' in navigator)) return
  acquire()
  document.addEventListener('visibilitychange', acquire)
  // Certains navigateurs refusent le verrou avant le premier geste : on retente à chaque appui.
  document.addEventListener('pointerdown', acquire, { capture: true, passive: true })
}
