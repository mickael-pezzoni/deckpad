import type { ReactNode } from 'react'

// Structure commune d'une page : titre fixe + contenu qui remplit le reste.
export function PageShell({ title, children }: { title: string; children: ReactNode }) {
  return (
    <section className="page">
      <h1 className="page-title">{title}</h1>
      <div className="page-body">{children}</div>
    </section>
  )
}
