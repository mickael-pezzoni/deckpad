// Chargement d'une page : une mini grille de tuiles qui s'allument en vague
// diagonale (clin d'œil aux tuiles de l'app), centrée sur la page.
export function Loader({ label = 'Chargement…' }: { label?: string }) {
  return (
    <div className="loader" role="status">
      <div className="loader-grid" aria-hidden>
        {Array.from({ length: 9 }, (_, i) => (
          <span key={i} style={{ '--d': (i % 3) + Math.floor(i / 3) } as React.CSSProperties} />
        ))}
      </div>
      <p className="loader-label">{label}</p>
    </div>
  )
}
