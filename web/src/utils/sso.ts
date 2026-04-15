import { post } from '@/utils/request'
import { getUserServiceBaseURL } from '@/config/apiConfig'

const CALLBACK_PATH = '/auth/callback'

interface OIDCStatePayload {
  next?: string
  ts: number
}

interface OIDCTokenResponse {
  access_token: string
  token_type?: string
  expires_in?: number
  scope?: string
}

function toBase64Url(value: string): string {
  const encoded = btoa(unescape(encodeURIComponent(value)))
  return encoded.replace(/\+/g, '-').replace(/\//g, '_').replace(/=+$/g, '')
}

function fromBase64Url(value: string): string {
  const base64 = value.replace(/-/g, '+').replace(/_/g, '/')
  const padded = base64 + '='.repeat((4 - (base64.length % 4)) % 4)
  return decodeURIComponent(escape(atob(padded)))
}

function buildState(next?: string): string {
  const payload: OIDCStatePayload = {
    next,
    ts: Date.now(),
  }
  return toBase64Url(JSON.stringify(payload))
}

function parseState(state?: string | null): OIDCStatePayload | null {
  if (!state) {
    return null
  }
  try {
    return JSON.parse(fromBase64Url(state)) as OIDCStatePayload
  } catch {
    return null
  }
}

function sanitizeNext(next?: string | null): string {
  if (!next) {
    return '/assistants'
  }
  // Only allow same-origin relative path
  if (!next.startsWith('/')) {
    return '/assistants'
  }
  if (next.startsWith('//')) {
    return '/assistants'
  }
  return next
}

function getSSOLoginURL(): string {
  return (
    import.meta.env.VITE_SSO_LOGIN_URL ||
    `${getUserServiceBaseURL()}/auth/login`
  )
}

function getSSOAuthorizeURL(): string {
  return (
    import.meta.env.VITE_SSO_AUTHORIZE_URL ||
    `${getUserServiceBaseURL()}/auth/oidc/authorize`
  )
}

function getSSOExchangeURL(): string {
  return (
    import.meta.env.VITE_SSO_EXCHANGE_URL ||
    `${getUserServiceBaseURL()}/auth/oidc/exchange`
  )
}

function getSSOLogoutURL(): string {
  return (
    import.meta.env.VITE_SSO_LOGOUT_URL ||
    `${getUserServiceBaseURL()}/auth/logout`
  )
}

export function getOIDCClientId(): string {
  return import.meta.env.VITE_OIDC_CLIENT_ID || 'soulnexus-web'
}

export function getOIDCCallbackURL(): string {
  return `${window.location.origin}${CALLBACK_PATH}`
}

export function beginSSOLogin(next?: string): void {
  const hasAuthorizeURL = Boolean(import.meta.env.VITE_SSO_AUTHORIZE_URL)
  const targetURL = new URL(hasAuthorizeURL ? getSSOAuthorizeURL() : getSSOLoginURL())
  targetURL.searchParams.set('client_id', getOIDCClientId())
  targetURL.searchParams.set('redirect_uri', getOIDCCallbackURL())
  targetURL.searchParams.set('state', buildState(sanitizeNext(next)))
  window.location.assign(targetURL.toString())
}

export function resolveNextPathFromState(state?: string | null): string {
  const parsed = parseState(state)
  return sanitizeNext(parsed?.next)
}

export async function exchangeOIDCCode(code: string): Promise<OIDCTokenResponse> {
  const payload: Record<string, string> = {
    grant_type: 'authorization_code',
    code,
    client_id: getOIDCClientId(),
    redirect_uri: getOIDCCallbackURL(),
  }

  const response = await post<OIDCTokenResponse>(getSSOExchangeURL(), payload)

  if (response.code !== 200 || !response.data?.access_token) {
    throw new Error(response.msg || 'SSO token exchange failed')
  }

  return response.data
}

export function buildSSOLogoutURL(next?: string, token?: string): string {
  const target = new URL(getSSOLogoutURL())
  const nextURL = next || `${window.location.origin}/`
  target.searchParams.set('next', nextURL)
  if (token) {
    target.searchParams.set('token', token)
  }
  return target.toString()
}
