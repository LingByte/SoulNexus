/** 与后端 pkg/logger.HeaderXReqID 一致 */
export const X_REQ_ID_HEADER = 'X-Reqid'

/** 为每个 HTTP 请求生成可追踪 id（后端会回写同一响应头） */
export function genReqId(): string {
  if (typeof crypto !== 'undefined' && typeof crypto.randomUUID === 'function') {
    return crypto.randomUUID().replace(/-/g, '')
  }
  return `${Date.now().toString(36)}${Math.random().toString(36).slice(2, 12)}`
}
