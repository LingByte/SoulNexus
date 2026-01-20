import { motion } from 'framer-motion'
import { useThemeStore } from '@/stores/themeStore.ts'
import { useI18nStore } from '@/stores/i18nStore.ts'
import { cn } from '@/utils/cn.ts'
import { Sun, Moon } from 'lucide-react'

interface EnhancedThemeToggleProps {
  className?: string
  size?: 'sm' | 'md' | 'lg'
}

const EnhancedThemeToggle = ({ className = "", size = 'md' }: EnhancedThemeToggleProps) => {
  const { isDark, toggleMode } = useThemeStore()
  const { t } = useI18nStore()

  const sizeClasses = {
    sm: 'w-10 h-10', // 球样式更大些
    md: 'w-12 h-12',
    lg: 'w-16 h-16',
  }
  const iconSizes = {
    sm: 'w-5 h-5',
    md: 'w-6 h-6',
    lg: 'w-8 h-8',
  }

  return (
    <motion.button
      onClick={toggleMode}
      title={isDark ? t('theme.light') : t('theme.dark')}
      className={cn(
        'flex items-center justify-center rounded-full bg-gradient-to-br from-gray-200 to-gray-400 dark:from-gray-600 dark:to-gray-900 border-2 border-gray-300 dark:border-gray-500 shadow-xl transition-all hover:scale-110 duration-200',
        sizeClasses[size],
        className
      )}
      whileHover={{ scale: 1.07 }}
      whileTap={{ scale: 0.93 }}
    >
      <span className="sr-only">{t('theme.toggle')}</span>
      {isDark
        ? <Sun className={cn('text-yellow-400', iconSizes[size])} />
        : <Moon className={cn('text-blue-400', iconSizes[size])} />}
    </motion.button>
  )
}

export default EnhancedThemeToggle
