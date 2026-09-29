import type { ReactNode } from 'react'
import type { Page } from '../pages'
import { NavMenu } from './NavMenu'
import { ThemeToggle } from '../components/ThemeToggle'

type Props = {
  pages: Page[]
  current: number
  onNavigate: (index: number) => void
  pcSwitch: ReactNode // nom du PC piloté, pour en changer
  children: ReactNode
}

// Cadre de l'app : les pages occupent tout l'espace, le menu (avec le PC piloté et le
// bouton de thème) reste fixe en bas.
export function AppShell({ pages, current, onNavigate, pcSwitch, children }: Props) {
  return (
    <div className="shell">
      <main className="shell-content">{children}</main>
      <div className="shell-bar">
        {pcSwitch}
        <NavMenu pages={pages} current={current} onSelect={onNavigate} />
        <ThemeToggle />
      </div>
    </div>
  )
}
