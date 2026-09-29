// Adresses de l'API du PC choisi : le hub relaie /api/pc/<id>/… vers l'agent de ce PC.
// Toute l'appli est recréée quand on change de PC, donc lire l'id au rendu suffit.

let current = ''

export function setCurrentPc(id: string) {
  current = id
}

export const currentPc = () => current

// api('/stats/stream') -> '/api/pc/<id>/stats/stream'
export function api(path: string) {
  return `/api/pc/${encodeURIComponent(current)}${path}`
}
