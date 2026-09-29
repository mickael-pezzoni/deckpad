import { ArrowLeftRight } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { usePc } from './pcContext'

// Bouton de la barre du bas : le nom du PC piloté, et un tap pour en changer.
// Masqué sur téléphone (la barre est pleine) : on passe alors par Paramètres.
export function PcSwitch() {
  const { t } = useTranslation()
  const { name, switchPc } = usePc()
  return (
    <button type="button" className="pc-switch" onClick={switchPc} aria-label={`${t('pcs.switch')} (${name})`} title={t('pcs.switch')}>
      <ArrowLeftRight size={22} strokeWidth={2} aria-hidden />
      <span className="pc-switch-name">{name}</span>
    </button>
  )
}
