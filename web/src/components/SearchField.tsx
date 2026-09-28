import { useEffect, useRef, useState } from 'react'
import { Search, X } from 'lucide-react'
import { useTranslation } from 'react-i18next'

type Props = {
  value: string
  onChange: (value: string) => void
  placeholder?: string
}

// Tuile loupe : un appui l'ouvre en champ de recherche, la croix l'efface et la referme.
// Ouverte vide, elle se replie quand on quitte le champ.
export function SearchField({ value, onChange, placeholder: custom }: Props) {
  const { t } = useTranslation()
  const placeholder = custom ?? t('common.search')
  const [open, setOpen] = useState(value !== '')
  const input = useRef<HTMLInputElement>(null)

  useEffect(() => {
    if (open) input.current?.focus()
  }, [open])

  if (!open) {
    return (
      <button type="button" className="search search-closed" aria-label={placeholder} onClick={() => setOpen(true)}>
        <Search size={26} strokeWidth={2.2} aria-hidden />
      </button>
    )
  }

  return (
    <label className="search search-open">
      <Search size={24} strokeWidth={2.2} aria-hidden />
      <input
        ref={input}
        type="search"
        value={value}
        placeholder={placeholder}
        enterKeyHint="search"
        autoComplete="off"
        spellCheck={false}
        onChange={(e) => onChange(e.target.value)}
        onBlur={() => value === '' && setOpen(false)}
        onKeyDown={(e) => e.key === 'Enter' && input.current?.blur()}
      />
      <button
        type="button"
        className="search-clear"
        aria-label={t('common.clearSearch')}
        // Garde le focus : sinon le blur replie avant le clic.
        onPointerDown={(e) => e.preventDefault()}
        onClick={() => {
          onChange('')
          setOpen(false)
        }}
      >
        <X size={24} strokeWidth={2.2} aria-hidden />
      </button>
    </label>
  )
}

// Comparaison sans casse ni accents : « tele » trouve « Télégram ».
export function matches(text: string, query: string) {
  return fold(text).includes(fold(query.trim()))
}

function fold(text: string) {
  return text.normalize('NFD').replace(/\p{M}/gu, '').toLowerCase()
}
