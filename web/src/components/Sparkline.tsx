type Props = { values: number[]; max?: number }

const W = 200
const H = 48

// Mini-courbe sur les 60 dernières secondes, échelle 0 → max.
export function Sparkline({ values, max = 100 }: Props) {
  if (values.length < 2) return <svg className="sparkline" viewBox={`0 0 ${W} ${H}`} aria-hidden />
  const step = W / 59
  const x0 = W - (values.length - 1) * step
  const pts = values.map((v, i) => [x0 + i * step, H - 1 - (Math.min(v, max) / max) * (H - 2)])
  const line = pts.map(([x, y]) => `${x.toFixed(1)},${y.toFixed(1)}`).join(' ')
  const area = `${x0},${H} ${line} ${W},${H}`
  return (
    <svg className="sparkline" viewBox={`0 0 ${W} ${H}`} preserveAspectRatio="none" aria-hidden>
      <polygon points={area} className="sparkline-area" />
      <polyline points={line} className="sparkline-line" />
    </svg>
  )
}
