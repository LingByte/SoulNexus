/** Converts Tailwind-style HSL token "262 83% 58%" to #rrggbb for Arco ConfigProvider. */
export function hslTokenToHex(hslToken: string, fallback = '#8B5CF6'): string {
  const parts = hslToken.trim().split(/\s+/)
  if (parts.length < 3) return fallback
  const h = parseFloat(parts[0]!)
  const s = parseFloat(parts[1]!) / 100
  const l = parseFloat(parts[2]!) / 100
  if (Number.isNaN(h) || Number.isNaN(s) || Number.isNaN(l)) return fallback

  const c = (1 - Math.abs(2 * l - 1)) * s
  const x = c * (1 - Math.abs(((h / 60) % 2) - 1))
  const m = l - c / 2
  let r = 0
  let g = 0
  let b = 0
  if (h < 60) {
    r = c
    g = x
  } else if (h < 120) {
    r = x
    g = c
  } else if (h < 180) {
    g = c
    b = x
  } else if (h < 240) {
    g = x
    b = c
  } else if (h < 300) {
    r = x
    b = c
  } else {
    r = c
    b = x
  }
  const toByte = (n: number) => Math.round((n + m) * 255)
    .toString(16)
    .padStart(2, '0')
  return `#${toByte(r)}${toByte(g)}${toByte(b)}`
}

export function readPrimaryColorFromDocument(fallback = '#8B5CF6'): string {
  if (typeof document === 'undefined') return fallback
  const token = getComputedStyle(document.documentElement).getPropertyValue('--primary').trim()
  return hslTokenToHex(token, fallback)
}
