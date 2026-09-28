import type { ComponentType } from 'react'
import { Activity, Clipboard, Folder, ListTree, Monitor, Music, Network, Power, Settings, Volume2, Zap, type LucideIcon } from 'lucide-react'
import { InfoPage } from './InfoPage'
import { StatsPage } from './StatsPage'
import { ProcessPage } from './ProcessPage'
import { NetworkPage } from './NetworkPage'
import { SystemPage } from './SystemPage'
import { FilesPage } from './FilesPage'
import { AudioPage } from './AudioPage'
import { MediaPage } from './MediaPage'
import { ShortcutsPage } from './ShortcutsPage'
import { ClipboardPage } from './ClipboardPage'
import { SettingsPage } from './SettingsPage'
import type { Messages } from '../i18n/fr'

// Le nom affiché vient des traductions : clé « pages.<id> ».
export type PageId = keyof Messages['pages']
export type Page = { id: PageId; icon: LucideIcon; Component: ComponentType }

// Ordre du swipe et du menu. Toutes les pages sont au même niveau.
export const pages: Page[] = [
  { id: 'info', icon: Monitor, Component: InfoPage },
  { id: 'stats', icon: Activity, Component: StatsPage },
  { id: 'process', icon: ListTree, Component: ProcessPage },
  { id: 'files', icon: Folder, Component: FilesPage },
  { id: 'network', icon: Network, Component: NetworkPage },
  { id: 'audio', icon: Volume2, Component: AudioPage },
  { id: 'media', icon: Music, Component: MediaPage },
  { id: 'shortcuts', icon: Zap, Component: ShortcutsPage },
  { id: 'clipboard', icon: Clipboard, Component: ClipboardPage },
  { id: 'system', icon: Power, Component: SystemPage },
  { id: 'settings', icon: Settings, Component: SettingsPage },
]
