import { createContext, useContext } from 'react'

// Vrai quand la page est celle affichée (une fois le swipe terminé).
export const PageActiveContext = createContext(true)

export function usePageActive() {
  return useContext(PageActiveContext)
}
