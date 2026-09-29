import type { Swiper } from 'swiper'

// Bloque le glissement entre les pages pendant un geste qui utilise le doigt
// (ex. le menu circulaire des fichiers). App enregistre le Swiper affiché.
let swiper: Swiper | null = null

export function registerSwiper(s: Swiper | null) {
  swiper = s
}

export function lockSwipe(locked: boolean) {
  if (swiper && !swiper.destroyed) swiper.allowTouchMove = !locked
}
