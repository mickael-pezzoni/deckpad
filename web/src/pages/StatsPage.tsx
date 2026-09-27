import { Cpu, Gauge, Gpu, MemoryStick } from 'lucide-react'
import { Tile, type Tone } from '../components/Tile'
import { Sparkline, type Thresholds } from '../components/Sparkline'
import { useStatsStream, type Stats } from '../hooks/useStatsStream'
import { Loader } from '../components/Loader'

export function StatsPage() {
  const { latest, history, connected } = useStatsStream()

  if (!latest) return <Loader label={connected ? 'Chargement…' : 'Connexion au PC…'} />

  const ramPct = (s: typeof latest) => (s.ramUsed / s.ramTotal) * 100
  const gpu = latest.gpu

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
