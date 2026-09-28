import { ClipboardCopy, ExternalLink, Keyboard } from 'lucide-react'
import { useLongPress } from '../hooks/useLongPress'
import type { HistoryItem, SendAction } from '../clipboard/history'

export const ACTION_ICONS: Record<SendAction, typeof ClipboardCopy> = {
  copy: ClipboardCopy,
  open: ExternalLink,
  type: Keyboard,
}

type Props = { item: HistoryItem; onSend: () => void; onRemove: () => void }

// Un envoi passé : un appui le renvoie de la même façon, un appui long l'efface.
export function HistoryChip({ item, onSend, onRemove }: Props) {
  const press = useLongPress(onSend, onRemove)
  const Icon = ACTION_ICONS[item.action]
  return (
    <button type="button" className="history-chip" {...press}>
      <Icon size={18} strokeWidth={2} aria-hidden />
      <span>{item.text.replace(/\s+/g, ' ')}</span>
    </button>
  )
}
