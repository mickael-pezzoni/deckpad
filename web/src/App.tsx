import { useEffect, useMemo, useState } from 'react'
import { Swiper, SwiperSlide } from 'swiper/react'
import type { Swiper as SwiperInstance } from 'swiper'
import 'swiper/css'
import { pages } from './pages'
import { AppShell } from './layout/AppShell'
import { PageShell } from './layout/PageShell'
import { swipeFromFields } from './layout/swipeFromFields'
import { registerSwiper } from './layout/swipeLock'
import { getStartPage } from './settings/startPage'
import { PairingScreen } from './pairing/PairingScreen'
import { SecureSetup } from './pairing/SecureSetup'
import { PcPicker } from './pcs/PcPicker'
import { fetchPcs, getStoredPc, storePc, type PC } from './pcs/pcs'
import { setCurrentPc } from './api'
import { alreadySecured, askedRecently, claimHandoff, goSecure, onInsecureLan, securePort, secureReachable } from './pairing/secure'

// En HTTP depuis la tablette : port HTTPS à proposer, ou null si rien à proposer.
// Une tablette qui a déjà basculé repart directement en HTTPS.
async function secureStep(afterPairing: boolean): Promise<string | null> {
  if (!onInsecureLan()) return null
  const port = await securePort()
  if (!port) return null
  // En dev (Vite), on reste sur le serveur de dev : le HTTPS sert l'appli construite, pas le code en cours.
  if (import.meta.env.DEV) return afterPairing ? port : null
  if (alreadySecured() && (await secureReachable(port)) && (await goSecure(port))) return null
  return afterPairing || !askedRecently() ? port : null
}

// Écran affiché : chargement, liste des PC, code d'un PC, ou les pages du PC choisi.
type Screen = { kind: 'loading' } | { kind: 'pick' } | { kind: 'pair'; pc: PC } | { kind: 'pc'; pc: PC }

export default function App() {
  const [screen, setScreen] = useState<Screen>({ kind: 'loading' })
  const [paired, setPaired] = useState(false) // tablette associée au hub
  const [securePortOffer, setSecurePortOffer] = useState<string | null>(null)
  const [swiper, setSwiper] = useState<SwiperInstance | null>(null)
  // Page de départ choisie dans Paramètres (Infos PC par défaut), lue une fois au lancement.
  const [startIndex] = useState(() => Math.max(0, pages.findIndex((p) => p.id === getStartPage())))
  const [current, setCurrent] = useState(startIndex) // suit le doigt : le menu réagit tout de suite
  const [shown, setShown] = useState(startIndex) // page arrivée : c'est elle seule qui se met à jour

  useEffect(() => {
    ;(async () => {
      await claimHandoff()
      const s = await fetch('/api/pair/status')
        .then((r) => r.json())
        .catch(() => ({ paired: false }))
      if (s.paired) setSecurePortOffer(await secureStep(false))
      setPaired(s.paired)
      // On reprend le PC de la dernière fois, sinon le seul PC associé s'il n'y en a qu'un.
      const pcs = s.paired ? ((await fetchPcs()) ?? []) : []
      const usable = pcs.filter((p) => p.paired)
      const stored = getStoredPc()
      const pc = usable.find((p) => p.id === stored) ?? (usable.length === 1 ? usable[0] : undefined)
      setScreen(pc ? openPc(pc) : { kind: 'pick' })
    })()
  }, [])

  useEffect(swipeFromFields, [])
  useEffect(() => registerSwiper(swiper), [swiper])

  const pick = (pc: PC) => setScreen(paired && pc.paired ? openPc(pc) : { kind: 'pair', pc })

  const onPaired = async (pc: PC) => {
    const wasPaired = paired
    setPaired(true)
    if (!wasPaired) setSecurePortOffer(await secureStep(true))
    setScreen(openPc({ ...pc, paired: true }))
  }

  const pcId = screen.kind === 'pc' ? screen.pc.id : ''

  // Créées une fois par PC : React ne re-rend une page que si son état ou sa visibilité change.
  // Changer de PC recrée tout (flux, états) pour ne rien mélanger entre deux PC.
  const content = useMemo(() => pages.map(({ Component }) => <Component key={pcId} />), [pcId])

  // Les pages ne sont recréées qu'une fois le swipe terminé (pas pendant l'animation),
  // et seules les deux pages dont l'état change se re-rendent (contexte PageActive).
  const deck = useMemo(
    () => (
      <Swiper
        key={pcId}
        className="deck"
        initialSlide={shown} // en changeant de PC, on reste sur la même page
        onSwiper={setSwiper}
        onSlideChange={(s) => setCurrent(s.activeIndex)}
        onSlideChangeTransitionEnd={(s) => setShown(s.activeIndex)}
      >
        {pages.map(({ id }, i) => (
          <SwiperSlide key={id}>
            <PageShell id={id} active={i === shown}>
              {content[i]}
            </PageShell>
          </SwiperSlide>
        ))}
      </Swiper>
    ),
    [shown, content, pcId],
  )

  if (screen.kind === 'loading') return null
  if (securePortOffer) return <SecureSetup port={securePortOffer} onSkip={() => setSecurePortOffer(null)} />
  if (screen.kind === 'pair')
    return <PairingScreen pc={screen.pc} onPaired={() => onPaired(screen.pc)} onBack={() => setScreen({ kind: 'pick' })} />
  if (screen.kind === 'pick') return <PcPicker current={getStoredPc()} onPick={pick} />

  return (
    <AppShell pages={pages} current={current} onNavigate={(i) => swiper?.slideTo(i)} pcName={screen.pc.name} onSwitchPc={() => setScreen({ kind: 'pick' })}>
      {deck}
    </AppShell>
  )
}

// Ouvre les pages d'un PC (et le retient pour le prochain lancement).
function openPc(pc: PC): Screen {
  setCurrentPc(pc.id)
  storePc(pc.id)
  return { kind: 'pc', pc }
}
