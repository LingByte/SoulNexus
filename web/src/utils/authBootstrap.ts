/** Consume one-time auth_token / refresh_token from URL after OAuth redirect. */
export function consumeAuthTokenFromURL(): string | null {
  if (typeof window === 'undefined') {
    return null
  }
  const params = new URLSearchParams(window.location.search)
  const token = params.get('auth_token')?.trim()
  const refreshToken = params.get('refresh_token')?.trim()
  if (!token && !refreshToken) {
    return null
  }
  if (refreshToken) {
    localStorage.setItem('refresh_token', refreshToken)
  }
  params.delete('auth_token')
  params.delete('refresh_token')
  const query = params.toString()
  const nextPath = `${window.location.pathname}${query ? `?${query}` : ''}${window.location.hash}`
  window.history.replaceState({}, '', nextPath)
  return token || null
}
