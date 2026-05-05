// Same semantics as web/src/utils/authUserProfile.ts — keep admin bundle self-contained.

export const AUTH_USER_PROFILE_FIELD_KEYS = [
  'phone',
  'firstName',
  'lastName',
  'displayName',
  'locale',
  'timezone',
  'avatar',
  'gender',
  'city',
  'region',
  'extra',
  'emailNotifications',
  'pushNotifications',
  'profileComplete',
  'wifiName',
  'wifiPassword',
  'fatherCallName',
  'motherCallName',
] as const

export function normalizeAuthUser<T extends Record<string, unknown>>(raw: T | null | undefined): T | null | undefined {
  if (raw == null || typeof raw !== 'object') return raw
  const p = raw.profile
  if (p == null || typeof p !== 'object') return raw
  const prof = p as Record<string, unknown>
  const out = { ...raw } as Record<string, unknown>
  for (const k of AUTH_USER_PROFILE_FIELD_KEYS) {
    if (prof[k] !== undefined) out[k] = prof[k]
  }
  return out as T
}

export function normalizeNestedProfileUsers<T>(data: T): T {
  const walk = (v: unknown): unknown => {
    if (v == null || typeof v !== 'object') return v
    if (Array.isArray(v)) return v.map(walk)
    const o = v as Record<string, unknown>
    const base =
      typeof o.email === 'string' && o.profile != null && typeof o.profile === 'object'
        ? (normalizeAuthUser(o as Record<string, unknown>) as Record<string, unknown>)
        : o
    const out: Record<string, unknown> = { ...base }
    for (const key of Object.keys(out)) {
      out[key] = walk(out[key])
    }
    return out
  }
  return walk(data) as T
}

export function mergeUserPartial<T extends Record<string, unknown>>(user: T, patch: Partial<T>): T {
  const next = { ...user, ...patch } as T & Record<string, unknown>
  const prof = next.profile
  if (prof != null && typeof prof === 'object') {
    const p = { ...(prof as Record<string, unknown>) }
    let changed = false
    for (const k of AUTH_USER_PROFILE_FIELD_KEYS) {
      const v = (patch as Record<string, unknown>)[k]
      if (v !== undefined) {
        p[k] = v
        changed = true
      }
    }
    if (changed) next.profile = p as typeof prof
  }
  return next as T
}
