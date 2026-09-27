import type { ReactNode } from 'react'
import { PageActiveContext } from './pageActive'

type Props = { title: string; active: boolean; children: ReactNode }

// Structure commune d'une page : le contenu remplit tout l'écran au-dessus du menu,
// qui affiche déjà le nom de la page (pas de titre ici).
export function PageShell({ title, active, children }: Props) {
  return (
    <section className="page" aria-label={title}>
      <PageActiveContext.Provider value={active}>
        <div className="page-body">{children}</div>
      </PageActiveContext.Provider>
    </section>
  )
}
