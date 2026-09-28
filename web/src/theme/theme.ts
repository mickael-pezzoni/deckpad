// Thème de l'appli (sombre par défaut, ou clair) : mémorisé sur la tablette et
// appliqué via l'attribut data-theme de <html>, que styles.css utilise pour ses couleurs.

export type Theme = 'dark' | 'light'

const KEY = 'deckpad.theme'
const BAR = { dark: '#0f1115', light: '#eceff4' } // couleur de la barre d'état, = --bg

export function getTheme(): Theme {
  try {
    return localStorage.getItem(KEY) === 'light' ? 'light' : 'dark'
  } catch {
    return 'dark'
  }
}

export function applyTheme(theme: Theme) {
  document.documentElement.dataset.theme = theme
  document.querySelector('meta[name="theme-color"]')?.setAttribute('content', BAR[theme])
}

export function setTheme(theme: Theme) {
  applyTheme(theme)
  try {
    localStorage.setItem(KEY, theme)
  } catch {
    // stockage indisponible : le thème vaut pour cette session seulement
  }
}
