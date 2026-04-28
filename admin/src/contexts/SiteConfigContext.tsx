import { createContext, useContext, useState, useEffect, ReactNode } from 'react'
import { type SiteConfig } from '@/services/adminApi'
import { buildLogoUrl } from '@/utils/logoUrl'

interface SiteConfigContextType {
  config: SiteConfig | null
  loading: boolean
  refresh: () => Promise<void>
  clearCache: () => void
}

const SiteConfigContext = createContext<SiteConfigContextType | undefined>(undefined)

export const SiteConfigProvider = ({ children }: { children: ReactNode }) => {
  const [config, setConfig] = useState<SiteConfig | null>(null)
  const [loading, setLoading] = useState(true)
  // 应用配置到页面
  const applyConfigToPage = (siteConfig: SiteConfig) => {
    // 更新页面标题
    if (siteConfig.SITE_NAME) {
      document.title = siteConfig.SITE_NAME
      // 同时更新 meta description
      const metaDescription = document.querySelector('meta[name="description"]')
      if (metaDescription) {
        metaDescription.setAttribute('content', siteConfig.SITE_NAME)
      }
      // 更新 apple-mobile-web-app-title
      const appleTitle = document.querySelector('meta[name="apple-mobile-web-app-title"]')
      if (appleTitle) {
        appleTitle.setAttribute('content', siteConfig.SITE_NAME)
      }
    }
    
    // 更新 favicon
    if (siteConfig.SITE_LOGO_URL) {
      const logoUrl = buildLogoUrl(siteConfig.SITE_LOGO_URL)
      
      // 更新 favicon
      let faviconLink = document.querySelector("link[rel~='icon']") as HTMLLinkElement
      if (!faviconLink) {
        faviconLink = document.createElement('link')
        faviconLink.rel = 'icon'
        document.head.appendChild(faviconLink)
      }
      faviconLink.href = logoUrl
      
      // 更新 apple-touch-icon
      let appleIcon = document.querySelector("link[rel='apple-touch-icon']") as HTMLLinkElement
      if (!appleIcon) {
        appleIcon = document.createElement('link')
        appleIcon.rel = 'apple-touch-icon'
        document.head.appendChild(appleIcon)
      }
      appleIcon.href = logoUrl
    }
  }

  // 获取默认配置
  const getDefaultConfig = (): SiteConfig => ({
    SITE_NAME: 'SoulNexus',
    SITE_DESCRIPTION: 'SoulNexus',
    SITE_TERMS_URL: '',
    SITE_URL: '',
    SITE_LOGO_URL: '',
    SHOULD_UPGRADE_DB: false,
    CENSOR_ENABLED: false,
  })

  const fetchConfig = async (_forceRefresh = false) => {
    // 后端暂无 site-config 接口，直接使用默认值
    const defaultConfig = getDefaultConfig()
    setConfig(defaultConfig)
    applyConfigToPage(defaultConfig)
    setLoading(false)
  }

  // 清除缓存
  const clearCache = () => {
    console.log('清除站点配置缓存')
  }

  useEffect(() => {
    fetchConfig()
  }, [])

    return (
    <SiteConfigContext.Provider value={{ 
      config, 
      loading,
      refresh: () => fetchConfig(true), // 强制刷新
      clearCache 
    }}>
      {children}
    </SiteConfigContext.Provider>
  )
}

export const useSiteConfig = () => {
  const context = useContext(SiteConfigContext)
  if (context === undefined) {
    throw new Error('useSiteConfig must be used within a SiteConfigProvider')
  }
  return context
}
