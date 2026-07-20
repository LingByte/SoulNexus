import type { ReactNode } from 'react'
import { motion } from 'framer-motion'
import { cn } from '@/utils/utils'

interface CardProps {
  children: ReactNode
  title?: string
  subtitle?: string
  actions?: ReactNode
  variant?: 'default' | 'outlined' | 'elevated' | 'filled' | 'glass'
  padding?: 'none' | 'sm' | 'md' | 'lg' | 'xl'
  className?: string
  hover?: boolean
  onClick?: () => void
  headerClassName?: string
  bodyClassName?: string
  footerClassName?: string
  footer?: ReactNode
  /** Enable SoulNexus-style lift animation on hover */
  animated?: boolean
}

const paddingMap: Record<NonNullable<CardProps['padding']>, string> = {
  none: 'p-0',
  sm: 'p-3',
  md: 'p-4',
  lg: 'p-6',
  xl: 'p-8',
}

export default function Card({
  children,
  title,
  subtitle,
  actions,
  variant = 'default',
  padding = 'md',
  className,
  hover = false,
  onClick,
  headerClassName,
  bodyClassName,
  footerClassName,
  footer,
  animated = false,
}: CardProps) {
  const variantClass =
    variant === 'outlined'
      ? 'border-2 border-gray-200 dark:border-gray-700 bg-white dark:bg-gray-900'
      : variant === 'elevated'
        ? 'bg-white dark:bg-gray-900 shadow-lg border border-gray-100 dark:border-gray-800'
        : variant === 'glass'
          ? 'bg-white/80 dark:bg-gray-900/80 backdrop-blur-sm border border-gray-200/60 dark:border-gray-700/60'
          : 'bg-white dark:bg-gray-900 border-2 border-gray-100 dark:border-gray-800'

  const inner = (
    <div
      className={cn(
        'rounded-xl',
        variantClass,
        paddingMap[padding],
        hover && 'transition-all duration-300 hover:shadow-xl hover:border-blue-200 dark:hover:border-blue-800',
        onClick && 'cursor-pointer',
        className,
      )}
      onClick={onClick}
      role={onClick ? 'button' : undefined}
    >
      {(title || subtitle || actions) && (
        <div className={cn('mb-3 flex items-start justify-between gap-3', headerClassName)}>
          <div>
            {title ? <h3 className="text-sm font-semibold text-gray-900 dark:text-white">{title}</h3> : null}
            {subtitle ? <p className="mt-0.5 text-xs text-gray-500 dark:text-gray-400">{subtitle}</p> : null}
          </div>
          {actions}
        </div>
      )}
      <div className={bodyClassName}>{children}</div>
      {footer ? (
        <div className={cn('mt-4 border-t border-gray-200 pt-4 dark:border-gray-700', footerClassName)}>{footer}</div>
      ) : null}
    </div>
  )

  if (animated) {
    return (
      <motion.div
        initial={{ opacity: 0, y: 20 }}
        animate={{ opacity: 1, y: 0 }}
        whileHover={{ y: -4 }}
        transition={{ duration: 0.2 }}
        className="h-full"
      >
        {inner}
      </motion.div>
    )
  }

  return inner
}
