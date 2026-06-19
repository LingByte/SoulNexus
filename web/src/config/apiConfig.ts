/**
 * API配置管理模块
 * 从环境变量统一读取后端地址配置
 */

interface ApiConfig {
  apiBaseURL: string
  wsBaseURL: string
}

/**
 * 将HTTP URL转换为WebSocket URL
 */
function convertToWebSocketURL(httpUrl: string): string {
  if (httpUrl.startsWith('https://')) {
    return httpUrl.replace('https://', 'wss://')
  } else if (httpUrl.startsWith('http://')) {
    return httpUrl.replace('http://', 'ws://')
  }
  if (httpUrl.startsWith('ws://') || httpUrl.startsWith('wss://')) {
    return httpUrl
  }
  return `ws://${httpUrl}`
}

/**
 * 获取API配置
 */
function getApiConfig(): ApiConfig {
  const apiBaseURL = import.meta.env.VITE_API_BASE_URL || '/api'

  let wsBaseURL = import.meta.env.VITE_WS_BASE_URL
  if (!wsBaseURL) {
    if (apiBaseURL.startsWith('/')) {
      const proto = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
      wsBaseURL = `${proto}//${window.location.host}`
    } else {
      wsBaseURL = convertToWebSocketURL(apiBaseURL.split('/api')[0] || apiBaseURL)
    }
  }

  return {
    apiBaseURL,
    wsBaseURL,
  }
}

let cachedConfig: ApiConfig | null = null

export function getConfig(): ApiConfig {
  if (!cachedConfig) {
    cachedConfig = getApiConfig()
  }
  return cachedConfig
}

export function getApiBaseURL(): string {
  return getConfig().apiBaseURL
}

/** 站点根（无 `/api` 后缀），用于打开服务端 HTML 页面等。 */
export function getApiOrigin(): string {
  const base = getApiBaseURL().trim().replace(/\/+$/, '')
  if (base.startsWith('/')) {
    return window.location.origin
  }
  if (base.endsWith('/api')) {
    return base.slice(0, -4).replace(/\/+$/, '')
  }
  return base.replace(/\/+$/, '')
}

export function getAccountDeletionRevokePageURL(email?: string): string {
  const origin = getApiOrigin()
  const u = new URL('/login/revoke-account-deletion', origin.endsWith('/') ? origin : `${origin}/`)
  const em = email?.trim()
  if (em) u.searchParams.set('email', em)
  return u.toString()
}

export function isAccountDeletionRevokeStandalonePage(): boolean {
  try {
    const expected = new URL(getApiOrigin())
    return (
      window.location.origin === expected.origin &&
      window.location.pathname === '/login/revoke-account-deletion'
    )
  } catch {
    return false
  }
}

export function getWebSocketBaseURL(): string {
  return getConfig().wsBaseURL
}

export function buildWebSocketURL(path: string): string {
  const wsBaseURL = getWebSocketBaseURL()
  const apiBaseURL = getApiBaseURL()

  if (wsBaseURL.includes('/api')) {
    return wsBaseURL + path.replace(/^\/api/, '')
  }

  const apiPath = apiBaseURL.replace(/^https?:\/\/[^\/]+/, '')
  return wsBaseURL + apiPath + path.replace(/^\/api/, '')
}

export function clearConfigCache(): void {
  cachedConfig = null
}
