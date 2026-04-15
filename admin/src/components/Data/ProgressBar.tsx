import { ReactNode } from 'react'
import { cn } from '@/utils/cn'

interface ProgressBarProps {
  value: number
  max?: number
  size?: 'sm' | 'md' | 'lg'
  variant?: 'default' | 'success' | 'warning' | 'error'
  showValue?: boolean
  label?: string
  description?: string
  className?: string
  children?: ReactNode
}

const ProgressBar = ({
  value,
  max = 100,
  size = 'md',
  variant = 'default',
  showValue = true,
  label,
  description,
  className,
  children
}: ProgressBarProps) => {
  const percentage = Math.min(100, Math.max(0, (value / max) * 100))

  const sizeClasses = {
    sm: 'h-2',
    md: 'h-3',
    lg: 'h-4'
  }

  const variantClasses = {
    default: 'bg-primary',
    success: 'bg-green-500',
    warning: 'bg-yellow-500',
    error: 'bg-red-500'
  }

  const getVariantTextColor = () => {
    switch (variant) {
      case 'success':
        return 'text-green-600 dark:text-green-400'
      case 'warning':
        return 'text-yellow-600 dark:text-yellow-400'
      case 'error':
        return 'text-red-600 dark:text-red-400'
      default:
        return 'text-primary'
    }
  }

  return (
    <div className={cn('w-full', className)}>
      {(label || showValue) && (
        <div className="flex items-center justify-between mb-2">
          {label && (
            <span className="text-sm font-medium text-neutral-700 dark:text-neutral-300">
              {label}
            </span>
          )}
          {showValue && (
            <span className={cn(
              'text-sm font-medium',
              getVariantTextColor()
            )}>
              {Math.round(percentage)}%
            </span>
          )}
        </div>
      )}

      <div className={cn(
        'w-full bg-neutral-200 dark:bg-neutral-700 rounded-full overflow-hidden',
        sizeClasses[size]
      )}>
        <div
          className={cn(
            'h-full transition-all duration-300 ease-in-out rounded-full',
            variantClasses[variant]
          )}
          style={{ width: `${percentage}%` }}
        >
          {children}
        </div>
      </div>

      {description && (
        <p className="text-xs text-neutral-500 dark:text-neutral-400 mt-1">
          {description}
        </p>
      )}
    </div>
  )
}

export default ProgressBar
