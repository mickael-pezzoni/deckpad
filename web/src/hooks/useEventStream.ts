import { useEffect, useRef, useState } from 'react'
import { usePageActive } from '../layout/pageActive'

type Options = {
  // Page cachée : garder le flux ouvert et mettre les mesures de côté (pour ne pas trouer
  // les courbes), au lieu de le couper. Dans les deux cas, rien n'est rendu en arrière-plan.
  keepAlive?: boolean
}

const MAX_BUFFERED = 120

// Écoute un flux Server-Sent Events de l'agent. EventSource se reconnecte tout seul
// si le PC redémarre ou si le Wi-Fi coupe.
export function useEventStream<T>(url: string, onData: (data: T) => void, { keepAlive = false }: Options = {}) {
  const active = usePageActive()
  const open = active || keepAlive
  const [connected, setConnected] = useState(false)
  const handler = useRef(onData)
  handler.current = onData
  const activeRef = useRef(active)
  activeRef.current = active
  const buffered = useRef<T[]>([])

  useEffect(() => {
    if (!open) return
    const es = new EventSource(url)
    es.onopen = () => setConnected(true)
    es.onerror = () => setConnected(false)
    es.onmessage = (e) => {
      const data: T = JSON.parse(e.data)
      if (activeRef.current) handler.current(data)
      else buffered.current = [...buffered.current.slice(-(MAX_BUFFERED - 1)), data]
    }
    return () => es.close()
  }, [url, open])

  // Retour sur la page : on rejoue d'un coup ce qui a été mis de côté (un seul rendu).
  useEffect(() => {
    if (!active || buffered.current.length === 0) return
    const pending = buffered.current
    buffered.current = []
    pending.forEach((d) => handler.current(d))
  }, [active])

  return connected
}
