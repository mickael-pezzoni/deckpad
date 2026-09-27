import type { ComponentType } from 'react'
import { InfoPage } from './InfoPage'
import { ComingSoon } from './ComingSoon'
import { StatsPage } from './StatsPage'
import { ProcessPage } from './ProcessPage'
import { NetworkPage } from './NetworkPage'

export type Page = { id: string; title: string; Component: ComponentType }

// Ordre du swipe. Toutes les pages sont au même niveau.
export const pages: Page[] = [
  { id: 'info', title: 'Infos PC', Component: InfoPage },
  { id: 'stats', title: 'Stats', Component: StatsPage },
  { id: 'process', title: 'Processus', Component: ProcessPage },
  { id: 'files', title: 'Fichiers', Component: ComingSoon },
  { id: 'network', title: 'Réseau', Component: NetworkPage },
  { id: 'system', title: 'Système', Component: ComingSoon },
  { id: 'settings', title: 'Paramètres', Component: ComingSoon },
]
