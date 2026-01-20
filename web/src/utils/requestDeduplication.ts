// 请求去重工具，避免同时发起多个相同的请求
interface PendingRequest {
  promise: Promise<any>
  timestamp: number
}

const pendingRequests = new Map<string, PendingRequest>()
const REQUEST_TIMEOUT = 30000 // 30秒超时

export const requestDeduplication = {
  // 去重请求
  dedupe: <T>(key: string, requestFn: () => Promise<T>): Promise<T> => {
    const existing = pendingRequests.get(key)
    
    // 如果存在未完成的请求且未超时，复用该请求
    if (existing && Date.now() - existing.timestamp < REQUEST_TIMEOUT) {
      return existing.promise as Promise<T>
    }
    
    // 创建新请求
    const promise = requestFn().finally(() => {
      // 请求完成后清除
      pendingRequests.delete(key)
    })
    
    pendingRequests.set(key, {
      promise,
      timestamp: Date.now()
    })
    
    return promise
  },
  
  // 清除所有待处理的请求
  clear: (): void => {
    pendingRequests.clear()
  },
  
  // 清除指定请求
  clearKey: (key: string): void => {
    pendingRequests.delete(key)
  }
}







