import {
  Calculator, Camera, Clipboard, Code, Folder, Gamepad2, Gauge, Globe, Headphones, Keyboard, Mail,
  MessageSquare, Mic, Monitor, Music, Rocket, Search, Star, Terminal, Video, Zap, type LucideIcon,
} from 'lucide-react'
import i18n from '../i18n'
import type { Messages } from '../i18n/fr'

export type Kind = 'keys' | 'launch' | 'open' | 'capture'

export type Shortcut = {
  id: string
  label: string
  icon: string
  color: string
  kind: Kind
  keys?: string[]
  command?: string
  target?: string
}

export type Availability = { ok: boolean; reason?: string }
export type ShortcutsState = { shortcuts: Shortcut[]; keys: Availability; capture: Availability }

// Indisponible sur ce PC : la raison, sinon null.
export function unavailable(s: Shortcut, state: ShortcutsState) {
  const a = s.kind === 'keys' ? state.keys : s.kind === 'capture' ? state.capture : null
  return a && !a.ok ? (a.reason ?? i18n.t('shortcuts.unavailableHere')) : null
}

export const ICONS: Record<string, LucideIcon> = {
  camera: Camera,
  video: Video,
  gauge: Gauge,
  folder: Folder,
  monitor: Monitor,
  calculator: Calculator,
  gamepad: Gamepad2,
  globe: Globe,
  terminal: Terminal,
  keyboard: Keyboard,
  music: Music,
  headphones: Headphones,
  mic: Mic,
  message: MessageSquare,
  mail: Mail,
  code: Code,
  search: Search,
  clipboard: Clipboard,
  zap: Zap,
  star: Star,
  rocket: Rocket,
}

export const COLORS: Record<string, string> = {
  blue: '#4f8cff',
  green: '#3fb27f',
  orange: '#e08a3c',
  red: '#d9605c',
  purple: '#9b7bf0',
  teal: '#36b3b3',
  pink: '#e06c9f',
  yellow: '#d9b341',
}

export const iconOf = (name: string) => ICONS[name] ?? Zap
export const colorOf = (name: string) => COLORS[name] ?? COLORS.blue

// Touches de modification, dans l'ordre de la combinaison.
export const MODIFIERS = ['ctrl', 'alt', 'shift', 'win']

// Touches spéciales : leur nom affiché est traduit (clés « keys.* »).
type Translated = keyof Messages['keys']
const SPECIAL_KEYS: Translated[] = [
  'esc', 'enter', 'tab', 'space', 'backspace', 'delete', 'insert', 'printscreen',
  'up', 'down', 'left', 'right', 'home', 'end', 'pageup', 'pagedown',
]
const translated = (id: string): id is Translated => id === 'shift' || (SPECIAL_KEYS as string[]).includes(id)

// Touches principales proposées, dans l'ordre de la liste déroulante.
export const KEYS: string[] = [
  ...'abcdefghijklmnopqrstuvwxyz0123456789'.split(''),
  ...Array.from({ length: 12 }, (_, i) => `f${i + 1}`),
  ...SPECIAL_KEYS,
]

const FIXED: Record<string, string> = { ctrl: 'Ctrl', alt: 'Alt', win: 'Win' }

// Nom affiché d'une touche : « Ctrl », « Maj » / « Shift », « F5 », « A ».
export function keyLabel(id: string) {
  if (FIXED[id]) return FIXED[id]
  if (translated(id)) return i18n.t(`keys.${id}`)
  return id.toUpperCase()
}

// Texte lisible d'une combinaison : « Ctrl + Maj + S ».
export function comboLabel(keys: string[] = []) {
  return keys.map(keyLabel).join(' + ')
}

// Ce que fait le raccourci, en une ligne.
export function describe(s: Shortcut) {
  if (s.kind === 'keys') return comboLabel(s.keys)
  if (s.kind === 'capture') return i18n.t('shortcuts.capturedDetail')
  if (s.kind === 'launch') return s.command ?? ''
  return s.target ?? ''
}
