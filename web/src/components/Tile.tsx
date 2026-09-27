import type { ReactNode } from 'react'
import type { LucideIcon } from 'lucide-react'

export type Tone = 'warning' | 'critical'

type Props = {
  label: string
  icon?: LucideIcon
  iconImage?: ReactNode // à la place d'icon, ex. l'icône d'un programme
  children: ReactNode
  detail?: ReactNode
  chart?: ReactNode
  wide?: boolean
  badge?: { text: string; tone: Tone }
  onClick?: () => void
}

export function Tile({ label, icon: Icon, iconImage, children, detail, chart, wide, badge, onClick }: Props) {
  const className = ['tile', wide && 'tile-wide', badge && `tile-${badge.tone}`, onClick && 'tile-button']
    .filter(Boolean)
    .join(' ')
  const content = (
    <>
      <span className="tile-head">
        <span className="tile-label">
          {iconImage ? (
            <span className="tile-icon tile-icon-image">{iconImage}</span>
          ) : (
            Icon && (
              <span className="tile-icon">
                <Icon size={20} strokeWidth={2} aria-hidden />
              </span>
            )
          )}
          {label}
        </span>
        {badge && <span className={`badge badge-${badge.tone}`}>{badge.text}</span>}
      </span>
      <span className="tile-content">
        <span className="tile-value">{children}</span>
        {detail && <span className="tile-detail">{detail}</span>}
      </span>
      {chart}
    </>
  )
  return onClick ? (
    <button type="button" className={className} onClick={onClick}>
      {content}
    </button>
  ) : (
    <div className={className}>{content}</div>
  )
}
