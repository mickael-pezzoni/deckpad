import { useEffect, useMemo, useState } from 'react'
import { Swiper, SwiperSlide } from 'swiper/react'
import type { Swiper as SwiperInstance } from 'swiper'
import 'swiper/css'
import { pages } from './pages'
import { AppShell } from './layout/AppShell'
import { PageShell } from './layout/PageShell'
import { PairingScreen } from './pairing/PairingScreen'

export default function App() {
  const [paired, setPaired] = useState<boolean | null>(null)
  const [swiper, setSwiper] = useState<SwiperInstance | null>(null)
  const [current, setCurrent] = useState(0)

  useEffect(() => {
    fetch('/api/pair/status')
      .then((r) => r.json())
      .then((s) => setPaired(s.paired))
      .catch(() => setPaired(false))
  }, [])

  // Les pages ne dépendent pas de la page active : on les crée une seule fois pour que
  // changer de page (swipe ou menu) ne re-rende que le menu, pas les 7 pages en pleine animation.
  const deck = useMemo(
    () => (
      <Swiper className="deck" onSwiper={setSwiper} onSlideChange={(s) => setCurrent(s.activeIndex)}>
        {pages.map(({ id, title, Component }) => (
          <SwiperSlide key={id}>
            <PageShell title={title}>
              <Component />
            </PageShell>
          </SwiperSlide>
        ))}
      </Swiper>
    ),
    [],
  )

  if (paired === null) return null
  if (!paired) return <PairingScreen onPaired={() => setPaired(true)} />

  return (
    <AppShell pages={pages} current={current} onNavigate={(i) => swiper?.slideTo(i)}>
      {deck}
    </AppShell>
  )
}
