import { useState, useEffect, ReactNode, useCallback, useRef } from 'react'
import { useLocalStorage } from '@/hooks/useLocalStorage'
import { getBrandTintedImageDataUrl } from '@/utils/brandLogo'
import { fetchSystemInit } from '@/api/systemInit'
import { isApiSuccess } from '@/api/types'
import { SiteConfigContext, type SiteConfig } from '@/contexts/siteConfigCore'

const SITE_CONFIG_CACHE_KEY = 'lingstorage_site_config'
const SITE_CONFIG_CACHE_TIMESTAMP_KEY = 'lingstorage_site_config_timestamp'
const CACHE_DURATION = 24 * 60 * 60 * 1000

const getDefaultConfig = (): SiteConfig => ({
  SITE_NAME: 'SoulNexus',
  SITE_DESCRIPTION: '管理后台登录',
  SITE_TERMS_URL: '',
  SITE_URL: '',
  SITE_LOGO_URL: '/icon-lingyu.png',
})

function mergeSiteConfig(raw: Partial<SiteConfig> | null | undefined): SiteConfig {
  const d = getDefaultConfig()
  if (!raw) return d
  return {
    SITE_NAME: raw.SITE_NAME?.trim() || d.SITE_NAME,
    SITE_DESCRIPTION: raw.SITE_DESCRIPTION?.trim() || d.SITE_DESCRIPTION,
    SITE_URL: raw.SITE_URL?.trim() || d.SITE_URL,
    SITE_LOGO_URL: raw.SITE_LOGO_URL?.trim() || d.SITE_LOGO_URL,
    SITE_TERMS_URL: raw.SITE_TERMS_URL?.trim() || d.SITE_TERMS_URL,
    SITE_FAVICON_URL: raw.SITE_FAVICON_URL,
    VOICE_CLONE_PROVIDER: raw.VOICE_CLONE_PROVIDER?.trim() || '',
    VOICE_CLONE_LABEL: raw.VOICE_CLONE_LABEL?.trim() || '',
    VOICEPRINT_PROVIDER: raw.VOICEPRINT_PROVIDER?.trim() || '',
    VOICEPRINT_LABEL: raw.VOICEPRINT_LABEL?.trim() || '',
    tenantSelfRegisterEnabled: Boolean(raw.tenantSelfRegisterEnabled),
    githubOAuthEnabled: Boolean(raw.githubOAuthEnabled),
    nluEnabled: Boolean(raw.nluEnabled),
    smsLoginEnabled: Boolean(raw.smsLoginEnabled),
    deploymentMode: raw.deploymentMode === 'community' ? 'community' : 'saas',
  }
}

