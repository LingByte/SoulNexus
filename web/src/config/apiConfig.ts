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
 * 获取API配置
 */
function getApiConfig(): ApiConfig {
  // 优先使用环境变量
  let apiBaseURL = import.meta.env.VITE_API_BASE_URL || 'http://localhost:7072/api'
  
  // 如果环境变量中有WebSocket URL，使用它；否则从API URL转换
  let wsBaseURL = import.meta.env.VITE_WS_BASE_URL
  if (!wsBaseURL) {
    // 从API URL提取host部分，转换为WebSocket URL
    const apiHost = apiBaseURL.replace(/^https?:\/\//, '').replace(/\/api.*$/, '')
    wsBaseURL = convertToWebSocketURL(apiBaseURL.split('/api')[0] || `http://${apiHost}`)
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
  
  // 如果wsBaseURL已经包含完整路径，直接使用
  if (wsBaseURL.includes('/api')) {
    return wsBaseURL + path.replace(/^\/api/, '')
  }
  
  // 否则从apiBaseURL提取路径部分
  const apiPath = apiBaseURL.replace(/^https?:\/\/[^\/]+/, '')
  return wsBaseURL + apiPath + path.replace(/^\/api/, '')
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

