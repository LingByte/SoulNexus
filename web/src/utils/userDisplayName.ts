import type { User } from '@/api/auth'

/** Best-effort display name for seat labels (Web ACD row, etc.). */
export function formatUserSeatBaseName(user: User | null | undefined): string {
  if (!user) return ''
  const prof = user.profile as { displayName?: string; firstName?: string; lastName?: string } | undefined
  const dn = typeof user.displayName === 'string'
    ? user.displayName.trim()
    : typeof prof?.displayName === 'string'
      ? prof.displayName.trim()
      : ''
  if (dn) return dn
  const fn = [
    user.firstName ?? prof?.firstName,
    user.lastName ?? prof?.lastName,
  ]
    .filter((x): x is string => typeof x === 'string' && x.trim().length > 0)
    .join(' ')
    .trim()
  if (fn) return fn
  const em = typeof user.email === 'string' ? user.email.trim() : ''
  if (em) {
    const at = em.indexOf('@')
    return at > 0 ? em.slice(0, at) : em
  }
  return ''
}
