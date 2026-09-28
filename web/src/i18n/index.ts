// Traductions de l'appli (react-i18next). Au premier lancement, on prend la langue
// de l'appareil (anglais si ce n'est ni le français ni l'anglais) ; elle se change
// ensuite dans Paramètres et reste mémorisée sur la tablette.
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

function fromDevice(): Lang {
  const langs = navigator.languages?.length ? navigator.languages : [navigator.language]
  for (const l of langs) {
    const base = l?.toLowerCase().split('-')[0]
    if (base === 'fr' || base === 'en') return base
  }
  return 'en'
}

export function getLang(): Lang {
  return i18n.language === 'fr' ? 'fr' : 'en'
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
  return getLang() === 'fr' ? 'fr-FR' : 'en-US'
}

i18n.on('languageChanged', (lng) => {
  document.documentElement.lang = lng
})

i18n.use(initReactI18next).init({
  resources: { fr: { translation: fr }, en: { translation: en } },
  lng: stored() ?? fromDevice(),
  fallbackLng: 'en',
  interpolation: { escapeValue: false }, // React échappe déjà
  returnNull: false,
})

export default i18n
