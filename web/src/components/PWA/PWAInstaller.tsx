import { useState, useEffect, useRef, useCallback } from 'react'
import { Zap, WifiOff, Sparkles, X } from 'lucide-react'
import { cn } from '@/utils/cn.ts'
import { useTranslation } from '@/i18n'

interface BeforeInstallPromptEvent extends Event {
  prompt(): Promise<void>
  userChoice: Promise<{ outcome: 'accepted' | 'dismissed' }>
}

declare global {
  interface Navigator {
    standalone?: boolean
  }
}

interface PWAInstallerProps {
  className?: string
  showOnLoad?: boolean
  delay?: number
  /** After “稍后” / close, show again after this many ms (default 30 min). */
  snoozeMs?: number
  position?: 'top-left' | 'top-right' | 'bottom-left' | 'bottom-right'
}

const SNOOZE_STORAGE_KEY = 'lingecho_pwa_install_snooze_until'
const ANIM_MS = 320

const PWAInstaller = ({
  className = '',
  showOnLoad = true,
  delay = 3000,
  snoozeMs = 30 * 60 * 1000,
  position = 'bottom-right',
}: PWAInstallerProps) => {
  const { t } = useTranslation()
  const [deferredPrompt, setDeferredPrompt] = useState<BeforeInstallPromptEvent | null>(null)
  const [panelOpen, setPanelOpen] = useState(false)
  const [panelActive, setPanelActive] = useState(false)
  const [isInstalled, setIsInstalled] = useState(false)
  const [isInstalling, setIsInstalling] = useState(false)
  const showTimerRef = useRef<number | null>(null)
  const animTimerRef = useRef<number | null>(null)

  const clearTimers = useCallback(() => {
    if (showTimerRef.current != null) {
      window.clearTimeout(showTimerRef.current)
      showTimerRef.current = null
    }
    if (animTimerRef.current != null) {
      window.clearTimeout(animTimerRef.current)
      animTimerRef.current = null
    }
  }, [])

  const scheduleShow = useCallback(
    (initialDelay = delay) => {
      clearTimers()
      const snoozeUntil = Number(localStorage.getItem(SNOOZE_STORAGE_KEY) || 0)
      const now = Date.now()
      const wait = Math.max(initialDelay, snoozeUntil > now ? snoozeUntil - now : 0)
      showTimerRef.current = window.setTimeout(() => {
        setPanelOpen(true)
        requestAnimationFrame(() => {
          requestAnimationFrame(() => setPanelActive(true))
        })
      }, wait)
    },
    [clearTimers, delay],
  )

  const dismissPanel = useCallback(() => {
    setPanelActive(false)
    animTimerRef.current = window.setTimeout(() => {
      setPanelOpen(false)
      localStorage.setItem(SNOOZE_STORAGE_KEY, String(Date.now() + snoozeMs))
      scheduleShow(snoozeMs)
    }, ANIM_MS)
  }, [scheduleShow, snoozeMs])

  useEffect(() => {
    const checkInstalled = () => {
      if (window.matchMedia('(display-mode: standalone)').matches) {
        setIsInstalled(true)
        return
      }
      if (window.navigator.standalone === true) {
        setIsInstalled(true)
      }
    }
    checkInstalled()
  }, [])

  useEffect(() => {
    const handleBeforeInstallPrompt = (e: Event) => {
      e.preventDefault()
      setDeferredPrompt(e as BeforeInstallPromptEvent)
      if (showOnLoad) {
        scheduleShow()
      }
    }
    window.addEventListener('beforeinstallprompt', handleBeforeInstallPrompt)
    return () => {
      window.removeEventListener('beforeinstallprompt', handleBeforeInstallPrompt)
      clearTimers()
    }
  }, [showOnLoad, scheduleShow, clearTimers])

  useEffect(() => {
    const handleAppInstalled = () => {
      setIsInstalled(true)
      setPanelActive(false)
      setPanelOpen(false)
      setDeferredPrompt(null)
      localStorage.removeItem(SNOOZE_STORAGE_KEY)
      clearTimers()
    }
    window.addEventListener('appinstalled', handleAppInstalled)
    return () => window.removeEventListener('appinstalled', handleAppInstalled)
  }, [clearTimers])

  const handleInstall = async () => {
    if (!deferredPrompt) return
    setIsInstalling(true)
    try {
      await deferredPrompt.prompt()
      await deferredPrompt.userChoice
    } catch (error) {
      console.error('安装过程中出错:', error)
    } finally {
      setIsInstalling(false)
      setDeferredPrompt(null)
      setPanelActive(false)
      setPanelOpen(false)
      clearTimers()
    }
  }

  const getPositionStyles = () => {
    const base = 'fixed z-50'
    switch (position) {
      case 'top-left':
        return `${base} top-4 left-4`
      case 'top-right':
        return `${base} top-4 right-4`
      case 'bottom-left':
        return `${base} bottom-4 left-4`
      case 'bottom-right':
      default:
        return `${base} bottom-4 right-4`
    }
  }

  const slideFromRight = position === 'bottom-right' || position === 'top-right'

  if (isInstalled || !deferredPrompt || !panelOpen) return null

  return (
    <div
      className={cn('max-w-sm w-full pointer-events-none', getPositionStyles(), className)}
      aria-live="polite"
    >
      <div
        className={cn(
          'rounded-xl shadow-2xl overflow-hidden border pointer-events-auto',
          'transition-all ease-out',
          slideFromRight ? 'origin-right' : 'origin-left',
        )}
        style={{
          background: 'var(--color-bg-2)',
          borderColor: 'var(--color-border)',
          transitionDuration: `${ANIM_MS}ms`,
          transform: panelActive
            ? 'translateX(0) scale(1)'
            : slideFromRight
              ? 'translateX(calc(100% + 1rem)) scale(0.96)'
              : 'translateX(calc(-100% - 1rem)) scale(0.96)',
          opacity: panelActive ? 1 : 0,
        }}
      >
        <div
          className="p-4 flex items-center justify-between"
          style={{
            background: 'linear-gradient(110deg, #8B5CF6 0%, #A78BFA 45%, #6D28D9 100%)',
            color: '#fff',
          }}
        >
          <div>
            <h3 className="font-bold text-sm">{t('pwa.title')}</h3>
            <p className="text-xs opacity-90">{t('pwa.subtitle')}</p>
          </div>
          <button type="button" onClick={dismissPanel} className="text-white/70 hover:text-white p-1">
            <X className="w-4 h-4" />
          </button>
        </div>

        <div className="p-4">
          <div className="space-y-3">
            <div className="flex items-start gap-3">
              <div className="w-8 h-8 rounded-full flex items-center justify-center shrink-0 mt-0.5 bg-green-100">
                <Zap className="w-4 h-4 text-green-600" />
              </div>
              <div>
                <h4 className="font-semibold text-sm" style={{ color: 'var(--color-text-1)' }}>
                  {t('pwa.fastAccess')}
                </h4>
                <p className="text-xs" style={{ color: 'var(--color-text-3)' }}>
                  {t('pwa.fastAccessDesc')}
                </p>
              </div>
            </div>
            <div className="flex items-start gap-3">
              <div className="w-8 h-8 rounded-full flex items-center justify-center shrink-0 mt-0.5 bg-blue-100">
                <WifiOff className="w-4 h-4 text-blue-600" />
              </div>
              <div>
                <h4 className="font-semibold text-sm" style={{ color: 'var(--color-text-1)' }}>
                  {t('pwa.offlineUse')}
                </h4>
                <p className="text-xs" style={{ color: 'var(--color-text-3)' }}>
                  {t('pwa.offlineUseDesc')}
                </p>
              </div>
            </div>
            <div className="flex items-start gap-3">
              <div className="w-8 h-8 rounded-full flex items-center justify-center shrink-0 mt-0.5 bg-sky-100">
                <Sparkles className="w-4 h-4 text-sky-600" />
              </div>
              <div>
                <h4 className="font-semibold text-sm" style={{ color: 'var(--color-text-1)' }}>
                  {t('pwa.nativeExperience')}
                </h4>
                <p className="text-xs" style={{ color: 'var(--color-text-3)' }}>
                  {t('pwa.nativeExperienceDesc')}
                </p>
              </div>
            </div>
          </div>

          <div className="mt-4 flex gap-2">
            <button
              type="button"
              onClick={handleInstall}
              disabled={isInstalling}
              className={cn(
                'flex-1 font-semibold py-2.5 px-4 rounded-lg text-sm text-white transition-all',
                'disabled:opacity-50 disabled:cursor-not-allowed active:scale-[0.98]',
              )}
              style={{
                background: 'linear-gradient(110deg, #8B5CF6 0%, #A78BFA 100%)',
              }}
            >
              {isInstalling ? (
                <span className="inline-flex items-center justify-center gap-2">
                  <span className="w-4 h-4 border-2 border-white/30 border-t-white rounded-full animate-spin" />
                  {t('pwa.installing')}
                </span>
              ) : (
                t('pwa.installNow')
              )}
            </button>
            <button
              type="button"
              onClick={dismissPanel}
              className="px-4 py-2.5 font-medium text-sm transition-colors"
              style={{ color: 'var(--color-text-3)' }}
            >
              {t('pwa.later')}
            </button>
          </div>
        </div>
      </div>
    </div>
  )
}

export default PWAInstaller
