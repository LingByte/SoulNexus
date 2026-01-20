import { LucideIcon } from 'lucide-react'
import { cn } from '@/utils/cn'
import Card from '../UI/Card'

interface StatCardProps {
  title: string
  value: string | number
  change?: {
    value: number
    type: 'increase' | 'decrease' | 'neutral'
  }
  icon?: LucideIcon
  iconColor?: string
  description?: string
  className?: string
  onClick?: () => void
}

const StatCard = ({
  title,
  value,
  change,
  icon: Icon,
  iconColor = 'text-primary',
  description,
  className,
  onClick
}: StatCardProps) => {
  const getChangeColor = (type: string) => {
    switch (type) {
      case 'increase':
        return 'text-green-600 dark:text-green-400'
      case 'decrease':
        return 'text-red-600 dark:text-red-400'
      case 'neutral':
      default:
        return 'text-neutral-500 dark:text-neutral-400'
    }
  }

  const getChangeIcon = (type: string) => {
    switch (type) {
      case 'increase':
        return '↗'
      case 'decrease':
        return '↘'
      case 'neutral':
      default:
        return '→'
    }
  }

  return (
    <Card
      className={cn(
        'p-6 transition-all hover:shadow-lg',
        onClick && 'cursor-pointer hover:scale-105',
        className
      )}
      onClick={onClick}
    >
      <div className="flex items-center justify-between">
        <div className="flex-1">
          <p className="text-sm font-medium text-neutral-600 dark:text-neutral-400 mb-1">
            {title}
          </p>
          <div className="flex items-baseline gap-2">
            <p className="text-2xl font-bold text-neutral-900 dark:text-white">
              {value}
            </p>
            {change && (
              <div className={cn(
                'flex items-center gap-1 text-sm font-medium',
                getChangeColor(change.type)
              )}>
                <span>{getChangeIcon(change.type)}</span>
                <span>{Math.abs(change.value)}%</span>
              </div>
            )}
          </div>
          {description && (
            <p className="text-xs text-neutral-500 dark:text-neutral-400 mt-1">
              {description}
            </p>
          )}
        </div>
        
        {Icon && (
          <div className={cn(
            'p-3 rounded-lg bg-neutral-100 dark:bg-neutral-800',
            iconColor
          )}>
            <Icon className="w-6 h-6" />
          </div>
        )}
      </div>
    </Card>
  )
}

export default StatCard
