import { useLayoutEffect, useRef, useState, type ReactNode } from 'react'

// Rangée qui défile de côté. Tant qu'elle tient à l'écran, glisser dessus change de
// page ; si elle déborde, glisser la fait défiler au lieu de changer de page.
export function SideScroll({ className, children }: { className: string; children: ReactNode }) {
  const ref = useRef<HTMLDivElement>(null)
  const [overflows, setOverflows] = useState(false)

  const check = () => {
    const el = ref.current
    if (el) setOverflows(el.scrollWidth > el.clientWidth + 1)
  }
  useLayoutEffect(check) // un élément en plus ou en moins
  useLayoutEffect(() => {
    const ro = new ResizeObserver(check) // rotation de la tablette
    ro.observe(ref.current!)
    return () => ro.disconnect()
  }, [])

  return (
    <div ref={ref} className={overflows ? `${className} swiper-no-swiping` : className}>
      {children}
    </div>
  )
}
