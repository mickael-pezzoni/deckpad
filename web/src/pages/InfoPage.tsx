import { useEffect, useState } from 'react'
import { AppWindow, Cpu, Monitor, Power, User } from 'lucide-react'
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

  if (error) return <p className="coming-soon">PC injoignable</p>
  if (!info || !now) return <Loader />

  // La durée avance avec l'horloge, entre deux rechargements.
  const uptime = info.uptimeSec + Math.max(0, (now.getTime() - new Date(info.now).getTime()) / 1000)

  const rows = [
    { icon: User, label: 'Utilisateur', value: shortUser(info.username) },
    { icon: Monitor, label: 'Nom du PC', value: info.hostname },
    { icon: AppWindow, label: 'Système', value: info.os },
    { icon: Power, label: 'Allumé depuis', value: formatUptime(uptime) },
    { icon: Cpu, label: 'Processeur', value: shortCPU(info.cpu) },
  ]

  // Grande horloge + fiche du PC en liste.
  return (
    <div className="info-page">
      <div className="tile info-clock">
        <span className="info-time">{now.toLocaleTimeString('fr-FR', { hour: '2-digit', minute: '2-digit' })}</span>
        <span className="info-date">
          {now.toLocaleDateString('fr-FR', { weekday: 'long', day: 'numeric', month: 'long' })}
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
  return d > 0 ? `${d} j ${h} h` : `${h} h ${m} min`
}
