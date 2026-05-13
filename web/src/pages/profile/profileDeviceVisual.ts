import type { UserDevice } from '@/api/profile'

export type DeviceVisualKind = 'desktop' | 'tablet' | 'phone' | 'laptop'

/** Map stored `deviceType` / UA to an icon category (backend: mobile | tablet | desktop, plus legacy laptop). */
export function resolveDeviceVisualKind(device: Pick<UserDevice, 'deviceType' | 'userAgent'>): DeviceVisualKind {
  const dt = (device.deviceType || '').toLowerCase().trim()
  if (dt === 'tablet') return 'tablet'
  if (dt === 'mobile') return 'phone'
  if (dt === 'laptop') return 'laptop'
  if (dt === 'desktop') return 'desktop'

  const ua = (device.userAgent || '').toLowerCase()
  if (ua.includes('ipad') || ua.includes('tablet')) return 'tablet'
  if (ua.includes('iphone') || ua.includes('mobile')) return 'phone'
  return 'desktop'
}
