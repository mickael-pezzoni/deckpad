import type { ReactNode } from 'react'

// Structure commune d'une page : le contenu remplit tout l'écran au-dessus du menu,
// qui affiche déjà le nom de la page (pas de titre ici).
export function PageShell({ title, children }: { title: string; children: ReactNode }) {
  return (
    <section className="page" aria-label={title}>
      <div className="page-body">{children}</div>
    </section>
  )
}
