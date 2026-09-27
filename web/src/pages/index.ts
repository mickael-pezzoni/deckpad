import type { ComponentType } from 'react'
import { Activity, Folder, ListTree, Monitor, Network, Power, Settings, type LucideIcon } from 'lucide-react'
import { InfoPage } from './InfoPage'
import { ComingSoon } from './ComingSoon'
import { StatsPage } from './StatsPage'
import { ProcessPage } from './ProcessPage'
import { NetworkPage } from './NetworkPage'
import { SystemPage } from './SystemPage'

export type Page = { id: string; title: string; icon: LucideIcon; Component: ComponentType }

// Ordre du swipe et du menu. Toutes les pages sont au même niveau.
export const pages: Page[] = [
  { id: 'info', title: 'Infos PC', icon: Monitor, Component: InfoPage },
  { id: 'stats', title: 'Stats', icon: Activity, Component: StatsPage },
  { id: 'process', title: 'Processus', icon: ListTree, Component: ProcessPage },
  { id: 'files', title: 'Fichiers', icon: Folder, Component: ComingSoon },
  { id: 'network', title: 'Réseau', icon: Network, Component: NetworkPage },
  { id: 'system', title: 'Système', icon: Power, Component: SystemPage },
  { id: 'settings', title: 'Paramètres', icon: Settings, Component: ComingSoon },
]
