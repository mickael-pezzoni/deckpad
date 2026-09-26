import { Swiper, SwiperSlide } from 'swiper/react'
import { Pagination } from 'swiper/modules'
import 'swiper/css'
import 'swiper/css/pagination'
import { pages } from './pages'

export default function App() {
  return (
    <Swiper modules={[Pagination]} pagination={{ clickable: true }} className="deck">
      {pages.map(({ id, title, Component }) => (
        <SwiperSlide key={id}>
          <section className="page">
            <h1 className="page-title">{title}</h1>
            <Component />
          </section>
        </SwiperSlide>
      ))}
    </Swiper>
  )
}
