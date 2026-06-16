/** 是否为占位邮箱（未绑定或 OAuth 临时邮箱） */
export function isPlaceholderEmail(email?: string | null): boolean {
  const e = (email || '').trim().toLowerCase()
  return !e || e.endsWith('@temp.local')
}

/** 是否可在账号安全页绑定邮箱 */
export function canBindEmail(email?: string | null): boolean {
  return isPlaceholderEmail(email)
}

/** 是否可换绑邮箱 */
export function canChangeEmail(email?: string | null): boolean {
  return !isPlaceholderEmail(email)
}
