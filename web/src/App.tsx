import { useEffect, useState } from 'react'
import { Swiper, SwiperSlide } from 'swiper/react'
import { Pagination } from 'swiper/modules'
import 'swiper/css'
import 'swiper/css/pagination'
import { pages } from './pages'
import { PairingScreen } from './pairing/PairingScreen'

export default function App() {
  const [paired, setPaired] = useState<boolean | null>(null)

  useEffect(() => {
    fetch('/api/pair/status')
      .then((r) => r.json())
      .then((s) => setPaired(s.paired))
      .catch(() => setPaired(false))
  }, [])

  if (paired === null) return null
  if (!paired) return <PairingScreen onPaired={() => setPaired(true)} />

  return (
    <Swiper modules={[Pagination]} pagination={{ clickable: true }} className="deck">
      {pages.map(({ id, title, Component }) => (
        <SwiperSlide key={id}>
          <section className="page">
            <h1 className="page-title">{title}</h1>
            <div className="page-body">
              <Component />
            </div>
          </section>
        </SwiperSlide>
      ))}
    </Swiper>
  )
}
