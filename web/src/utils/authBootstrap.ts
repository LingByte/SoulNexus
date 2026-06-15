/** Consume one-time auth_token from URL after cross-origin signup/login redirect. */
export function consumeAuthTokenFromURL(): string | null {
  if (typeof window === 'undefined') {
    return null
  }
  const params = new URLSearchParams(window.location.search)
  const token = params.get('auth_token')?.trim()
  if (!token) {
    return null
  }
  params.delete('auth_token')
  const query = params.toString()
  const nextPath = `${window.location.pathname}${query ? `?${query}` : ''}${window.location.hash}`
  window.history.replaceState({}, '', nextPath)
  return token
}
