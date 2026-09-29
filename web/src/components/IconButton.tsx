import type { LucideIcon } from 'lucide-react'

type Props = {
  icon: LucideIcon
  label: string // lu par les lecteurs d'écran, l'icône seule s'affiche
  onClick: () => void
}

// Bouton carré avec une seule icône (retour, accueil…).
export function IconButton({ icon: Icon, label, onClick }: Props) {
  return (
    <button type="button" className="icon-button" onClick={onClick} aria-label={label} title={label}>
      <Icon size={28} aria-hidden />
    </button>
  )
}
