// Derniers envois vers le PC, gardés sur la tablette seulement.

export type SendAction = 'copy' | 'open' | 'type'
export type HistoryItem = { id: string; action: SendAction; text: string }

const KEY = 'deckpad.clipboard.history'
const MAX = 10

export function loadHistory(): HistoryItem[] {
  try {
    const list = JSON.parse(localStorage.getItem(KEY) ?? '[]')
    return Array.isArray(list) ? list.slice(0, MAX) : []
  } catch {
    return []
  }
}

function save(list: HistoryItem[]) {
  try {
    localStorage.setItem(KEY, JSON.stringify(list))
  } catch {
    // stockage indisponible (navigation privée) : l'historique dure le temps de la page
  }
}

// Ajoute un envoi en tête ; le même texte envoyé de la même façon remonte au lieu d'être doublé.
export function pushHistory(list: HistoryItem[], action: SendAction, text: string): HistoryItem[] {
  const rest = list.filter((h) => !(h.action === action && h.text === text))
  const next = [{ id: `${Date.now()}-${Math.random().toString(36).slice(2, 7)}`, action, text }, ...rest].slice(0, MAX)
  save(next)
  return next
}

export function removeHistory(list: HistoryItem[], id: string): HistoryItem[] {
  const next = list.filter((h) => h.id !== id)
  save(next)
  return next
}
