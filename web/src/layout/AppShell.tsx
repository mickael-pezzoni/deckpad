import type { ReactNode } from 'react'
import type { Page } from '../pages'
import { NavMenu } from './NavMenu'

type Props = {
  pages: Page[]
  current: number
  onNavigate: (index: number) => void
  children: ReactNode
}

// Cadre de l'app : les pages occupent tout l'espace, le menu reste fixe en bas.
export function AppShell({ pages, current, onNavigate, children }: Props) {
  return (
    <div className="shell">
      <main className="shell-content">{children}</main>
      <NavMenu pages={pages} current={current} onSelect={onNavigate} />
    </div>
  )
}
