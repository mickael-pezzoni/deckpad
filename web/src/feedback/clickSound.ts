// Petit « clic » joué sur la tablette à chaque appui sur un bouton d'action, sur
// toutes les pages. Généré avec Web Audio : aucun fichier son à charger.
// Le menu du bas (changer de page) reste silencieux, et les boutons marqués
// data-silent jouent le son eux-mêmes (appui long : au moment où l'action part).

let ctx: AudioContext | null = null

export function playClick() {
  // Créé au premier appui : les navigateurs n'autorisent le son qu'après un geste.
  try {
    ctx ??= new AudioContext()
  } catch {
    return // pas de son disponible : l'appui marche quand même
  }
  if (ctx.state === 'suspended') ctx.resume()
  const t = ctx.currentTime
  const osc = ctx.createOscillator()
  const gain = ctx.createGain()
  osc.type = 'sine'
  osc.frequency.setValueAtTime(1400, t)
  osc.frequency.exponentialRampToValueAtTime(700, t + 0.05)
  gain.gain.setValueAtTime(0.0001, t)
  gain.gain.exponentialRampToValueAtTime(0.25, t + 0.005)
  gain.gain.exponentialRampToValueAtTime(0.0001, t + 0.07)
  osc.connect(gain).connect(ctx.destination)
  osc.start(t)
  osc.stop(t + 0.08)
}

// installClickSound écoute tous les clics de l'appli (un seul écouteur).
export function installClickSound() {
  document.addEventListener(
    'click',
    (e) => {
      const button = (e.target as Element | null)?.closest('button')
      if (!button || button.disabled || button.dataset.silent !== undefined || button.closest('.nav-menu')) return
      playClick()
    },
    { capture: true },
  )
}
