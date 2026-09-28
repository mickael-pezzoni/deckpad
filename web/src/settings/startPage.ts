// Page affichée au lancement de l'appli, choisie dans Paramètres et mémorisée sur la tablette.

const KEY = 'deckpad.startPage'
export const DEFAULT_START_PAGE = 'info'

export function getStartPage(): string {
  try {
    return localStorage.getItem(KEY) ?? DEFAULT_START_PAGE
  } catch {
    return DEFAULT_START_PAGE
  }
}

export function setStartPage(id: string) {
  try {
    localStorage.setItem(KEY, id)
  } catch {
    // stockage indisponible : le choix vaut pour cette session seulement
  }
}
