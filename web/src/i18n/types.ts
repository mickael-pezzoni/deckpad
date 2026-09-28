import type { Messages } from './fr'

// Clés de traduction vérifiées à la compilation : une faute de frappe dans t('…') est une erreur.
declare module 'i18next' {
  interface CustomTypeOptions {
    resources: { translation: Messages }
  }
}
