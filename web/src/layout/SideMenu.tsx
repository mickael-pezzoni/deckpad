import { useEffect, useState } from 'react'
import { ArrowLeftRight, Menu } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import type { Page } from '../pages'
import { ThemeToggle } from '../components/ThemeToggle'

type Props = {
  pages: Page[]
  current: number
  pcName: string
  onSelect: (index: number) => void
  onSwitchPc: () => void
}

// Menu du téléphone : en bas, seulement la page affichée ; un tap ouvre le menu
// latéral avec toutes les pages, le changement de PC et le thème.
export function SideMenu({ pages, current, pcName, onSelect, onSwitchPc }: Props) {
  const { t } = useTranslation()
  const [open, setOpen] = useState(false)
  const { id, icon: Icon } = pages[current]

  // Bouton retour d'Android : ferme le menu au lieu de quitter l'appli.
  useEffect(() => {
    if (!open) return
    const onKey = (e: KeyboardEvent) => e.key === 'Escape' && setOpen(false)
    window.addEventListener('keydown', onKey)
    return () => window.removeEventListener('keydown', onKey)
  }, [open])

  return (
    <>
      <button type="button" className="side-current" aria-expanded={open} onClick={() => setOpen(true)}>
        <Icon size={24} strokeWidth={2} aria-hidden />
        <span>{t(`pages.${id}`)}</span>
        <Menu size={22} strokeWidth={2} aria-hidden className="side-current-menu" />
      </button>

      <div className={`sidebar-scrim${open ? ' is-open' : ''}`} onClick={() => setOpen(false)} aria-hidden />
      <aside className={`sidebar${open ? ' is-open' : ''}`} aria-label={t('common.pagesNav')} inert={!open}>
        <button
          type="button"
          className="sidebar-pc"
          onClick={() => {
            setOpen(false)
            onSwitchPc()
          }}
        >
          <span className="sidebar-pc-name">{pcName}</span>
          <span className="sidebar-pc-switch">
            <ArrowLeftRight size={20} strokeWidth={2} aria-hidden />
            {t('pcs.switch')}
          </span>
        </button>
        <nav className="sidebar-pages">
          {pages.map(({ id, icon: PageIcon }, i) => (
            <button
              key={id}
              type="button"
              className={i === current ? 'sidebar-item active' : 'sidebar-item'}
              aria-current={i === current ? 'page' : undefined}
              onClick={() => {
                setOpen(false)
                onSelect(i)
              }}
            >
              <PageIcon size={24} strokeWidth={2} aria-hidden />
              {t(`pages.${id}`)}
            </button>
          ))}
        </nav>
        <div className="sidebar-foot">
          <ThemeToggle />
        </div>
      </aside>
    </>
  )
}
