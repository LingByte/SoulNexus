import { useState, useEffect } from 'react'
import { motion, AnimatePresence } from 'framer-motion'
import { X, Cookie } from 'lucide-react'
import { Link } from 'react-router-dom'
import { useI18nStore } from '@/stores/i18nStore'

const COOKIE_CONSENT_KEY = 'cookie_consent_accepted'

const CookieConsent = () => {
  const [showConsent, setShowConsent] = useState(false)
  const { t } = useI18nStore()

  useEffect(() => {
    // 检查用户是否已经同意过
    const hasConsented = localStorage.getItem(COOKIE_CONSENT_KEY)
    if (!hasConsented) {
      // 延迟显示，避免影响页面加载体验
      const timer = setTimeout(() => {
        setShowConsent(true)
      }, 1000)
      return () => clearTimeout(timer)
    }
  }, [])

  const handleAccept = () => {
    localStorage.setItem(COOKIE_CONSENT_KEY, 'true')
    setShowConsent(false)
  }

  const handleReject = () => {
    // 用户拒绝，但仍需记录选择
    localStorage.setItem(COOKIE_CONSENT_KEY, 'rejected')
    setShowConsent(false)
    // 可以在这里添加禁用非必要 Cookie 的逻辑
  }

  return (
    <AnimatePresence>
      {showConsent && (
        <motion.div
          initial={{ y: 100, opacity: 0 }}
          animate={{ y: 0, opacity: 1 }}
          exit={{ y: 100, opacity: 0 }}
          transition={{ duration: 0.3, ease: 'easeOut' }}
          className="fixed bottom-0 left-0 right-0 z-[9999] p-4 md:p-6"
        >
          <div className="max-w-6xl mx-auto">
            <div className="relative bg-white dark:bg-gray-800 rounded-2xl shadow-2xl border border-gray-200 dark:border-gray-700 overflow-hidden">
              {/* 装饰性渐变 */}
              <div className="absolute inset-0 bg-gradient-to-r from-blue-50/50 via-purple-50/50 to-pink-50/50 dark:from-blue-950/20 dark:via-purple-950/20 dark:to-pink-950/20 pointer-events-none" />
              
              <div className="relative p-6 md:p-8">
                <div className="flex flex-col md:flex-row items-start md:items-center gap-4">
                  {/* 图标 */}
                  <div className="flex-shrink-0">
                    <div className="w-12 h-12 rounded-full bg-blue-100 dark:bg-blue-900/30 flex items-center justify-center">
                      <Cookie className="w-6 h-6 text-blue-600 dark:text-blue-400" />
                    </div>
                  </div>

                  {/* 内容 */}
                  <div className="flex-1 min-w-0">
                    <h3 className="text-lg font-semibold text-gray-900 dark:text-white mb-2">
                      {t('cookie.title')}
                    </h3>
                    <p className="text-sm text-gray-600 dark:text-gray-400 leading-relaxed">
                      {t('cookie.description')}{' '}
                      <Link
                        to="/privacy"
                        className="text-blue-600 dark:text-blue-400 hover:underline font-medium"
                      >
                        {t('cookie.privacyLink')}
                      </Link>
                      {t('cookie.descriptionEnd')}
                    </p>
                  </div>

                  {/* 按钮组 */}
                  <div className="flex flex-col sm:flex-row gap-3 w-full md:w-auto">
                    <button
                      onClick={handleReject}
                      className="px-6 py-2.5 rounded-lg text-sm font-medium text-gray-700 dark:text-gray-300 bg-gray-100 dark:bg-gray-700 hover:bg-gray-200 dark:hover:bg-gray-600 transition-colors duration-200 whitespace-nowrap"
                    >
                      {t('cookie.reject')}
                    </button>
                    <button
                      onClick={handleAccept}
                      className="px-6 py-2.5 rounded-lg text-sm font-medium bg-gradient-to-r from-blue-600 to-purple-600 hover:from-blue-700 hover:to-purple-700 transition-all duration-200 shadow-lg shadow-blue-500/30 whitespace-nowrap"
                    >
                      {t('cookie.accept')}
                    </button>
                  </div>

                  {/* 关闭按钮 */}
                  <button
                    onClick={handleReject}
                    className="absolute top-4 right-4 p-1 rounded-lg text-gray-400 hover:text-gray-600 dark:hover:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700 transition-colors duration-200"
                    aria-label="Close"
                  >
                    <X className="w-5 h-5" />
                  </button>
                </div>
              </div>
            </div>
          </div>
        </motion.div>
      )}
    </AnimatePresence>
  )
}

export default CookieConsent
