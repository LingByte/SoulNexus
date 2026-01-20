// 概览配置缓存工具
interface CacheItem<T> {
  data: T
  timestamp: number
  organizationId: number
}

const CACHE_DURATION = 10 * 60 * 1000 // 10分钟缓存（增加缓存时间）
const STATS_CACHE_DURATION = 2 * 60 * 1000 // 统计数据2分钟缓存（统计数据变化更频繁）
const configCache = new Map<string, CacheItem<any>>()
const statsCache = new Map<string, CacheItem<any>>()

export const overviewCache = {
  // 获取配置缓存
  getConfig: (organizationId: number): any | null => {
    const key = `config-${organizationId}`
    const cached = configCache.get(key)
    if (cached && Date.now() - cached.timestamp < CACHE_DURATION) {
      return cached.data
    }
    return null
  },

  // 设置配置缓存
  setConfig: (organizationId: number, data: any): void => {
    const key = `config-${organizationId}`
    configCache.set(key, {
      data,
      timestamp: Date.now(),
      organizationId
    })
  },

  // 获取统计数据缓存
  getStats: (organizationId: number): any | null => {
    const key = `stats-${organizationId}`
    const cached = statsCache.get(key)
    if (cached && Date.now() - cached.timestamp < STATS_CACHE_DURATION) {
      return cached.data
    }
    return null
  },

  // 设置统计数据缓存
  setStats: (organizationId: number, data: any): void => {
    const key = `stats-${organizationId}`
    statsCache.set(key, {
      data,
      timestamp: Date.now(),
      organizationId
    })
  },

  // 清除指定组织的缓存
  clear: (organizationId: number): void => {
    configCache.delete(`config-${organizationId}`)
    statsCache.delete(`stats-${organizationId}`)
  },

  // 清除所有缓存
  clearAll: (): void => {
    configCache.clear()
    statsCache.clear()
  }
}

