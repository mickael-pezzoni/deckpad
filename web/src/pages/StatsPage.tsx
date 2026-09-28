import { Cpu, Gauge, Gpu, MemoryStick, type LucideIcon } from 'lucide-react'
import type { Tone } from '../components/Tile'
import { Sparkline, type Thresholds } from '../components/Sparkline'
import { useStatsStream, type Stats } from '../hooks/useStatsStream'
import { Loader } from '../components/Loader'

type Metric = {
  label: string
  icon: LucideIcon
  value: number | null // en %, null si non détecté
  detail?: string
  thresholds: Thresholds
  series: number[]
}

// Une ligne par mesure : le chiffre à gauche, la courbe de la dernière minute sur toute la largeur.
export function StatsPage() {
  const { latest, history, connected } = useStatsStream()

  if (!latest) return <Loader label={connected ? 'Chargement…' : 'Connexion au PC…'} />

  const ramPct = (s: Stats) => (s.ramUsed / s.ramTotal) * 100
  const gpu = latest.gpu
  const metrics: Metric[] = [
    { label: 'CPU', icon: Cpu, value: latest.cpu, thresholds: LOAD, series: history.map((s) => s.cpu) },
    {
      label: 'RAM',
      icon: MemoryStick,
      value: ramPct(latest),
      detail: `${gb(latest.ramUsed)} / ${gb(latest.ramTotal)} Go`,
      thresholds: RAM,
      series: history.map(ramPct),
    },
    {
      label: 'GPU',
      icon: Gpu,
      value: gpu ? gpu.usage : null,
      detail: gpu ? gpuDetail(gpu) : 'Non détecté',
      thresholds: LOAD,
      series: history.map((s) => s.gpu?.usage ?? 0),
    },
  ]
  const fps = latest.fps

  return (
    <div className="stats-rows">
      {metrics.map((m) => {
        const badge = m.value == null ? undefined : level(m.value, m.thresholds)
        return (
          <div key={m.label} className={badge ? `tile stats-row tile-${badge.tone}` : 'tile stats-row'}>
            <div className="stats-row-info">
              <span className="tile-head">
                <Label icon={m.icon} label={m.label} />
                {badge && <span className={`badge badge-${badge.tone}`}>{badge.text}</span>}
              </span>
              <span className="stats-row-value">{m.value == null ? '—' : `${Math.round(m.value)} %`}</span>
              {m.detail && <span className="tile-detail">{m.detail}</span>}
            </div>
            <div className="stats-row-chart">
              {m.value != null && <Sparkline values={m.series} thresholds={m.thresholds} />}
            </div>
          </div>
        )
      })}
      <div className="tile stats-row">
        <div className="stats-row-info">
          <span className="tile-head">
            <Label icon={Gauge} label="FPS" />
          </span>
          <span className="stats-row-value">{fps ?? '—'}</span>
        </div>
        {/* Place réservée à la courbe des FPS, pas encore mesurés. */}
        <div className="stats-row-chart stats-row-soon">{fps == null && 'Bientôt disponible'}</div>
      </div>
      {!connected && <p className="stats-lost">Connexion perdue, reconnexion…</p>}
    </div>
  )
}

function Label({ icon: Icon, label }: { icon: LucideIcon; label: string }) {
  return (
    <span className="tile-label">
      <span className="tile-icon">
        <Icon size={20} strokeWidth={2} aria-hidden />
      </span>
      {label}
    </span>
  )
}

// Seuils en % : couleur (fond teinté, chiffre, courbe) + badge texte, jamais la couleur seule.
const LOAD: Thresholds = { warning: 70, critical: 90 }
const RAM: Thresholds = { warning: 80, critical: 90 }

function level(value: number, t: Thresholds): { text: string; tone: Tone } | undefined {
  if (value >= t.critical) return { text: 'Très élevé', tone: 'critical' }
  if (value >= t.warning) return { text: 'Élevé', tone: 'warning' }
  return undefined
}

function gpuDetail(gpu: NonNullable<Stats['gpu']>) {
  const vram = gpu.memTotal > 0 ? `${gb(gpu.memUsed)} / ${gb(gpu.memTotal)} Go` : `${gb(gpu.memUsed)} Go`
  return gpu.temp == null ? vram : `${Math.round(gpu.temp)} °C · ${vram}`
}

function gb(bytes: number) {
  return (bytes / 1024 ** 3).toFixed(1).replace('.', ',')
}
