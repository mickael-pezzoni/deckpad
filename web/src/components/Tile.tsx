import type { ReactNode } from 'react'

type Props = {
  label: string
  children: ReactNode
  wide?: boolean
}

export function Tile({ label, children, wide }: Props) {
  return (
    <div className={wide ? 'tile tile-wide' : 'tile'}>
      <span className="tile-label">{label}</span>
      <span className="tile-value">{children}</span>
    </div>
  )
}
