import { useTranslation } from 'react-i18next'
import type { Page } from '../pages'

type Props = {
  pages: Page[]
  current: number
  onSelect: (index: number) => void
}

// Menu horizontal d'icônes : montre la page affichée et permet d'y sauter d'un tap.
export function NavMenu({ pages, current, onSelect }: Props) {
  const { t } = useTranslation()
  return (
    <nav className="nav-menu" aria-label={t('common.pagesNav')}>
      {pages.map(({ id, icon: Icon }, i) => (
        <button
          key={id}
          type="button"
          className={i === current ? 'nav-item active' : 'nav-item'}
          aria-label={t(`pages.${id}`)}
          aria-current={i === current ? 'page' : undefined}
          onClick={() => onSelect(i)}
        >
          <Icon size={24} strokeWidth={2} aria-hidden />
          {i === current && <span className="nav-label">{t(`pages.${id}`)}</span>}
        </button>
      ))}
    </nav>
  )
}
