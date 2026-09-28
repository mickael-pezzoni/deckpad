import { useEffect, useState, type ReactNode } from 'react'
import { ArrowDown, ArrowUp, EthernetPort, Globe, House, Timer, Wifi, type LucideIcon } from 'lucide-react'
import { Tile } from '../components/Tile'
import { Sparkline } from '../components/Sparkline'
import { useEventStream } from '../hooks/useEventStream'
import { Loader } from '../components/Loader'

type Net = {
  interface: string
  kind: 'wifi' | 'ethernet' | ''
  localIP: string
  rxRate: number
  txRate: number
  latencyMs: number | null
}

const HISTORY = 60

// PROTOTYPE : ?net=a|b|c pour comparer les maquettes.
const variant = new URLSearchParams(location.search).get('net') ?? 'a'

export function NetworkPage() {
  const [history, setHistory] = useState<Net[]>([])
  const [publicIP, setPublicIP] = useState<string | null>(null)
  const connected = useEventStream<Net>('/api/network/stream', (n) =>
    setHistory((h) => [...h.slice(-(HISTORY - 1)), n]),
    { keepAlive: true }, // garde les courbes complètes
  )

  useEffect(() => {
    fetch('/api/network/public-ip')
      .then((r) => (r.ok ? r.json() : Promise.reject()))
      .then((d) => setPublicIP(d.ip))
      .catch(() => setPublicIP('Indisponible'))
  }, [])

  const net = history.at(-1)
  if (!net) return <Loader label={connected ? 'Chargement…' : 'Connexion au PC…'} />

  // Même échelle pour les deux courbes, avec un plancher pour ne pas amplifier le bruit.
  const max = Math.max(128 * 1024, ...history.flatMap((n) => [n.rxRate, n.txRate]))
  const rx = history.map((n) => n.rxRate)
  const tx = history.map((n) => n.txRate)
  // B et C : chaque courbe a sa propre échelle.
  const maxRx = Math.max(128 * 1024, ...rx)
  const maxTx = Math.max(128 * 1024, ...tx)
  const KindIcon = net.kind === 'wifi' ? Wifi : EthernetPort
  const kind = net.kind === 'wifi' ? 'Wi-Fi' : net.kind === 'ethernet' ? 'Câble' : '—'
  const latency = net.latencyMs == null ? '—' : `${net.latencyMs} ms`

  if (variant === 'a') {
    return (
      <div className="net-a">
        <div className="tile net-a-rates">
          <div className="net-a-values">
            <BigRate icon={ArrowDown} label="Réception" value={rate(net.rxRate)} />
            <BigRate icon={ArrowUp} label="Envoi" value={rate(net.txRate)} up />
          </div>
          <div className="net-chart">
            <Sparkline values={rx} max={max} />
            <span className="net-up">
              <Sparkline values={tx} max={max} />
            </span>
          </div>
        </div>
        <List
          rows={[
            { icon: KindIcon, label: 'Connexion', value: kind },
            { icon: Timer, label: 'Latence', value: latency },
            { icon: House, label: 'IP locale', value: net.localIP || '—' },
            { icon: Globe, label: 'IP publique', value: publicIP ?? '…' },
          ]}
        />
      </div>
    )
  }

  if (variant === 'b') {
    return (
      <div className="net-b">
        <div className="tile net-b-rate">
          <Label icon={ArrowDown}>Réception</Label>
          <span className="net-big">{rate(net.rxRate)}</span>
          <div className="net-chart">
            <Sparkline values={rx} max={maxRx} />
          </div>
        </div>
        <div className="tile net-b-rate net-up">
          <Label icon={ArrowUp}>Envoi</Label>
          <span className="net-big">{rate(net.txRate)}</span>
          <div className="net-chart">
            <Sparkline values={tx} max={maxTx} />
          </div>
        </div>
        <div className="tile net-b-strip">
          {[
            { icon: KindIcon, label: 'Connexion', value: kind },
            { icon: Timer, label: 'Latence', value: latency },
            { icon: House, label: 'IP locale', value: net.localIP || '—' },
            { icon: Globe, label: 'IP publique', value: publicIP ?? '…' },
          ].map((r) => (
            <div className="net-b-item" key={r.label}>
              <Label icon={r.icon}>{r.label}</Label>
              <span className="info-value">{r.value}</span>
            </div>
          ))}
        </div>
      </div>
    )
  }

  const quality = net.latencyMs == null ? 'Hors ligne' : net.latencyMs < 40 ? 'Excellente' : net.latencyMs < 100 ? 'Correcte' : 'Lente'
  return (
    <div className="net-c">
      <div className="tile net-c-hero">
        <span className="net-c-icon">
          <KindIcon size={64} strokeWidth={1.75} aria-hidden />
        </span>
        <span className="net-c-kind">{kind}</span>
        <span className="net-c-quality">
          Connexion {quality.toLowerCase()} · {latency}
        </span>
        <div className="net-c-ips">
          <span>
            <House size={18} aria-hidden /> {net.localIP || '—'}
          </span>
          <span>
            <Globe size={18} aria-hidden /> {publicIP ?? '…'}
          </span>
        </div>
      </div>
      <Tile label="Réception" icon={ArrowDown} chart={<Sparkline values={rx} max={maxRx} />}>
        {rate(net.rxRate)}
      </Tile>
      <Tile label="Envoi" icon={ArrowUp} chart={<Sparkline values={tx} max={maxTx} />}>
        {rate(net.txRate)}
      </Tile>
    </div>
  )
}

function Label({ icon: Icon, children }: { icon: LucideIcon; children: ReactNode }) {
  return (
    <span className="tile-label">
      <span className="tile-icon">
        <Icon size={20} strokeWidth={2} aria-hidden />
      </span>
      {children}
    </span>
  )
}

function BigRate({ icon, label, value, up }: { icon: LucideIcon; label: string; value: string; up?: boolean }) {
  return (
    <div className={up ? 'net-a-value net-up' : 'net-a-value'}>
      <Label icon={icon}>{label}</Label>
      <span className="net-big">{value}</span>
    </div>
  )
}

function List({ rows }: { rows: { icon: LucideIcon; label: string; value: string }[] }) {
  return (
    <div className="tile info-list">
      {rows.map(({ icon, label, value }) => (
        <div className="info-row" key={label}>
          <Label icon={icon}>{label}</Label>
          <span className="info-value">{value}</span>
        </div>
      ))}
    </div>
  )
}

// Débit lisible en Ko/s ou Mo/s.
function rate(bytesPerSec: number) {
  if (bytesPerSec < 1024 * 1024) return `${Math.round(bytesPerSec / 1024)} Ko/s`
  return `${(bytesPerSec / 1024 ** 2).toFixed(1).replace('.', ',')} Mo/s`
}
