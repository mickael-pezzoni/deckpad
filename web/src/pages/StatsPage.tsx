import { Tile } from '../components/Tile'
import { Sparkline } from '../components/Sparkline'
import { useStatsStream, type Stats } from '../hooks/useStatsStream'

export function StatsPage() {
  const { latest, history, connected } = useStatsStream()

  if (!latest) return <p className="coming-soon">{connected ? 'Chargement…' : 'Connexion au PC…'}</p>

  const ramPct = (s: typeof latest) => (s.ramUsed / s.ramTotal) * 100
  const gpu = latest.gpu

  return (
    <div className="grid grid-large">
      <Tile
        label="CPU"
        chart={<Sparkline values={history.map((s) => s.cpu)} />}
      >
        {Math.round(latest.cpu)} %
      </Tile>
      <Tile
        label="RAM"
        detail={`${gb(latest.ramUsed)} / ${gb(latest.ramTotal)} Go`}
        chart={<Sparkline values={history.map(ramPct)} />}
      >
        {Math.round(ramPct(latest))} %
      </Tile>
      <Tile
        label="GPU"
        detail={gpu ? gpuDetail(gpu) : 'Non détecté'}
        chart={gpu && <Sparkline values={history.map((s) => s.gpu?.usage ?? 0)} />}
      >
        {gpu ? `${Math.round(gpu.usage)} %` : '—'}
      </Tile>
      <Tile label="FPS" detail={latest.fps == null ? 'Bientôt disponible' : undefined}>
        {latest.fps ?? '—'}
      </Tile>
      {!connected && <p className="coming-soon">Connexion perdue, reconnexion…</p>}
    </div>
  )
}

function gpuDetail(gpu: NonNullable<Stats['gpu']>) {
  const vram = gpu.memTotal > 0 ? `${gb(gpu.memUsed)} / ${gb(gpu.memTotal)} Go` : `${gb(gpu.memUsed)} Go`
  return gpu.temp == null ? vram : `${Math.round(gpu.temp)} °C · ${vram}`
}

function gb(bytes: number) {
  return (bytes / 1024 ** 3).toFixed(1).replace('.', ',')
}
