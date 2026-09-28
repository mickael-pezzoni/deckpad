import type { LucideIcon } from 'lucide-react'

export type InfoRow = { icon: LucideIcon; label: string; value: string }

// Fiche en liste : libellé à gauche, valeur à droite (Infos PC, Réseau).
export function InfoList({ rows }: { rows: InfoRow[] }) {
  return (
    <div className="tile info-list">
      {rows.map(({ icon: Icon, label, value }) => (
        <div className="info-row" key={label}>
          <span className="tile-label">
            <span className="tile-icon">
              <Icon size={20} strokeWidth={2} aria-hidden />
            </span>
            {label}
          </span>
          <span className="info-value">{value}</span>
        </div>
      ))}
    </div>
  )
}
