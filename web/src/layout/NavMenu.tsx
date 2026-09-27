import type { Page } from '../pages'

type Props = {
  pages: Page[]
  current: number
  onSelect: (index: number) => void
}

// Menu horizontal d'icônes : montre la page affichée et permet d'y sauter d'un tap.
export function NavMenu({ pages, current, onSelect }: Props) {
  return (
    <nav className="nav-menu" aria-label="Pages">
      {pages.map(({ id, title, icon: Icon }, i) => (
        <button
          key={id}
          type="button"
          className={i === current ? 'nav-item active' : 'nav-item'}
          aria-label={title}
          aria-current={i === current ? 'page' : undefined}
          onClick={() => onSelect(i)}
        >
          <Icon size={24} strokeWidth={2} aria-hidden />
          {i === current && <span className="nav-label">{title}</span>}
        </button>
      ))}
    </nav>
  )
}
