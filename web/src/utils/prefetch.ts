// 预加载工具，在用户可能访问前预加载数据
import { getOverviewConfig, getOrganizationStats } from '@/api/overview'
import { getGroup } from '@/api/group'
import { overviewCache } from './overviewCache'
import { requestDeduplication } from './requestDeduplication'

export const prefetch = {
  // 预加载组织概览数据
  prefetchOverview: async (organizationId: number): Promise<void> => {
    // 如果已有缓存，不需要预加载
    if (overviewCache.getConfig(organizationId) && overviewCache.getStats(organizationId)) {
      return
    }

    try {
      // 使用低优先级预加载（不阻塞主线程）
      if ('requestIdleCallback' in window) {
        requestIdleCallback(async () => {
          await Promise.allSettled([
            requestDeduplication.dedupe(
              `group-${organizationId}`,
              () => getGroup(organizationId)
            ),
            requestDeduplication.dedupe(
              `config-${organizationId}`,
              () => getOverviewConfig(organizationId)
            ),
            requestDeduplication.dedupe(
              `stats-${organizationId}`,
              () => getOrganizationStats(organizationId)
            )
          ])
        })
      } else {
        // 降级方案：延迟加载
        setTimeout(async () => {
          await Promise.allSettled([
            requestDeduplication.dedupe(
              `group-${organizationId}`,
              () => getGroup(organizationId)
            ),
            requestDeduplication.dedupe(
              `config-${organizationId}`,
              () => getOverviewConfig(organizationId)
            ),
            requestDeduplication.dedupe(
              `stats-${organizationId}`,
              () => getOrganizationStats(organizationId)
            )
          ])
        }, 1000)
      }
    } catch (err) {
      // 预加载失败不影响主流程
      console.debug('预加载失败:', err)
    }
  }
}





