import { useEffect, useState } from 'react'

// Suit une media query CSS (ex. largeur de téléphone), mise à jour à la rotation.
export function useMediaQuery(query: string) {
  const [matches, setMatches] = useState(() => matchMedia(query).matches)
  useEffect(() => {
    const mq = matchMedia(query)
    const onChange = () => setMatches(mq.matches)
    onChange()
    mq.addEventListener('change', onChange)
    return () => mq.removeEventListener('change', onChange)
  }, [query])
  return matches
}
