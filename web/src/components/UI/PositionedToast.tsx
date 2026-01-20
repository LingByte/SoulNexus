import { useEffect, useState } from 'react'
import { motion, AnimatePresence } from 'framer-motion'
import { X, CheckCircle, AlertCircle, Info, AlertTriangle } from 'lucide-react'
import { cn } from '@/utils/cn.ts'

interface PositionedToastProps {
  id: string
  type: 'success' | 'error' | 'warning' | 'info'
  title: string
  message?: string
  duration?: number
  onClose: (id: string) => void
  position?: 'top-right' | 'top-left' | 'bottom-right' | 'bottom-left' | 'top-center' | 'bottom-center'
  scrollToPosition?: {
    x: number
    y: number
    behavior?: 'smooth' | 'instant' | 'auto'
  }
  className?: string
}

const PositionedToast = ({
  id,
  type,
  title,
  message,
  duration = 5000,
  onClose,
  position = 'top-right',
  scrollToPosition,
  className
}: PositionedToastProps) => {
  const [isVisible, setIsVisible] = useState(true)

  useEffect(() => {
    if (duration > 0) {
      const timer = setTimeout(() => {
        setIsVisible(false)
        setTimeout(() => onClose(id), 300)
      }, duration)

      return () => clearTimeout(timer)
    }
  }, [duration, id, onClose])

  // 滚动到指定位置
  useEffect(() => {
    if (scrollToPosition) {
      const { x, y, behavior = 'smooth' } = scrollToPosition
      window.scrollTo({
        left: x,
        top: y,
        behavior: behavior as ScrollBehavior
      })
    }
  }, [scrollToPosition])

  const handleClose = () => {
    setIsVisible(false)
    setTimeout(() => onClose(id), 300)
  }

  const getIcon = () => {
    switch (type) {
      case 'success':
        return <CheckCircle className="w-5 h-5" />
      case 'error':
        return <AlertCircle className="w-5 h-5" />
      case 'warning':
        return <AlertTriangle className="w-5 h-5" />
      case 'info':
        return <Info className="w-5 h-5" />
      default:
        return <Info className="w-5 h-5" />
    }
  }

  const getStyles = () => {
    switch (type) {
      case 'success':
        return {
          container: 'bg-green-50 dark:bg-green-900/20 border-green-200 dark:border-green-800',
          icon: 'text-green-600 dark:text-green-400',
          title: 'text-green-900 dark:text-green-100',
          message: 'text-green-700 dark:text-green-300'
        }
      case 'error':
        return {
          container: 'bg-red-50 dark:bg-red-900/20 border-red-200 dark:border-red-800',
          icon: 'text-red-600 dark:text-red-400',
          title: 'text-red-900 dark:text-red-100',
          message: 'text-red-700 dark:text-red-300'
        }
      case 'warning':
        return {
          container: 'bg-yellow-50 dark:bg-yellow-900/20 border-yellow-200 dark:border-yellow-800',
          icon: 'text-yellow-600 dark:text-yellow-400',
          title: 'text-yellow-900 dark:text-yellow-100',
          message: 'text-yellow-700 dark:text-yellow-300'
        }
      case 'info':
        return {
          container: 'bg-blue-50 dark:bg-blue-900/20 border-blue-200 dark:border-blue-800',
          icon: 'text-blue-600 dark:text-blue-400',
          title: 'text-blue-900 dark:text-blue-100',
          message: 'text-blue-700 dark:text-blue-300'
        }
      default:
        return {
          container: 'bg-neutral-50 dark:bg-gray-700 border-neutral-200 dark:border-gray-600',
          icon: 'text-neutral-600 dark:text-neutral-400',
          title: 'text-neutral-900 dark:text-white',
          message: 'text-neutral-700 dark:text-neutral-300'
        }
    }
  }

  const getPositionClasses = () => {
    switch (position) {
      case 'top-left':
        return 'top-4 left-4'
      case 'top-center':
        return 'top-4 left-1/2 transform -translate-x-1/2'
      case 'top-right':
        return 'top-4 right-4'
      case 'bottom-left':
        return 'bottom-4 left-4'
      case 'bottom-center':
        return 'bottom-4 left-1/2 transform -translate-x-1/2'
      case 'bottom-right':
        return 'bottom-4 right-4'
      default:
        return 'top-4 right-4'
    }
  }

  const styles = getStyles()

  return (
    <AnimatePresence>
      {isVisible && (
        <motion.div
          initial={{ opacity: 0, scale: 0.8, y: -20 }}
          animate={{ opacity: 1, scale: 1, y: 0 }}
          exit={{ opacity: 0, scale: 0.8, y: -20 }}
          transition={{ duration: 0.3, ease: [0.16, 1, 0.3, 1] }}
          className={cn(
            'fixed z-[10000] max-w-sm w-full shadow-lg rounded-xl border backdrop-blur-sm',
            styles.container,
            getPositionClasses(),
            className
          )}
        >
          <div className="p-4">
            <div className="flex items-start gap-3">
              <div className={cn('flex-shrink-0', styles.icon)}>
                {getIcon()}
              </div>
              
              <div className="flex-1 min-w-0">
                <h4 className={cn('text-sm font-semibold', styles.title)}>
                  {title}
                </h4>
                {message && (
                  <p className={cn('text-sm mt-1', styles.message)}>
                    {message}
                  </p>
                )}
              </div>
              
              <button
                onClick={handleClose}
                className={cn(
                  'flex-shrink-0 p-1 rounded-lg transition-colors',
                  'hover:bg-black/5 dark:hover:bg-white/5',
                  styles.icon
                )}
              >
                <X className="w-4 h-4" />
              </button>
            </div>
          </div>
        </motion.div>
      )}
    </AnimatePresence>
  )
}
export default PositionedToast
