// @refresh reset
import { createContext, useContext } from 'react'

export interface SiteConfig {
  SITE_NAME?: string
  SITE_DESCRIPTION?: string
  SITE_TERMS_URL?: string
  SITE_URL?: string
  SITE_LOGO_URL?: string
  SITE_FAVICON_URL?: string
  VOICE_CLONE_PROVIDER?: string
  VOICE_CLONE_LABEL?: string
  VOICEPRINT_PROVIDER?: string
  VOICEPRINT_LABEL?: string
  tenantSelfRegisterEnabled?: boolean
  githubOAuthEnabled?: boolean
  nluEnabled?: boolean
  smsLoginEnabled?: boolean
  deploymentMode?: 'community' | 'saas'
}

export interface SiteConfigContextType {
  config: SiteConfig
  loading: boolean
  ready: boolean
  error: Error | null
  refresh: () => Promise<void>
  clearCache: () => void
}

export const SiteConfigContext = createContext<SiteConfigContextType | undefined>(undefined)

export const useSiteConfig = () => {
  const context = useContext(SiteConfigContext)
  if (context === undefined) {
    throw new Error('useSiteConfig must be used within a SiteConfigProvider')
  }
  return context
}
