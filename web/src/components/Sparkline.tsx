import { useId } from 'react'

// Seuils en valeur de la courbe (ex. %) : au-dessus, la courbe passe en orange puis en rouge.
export type Thresholds = { warning: number; critical: number }

type Props = { values: number[]; max?: number; thresholds?: Thresholds }

const W = 200
const H = 48

// Mini-courbe sur les 60 dernières secondes, échelle 0 → max.
export function Sparkline({ values, max = 100, thresholds }: Props) {
  const id = `spark${useId().replace(/[^\w-]/g, '')}`
  if (values.length < 2) return <svg className="sparkline" viewBox={`0 0 ${W} ${H}`} aria-hidden />
  const y = (v: number) => H - 1 - (Math.min(v, max) / max) * (H - 2)
  const step = W / 59
  const x0 = W - (values.length - 1) * step
  const pts = values.map((v, i) => [x0 + i * step, y(v)])
  const line = pts.map(([x, py]) => `${x.toFixed(1)},${py.toFixed(1)}`).join(' ')
  const area = `${x0},${H} ${line} ${W},${H}`
  // Dégradé vertical à paliers nets : seule la partie de la courbe au-dessus d'un seuil change de couleur.
  const paint = thresholds ? `url(#${id})` : undefined
  return (
    <svg className="sparkline" viewBox={`0 0 ${W} ${H}`} preserveAspectRatio="none" aria-hidden>
      {thresholds && (
        <defs>
          <linearGradient id={id} gradientUnits="userSpaceOnUse" x1="0" y1="0" x2="0" y2={H}>
            <stop offset={y(thresholds.critical) / H} stopColor="var(--critical)" />
            <stop offset={y(thresholds.critical) / H} stopColor="var(--warning)" />
            <stop offset={y(thresholds.warning) / H} stopColor="var(--warning)" />
            <stop offset={y(thresholds.warning) / H} stopColor="var(--accent)" />
          </linearGradient>
        </defs>
      )}
      <polygon points={area} className="sparkline-area" style={paint ? { fill: paint } : undefined} />
      <polyline points={line} className="sparkline-line" style={paint ? { stroke: paint } : undefined} />
    </svg>
  )
}
