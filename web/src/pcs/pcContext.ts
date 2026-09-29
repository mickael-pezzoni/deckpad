import { createContext, useContext } from 'react'

// PC piloté, et de quoi revenir à la liste des PC (bouton du bas, Paramètres).
export const PcContext = createContext<{ name: string; switchPc: () => void }>({ name: '', switchPc: () => {} })

export const usePc = () => useContext(PcContext)
