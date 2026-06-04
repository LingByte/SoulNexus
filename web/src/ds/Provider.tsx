/**
 * DSProvider - SoulNexus 双端设计系统根 Provider
 * - 同时挂载 ArcoDesign PC ConfigProvider 与 Mobile ContextProvider
 * - 桥接 themeStore (light/dark) 到 Arco 的 darkMode
 * - 桥接 i18nStore 到 Arco 的 locale
 */
import React, { useEffect, useMemo } from 'react'
import ArcoConfigProvider from '@arco-design/web-react/es/ConfigProvider'
import arcoZhCN from '@arco-design/web-react/es/locale/zh-CN'
import arcoEnUS from '@arco-design/web-react/es/locale/en-US'
import ArcoMobileContextProvider from '@arco-design/mobile-react/esm/context-provider'
import { useThemeStore } from '@/stores/themeStore'
import { useI18nStore } from '@/stores/i18nStore'

interface DSProviderProps {
    children: React.ReactNode
}

const ARCO_DARK_BODY_ATTR = 'arco-theme'

/** 将 themeStore.isDark 同步到 body 的 arco-theme 属性，让 PC 端 Arco 自动切换暗黑色板 */
function useSyncArcoTheme(isDark: boolean) {
    useEffect(() => {
        if (typeof document === 'undefined') return
        if (isDark) document.body.setAttribute(ARCO_DARK_BODY_ATTR, 'dark')
        else document.body.removeAttribute(ARCO_DARK_BODY_ATTR)
    }, [isDark])
}

function mapLocaleToArco(lng?: string) {
    if (!lng) return arcoZhCN
    if (lng.startsWith('zh')) return arcoZhCN
    return arcoEnUS
}

export function DSProvider({ children }: DSProviderProps) {
    const isDark = useThemeStore(s => s.isDark)
    const language = useI18nStore(s => s.language)

    useSyncArcoTheme(isDark)

    const arcoLocale = useMemo(() => mapLocaleToArco(language), [language])

    return (
        <ArcoConfigProvider locale={arcoLocale}>
            <ArcoMobileContextProvider isDarkMode={isDark}>
                {children as any}
            </ArcoMobileContextProvider>
        </ArcoConfigProvider>
    )
}

export default DSProvider
