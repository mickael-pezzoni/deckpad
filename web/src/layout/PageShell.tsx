import type { ReactNode } from 'react'
import { useTranslation } from 'react-i18next'
import type { PageId } from '../pages'
import { PageActiveContext } from './pageActive'

type Props = { id: PageId; active: boolean; children: ReactNode }

// Structure commune d'une page : le contenu remplit tout l'écran au-dessus du menu,
// qui affiche déjà le nom de la page (pas de titre ici).
export function PageShell({ id, active, children }: Props) {
  const { t } = useTranslation()
  return (
    <section className="page" aria-label={t(`pages.${id}`)}>
      <PageActiveContext.Provider value={active}>
        <div className="page-body">{children}</div>
      </PageActiveContext.Provider>
    </section>
  )
}
