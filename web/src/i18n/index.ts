// Traductions de l'appli (react-i18next). La langue est choisie dans Paramètres et
// mémorisée sur la tablette ; au premier lancement on suit la langue du navigateur,
// et le français sert de repli.
import i18n from 'i18next'
import { initReactI18next } from 'react-i18next'
import { fr } from './fr'
import { en } from './en'

export type Lang = 'fr' | 'en'

export const LANGS: { id: Lang; label: string }[] = [
  { id: 'fr', label: 'Français' },
  { id: 'en', label: 'English' },
]

const KEY = 'deckpad.lang'

function stored(): Lang | null {
  try {
    const v = localStorage.getItem(KEY)
    return v === 'fr' || v === 'en' ? v : null
  } catch {
    return null
  }
}

function fromBrowser(): Lang {
  const langs = navigator.languages?.length ? navigator.languages : [navigator.language]
  for (const l of langs) {
    const base = l?.toLowerCase().split('-')[0]
    if (base === 'fr' || base === 'en') return base
  }
  return 'fr'
}

export function getLang(): Lang {
  return i18n.language === 'en' ? 'en' : 'fr'
}

export function setLang(lang: Lang) {
  try {
    localStorage.setItem(KEY, lang)
  } catch {
    // stockage indisponible : la langue vaut pour cette session seulement
  }
  i18n.changeLanguage(lang)
}

// Locale des dates et des nombres (virgule décimale en français, point en anglais).
export function locale() {
  return getLang() === 'en' ? 'en-US' : 'fr-FR'
}

i18n.on('languageChanged', (lng) => {
  document.documentElement.lang = lng
})

i18n.use(initReactI18next).init({
  resources: { fr: { translation: fr }, en: { translation: en } },
  lng: stored() ?? fromBrowser(),
  fallbackLng: 'fr',
  interpolation: { escapeValue: false }, // React échappe déjà
  returnNull: false,
})

export default i18n
