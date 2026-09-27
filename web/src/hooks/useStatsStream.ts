import { useEffect, useState } from 'react'

export type Stats = {
  cpu: number
  ramUsed: number
  ramTotal: number
  gpu: { name: string; usage: number; memUsed: number; memTotal: number; temp: number | null } | null
  fps: number | null
}

const HISTORY = 60 // une minute à 1 mesure/s

// Reçoit les stats poussées par l'agent et garde un historique court pour les courbes.
export function useStatsStream() {
  const [history, setHistory] = useState<Stats[]>([])
  const [connected, setConnected] = useState(false)

  useEffect(() => {
    // EventSource se reconnecte tout seul si le PC redémarre ou le Wi-Fi coupe.
    const es = new EventSource('/api/stats/stream')
    es.onopen = () => setConnected(true)
    es.onerror = () => setConnected(false)
    es.onmessage = (e) => {
      const s: Stats = JSON.parse(e.data)
      setHistory((h) => [...h.slice(-(HISTORY - 1)), s])
    }
    return () => es.close()
  }, [])

  return { latest: history.at(-1), history, connected }
}
