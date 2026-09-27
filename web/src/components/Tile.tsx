import type { ReactNode } from 'react'

type Props = {
  label: string
  children: ReactNode
  detail?: ReactNode
  chart?: ReactNode
  wide?: boolean
  onClick?: () => void
}

export function Tile({ label, children, detail, chart, wide, onClick }: Props) {
  const className = ['tile', wide && 'tile-wide', onClick && 'tile-button'].filter(Boolean).join(' ')
  const content = (
    <>
      <span className="tile-label">{label}</span>
      <span className="tile-value">{children}</span>
      {detail && <span className="tile-detail">{detail}</span>}
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
