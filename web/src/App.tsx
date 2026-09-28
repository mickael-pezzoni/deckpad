import { useEffect, useMemo, useState } from 'react'
import { Swiper, SwiperSlide } from 'swiper/react'
import type { Swiper as SwiperInstance } from 'swiper'
import 'swiper/css'
import { pages } from './pages'
import { AppShell } from './layout/AppShell'
import { PageShell } from './layout/PageShell'
import { swipeFromFields } from './layout/swipeFromFields'
import { PairingScreen } from './pairing/PairingScreen'
import { SecureSetup } from './pairing/SecureSetup'
import { alreadySecured, askedRecently, claimHandoff, goSecure, onInsecureLan, securePort, secureReachable } from './pairing/secure'

// En HTTP depuis la tablette : port HTTPS à proposer, ou null si rien à proposer.
// Une tablette qui a déjà basculé repart directement en HTTPS.
async function secureStep(afterPairing: boolean): Promise<string | null> {
  if (!onInsecureLan()) return null
  const port = await securePort()
  if (!port) return null
  if (alreadySecured() && (await secureReachable(port)) && (await goSecure(port))) return null
  return afterPairing || !askedRecently() ? port : null
}

export default function App() {
  const [paired, setPaired] = useState<boolean | null>(null)
  const [securePortOffer, setSecurePortOffer] = useState<string | null>(null)
  const [swiper, setSwiper] = useState<SwiperInstance | null>(null)
  const [current, setCurrent] = useState(0) // suit le doigt : le menu réagit tout de suite
  const [shown, setShown] = useState(0) // page arrivée : c'est elle seule qui se met à jour

  useEffect(() => {
    ;(async () => {
      await claimHandoff()
      const s = await fetch('/api/pair/status')
        .then((r) => r.json())
        .catch(() => ({ paired: false }))
      if (s.paired) setSecurePortOffer(await secureStep(false))
      setPaired(s.paired)
    })()
  }, [])

  useEffect(swipeFromFields, [])

  const onPaired = async () => {
    setSecurePortOffer(await secureStep(true))
    setPaired(true)
  }

  // Créées une seule fois : React ne re-rend une page que si son état ou sa visibilité change.
  const content = useMemo(() => pages.map(({ Component }) => <Component />), [])

  // Les pages ne sont recréées qu'une fois le swipe terminé (pas pendant l'animation),
  // et seules les deux pages dont l'état change se re-rendent (contexte PageActive).
  const deck = useMemo(
    () => (
      <Swiper
        className="deck"
        onSwiper={setSwiper}
        onSlideChange={(s) => setCurrent(s.activeIndex)}
        onSlideChangeTransitionEnd={(s) => setShown(s.activeIndex)}
      >
        {pages.map(({ id, title }, i) => (
          <SwiperSlide key={id}>
            <PageShell title={title} active={i === shown}>
              {content[i]}
            </PageShell>
          </SwiperSlide>
        ))}
      </Swiper>
    ),
    [shown, content],
  )

  if (paired === null) return null
  if (!paired) return <PairingScreen onPaired={onPaired} />
  if (securePortOffer) return <SecureSetup port={securePortOffer} onSkip={() => setSecurePortOffer(null)} />

  return (
    <AppShell pages={pages} current={current} onNavigate={(i) => swiper?.slideTo(i)}>
      {deck}
    </AppShell>
  )
}
