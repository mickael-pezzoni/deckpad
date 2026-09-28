import { useEffect, useState } from 'react'
import { AppWindow, CalendarDays, Clock, Cpu, Monitor, Power, User } from 'lucide-react'
import { Tile } from '../components/Tile'
import { Loader } from '../components/Loader'
import { usePageActive } from '../layout/pageActive'
import './info-variants.css'

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

  const time = now.toLocaleTimeString('fr-FR', { hour: '2-digit', minute: '2-digit' })
  const date = now.toLocaleDateString('fr-FR', { weekday: 'long', day: 'numeric', month: 'long' })
  const v = new URLSearchParams(location.search).get('info')
  const user = shortUser(info.username)
  const cpu = shortCPU(info.cpu)
  const up = formatUptime(uptime)

  if (v === 'a')
    return (
      <div className="iv-a">
        <div className="tile iv-hero">
          <span className="iv-time">{time}</span>
          <span className="iv-date">{date}</span>
        </div>
        <div className="iv-a-side">
          <Tile label="Utilisateur" icon={User}>{user}</Tile>
          <Tile label="Nom du PC" icon={Monitor}>{info.hostname}</Tile>
          <Tile label="Système" icon={AppWindow}>{info.os}</Tile>
          <Tile label="Allumé depuis" icon={Power}>{up}</Tile>
          <Tile label="Processeur" icon={Cpu} wide>{cpu}</Tile>
        </div>
      </div>
    )

  if (v === 'b')
    return (
      <div className="iv-b">
        <div className="tile iv-id">
          <span className="iv-id-icon"><Monitor size={56} strokeWidth={1.6} /></span>
          <span className="iv-id-text">
            <span className="iv-id-name">{info.hostname}</span>
            <span className="iv-id-sub"><User size={20} /> {user}<span className="iv-dot">·</span><AppWindow size={20} /> {info.os}</span>
          </span>
        </div>
        <div className="iv-b-row">
          <Tile label="Heure" icon={Clock} detail={date}>{time}</Tile>
          <Tile label="Allumé depuis" icon={Power}>{up}</Tile>
          <Tile label="Processeur" icon={Cpu}>{cpu}</Tile>
        </div>
      </div>
    )

  if (v === 'c') {
    const rows = [
      { icon: User, label: 'Utilisateur', value: user },
      { icon: Monitor, label: 'Nom du PC', value: info.hostname },
      { icon: AppWindow, label: 'Système', value: info.os },
      { icon: Power, label: 'Allumé depuis', value: up },
      { icon: Cpu, label: 'Processeur', value: cpu },
    ]
    return (
      <div className="iv-c">
        <div className="tile iv-hero">
          <span className="iv-time">{time}</span>
          <span className="iv-date">{date}</span>
        </div>
        <div className="tile iv-list">
          {rows.map(({ icon: I, label, value }) => (
            <div className="iv-row" key={label}>
              <span className="tile-label"><span className="tile-icon"><I size={20} /></span>{label}</span>
              <span className="iv-row-value">{value}</span>
            </div>
          ))}
        </div>
      </div>
    )
  }

  return (
    <div className="grid">
      <Tile label="Heure" icon={Clock} wide>
        {now.toLocaleTimeString('fr-FR', { hour: '2-digit', minute: '2-digit' })}
      </Tile>
      <Tile label="Date" icon={CalendarDays} wide>
        {now.toLocaleDateString('fr-FR', { weekday: 'long', day: 'numeric', month: 'long' })}
      </Tile>
      <Tile label="Utilisateur" icon={User}>{shortUser(info.username)}</Tile>
      <Tile label="Nom du PC" icon={Monitor}>{info.hostname}</Tile>
      <Tile label="Système" icon={AppWindow}>{info.os}</Tile>
      <Tile label="Allumé depuis" icon={Power}>{formatUptime(uptime)}</Tile>
      <Tile label="Processeur" icon={Cpu}>{shortCPU(info.cpu)}</Tile>
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
