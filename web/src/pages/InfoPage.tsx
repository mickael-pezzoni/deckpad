import { useEffect, useState } from 'react'
import { Tile } from '../components/Tile'

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
  const now = usePcClock(info)

  useEffect(() => {
    fetch('/api/info')
      .then((r) => (r.ok ? r.json() : Promise.reject(r.status)))
      .then(setInfo)
      .catch(() => setError(true))
  }, [])

  if (error) return <p className="coming-soon">PC injoignable</p>
  if (!info || !now) return <p className="coming-soon">Chargement…</p>

  return (
    <div className="grid">
      <Tile label="Heure" wide>
        {now.toLocaleTimeString('fr-FR', { hour: '2-digit', minute: '2-digit' })}
      </Tile>
      <Tile label="Date" wide>
        {now.toLocaleDateString('fr-FR', { weekday: 'long', day: 'numeric', month: 'long' })}
      </Tile>
      <Tile label="Utilisateur">{shortUser(info.username)}</Tile>
      <Tile label="Nom du PC">{info.hostname}</Tile>
      <Tile label="Système">{info.os}</Tile>
      <Tile label="Allumé depuis">{formatUptime(info.uptimeSec)}</Tile>
      <Tile label="Processeur" wide>{info.cpu}</Tile>
    </div>
  )
}

// Horloge du PC : on part de l'heure envoyée par l'agent et on avance localement.
function usePcClock(info: Info | null) {
  const [now, setNow] = useState<Date | null>(null)
  useEffect(() => {
    if (!info) return
    const offset = new Date(info.now).getTime() - Date.now()
    const tick = () => setNow(new Date(Date.now() + offset))
    tick()
    const id = setInterval(tick, 1000)
    return () => clearInterval(id)
  }, [info])
  return now
}

// Windows renvoie « MACHINE\utilisateur ».
function shortUser(username: string) {
  return username.split('\\').pop() ?? username
}

function formatUptime(sec: number) {
  const d = Math.floor(sec / 86400)
  const h = Math.floor((sec % 86400) / 3600)
  const m = Math.floor((sec % 3600) / 60)
  return d > 0 ? `${d} j ${h} h` : `${h} h ${m} min`
}
