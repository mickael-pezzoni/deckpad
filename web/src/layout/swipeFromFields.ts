// Swiper ignore un glissement qui démarre sur le champ de saisie en cours d'édition
// (pour laisser sélectionner le texte). Sur la tablette, on veut quand même changer
// de page : un geste nettement horizontal sur ce champ le quitte (le clavier se ferme)
// et Swiper prend le relais. Un appui ou un geste vertical ne change rien.
const FIELDS = 'input, textarea'
const START_PX = 12

export function swipeFromFields(): () => void {
  let start: { x: number; y: number } | null = null

  const down = (e: PointerEvent) => {
    const el = e.target as Element
    start = e.pointerType !== 'mouse' && el === document.activeElement && el.matches(FIELDS) ? { x: e.clientX, y: e.clientY } : null
  }
  const move = (e: PointerEvent) => {
    if (!start || e.target !== document.activeElement) return
    const dx = Math.abs(e.clientX - start.x)
    const dy = Math.abs(e.clientY - start.y)
    if (dx > START_PX && dx > dy * 2) {
      start = null
      ;(document.activeElement as HTMLElement).blur()
    }
  }
  const up = () => (start = null)

  // Phase de capture : avant Swiper, qui écoute les mêmes événements.
  document.addEventListener('pointerdown', down, true)
  document.addEventListener('pointermove', move, true)
  document.addEventListener('pointerup', up, true)
  document.addEventListener('pointercancel', up, true)
  return () => {
    document.removeEventListener('pointerdown', down, true)
    document.removeEventListener('pointermove', move, true)
    document.removeEventListener('pointerup', up, true)
    document.removeEventListener('pointercancel', up, true)
  }
}
