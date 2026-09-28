import { useState } from 'react'
import { Moon, Sun } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { getTheme, setTheme } from '../theme/theme'

// Bouton rond soleil/lune : bascule entre le thème sombre et le thème clair.
export function ThemeToggle() {
  const { t } = useTranslation()
  const [theme, set] = useState(getTheme)
  const next = theme === 'dark' ? 'light' : 'dark'
  const Icon = theme === 'dark' ? Sun : Moon

  return (
    <button
      type="button"
      className="theme-toggle"
      aria-label={next === 'light' ? t('theme.toLight') : t('theme.toDark')}
      onClick={() => {
        setTheme(next)
        set(next)
      }}
    >
      <Icon size={24} strokeWidth={2} aria-hidden />
    </button>
  )
}
