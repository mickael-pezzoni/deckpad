import { useState } from 'react'
import { AppWindow } from 'lucide-react'

// Icône d'un programme, extraite de son .exe par l'agent (Windows).
// Si elle n'existe pas, on affiche une icône générique.
export function AppIcon({ name }: { name: string }) {
  const [failed, setFailed] = useState(false)
  if (failed) return <AppWindow size={20} strokeWidth={2} aria-hidden className="app-icon-fallback" />
  return (
    <img
      src={`/api/processes/icon?name=${encodeURIComponent(name)}`}
      alt=""
      width={28}
      height={28}
      onError={() => setFailed(true)}
    />
  )
}
