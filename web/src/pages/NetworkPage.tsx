import { useEffect, useState } from 'react'
import { ArrowDown, ArrowUp, EthernetPort, Globe, House, Timer, Wifi, type LucideIcon } from 'lucide-react'
import { Sparkline } from '../components/Sparkline'
import { InfoList } from '../components/InfoList'
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
  const wifi = net.kind === 'wifi'

  // Débits en grand avec leur courbe commune, fiche de la connexion à côté.
  return (
    <div className="net-page">
      <div className="tile net-rates">
        <div className="net-values">
          <Rate icon={ArrowDown} label="Réception" value={net.rxRate} />
          <Rate icon={ArrowUp} label="Envoi" value={net.txRate} upload />
        </div>
        <div className="net-chart">
          <Sparkline values={history.map((n) => n.rxRate)} max={max} />
          <span className="net-upload">
            <Sparkline values={history.map((n) => n.txRate)} max={max} />
          </span>
        </div>
      </div>
      <InfoList
        rows={[
          {
            icon: wifi ? Wifi : EthernetPort,
            label: 'Connexion',
            value: wifi ? 'Wi-Fi' : net.kind === 'ethernet' ? 'Câble' : '—',
          },
          { icon: Timer, label: 'Latence', value: net.latencyMs == null ? '—' : `${net.latencyMs} ms` },
          { icon: House, label: 'IP locale', value: net.localIP || '—' },
          { icon: Globe, label: 'IP publique', value: publicIP ?? '…' },
        ]}
      />
    </div>
  )
}

function Rate({ icon: Icon, label, value, upload }: { icon: LucideIcon; label: string; value: number; upload?: boolean }) {
  return (
    <div className={upload ? 'net-rate net-upload' : 'net-rate'}>
      <span className="tile-label">
        <span className="tile-icon">
          <Icon size={20} strokeWidth={2} aria-hidden />
        </span>
        {label}
      </span>
      <span className="net-value">{rate(value)}</span>
    </div>
  )
}

// Débit lisible en Ko/s ou Mo/s.
function rate(bytesPerSec: number) {
  if (bytesPerSec < 1024 * 1024) return `${Math.round(bytesPerSec / 1024)} Ko/s`
  return `${(bytesPerSec / 1024 ** 2).toFixed(1).replace('.', ',')} Mo/s`
}
