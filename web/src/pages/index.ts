import type { ComponentType } from 'react'
import { Activity, Clipboard, Folder, ListTree, Monitor, Music, Network, Power, Settings, Volume2, Zap, type LucideIcon } from 'lucide-react'
import { InfoPage } from './InfoPage'
import { ComingSoon } from './ComingSoon'
import { StatsPage } from './StatsPage'
import { ProcessPage } from './ProcessPage'
import { NetworkPage } from './NetworkPage'
import { SystemPage } from './SystemPage'
import { FilesPage } from './FilesPage'
import { AudioPage } from './AudioPage'
import { MediaPage } from './MediaPage'
import { ShortcutsPage } from './ShortcutsPage'
import { ClipboardPage } from './ClipboardPage'

export type Page = { id: string; title: string; icon: LucideIcon; Component: ComponentType }

// Ordre du swipe et du menu. Toutes les pages sont au même niveau.
export const pages: Page[] = [
  { id: 'info', title: 'Infos PC', icon: Monitor, Component: InfoPage },
  { id: 'stats', title: 'Stats', icon: Activity, Component: StatsPage },
  { id: 'process', title: 'Processus', icon: ListTree, Component: ProcessPage },
  { id: 'files', title: 'Fichiers', icon: Folder, Component: FilesPage },
  { id: 'network', title: 'Réseau', icon: Network, Component: NetworkPage },
  { id: 'audio', title: 'Audio', icon: Volume2, Component: AudioPage },
  { id: 'media', title: 'Médias', icon: Music, Component: MediaPage },
  { id: 'shortcuts', title: 'Raccourcis', icon: Zap, Component: ShortcutsPage },
  { id: 'clipboard', title: 'Presse-papiers', icon: Clipboard, Component: ClipboardPage },
  { id: 'system', title: 'Système', icon: Power, Component: SystemPage },
  { id: 'settings', title: 'Paramètres', icon: Settings, Component: ComingSoon },
]
