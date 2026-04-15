import { getApiBaseURL } from '@/config/apiConfig'

/**
 * 构建完整的 logo URL
 * 如果 logoUrl 是相对路径，则构建完整的后端 URL
 * 如果 logoUrl 是完整 URL，则直接返回
 */
export function buildLogoUrl(logoUrl: string): string {
  if (!logoUrl) {
    return '/static/img/favicon.png'
  }

  // 如果是完整 URL（http:// 或 https:// 开头），直接返回
  if (logoUrl.startsWith('http://') || logoUrl.startsWith('https://')) {
    return logoUrl
  }

  // 如果是相对路径，构建完整的后端 URL
  if (logoUrl.startsWith('/')) {
    // 从 BACKEND_BASE 中提取基础 URL（去掉 /api 前缀）
    const backendBase = getApiBaseURL()
    const baseUrl = backendBase.replace(/\/api$/, '')
    return `${baseUrl}${logoUrl}`
  }

  // 其他情况，直接返回
  return logoUrl
}
