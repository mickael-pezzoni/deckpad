// Copie un texte dans le presse-papiers de la tablette. Sans HTTPS, le navigateur
// refuse navigator.clipboard : on passe alors par un champ caché et la commande
// « copier » classique, encore permise pendant un appui.
export async function copyToDevice(text: string): Promise<boolean> {
  if (window.isSecureContext && navigator.clipboard) {
    try {
      await navigator.clipboard.writeText(text)
      return true
    } catch {
      // on essaie l'ancienne méthode
    }
  }
  const area = document.createElement('textarea')
  area.value = text
  area.setAttribute('readonly', '')
  area.style.position = 'fixed'
  area.style.opacity = '0'
  document.body.appendChild(area)
  area.select()
  area.setSelectionRange(0, text.length)
  let ok = false
  try {
    ok = document.execCommand('copy')
  } catch {
    ok = false
  }
  area.remove()
  return ok
}

// Un texte qui ressemble à une adresse web (à ouvrir dans le navigateur du PC).
export function looksLikeURL(text: string): boolean {
  const t = text.trim()
  if (!t || /\s/.test(t)) return false
  return /^https?:\/\/\S+$/i.test(t) || /^[\w-]+(\.[\w-]+)+(:\d+)?(\/\S*)?$/.test(t)
}
