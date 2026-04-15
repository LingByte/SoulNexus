/**
 * 站点配置缓存管理工具
 */

// 站点配置键名列表
export const SITE_CONFIG_KEYS = [
  'SITE_NAME',
  'SITE_DESCRIPTION', 
  'SITE_TERMS_URL',
  'SITE_URL',
  'SITE_LOGO_URL'
] as const

// 站点配置键名类型
export type SiteConfigKey = typeof SITE_CONFIG_KEYS[number]

/**
 * 检查是否是站点配置
 * @param key 配置键名
 * @returns 是否是站点配置
 */
export const isSiteConfig = (key: string): key is SiteConfigKey => {
  return SITE_CONFIG_KEYS.includes(key as SiteConfigKey)
}

/**
 * 清除站点配置缓存
 * 直接操作localStorage，不依赖React hooks
 */
export const clearSiteConfigCache = (): void => {
  try {
    localStorage.removeItem('lingstorage_site_config')
    localStorage.removeItem('lingstorage_site_config_timestamp')
    console.log('已清除站点配置缓存')
  } catch (error) {
    console.warn('清除站点配置缓存失败:', error)
  }
}

/**
 * 在配置更新后处理缓存清除
 * @param configKey 更新的配置键名
 * @param action 操作类型
 */
export const handleConfigCacheUpdate = (
  configKey: string, 
  action: 'create' | 'update' | 'delete' = 'update'
): void => {
  if (isSiteConfig(configKey)) {
    console.log(`${action === 'create' ? '创建' : action === 'delete' ? '删除' : '更新'}了站点配置 ${configKey}，清除缓存`)
    clearSiteConfigCache()
  }
}