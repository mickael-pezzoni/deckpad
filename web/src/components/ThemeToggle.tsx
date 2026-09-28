import { useState } from 'react'
import { Moon, Sun } from 'lucide-react'
import { getTheme, setTheme } from '../theme/theme'

// Bouton rond soleil/lune : bascule entre le thème sombre et le thème clair.
export function ThemeToggle() {
  const [theme, set] = useState(getTheme)
  const next = theme === 'dark' ? 'light' : 'dark'
  const Icon = theme === 'dark' ? Sun : Moon

  return (
    <button
      type="button"
      className="theme-toggle"
      aria-label={next === 'light' ? 'Passer en thème clair' : 'Passer en thème sombre'}
      onClick={() => {
        setTheme(next)
        set(next)
      }}
    >
      <Icon size={24} strokeWidth={2} aria-hidden />
    </button>
  )
}
