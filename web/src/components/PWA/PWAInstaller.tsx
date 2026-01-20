import { useState, useEffect } from 'react'
import { motion, AnimatePresence } from 'framer-motion'
import { Smartphone, Zap, Bell, Download, X } from 'lucide-react'
import { cn } from '@/utils/cn.ts'

interface BeforeInstallPromptEvent extends Event {
  prompt(): Promise<void>
  userChoice: Promise<{ outcome: 'accepted' | 'dismissed' }>
}

// 扩展 Navigator 接口以包含 iOS Safari 的 standalone 属性
declare global {
  interface Navigator {
    standalone?: boolean
  }
}

interface PWAInstallerProps {
  className?: string
  showOnLoad?: boolean
  delay?: number
  position?: 'top-left' | 'top-right' | 'bottom-left' | 'bottom-right'
}

const PWAInstaller = ({
                        className = "",
                        showOnLoad = true,
                        delay = 3000,
                        position = 'bottom-right'
                      }: PWAInstallerProps) => {
  const [deferredPrompt, setDeferredPrompt] = useState<BeforeInstallPromptEvent | null>(null)
  const [isVisible, setIsVisible] = useState(false)
  const [isInstalled, setIsInstalled] = useState(false)
  const [isInstalling, setIsInstalling] = useState(false)

  // 检查是否已安装
  useEffect(() => {
    const checkInstalled = () => {
      // 检查是否在独立模式下运行（已安装）
      if (window.matchMedia('(display-mode: standalone)').matches) {
        setIsInstalled(true)
        return
      }

      // 检查是否在iOS Safari中
      if (window.navigator.standalone === true) {
        setIsInstalled(true)
        return
      }
    }

    checkInstalled()
  }, [])

  // 监听安装提示事件
  useEffect(() => {
    const handleBeforeInstallPrompt = (e: Event) => {
      e.preventDefault()
      setDeferredPrompt(e as BeforeInstallPromptEvent)

      if (showOnLoad) {
        setTimeout(() => {
          setIsVisible(true)
        }, delay)
      }
    }

    window.addEventListener('beforeinstallprompt', handleBeforeInstallPrompt)

    return () => {
      window.removeEventListener('beforeinstallprompt', handleBeforeInstallPrompt)
    }
  }, [showOnLoad, delay])

  // 监听安装完成事件
  useEffect(() => {
    const handleAppInstalled = () => {
      setIsInstalled(true)
      setIsVisible(false)
      setDeferredPrompt(null)
    }

    window.addEventListener('appinstalled', handleAppInstalled)

    return () => {
      window.removeEventListener('appinstalled', handleAppInstalled)
    }
  }, [])

  // 处理安装
  const handleInstall = async () => {
    if (!deferredPrompt) return

    setIsInstalling(true)

    try {
      await deferredPrompt.prompt()
      const { outcome } = await deferredPrompt.userChoice

      if (outcome === 'accepted') {
        console.log('用户接受了安装提示')
      } else {
        console.log('用户拒绝了安装提示')
      }
    } catch (error) {
      console.error('安装过程中出错:', error)
    } finally {
      setIsInstalling(false)
      setDeferredPrompt(null)
      setIsVisible(false)
    }
  }

  // 处理关闭
  const handleClose = () => {
    setIsVisible(false)
  }

  // 获取位置样式
  const getPositionStyles = () => {
    const baseStyles = 'fixed z-50'

    switch (position) {
      case 'top-left':
        return `${baseStyles} top-4 left-4`
      case 'top-right':
        return `${baseStyles} top-4 right-4`
      case 'bottom-left':
        return `${baseStyles} bottom-4 left-4`
      case 'bottom-right':
      default:
        return `${baseStyles} bottom-4 right-4`
    }
  }

  // 如果已安装或没有安装提示，不显示
  if (isInstalled || !deferredPrompt) return null

  return (
      <AnimatePresence>
        {isVisible && (
            <motion.div
                className={cn(
                    'max-w-sm w-full',
                    getPositionStyles(),
                    className
                )}
                initial={{ opacity: 0, scale: 0.8, y: 20 }}
                animate={{ opacity: 1, scale: 1, y: 0 }}
                exit={{ opacity: 0, scale: 0.8, y: 20 }}
                transition={{ duration: 0.3, ease: "easeOut" }}
            >
              <div className="bg-white border border-gray-200 rounded-xl shadow-2xl overflow-hidden">
                {/* 头部 */}
                <div className="bg-gradient-to-r from-blue-500 to-purple-600 p-4">
                  <div className="flex items-center justify-between">
                    <div className="flex items-center gap-3">
                      <div className="w-10 h-10 bg-white/20 rounded-lg flex items-center justify-center">
                        <Smartphone className="w-6 h-6" />
                      </div>
                      <div>
                        <h3 className="font-bold text-sm">安装应用</h3>
                        <p className="text-xs">获得更好的体验</p>
                      </div>
                    </div>
                    <button
                        onClick={handleClose}
                        className="text-white/60 hover:text-white transition-colors p-1"
                    >
                      <X className="w-4 h-4" />
                    </button>
                  </div>
                </div>

                {/* 内容 */}
                <div className="p-4">
                  <div className="space-y-3">
                    <div className="flex items-start gap-3">
                      <div className="w-8 h-8 bg-green-100 rounded-full flex items-center justify-center flex-shrink-0 mt-0.5">
                        <Zap className="w-4 h-4 text-green-600" />
                      </div>
                      <div>
                        <h4 className="font-semibold text-sm text-gray-900">更快访问</h4>
                        <p className="text-xs text-gray-600">无需打开浏览器，直接启动应用</p>
                      </div>
                    </div>

                    <div className="flex items-start gap-3">
                      <div className="w-8 h-8 bg-blue-100 rounded-full flex items-center justify-center flex-shrink-0 mt-0.5">
                        <Bell className="w-4 h-4 text-blue-600" />
                      </div>
                      <div>
                        <h4 className="font-semibold text-sm text-gray-900">离线使用</h4>
                        <p className="text-xs text-gray-600">支持离线访问，随时随地使用</p>
                      </div>
                    </div>

                    <div className="flex items-start gap-3">
                      <div className="w-8 h-8 bg-purple-100 rounded-full flex items-center justify-center flex-shrink-0 mt-0.5">
                        <Download className="w-4 h-4 text-purple-600" />
                      </div>
                      <div>
                        <h4 className="font-semibold text-sm text-gray-900">原生体验</h4>
                        <p className="text-xs text-gray-600">享受接近原生应用的流畅体验</p>
                      </div>
                    </div>
                  </div>

                  {/* 按钮 */}
                  <div className="mt-4 flex gap-2">
                    <button
                        onClick={handleInstall}
                        disabled={isInstalling}
                        className={cn(
                            'flex-1 bg-gradient-to-r from-blue-500 to-purple-600 font-semibold py-2.5 px-4 rounded-lg text-sm transition-all duration-200',
                            'hover:from-blue-600 hover:to-purple-700 hover:shadow-lg',
                            'disabled:opacity-50 disabled:cursor-not-allowed',
                            'active:scale-95'
                        )}
                    >
                      {isInstalling ? (
                          <div className="flex items-center justify-center gap-2">
                            <div className="w-4 h-4 border-2 border-white/30 border-t-white rounded-full animate-spin"></div>
                            <span>安装中...</span>
                          </div>
                      ) : (
                          '立即安装'
                      )}
                    </button>
                    <button
                        onClick={handleClose}
                        className="px-4 py-2.5 text-gray-500 hover:text-gray-700 font-medium text-sm transition-colors"
                    >
                      稍后
                    </button>
                  </div>
                </div>
              </div>
            </motion.div>
        )}
      </AnimatePresence>
  )
}

export default PWAInstaller
