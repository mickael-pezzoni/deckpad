import { useEffect, useState } from 'react'
import { Monitor, MonitorOff } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { Loader } from '../components/Loader'
import { fetchPcs, type PC } from './pcs'

type Props = {
  current: string | null // PC affiché avant d'ouvrir la liste
  onPick: (pc: PC) => void
}

// Liste des PC trouvés par le hub : une grande tuile par PC. Un PC éteint reste
// affiché (grisé) s'il est déjà associé, pour qu'on sache qu'il existe.
export function PcPicker({ current, onPick }: Props) {
  const { t } = useTranslation()
  const [pcs, setPcs] = useState<PC[] | null>(null)

  useEffect(() => {
    let alive = true
    const load = () =>
      fetchPcs().then((list) => {
        if (alive && list) setPcs(list)
      })
    load()
    // Un PC qu'on vient d'allumer apparaît tout seul.
    const timer = setInterval(() => document.hidden || load(), 5000)
    return () => {
      alive = false
      clearInterval(timer)
    }
  }, [])

  if (pcs === null) {
    return (
      <main className="pcs">
        <Loader label={t('pcs.searching')} />
      </main>
    )
  }

  if (pcs.length === 0) {
    return (
      <main className="pcs">
        <div className="pcs-empty">
          <Loader label={t('pcs.empty')} />
          <p>{t('pcs.emptyHint')}</p>
        </div>
      </main>
    )
  }

  return (
    <main className="pcs">
      <h1 className="pcs-title">{t('pcs.title')}</h1>
      <div className="pcs-grid">
        {pcs.map((pc) => {
          const Icon = pc.online ? Monitor : MonitorOff
          const status = !pc.online ? t('pcs.off') : !pc.paired ? t('pcs.toPair') : pc.id === current ? t('pcs.current') : t('pcs.ready')
          return (
            <button
              key={pc.id}
              type="button"
              className={`tile tile-button pc-tile${pc.id === current ? ' is-current' : ''}${pc.paired ? '' : ' is-new'}`}
              disabled={!pc.online}
              onClick={() => onPick(pc)}
            >
              <Icon size={48} strokeWidth={1.5} aria-hidden className="pc-icon" />
              <span className="pc-name">{pc.name}</span>
              <span className="pc-status">{status}</span>
            </button>
          )
        })}
      </div>
    </main>
  )
}
