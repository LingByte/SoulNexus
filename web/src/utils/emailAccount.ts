/** 是否为占位邮箱（未绑定或 OAuth 临时邮箱） */
export function isPlaceholderEmail(email?: string | null): boolean {
  const e = (email || '').trim().toLowerCase()
  return !e || e.endsWith('@temp.local')
}

/** 是否可在账号安全页绑定邮箱 */
export function canBindEmail(email?: string | null, emailVerified?: boolean): boolean {
  return isPlaceholderEmail(email) || !emailVerified
}

/** 是否可换绑邮箱 */
export function canChangeEmail(email?: string | null, emailVerified?: boolean): boolean {
  return !isPlaceholderEmail(email) && !!emailVerified
}
