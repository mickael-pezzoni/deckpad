import { useEffect, useState } from 'react'

// Valeur venant du PC, remplacée un instant par celle choisie sur la tablette :
// le temps que le PC l'applique, la valeur affichée ne revient pas en arrière.
export function useLatched<T>(server: T, holdMs = 1500): [T, (v: T) => void] {
  const [local, setLocal] = useState<{ v: T } | null>(null)

  useEffect(() => {
    if (!local) return
    const id = setTimeout(() => setLocal(null), holdMs)
    return () => clearTimeout(id)
  }, [local, holdMs])

  return [local ? local.v : server, (v) => setLocal({ v })]
}