export const SiteConfigProvider = ({ children }: { children: ReactNode }) => {
  const [cachedConfig, setCachedConfig] = useLocalStorage<SiteConfig | null>(SITE_CONFIG_CACHE_KEY, null)
  const [cacheTimestamp, setCacheTimestamp] = useLocalStorage<number>(SITE_CONFIG_CACHE_TIMESTAMP_KEY, 0)

  const [config, setConfig] = useState<SiteConfig>(() =>
    mergeSiteConfig(cachedConfig ?? getDefaultConfig()),
  )
  const [loading, setLoading] = useState(true)
  const [ready, setReady] = useState(false)
  const [error, setError] = useState<Error | null>(null)
  const fetchedRef = useRef(false)

  const isCacheValid = useCallback(() => {
    if (!cachedConfig || !cacheTimestamp) return false
    return Date.now() - cacheTimestamp < CACHE_DURATION
  }, [cachedConfig, cacheTimestamp])

  const applyThemeHeadLogo = useCallback((logoUrl?: string) => {
    const src = logoUrl || config.SITE_LOGO_URL || '/icon-lingyu.png'

    void getBrandTintedImageDataUrl(src)
      .then((tintedUrl) => {
        let faviconLink = document.querySelector("link[rel~='icon']") as HTMLLinkElement | null
        if (!faviconLink) {
          faviconLink = document.createElement('link')
          faviconLink.rel = 'icon'
          document.head.appendChild(faviconLink)
        }
        faviconLink.type = 'image/png'
        faviconLink.href = tintedUrl

        let appleIcon = document.querySelector("link[rel='apple-touch-icon']") as HTMLLinkElement | null
        if (!appleIcon) {
          appleIcon = document.createElement('link')
          appleIcon.rel = 'apple-touch-icon'
          document.head.appendChild(appleIcon)
        }
        appleIcon.href = tintedUrl
      })
      .catch(() => {
        let faviconLink = document.querySelector("link[rel~='icon']") as HTMLLinkElement | null
        if (!faviconLink) {
          faviconLink = document.createElement('link')
          faviconLink.rel = 'icon'
          document.head.appendChild(faviconLink)
        }
        faviconLink.href = src

        let appleIcon = document.querySelector("link[rel='apple-touch-icon']") as HTMLLinkElement | null
        if (!appleIcon) {
          appleIcon = document.createElement('link')
          appleIcon.rel = 'apple-touch-icon'
          document.head.appendChild(appleIcon)
        }
        appleIcon.href = src
      })
  }, [config.SITE_LOGO_URL])

  const applyConfigToPage = useCallback(
    (siteConfig: SiteConfig) => {
      if (siteConfig.SITE_NAME) {
        document.title = siteConfig.SITE_NAME
        const metaDescription = document.querySelector('meta[name="description"]')
        if (metaDescription) {
          metaDescription.setAttribute('content', siteConfig.SITE_DESCRIPTION || siteConfig.SITE_NAME)
        }
        const appleTitle = document.querySelector('meta[name="apple-mobile-web-app-title"]')
        if (appleTitle) {
          appleTitle.setAttribute('content', siteConfig.SITE_NAME)
        }
      }
      applyThemeHeadLogo(siteConfig.SITE_LOGO_URL)
    },
    [applyThemeHeadLogo],
  )

  const commitConfig = useCallback(
    (siteConfig: SiteConfig, persist = true) => {
      const normalized = mergeSiteConfig(siteConfig)
      setConfig(normalized)
      applyConfigToPage(normalized)
      if (persist) {
        setCachedConfig(normalized)
        setCacheTimestamp(Date.now())
      }
      setReady(true)
    },
    [applyConfigToPage, setCachedConfig, setCacheTimestamp],
  )

  const fetchConfig = useCallback(
    async (forceRefresh = false) => {
      try {
        setLoading(true)
        setError(null)

        if (!forceRefresh && isCacheValid() && cachedConfig) {
          commitConfig(cachedConfig, false)
          return
        }

        const res = await fetchSystemInit()
        if (isApiSuccess(res.code) && res.data) {
          commitConfig({
            SITE_NAME: res.data.SITE_NAME,
            SITE_DESCRIPTION: res.data.SITE_DESCRIPTION,
            SITE_URL: res.data.SITE_URL,
            SITE_LOGO_URL: res.data.SITE_LOGO_URL,
            SITE_TERMS_URL: res.data.SITE_TERMS_URL,
            VOICE_CLONE_PROVIDER: res.data.VOICE_CLONE_PROVIDER,
            VOICE_CLONE_LABEL: res.data.VOICE_CLONE_LABEL,
            VOICEPRINT_PROVIDER: res.data.VOICEPRINT_PROVIDER,
            VOICEPRINT_LABEL: res.data.VOICEPRINT_LABEL,
            tenantSelfRegisterEnabled: res.data.tenantSelfRegisterEnabled,
            githubOAuthEnabled: res.data.githubOAuthEnabled,
            nluEnabled: res.data.nluEnabled,
            smsLoginEnabled: res.data.smsLoginEnabled,
            deploymentMode: res.data.deploymentMode === 'community' ? 'community' : 'saas',
          })
        } else {
          throw new Error(res.msg || 'Failed to load site config')
        }
      } catch (err) {
        console.error('获取站点配置失败:', err)
        setError(err instanceof Error ? err : new Error('Failed to load site config'))
        if (cachedConfig) {
          commitConfig(cachedConfig, false)
        } else {
          commitConfig(getDefaultConfig(), false)
        }
      } finally {
        setLoading(false)
      }
    },
    [cachedConfig, commitConfig, isCacheValid],
  )

  const clearCache = () => {
    setCachedConfig(null)
    setCacheTimestamp(0)
  }

  useEffect(() => {
    if (fetchedRef.current) return
    fetchedRef.current = true
    if (cachedConfig) {
      commitConfig(cachedConfig, false)
    } else {
      commitConfig(getDefaultConfig(), false)
    }
    void fetchConfig(false)
  }, [cachedConfig, commitConfig, fetchConfig])

  useEffect(() => {
    const observer = new MutationObserver(() => {
      applyThemeHeadLogo()
    })
    observer.observe(document.documentElement, { attributes: true, attributeFilter: ['class'] })

    const media = window.matchMedia('(prefers-color-scheme: dark)')
    const onMediaChange = () => applyThemeHeadLogo()
    media.addEventListener('change', onMediaChange)

    return () => {
      observer.disconnect()
      media.removeEventListener('change', onMediaChange)
    }
  }, [applyThemeHeadLogo])

  return (
    <SiteConfigContext.Provider
      value={{
        config,
        loading,
        ready,
        error,
        refresh: () => fetchConfig(true),
        clearCache,
      }}
    >
      {children}
    </SiteConfigContext.Provider>
  )
}
