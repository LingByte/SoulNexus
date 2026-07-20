/**
 * API配置管理模块
 * 从环境变量统一读取后端地址配置
 */

interface ApiConfig {
  apiBaseURL: string
  wsBaseURL: string
  uploadsBaseURL: string
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
  // 如果已经有ws://或wss://，直接返回
  if (httpUrl.startsWith('ws://') || httpUrl.startsWith('wss://')) {
    return httpUrl
  }
  // 默认使用ws://
  return `ws://${httpUrl}`
}

/**
 * Normalized path: leading slash, no trailing slash, or '' when URL path is empty/root.
 */
function parseMountPath(p: string): string {
  let s = (p || '').trim()
  if (!s || s === '/') return ''
  s = s.replace(/\/+$/, '')
  if (!s.startsWith('/')) s = `/${s}`
  return s
}

/**
 * HTTP API mount path (e.g. /api) derived from VITE_API_BASE_URL.
 * If the value is an origin-only URL (https://host with no path), default to /api to match backend API_PREFIX.
 */
export function apiMountPathFromApiBase(apiBaseURL: string): string {
  const t = (apiBaseURL || '').trim() || '/api'
  if (t.startsWith('https://') || t.startsWith('http://')) {
    try {
      const norm = parseMountPath(new URL(t).pathname)
      return norm || '/api'
    } catch {
      return '/api'
    }
  }
  if (t.startsWith('/')) {
    const norm = parseMountPath(t)
    return norm || '/api'
  }
  if (t === 'api' || t.startsWith('api/')) {
    const norm = parseMountPath(`/${t}`)
    return norm || '/api'
  }
  return '/api'
}

/** Exported mount for callers that build paths under the same prefix as Gin. */
export function getApiMountPath(): string {
  return apiMountPathFromApiBase(getApiBaseURL())
}

/** Strip one leading mount prefix from a full API path (e.g. /api/lingecho/... → /lingecho/...). */
function stripLeadingMount(path: string, mount: string): string {
  const m = parseMountPath(mount) || '/api'
  const p = path.startsWith('/') ? path : `/${path}`
  if (p === m) return '/'
  const sep = m.endsWith('/') ? m : `${m}/`
  if (p.startsWith(sep)) {
    return p.slice(m.length)
  }
  if (p.startsWith('/api/')) {
    return p.slice('/api'.length)
  }
  if (p === '/api') return '/'
  return p
}

/** True only when ws URL pathname equals the API mount (e.g. wss://host/api), not longer paths like .../api/lingecho. */
function wsBasePathEqualsApiMount(wsBase: string, apiMount: string): boolean {
  try {
    const u = new URL(wsBase)
    return parseMountPath(u.pathname) === parseMountPath(apiMount)
  } catch {
    return false
  }
}

/**
 * Derive WebSocket origin (no path) from VITE_API_BASE_URL when VITE_WS_BASE_URL is unset.
 * Relative /api must use the page origin so HTTPS pages get wss://same-host, not ws://api.
 */
function deriveWsBaseFromApi(apiBaseURL: string): string {
  const trimmed = (apiBaseURL || '').trim()
  if (trimmed.startsWith('https://')) {
    const u = new URL(trimmed)
    const port = u.port ? `:${u.port}` : ''
    return `wss://${u.hostname}${port}`
  }
  if (trimmed.startsWith('http://')) {
    const u = new URL(trimmed)
    const port = u.port ? `:${u.port}` : ''
    return `ws://${u.hostname}${port}`
  }
  if (trimmed.startsWith('//') && typeof window !== 'undefined' && window.location?.protocol) {
    return convertToWebSocketURL(`${window.location.protocol}${trimmed}`)
  }
  // Path-only prefix (/api) or common typo "api" without leading slash
  if (trimmed.startsWith('/') || trimmed === 'api' || trimmed.startsWith('api/')) {
    if (typeof window !== 'undefined' && window.location?.origin) {
      return convertToWebSocketURL(window.location.origin)
    }
    return 'ws://localhost'
  }
  // Bare host[:port][/path] (legacy)
  try {
    const u = new URL(`http://${trimmed}`)
    const port = u.port ? `:${u.port}` : ''
    return convertToWebSocketURL(`${u.protocol}//${u.hostname}${port}`)
  } catch {
    return convertToWebSocketURL(trimmed)
  }
}

/**
 * 获取API配置
 */
function getApiConfig(): ApiConfig {
  // 优先使用环境变量；如果未配置，则使用 /api 作为基础前缀
  // 这个值会作为 BACKEND_BASE（如 /api），由各个服务自行在其后追加路径，避免出现 /api/api 的情况
  let apiBaseURL = import.meta.env.VITE_API_BASE_URL || '/api'
  
  // 如果环境变量中有WebSocket URL，使用它；否则从API URL转换
  let wsBaseURL = import.meta.env.VITE_WS_BASE_URL
  if (!wsBaseURL) {
    wsBaseURL = deriveWsBaseFromApi(apiBaseURL)
  }
  
  const uploadsBaseURL = import.meta.env.VITE_UPLOADS_BASE_URL || apiBaseURL.replace('/api', '/uploads')

  return {
    apiBaseURL,
    wsBaseURL,
    uploadsBaseURL,
  }
}

// 缓存配置
let cachedConfig: ApiConfig | null = null

/**
 * 获取配置（带缓存）
 */
export function getConfig(): ApiConfig {
  if (!cachedConfig) {
    cachedConfig = getApiConfig()
  }
  return cachedConfig
}

/**
 * 获取API基础URL
 */
export function getApiBaseURL(): string {
  return getConfig().apiBaseURL
}

/**
 * 获取WebSocket基础URL
 */
export function getWebSocketBaseURL(): string {
  return getConfig().wsBaseURL
}

/**
 * 构建完整的WebSocket URL
 * @param path API路径，例如 '/api/voice/websocket' 或 '/api/chat/call'
 */
export function buildWebSocketURL(path: string): string {
  const wsBaseURL = getWebSocketBaseURL()
  const apiBaseURL = getApiBaseURL()
  const mount = apiMountPathFromApiBase(apiBaseURL)
  const suffix = stripLeadingMount(path, mount)

  // 仅当 WS 基址的 path 与 API 挂载点完全一致（如 wss://host/api）时才去掉 path 里的 mount，避免
  // wss://host/api/lingecho/... 被误判后只剥一层 /api，拼成 .../lingecho/... 落在静态站（握手 200）。
  if (wsBasePathEqualsApiMount(wsBaseURL, mount)) {
    return `${wsBaseURL.replace(/\/$/, '')}${suffix}`
  }

  return `${wsBaseURL.replace(/\/$/, '')}${mount}${suffix}`
}

/**
 * 获取上传文件基础URL
 */
export function getUploadsBaseURL(): string {
  return getConfig().uploadsBaseURL
}

/**
 * 清除配置缓存（用于重新加载配置）
 */
export function clearConfigCache(): void {
  cachedConfig = null
}

