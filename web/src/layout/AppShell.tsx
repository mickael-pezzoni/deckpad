import type { ReactNode } from 'react'
import type { Page } from '../pages'
import { NavMenu } from './NavMenu'
import { SideMenu } from './SideMenu'
import { ThemeToggle } from '../components/ThemeToggle'
import { useMediaQuery } from '../hooks/useMediaQuery'

// Même seuil que la CSS « téléphone ».
const PHONE = '(max-width: 560px)'

type Props = {
  pages: Page[]
  current: number
  onNavigate: (index: number) => void
  pcName: string
  onSwitchPc: () => void // retour à la liste des PC
  children: ReactNode
}

// Cadre de l'app : les pages occupent tout l'espace, le menu (et le bouton de thème) reste fixe en bas.
// Sur téléphone, la barre ne montre que la page affichée et ouvre un menu latéral.
export function AppShell({ pages, current, onNavigate, pcName, onSwitchPc, children }: Props) {
  const phone = useMediaQuery(PHONE)
  return (
    <div className="shell">
      <main className="shell-content">{children}</main>
      <div className="shell-bar">
        {phone ? (
          <SideMenu pages={pages} current={current} pcName={pcName} onSelect={onNavigate} onSwitchPc={onSwitchPc} />
        ) : (
          <>
            <NavMenu pages={pages} current={current} onSelect={onNavigate} onSwitchPc={onSwitchPc} />
            <ThemeToggle />
          </>
        )}
      </div>
    </div>
  )
}
