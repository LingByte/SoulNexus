/** SoulNexus SIP web seat gateway (cmd/sip SIP_WEBSEAT_HTTP_ADDR) — Vite env. */
export function webSeatHttpBase(): string {
  const v = import.meta.env.VITE_SIP_WEBSEAT_HTTP_BASE
  const raw = typeof v === 'string' ? v.trim().replace(/\/$/, '') : ''
  if (!raw) {
    // Default to same-origin deployment (recommended behind reverse proxy).
    if (typeof window !== 'undefined' && window.location?.origin) {
      return window.location.origin.replace(/\/$/, '')
    }
    return ''
  }
  return normalizeLoopbackBase(normalizeBaseURL(raw))
}

/** Same secret as gateway SIP_WEBSEAT_WS_TOKEN; empty if gateway accepts any client. */
export function webSeatWsToken(): string {
  const v = import.meta.env.VITE_SIP_WEBSEAT_WS_TOKEN
  return typeof v === 'string' ? v.trim() : ''
}

/** Optional dedicated WS base (e.g. https://domain/ws) to separate WS and HTTP prefixes. */
export function webSeatWsBase(): string {
  const v = import.meta.env.VITE_SIP_WEBSEAT_WS_BASE
  const raw = typeof v === 'string' ? v.trim().replace(/\/$/, '') : ''
  if (!raw) return ''
  return normalizeLoopbackBase(normalizeBaseURL(raw))
}

export function buildWebSeatWebSocketURL(httpBase: string, token: string, wsBase?: string): string {
  const pickedBase = (wsBase || '').trim() || httpBase
  const root = normalizeBaseURL(pickedBase).replace(/\/$/, '')
  const base = root.endsWith('/') ? root : `${root}/`
  let u: URL
  try {
    // Important: do NOT start with "/" here, otherwise URL() discards base pathname (e.g. "/ws").
    u = new URL('webseat/v1/ws', base)
  } catch {
    // Last-resort fallback to same-origin if env base is malformed.
    const origin = typeof window !== 'undefined' && window.location?.origin ? window.location.origin : 'http://localhost'
    u = new URL('webseat/v1/ws', `${origin}/`)
  }
  u.protocol = u.protocol === 'https:' ? 'wss:' : 'ws:'
  if (token) u.searchParams.set('token', token)
  return u.toString()
}

// Accept host:port without scheme (e.g. "1.2.3.4:9080") and normalize to absolute URL.
function normalizeBaseURL(base: string): string {
  const s = (base || '').trim()
  if (!s) return s
  // Relative path (recommended with reverse proxy): "/webseat" -> "https://current-host/webseat"
  if (s.startsWith('/')) {
    if (typeof window !== 'undefined' && window.location?.origin) {
      return `${window.location.origin}${s}`.replace(/\/$/, '')
    }
    return s
  }
  // Protocol-relative URL: "//example.com" -> "https://example.com" on HTTPS pages
  if (s.startsWith('//')) {
    if (typeof window !== 'undefined' && window.location?.protocol) {
      return `${window.location.protocol}${s}`.replace(/\/$/, '')
    }
    return `https:${s}`.replace(/\/$/, '')
  }
  if (/^https?:\/\//i.test(s)) return s
  if (/^wss?:\/\//i.test(s)) return s.replace(/^ws/i, 'http')
  return `http://${s}`
}

// If frontend is opened remotely, a loopback gateway like http://127.0.0.1:9080
// points to the agent browser machine, not the server. Rewrite host to current page host.
function normalizeLoopbackBase(base: string): string {
  if (typeof window === 'undefined' || !window.location?.hostname) return base
  let u: URL
  try {
    u = new URL(base)
  } catch {
    return base
  }
  const h = u.hostname.toLowerCase()
  const isLoopback = h === '127.0.0.1' || h === 'localhost' || h === '::1'
  if (!isLoopback) return base
  const pageHost = window.location.hostname
  if (!pageHost || pageHost === 'localhost' || pageHost === '127.0.0.1') return base
  u.hostname = pageHost
  return u.toString().replace(/\/$/, '')
}
