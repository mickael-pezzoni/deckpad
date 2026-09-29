// PC connus du hub, et celui choisi sur cette tablette.

export type PC = {
  id: string
  name: string
  online: boolean // vu sur le réseau
  paired: boolean // le hub a sa clé
}

const KEY = 'deckpad.pc'

export async function fetchPcs(): Promise<PC[] | null> {
  const r = await fetch('/api/agents').catch(() => null)
  return r?.ok ? r.json().catch(() => null) : null
}

export function getStoredPc() {
  try {
    return localStorage.getItem(KEY)
  } catch {
    return null
  }
}

export function storePc(id: string) {
  try {
    localStorage.setItem(KEY, id)
  } catch {
    // stockage indisponible : il faudra rechoisir le PC au prochain lancement
  }
}
