import { useEffect, useRef, useState } from 'react'

// Écoute un flux Server-Sent Events de l'agent. EventSource se reconnecte tout seul
// si le PC redémarre ou si le Wi-Fi coupe.
export function useEventStream<T>(url: string, onData: (data: T) => void) {
  const [connected, setConnected] = useState(false)
  const handler = useRef(onData)
  handler.current = onData

  useEffect(() => {
    const es = new EventSource(url)
    es.onopen = () => setConnected(true)
    es.onerror = () => setConnected(false)
    es.onmessage = (e) => handler.current(JSON.parse(e.data))
    return () => es.close()
  }, [url])

  return connected
}
