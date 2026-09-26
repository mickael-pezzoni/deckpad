import type { ComponentType } from 'react'
import { InfoPage } from './InfoPage'
import { ComingSoon } from './ComingSoon'

export type Page = { id: string; title: string; Component: ComponentType }

// Ordre du swipe. Toutes les pages sont au même niveau.
export const pages: Page[] = [
  { id: 'info', title: 'Infos PC', Component: InfoPage },
  { id: 'stats', title: 'Stats', Component: ComingSoon },
  { id: 'process', title: 'Processus', Component: ComingSoon },
  { id: 'files', title: 'Fichiers', Component: ComingSoon },
  { id: 'network', title: 'Réseau', Component: ComingSoon },
  { id: 'system', title: 'Système', Component: ComingSoon },
  { id: 'settings', title: 'Paramètres', Component: ComingSoon },
]
