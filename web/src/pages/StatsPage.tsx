import { Cpu, Gauge, Gpu, MemoryStick, type LucideIcon } from 'lucide-react'
import { Tile, type Tone } from '../components/Tile'
import { Sparkline, type Thresholds } from '../components/Sparkline'
import { UsageBar } from '../components/UsageBar'
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

// Prototypes de mise en page, choisis via ?stats=a|b|c (sinon la page actuelle).
const variant = new URLSearchParams(location.search).get('stats')

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

  if (variant === 'a') return <Gauges metrics={metrics} fps={latest.fps} />
  if (variant === 'b') return <Bars metrics={metrics} fps={latest.fps} />
  if (variant === 'c') return <Rows metrics={metrics} fps={latest.fps} />

  return (
    <div className="grid grid-large">
      <Tile
        label="CPU"
        icon={Cpu}
        badge={level(latest.cpu, LOAD)}
        chart={<Sparkline values={history.map((s) => s.cpu)} thresholds={LOAD} />}
      >
        {Math.round(latest.cpu)} %
      </Tile>
      <Tile
        label="RAM"
        icon={MemoryStick}
        detail={`${gb(latest.ramUsed)} / ${gb(latest.ramTotal)} Go`}
        badge={level(ramPct(latest), RAM)}
        chart={<Sparkline values={history.map(ramPct)} thresholds={RAM} />}
      >
        {Math.round(ramPct(latest))} %
      </Tile>
      <Tile
        label="GPU"
        icon={Gpu}
        detail={gpu ? gpuDetail(gpu) : 'Non détecté'}
        badge={gpu ? level(gpu.usage, LOAD) : undefined}
        chart={gpu && <Sparkline values={history.map((s) => s.gpu?.usage ?? 0)} thresholds={LOAD} />}
      >
        {gpu ? `${Math.round(gpu.usage)} %` : '—'}
      </Tile>
      <Tile label="FPS" icon={Gauge} detail={latest.fps == null ? 'Bientôt disponible' : undefined}>
        {latest.fps ?? '—'}
      </Tile>
      {!connected && <p className="coming-soon">Connexion perdue, reconnexion…</p>}
    </div>
  )
}

// A : quatre jauges rondes côte à côte (2 × 2 en portrait).
function Gauges({ metrics, fps }: { metrics: Metric[]; fps: number | null }) {
  return (
    <div className="stats-gauges">
      {metrics.map((m) => {
        const badge = m.value == null ? undefined : level(m.value, m.thresholds)
        return (
          <div key={m.label} className={badge ? `tile stats-gauge tile-${badge.tone}` : 'tile stats-gauge'}>
            <Head m={m} badge={badge} />
            <Ring value={m.value} tone={badge?.tone} />
            <span className="tile-detail">{m.detail ?? ' '}</span>
          </div>
        )
      })}
      <div className="tile stats-gauge">
        <span className="tile-head">
          <Label icon={Gauge} label="FPS" />
        </span>
        <div className="stats-ring stats-ring-empty">
          <span className="stats-ring-value">{fps ?? '—'}</span>
        </div>
        <span className="tile-detail">{fps == null ? 'Bientôt disponible' : ' '}</span>
      </div>
    </div>
  )
}

function Ring({ value, tone }: { value: number | null; tone?: Tone }) {
  const r = 44
  const c = 2 * Math.PI * r
  const pct = Math.min(100, Math.max(0, value ?? 0))
  return (
    <div className={tone ? `stats-ring stats-ring-${tone}` : 'stats-ring'}>
      <svg viewBox="0 0 100 100" aria-hidden>
        <circle cx="50" cy="50" r={r} className="stats-ring-track" />
        <circle
          cx="50"
          cy="50"
          r={r}
          className="stats-ring-fill"
          strokeDasharray={`${(pct / 100) * c} ${c}`}
          transform="rotate(-90 50 50)"
        />
      </svg>
      <span className="stats-ring-value">{value == null ? '—' : `${Math.round(value)} %`}</span>
    </div>
  )
}

// B : une fiche avec une barre par mesure, FPS en grand à côté.
function Bars({ metrics, fps }: { metrics: Metric[]; fps: number | null }) {
  return (
    <div className="stats-bars-page">
      <div className="tile stats-bars">
        {metrics.map((m) => {
          const badge = m.value == null ? undefined : level(m.value, m.thresholds)
          return (
            <div key={m.label} className={badge ? `stats-bar stats-${badge.tone}` : 'stats-bar'}>
              <div className="stats-bar-head">
                <Label icon={m.icon} label={m.label} />
                {badge && <span className={`badge badge-${badge.tone}`}>{badge.text}</span>}
                <span className="stats-bar-value">{m.value == null ? '—' : `${Math.round(m.value)} %`}</span>
              </div>
              <UsageBar percent={m.value ?? 0} />
              {m.detail && <span className="tile-detail">{m.detail}</span>}
            </div>
          )
        })}
      </div>
      <div className="tile stats-fps">
        <span className="tile-head">
          <Label icon={Gauge} label="FPS" />
        </span>
        <span className={fps == null ? 'stats-fps-value stats-fps-empty' : 'stats-fps-value'}>{fps ?? '—'}</span>
        <span className="tile-detail">{fps == null ? 'Bientôt disponible' : 'images / seconde'}</span>
      </div>
    </div>
  )
}

// C : une ligne par mesure, chiffre à gauche et grande courbe sur toute la largeur.
function Rows({ metrics, fps }: { metrics: Metric[]; fps: number | null }) {
  return (
    <div className="stats-rows">
      {metrics.map((m) => {
        const badge = m.value == null ? undefined : level(m.value, m.thresholds)
        return (
          <div key={m.label} className={badge ? `tile stats-row tile-${badge.tone}` : 'tile stats-row'}>
            <div className="stats-row-info">
              <Head m={m} badge={badge} />
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
        <div className="stats-row-chart stats-row-soon">{fps == null && 'Bientôt disponible'}</div>
      </div>
    </div>
  )
}

function Head({ m, badge }: { m: Metric; badge?: { text: string; tone: Tone } }) {
  return (
    <span className="tile-head">
      <Label icon={m.icon} label={m.label} />
      {badge && <span className={`badge badge-${badge.tone}`}>{badge.text}</span>}
    </span>
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
