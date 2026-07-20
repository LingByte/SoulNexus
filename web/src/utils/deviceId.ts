const STORAGE_KEY = 'soulnexus_device_id'

export function getOrCreateDeviceId(): string {
  if (typeof localStorage === 'undefined') return ''
  let id = localStorage.getItem(STORAGE_KEY)
  if (!id) {
    id = typeof crypto !== 'undefined' && crypto.randomUUID ? crypto.randomUUID() : `${Date.now()}-${Math.random()}`
    localStorage.setItem(STORAGE_KEY, id)
  }
  return id
}

export function getDeviceIdHeader(): Record<string, string> {
  const id = getOrCreateDeviceId()
  return id ? { 'X-Device-Id': id } : {}
}
