import i18n, { locale } from './i18n'

const GB = 1024 ** 3

// Nombre avec la virgule ou le point selon la langue : « 1,2 » / « 1.2 ».
export function decimal(value: number, digits = 1) {
  return value.toLocaleString(locale(), { minimumFractionDigits: digits, maximumFractionDigits: digits, useGrouping: false })
}

// Pourcentage : « 12,5 % » en français, « 12.5% » en anglais.
export function percent(value: number, digits = 1) {
  return i18n.t('units.percent', { value: decimal(value, digits) })
}

// Taille lisible : « 512 Mo », « 1,2 To » (« 512 MB », « 1.2 TB » en anglais).
export function formatBytes(bytes: number) {
  const units = ['bytes', 'KB', 'MB', 'GB', 'TB'] as const
  let value = bytes
  let i = 0
  while (value >= 1024 && i < units.length - 1) {
    value /= 1024
    i++
  }
  const digits = i === 0 || value >= 100 ? 0 : 1
  return `${decimal(value, digits)} ${i18n.t(`units.${units[i]}`)}`
}

// Gigaoctets sans unité, avec une décimale : « 12,4 ».
export function gb(bytes: number) {
  return decimal(bytes / GB)
}

// Mémoire d'un programme : « 850 Mo », « 1,2 Go ».
export function mb(bytes: number) {
  return bytes >= GB ? `${gb(bytes)} ${i18n.t('units.GB')}` : `${Math.round(bytes / 1024 ** 2)} ${i18n.t('units.MB')}`
}
