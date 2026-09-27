// Taille lisible en français : « 512 Mo », « 1,2 To ».
export function formatBytes(bytes: number) {
  const units = ['octets', 'Ko', 'Mo', 'Go', 'To']
  let value = bytes
  let i = 0
  while (value >= 1024 && i < units.length - 1) {
    value /= 1024
    i++
  }
  const digits = i === 0 || value >= 100 ? 0 : 1
  return `${value.toFixed(digits).replace('.', ',')} ${units[i]}`
}
