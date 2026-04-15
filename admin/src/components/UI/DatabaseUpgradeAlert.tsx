import { useState, useEffect } from 'react'
import { motion, AnimatePresence } from 'framer-motion'
import { AlertTriangle, X, ExternalLink } from 'lucide-react'
import Button from './Button'
import { useLocalStorage } from '@/hooks/useLocalStorage'

interface DatabaseUpgradeAlertProps {
  shouldShow: boolean
  className?: string
}

/**
 * 数据库升级提示组件
 * 显示在页面左下角，提醒用户升级数据库
 */
const DatabaseUpgradeAlert = ({ shouldShow, className = '' }: DatabaseUpgradeAlertProps) => {
  const [dismissed, setDismissed] = useLocalStorage('lingstorage_db_upgrade_dismissed', false)
  const [isVisible, setIsVisible] = useState(false)

  useEffect(() => {
    // 只有在需要显示且用户没有点击"不再提醒"时才显示
    setIsVisible(shouldShow && !dismissed)
  }, [shouldShow, dismissed])

  const handleDismiss = () => {
    setIsVisible(false)
  }

  const handleNeverShow = () => {
    setDismissed(true)
    setIsVisible(false)
  }

  if (!isVisible) return null

  return (
    <AnimatePresence>
      <motion.div
        initial={{ opacity: 0, y: 20, scale: 0.95 }}
        animate={{ opacity: 1, y: 0, scale: 1 }}
        exit={{ opacity: 0, y: 20, scale: 0.95 }}
        transition={{ duration: 0.3, type: "spring", stiffness: 300 }}
        className={`fixed bottom-4 left-4 z-50 w-full max-w-sm ${className}`}
      >
        <div className="bg-gradient-to-r from-amber-50 to-orange-50 dark:from-amber-900/20 dark:to-orange-900/20 border border-amber-200 dark:border-amber-800 rounded-xl shadow-lg backdrop-blur-sm">
          <div className="p-4">
            {/* 关闭按钮 */}
            <button
              onClick={handleDismiss}
              className="absolute top-2 right-2 text-amber-600 dark:text-amber-400 hover:text-amber-700 dark:hover:text-amber-300 transition-colors p-1 rounded-full hover:bg-amber-100 dark:hover:bg-amber-900/30"
            >
              <X className="w-4 h-4" />
            </button>

            {/* 警告图标和标题 */}
            <div className="flex items-start gap-3 mb-3">
              <div className="flex-shrink-0 mt-0.5">
                <div className="w-8 h-8 rounded-full bg-amber-100 dark:bg-amber-900/50 flex items-center justify-center">
                  <AlertTriangle className="w-4 h-4 text-amber-600 dark:text-amber-400" />
                </div>
              </div>
              <div className="flex-1 min-w-0">
                <h3 className="text-sm font-semibold text-amber-900 dark:text-amber-100 mb-1">
                  数据库升级建议
                </h3>
                <p className="text-xs text-amber-800 dark:text-amber-200 leading-relaxed">
                  检测到您目前使用的是 SQLite 数据库，在生产环境中可能存在性能瓶颈和数据丢失风险。
                  建议升级到 MySQL 或 PostgreSQL 以获得更好的性能和稳定性。
                </p>
              </div>
            </div>

            {/* 操作按钮 */}
            <div className="flex items-center justify-between gap-2 mt-4">
              <a
                href="https://docs.lingecho.com/deployment/database"
                target="_blank"
                rel="noopener noreferrer"
                className="flex items-center gap-1 text-xs text-amber-700 dark:text-amber-300 hover:text-amber-800 dark:hover:text-amber-200 transition-colors"
              >
                <span>查看升级指南</span>
                <ExternalLink className="w-3 h-3" />
              </a>
              <div className="flex gap-2">
                <Button
                  variant="ghost"
                  size="xs"
                  onClick={handleNeverShow}
                  className="text-amber-600 dark:text-amber-400 hover:text-amber-700 dark:hover:text-amber-300 hover:bg-amber-100 dark:hover:bg-amber-900/30"
                >
                  不再提醒
                </Button>
                <Button
                  variant="outline"
                  size="xs"
                  onClick={handleDismiss}
                  className="border-amber-300 dark:border-amber-700 text-amber-700 dark:text-amber-300 hover:bg-amber-100 dark:hover:bg-amber-900/30"
                >
                  知道了
                </Button>
              </div>
            </div>
          </div>
        </div>
      </motion.div>
    </AnimatePresence>
  )
}

export default DatabaseUpgradeAlert