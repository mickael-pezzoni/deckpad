import type { ReactNode } from 'react'

type Props = {
  label: string
  children: ReactNode
  detail?: ReactNode
  chart?: ReactNode
  wide?: boolean
}

export function Tile({ label, children, detail, chart, wide }: Props) {
  return (
    <div className={wide ? 'tile tile-wide' : 'tile'}>
      <span className="tile-label">{label}</span>
      <span className="tile-value">{children}</span>
      {detail && <span className="tile-detail">{detail}</span>}
      {chart}
    </div>
  )
}
