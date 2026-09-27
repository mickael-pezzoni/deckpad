import { useState } from 'react'
import { useEventStream } from './useEventStream'

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
  const connected = useEventStream<Stats>('/api/stats/stream', (s) =>
    setHistory((h) => [...h.slice(-(HISTORY - 1)), s]),
  )
  return { latest: history.at(-1), history, connected }
}
