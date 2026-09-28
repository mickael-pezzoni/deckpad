import { useEffect, useState } from 'react'
import { AppWindow, Cpu, Monitor, Power, User } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import i18n, { locale } from '../i18n'
import { Loader } from '../components/Loader'
import { InfoList } from '../components/InfoList'
import { usePageActive } from '../layout/pageActive'

type Info = {
  hostname: string
  username: string
  os: string
  cpu: string
  uptimeSec: number
  now: string
}

export function InfoPage() {
  const { t } = useTranslation()
  const [info, setInfo] = useState<Info | null>(null)
  const [error, setError] = useState(false)
  const active = usePageActive()
  const now = usePcClock(info, active)

  // Rechargé chaque minute (et au retour sur la page) : suit un redémarrage du PC
  // ou un changement de session. En pause quand la page est cachée.
  useEffect(() => {
    if (!active) return
    const load = () =>
      fetch('/api/info')
        .then((r) => (r.ok ? r.json() : Promise.reject(r.status)))
        .then((i) => {
          setInfo(i)
          setError(false)
        })
        .catch(() => setError(true))
    load()
    const id = setInterval(load, 60_000)
    return () => clearInterval(id)
  }, [active])

  if (error) return <p className="coming-soon">{t('common.unreachable')}</p>
  if (!info || !now) return <Loader />

  // La durée avance avec l'horloge, entre deux rechargements.
  const uptime = info.uptimeSec + Math.max(0, (now.getTime() - new Date(info.now).getTime()) / 1000)

  const rows = [
    { icon: User, label: t('info.user'), value: shortUser(info.username) },
    { icon: Monitor, label: t('info.pcName'), value: info.hostname },
    { icon: AppWindow, label: t('info.os'), value: info.os },
    { icon: Power, label: t('info.uptime'), value: formatUptime(uptime) },
    { icon: Cpu, label: t('info.cpu'), value: shortCPU(info.cpu) },
  ]

  // Grande horloge + fiche du PC en liste.
  return (
    <div className="info-page">
      <div className="tile info-clock">
        <span className="info-time">{now.toLocaleTimeString(locale(), { hour: '2-digit', minute: '2-digit', hourCycle: 'h23' })}</span>
        <span className="info-date">
          {now.toLocaleDateString(locale(), { weekday: 'long', day: 'numeric', month: 'long' })}
        </span>
      </div>
      <InfoList rows={rows} />
    </div>
  )
}

// Horloge du PC : on part de l'heure envoyée par l'agent et on avance localement.
function usePcClock(info: Info | null, active: boolean) {
  const [now, setNow] = useState<Date | null>(null)
  useEffect(() => {
    if (!info || !active) return
    const offset = new Date(info.now).getTime() - Date.now()
    const tick = () => setNow(new Date(Date.now() + offset))
    tick()
    const id = setInterval(tick, 1000)
    return () => clearInterval(id)
  }, [info, active])
  return now
}

// Windows renvoie « MACHINE\utilisateur ».
function shortUser(username: string) {
  return username.split('\\').pop() ?? username
}

// « AMD Ryzen 7 5800X 8-Core Processor » → « AMD Ryzen 7 5800X ».
function shortCPU(name: string) {
  return name
    .replace(/\((R|TM)\)/gi, '')
    .replace(/\d+-Core|Processor|CPU|@.*$/gi, '')
    .replace(/\s+/g, ' ')
    .trim()
}

function formatUptime(sec: number) {
  const d = Math.floor(sec / 86400)
  const h = Math.floor((sec % 86400) / 3600)
  const m = Math.floor((sec % 3600) / 60)
  return d > 0 ? i18n.t('info.days', { d, h }) : i18n.t('info.hours', { h, m })
}
