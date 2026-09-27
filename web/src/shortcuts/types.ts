import {
  Calculator, Camera, Clipboard, Code, Folder, Gamepad2, Gauge, Globe, Headphones, Keyboard, Mail,
  MessageSquare, Mic, Monitor, Music, Rocket, Search, Star, Terminal, Video, Zap, type LucideIcon,
} from 'lucide-react'

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
  return a && !a.ok ? (a.reason ?? 'Indisponible sur ce PC') : null
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

export const MODIFIERS = [
  { id: 'ctrl', label: 'Ctrl' },
  { id: 'alt', label: 'Alt' },
  { id: 'shift', label: 'Maj' },
  { id: 'win', label: 'Win' },
]

const SPECIAL_KEYS: [string, string][] = [
  ['esc', 'Échap'],
  ['enter', 'Entrée'],
  ['tab', 'Tab'],
  ['space', 'Espace'],
  ['backspace', 'Retour arrière'],
  ['delete', 'Suppr'],
  ['insert', 'Inser'],
  ['printscreen', 'Impr. écran'],
  ['up', 'Flèche haut'],
  ['down', 'Flèche bas'],
  ['left', 'Flèche gauche'],
  ['right', 'Flèche droite'],
  ['home', 'Début'],
  ['end', 'Fin'],
  ['pageup', 'Page préc.'],
  ['pagedown', 'Page suiv.'],
]

// Touches principales proposées, dans l'ordre de la liste déroulante.
export const KEYS: [string, string][] = [
  ...'abcdefghijklmnopqrstuvwxyz0123456789'.split('').map((c): [string, string] => [c, c.toUpperCase()]),
  ...Array.from({ length: 12 }, (_, i): [string, string] => [`f${i + 1}`, `F${i + 1}`]),
  ...SPECIAL_KEYS,
]

const keyLabels = new Map<string, string>([...MODIFIERS.map((m): [string, string] => [m.id, m.label]), ...KEYS])

// Texte lisible d'une combinaison : « Ctrl + Maj + S ».
export function comboLabel(keys: string[] = []) {
  return keys.map((k) => keyLabels.get(k) ?? k).join(' + ')
}

// Ce que fait le raccourci, en une ligne.
export function describe(s: Shortcut) {
  if (s.kind === 'keys') return comboLabel(s.keys)
  if (s.kind === 'capture') return 'Copiée dans le presse-papiers'
  if (s.kind === 'launch') return s.command ?? ''
  return s.target ?? ''
}
