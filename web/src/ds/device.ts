/**
 * 设备检测：PC vs Mobile
 * - SSR 安全：server 默认返回 'pc'
 * - 默认断点 768px，可由 DSProvider 覆盖
 */
import { useEffect, useState } from 'react'

export type DeviceKind = 'pc' | 'mobile'

const MOBILE_BREAKPOINT = 768
const UA_MOBILE_RE = /Mobi|Android|iPhone|iPad|iPod|Windows Phone|webOS|BlackBerry|IEMobile|Opera Mini/i

export function detectDevice(breakpoint: number = MOBILE_BREAKPOINT): DeviceKind {
    if (typeof window === 'undefined') return 'pc'
    const byUA = UA_MOBILE_RE.test(navigator.userAgent)
    const byWidth = window.innerWidth < breakpoint
    return (byUA || byWidth) ? 'mobile' : 'pc'
}

export function useDevice(breakpoint: number = MOBILE_BREAKPOINT): DeviceKind {
    const [device, setDevice] = useState<DeviceKind>(() => detectDevice(breakpoint))

    useEffect(() => {
        if (typeof window === 'undefined') return
        const mql = window.matchMedia(`(max-width: ${breakpoint - 1}px)`)
        const update = () => setDevice(detectDevice(breakpoint))
        update()
        mql.addEventListener?.('change', update)
        window.addEventListener('resize', update)
        return () => {
            mql.removeEventListener?.('change', update)
            window.removeEventListener('resize', update)
        }
    }, [breakpoint])

    return device
}

/** 静态判断（非 hook 场景，例如 toast / modal 静态方法） */
export const isMobile = () => detectDevice() === 'mobile'
