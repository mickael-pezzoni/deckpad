// Barre de remplissage (ex. espace disque occupé), rouge quand c'est presque plein.
export function UsageBar({ percent, critical }: { percent: number; critical?: boolean }) {
  return (
    <span className={critical ? 'usage-bar usage-bar-critical' : 'usage-bar'} aria-hidden>
      <span style={{ width: `${Math.min(100, Math.max(0, percent))}%` }} />
    </span>
  )
}
