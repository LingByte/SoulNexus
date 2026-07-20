const AUTH_TOKEN_KEY = 'auth_token'

/** Keep legacy localStorage key in sync with zustand auth store (WebSocket, etc.). */
export function syncAuthToken(token: string | null | undefined) {
  if (typeof window === 'undefined') return
  if (token) {
    localStorage.setItem(AUTH_TOKEN_KEY, token)
  } else {
    localStorage.removeItem(AUTH_TOKEN_KEY)
  }
}

export function readAuthToken(): string | null {
  if (typeof window === 'undefined') return null
  return localStorage.getItem(AUTH_TOKEN_KEY)
}
