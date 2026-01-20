import { ReactNode } from 'react'
import { cn } from '@/utils/cn'

interface TimelineItem {
  id: string
  title: string
  description?: string
  time: string
  status?: 'completed' | 'current' | 'upcoming'
  icon?: ReactNode
  color?: string
}

interface TimelineProps {
  items: TimelineItem[]
  orientation?: 'vertical' | 'horizontal'
  className?: string
}

const Timeline = ({
  items,
  orientation = 'vertical',
  className
}: TimelineProps) => {
  const isVertical = orientation === 'vertical'

  const getStatusColor = (status?: string) => {
    switch (status) {
      case 'completed':
        return 'bg-green-500 border-green-500'
      case 'current':
        return 'bg-primary border-primary'
      case 'upcoming':
        return 'bg-neutral-300 dark:bg-neutral-600 border-neutral-300 dark:border-neutral-600'
      default:
        return 'bg-neutral-300 dark:bg-neutral-600 border-neutral-300 dark:border-neutral-600'
    }
  }

  const getStatusTextColor = (status?: string) => {
    switch (status) {
      case 'completed':
        return 'text-green-600 dark:text-green-400'
      case 'current':
        return 'text-primary'
      case 'upcoming':
        return 'text-neutral-500 dark:text-neutral-400'
      default:
        return 'text-neutral-500 dark:text-neutral-400'
    }
  }

  if (isVertical) {
    return (
      <div className={cn('space-y-6', className)}>
        {items.map((item, index) => (
          <div key={item.id} className="relative flex gap-4">
            {/* 连接线 */}
            {index < items.length - 1 && (
              <div className="absolute left-4 top-8 w-0.5 h-full bg-neutral-200 dark:bg-neutral-700" />
            )}
            
            {/* 图标/圆点 */}
            <div className={cn(
              'flex-shrink-0 w-8 h-8 rounded-full border-2 flex items-center justify-center',
              getStatusColor(item.status)
            )}>
              {item.icon ? (
                <div className="w-4 h-4 text-white">
                  {item.icon}
                </div>
              ) : (
                <div className="w-2 h-2 bg-white rounded-full" />
              )}
            </div>

            {/* 内容 */}
            <div className="flex-1 pb-6">
              <div className="flex items-center justify-between">
                <h3 className={cn(
                  'text-sm font-medium',
                  getStatusTextColor(item.status)
                )}>
                  {item.title}
                </h3>
                <time className="text-xs text-neutral-500 dark:text-neutral-400">
                  {item.time}
                </time>
              </div>
              {item.description && (
                <p className="mt-1 text-sm text-neutral-600 dark:text-neutral-400">
                  {item.description}
                </p>
              )}
            </div>
          </div>
        ))}
      </div>
    )
  }

  return (
    <div className={cn('flex items-center space-x-4 overflow-x-auto pb-4', className)}>
      {items.map((item, index) => (
        <div key={item.id} className="flex-shrink-0 flex flex-col items-center">
          {/* 图标/圆点 */}
          <div className={cn(
            'w-8 h-8 rounded-full border-2 flex items-center justify-center mb-2',
            getStatusColor(item.status)
          )}>
            {item.icon ? (
              <div className="w-4 h-4 text-white">
                {item.icon}
              </div>
            ) : (
              <div className="w-2 h-2 bg-white rounded-full" />
            )}
          </div>

          {/* 内容 */}
          <div className="text-center max-w-32">
            <h3 className={cn(
              'text-sm font-medium mb-1',
              getStatusTextColor(item.status)
            )}>
              {item.title}
            </h3>
            <p className="text-xs text-neutral-500 dark:text-neutral-400 mb-1">
              {item.time}
            </p>
            {item.description && (
              <p className="text-xs text-neutral-600 dark:text-neutral-400">
                {item.description}
              </p>
            )}
          </div>

          {/* 连接线 */}
          {index < items.length - 1 && (
            <div className="absolute left-1/2 top-4 w-full h-0.5 bg-neutral-200 dark:bg-neutral-700 transform translate-x-4" />
          )}
        </div>
      ))}
    </div>
  )
}

export default Timeline
