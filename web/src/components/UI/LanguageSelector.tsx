import { useState, useRef, useEffect } from 'react'
import { motion, AnimatePresence } from 'framer-motion'
import { Globe, Check } from 'lucide-react'
import { useI18nStore, Language } from '@/stores/i18nStore'
import { cn } from '@/utils/cn'

const languages: { code: Language; label: string; nativeLabel: string }[] = [
  { code: 'zh', label: '中文', nativeLabel: '中文' },
  { code: 'en', label: 'English', nativeLabel: 'English' },
  { code: 'ja', label: '日本語', nativeLabel: '日本語' },
]

interface LanguageSelectorProps {
  className?: string
  size?: 'sm' | 'md' | 'lg'
}

const LanguageSelector = ({ className = '', size = 'md' }: LanguageSelectorProps) => {
  const { language, setLanguage, t } = useI18nStore()
  const [isOpen, setIsOpen] = useState(false)
  const dropdownRef = useRef<HTMLDivElement>(null)

  // 点击外部关闭下拉菜单
  useEffect(() => {
    const handleClickOutside = (event: MouseEvent) => {
      if (dropdownRef.current && !dropdownRef.current.contains(event.target as Node)) {
        setIsOpen(false)
      }
    }

    if (isOpen) {
      document.addEventListener('mousedown', handleClickOutside)
    }

    return () => {
      document.removeEventListener('mousedown', handleClickOutside)
    }
  }, [isOpen])

  const sizeClasses = {
    sm: 'w-10 h-10',
    md: 'w-12 h-12',
    lg: 'w-16 h-16',
  }

  const iconSizes = {
    sm: 'w-4 h-4',
    md: 'w-5 h-5',
    lg: 'w-6 h-6',
  }

  return (
    <div className={cn('relative', className)} ref={dropdownRef}>
      <motion.button
        onClick={() => setIsOpen(!isOpen)}
        title={t('lang.select')}
        className={cn(
          'flex items-center justify-center rounded-full bg-gradient-to-br from-gray-200 to-gray-400 dark:from-gray-600 dark:to-gray-900 border-2 border-gray-300 dark:border-gray-500 shadow-xl transition-all hover:scale-110 duration-200',
          sizeClasses[size]
        )}
        whileHover={{ scale: 1.07 }}
        whileTap={{ scale: 0.93 }}
      >
        <Globe className={cn('text-gray-700 dark:text-gray-200', iconSizes[size])} />
      </motion.button>

      <AnimatePresence>
        {isOpen && (
          <motion.div
            initial={{ opacity: 0, y: -10, scale: 0.95 }}
            animate={{ opacity: 1, y: 0, scale: 1 }}
            exit={{ opacity: 0, y: -10, scale: 0.95 }}
            transition={{ duration: 0.2 }}
            className="absolute right-0 mt-2 w-48 bg-white dark:bg-gray-800 rounded-lg shadow-xl border border-gray-200 dark:border-gray-700 overflow-hidden z-50"
          >
            <div className="py-1">
              {languages.map((lang) => (
                <button
                  key={lang.code}
                  onClick={() => {
                    setLanguage(lang.code)
                    setIsOpen(false)
                  }}
                  className={cn(
                    'w-full px-4 py-2 text-left flex items-center justify-between hover:bg-gray-100 dark:hover:bg-gray-700 transition-colors',
                    language === lang.code && 'bg-gray-100 dark:bg-gray-700'
                  )}
                >
                  <div className="flex items-center gap-2">
                    <span className="text-sm font-medium text-gray-900 dark:text-gray-100">
                      {lang.nativeLabel}
                    </span>
                    {lang.label !== lang.nativeLabel && (
                      <span className="text-xs text-gray-500 dark:text-gray-400">
                        {lang.label}
                      </span>
                    )}
                  </div>
                  {language === lang.code && (
                    <Check className="w-4 h-4 text-primary" />
                  )}
                </button>
              ))}
            </div>
          </motion.div>
        )}
      </AnimatePresence>
    </div>
  )
}

export default LanguageSelector

