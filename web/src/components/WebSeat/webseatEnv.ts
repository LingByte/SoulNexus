/** SoulNexus SIP web seat gateway (cmd/sip SIP_WEBSEAT_HTTP_ADDR) — Vite env. */
export function webSeatHttpBase(): string {
  const v = import.meta.env.VITE_SIP_WEBSEAT_HTTP_BASE
  return typeof v === 'string' ? v.trim().replace(/\/$/, '') : ''
}

/** Same secret as gateway SIP_WEBSEAT_WS_TOKEN; empty if gateway accepts any client. */
export function webSeatWsToken(): string {
  const v = import.meta.env.VITE_SIP_WEBSEAT_WS_TOKEN
  return typeof v === 'string' ? v.trim() : ''
}

export function buildWebSeatWebSocketURL(httpBase: string, token: string): string {
  const root = httpBase.replace(/\/$/, '')
  const base = root.endsWith('/') ? root : `${root}/`
  const u = new URL('/webseat/v1/ws', base)
  u.protocol = u.protocol === 'https:' ? 'wss:' : 'ws:'
  if (token) u.searchParams.set('token', token)
  return u.toString()
}
