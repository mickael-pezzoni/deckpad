import { useEffect, useState } from 'react'
import { ArrowDown, ArrowUp, EthernetPort, Globe, House, Timer, Wifi } from 'lucide-react'
import { Tile } from '../components/Tile'
import { Sparkline } from '../components/Sparkline'
import { useEventStream } from '../hooks/useEventStream'

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
  )

  useEffect(() => {
    fetch('/api/network/public-ip')
      .then((r) => (r.ok ? r.json() : Promise.reject()))
      .then((d) => setPublicIP(d.ip))
      .catch(() => setPublicIP('Indisponible'))
  }, [])

  const net = history.at(-1)
  if (!net) return <p className="coming-soon">{connected ? 'Chargement…' : 'Connexion au PC…'}</p>

  // Même échelle pour les deux courbes, avec un plancher pour ne pas amplifier le bruit.
  const max = Math.max(128 * 1024, ...history.flatMap((n) => [n.rxRate, n.txRate]))

  return (
    <div className="grid">
      <Tile label="Réception" icon={ArrowDown} wide chart={<Sparkline values={history.map((n) => n.rxRate)} max={max} />}>
        {rate(net.rxRate)}
      </Tile>
      <Tile label="Envoi" icon={ArrowUp} wide chart={<Sparkline values={history.map((n) => n.txRate)} max={max} />}>
        {rate(net.txRate)}
      </Tile>
      <Tile label="Latence" icon={Timer} detail="vers Internet">
        {net.latencyMs == null ? '—' : `${net.latencyMs} ms`}
      </Tile>
      <Tile label="Connexion" icon={net.kind === 'wifi' ? Wifi : EthernetPort} detail={net.interface || undefined}>
        {net.kind === 'wifi' ? 'Wi-Fi' : net.kind === 'ethernet' ? 'Câble' : '—'}
      </Tile>
      <Tile label="IP locale" icon={House}>{net.localIP || '—'}</Tile>
      <Tile label="IP publique" icon={Globe}>{publicIP ?? '…'}</Tile>
    </div>
  )
}

// Débit lisible en Ko/s ou Mo/s.
function rate(bytesPerSec: number) {
  if (bytesPerSec < 1024 * 1024) return `${Math.round(bytesPerSec / 1024)} Ko/s`
  return `${(bytesPerSec / 1024 ** 2).toFixed(1).replace('.', ',')} Mo/s`
}
