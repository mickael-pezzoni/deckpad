import { useState } from 'react'
import { AppWindow } from 'lucide-react'
import { api } from '../api'

// Icône d'un programme, extraite par l'agent (de l'.exe sous Windows, des
// fichiers .desktop sous Linux). Si elle n'existe pas, on affiche une icône générique.
// src permet une autre source que la liste des processus (ex. la page Audio).
export function AppIcon({ name, src }: { name: string; src?: string }) {
  const [failed, setFailed] = useState(false)
  if (failed) return <AppWindow size={20} strokeWidth={2} aria-hidden className="app-icon-fallback" />
  return (
    <img
      src={src ?? api(`/processes/icon?name=${encodeURIComponent(name)}`)}
      alt=""
      width={28}
      height={28}
      onError={() => setFailed(true)}
    />
  )
}
