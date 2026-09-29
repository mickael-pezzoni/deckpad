import { useEffect, useState } from 'react'
import { ArrowDown, ArrowUp, EthernetPort, Globe, House, Timer, Wifi, type LucideIcon } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import i18n from '../i18n'
import { decimal } from '../format'
import { Sparkline } from '../components/Sparkline'
import { InfoList } from '../components/InfoList'
import { useEventStream } from '../hooks/useEventStream'
import { Loader } from '../components/Loader'
import { api } from '../api'

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
  const { t } = useTranslation()
  const [history, setHistory] = useState<Net[]>([])
  const [publicIP, setPublicIP] = useState<string | null>(null)
  const connected = useEventStream<Net>(api('/network/stream'), (n) =>
    setHistory((h) => [...h.slice(-(HISTORY - 1)), n]),
    { keepAlive: true }, // garde les courbes complètes
  )

  useEffect(() => {
    fetch(api('/network/public-ip'))
      .then((r) => (r.ok ? r.json() : Promise.reject()))
      .then((d) => setPublicIP(d.ip))
      .catch(() => setPublicIP('')) // vide : affiché « Indisponible »
  }, [])

  const net = history.at(-1)
  if (!net) return <Loader label={connected ? undefined : t('common.connecting')} />

  // Même échelle pour les deux courbes, avec un plancher pour ne pas amplifier le bruit.
  const max = Math.max(128 * 1024, ...history.flatMap((n) => [n.rxRate, n.txRate]))
  const wifi = net.kind === 'wifi'

  // Débits en grand avec leur courbe commune, fiche de la connexion à côté.
  return (
    <div className="net-page">
      <div className="tile net-rates">
        <div className="net-values">
          <Rate icon={ArrowDown} label={t('network.download')} value={net.rxRate} />
          <Rate icon={ArrowUp} label={t('network.upload')} value={net.txRate} upload />
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
            label: t('network.connection'),
            value: wifi ? t('network.wifi') : net.kind === 'ethernet' ? t('network.cable') : '—',
          },
          { icon: Timer, label: t('network.latency'), value: net.latencyMs == null ? '—' : t('units.ms', { value: net.latencyMs }) },
          { icon: House, label: t('network.localIP'), value: net.localIP || '—' },
          { icon: Globe, label: t('network.publicIP'), value: publicIP === null ? '…' : publicIP || t('network.unavailable') },
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

// Débit lisible en Ko/s ou Mo/s (KB/s, MB/s en anglais).
function rate(bytesPerSec: number) {
  const perSec = (unit: 'KB' | 'MB') => i18n.t('units.perSec', { unit: i18n.t(`units.${unit}`) })
  if (bytesPerSec < 1024 * 1024) return `${Math.round(bytesPerSec / 1024)} ${perSec('KB')}`
  return `${decimal(bytesPerSec / 1024 ** 2)} ${perSec('MB')}`
}
