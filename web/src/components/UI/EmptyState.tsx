import { LucideIcon } from 'lucide-react'
import Button from './Button'
import { cn } from '@/utils/cn.ts'

// 优化的空状态组件
interface EmptyStateProps {
  icon?: LucideIcon
  title: string
  description?: string
  action?: {
    label: string
    onClick: () => void
  }
  className?: string
  iconClassName?: string
  buttonClassName?: string
}

const EmptyState = ({
                      icon: Icon,
                      title,
                      description,
                      action,
                      className,
                      iconClassName = "text-neutral-400",  // 默认图标颜色
                      buttonClassName = "mt-4"  // 默认按钮位置
                    }: EmptyStateProps) => {
  return (
      <div className={cn(
          'flex flex-col items-center justify-center py-12 px-4 text-center max-w-sm mx-auto',
          className
      )}>
        {/* 如果有传入图标，则展示图标 */}
        {Icon && (
            <div className={cn("w-16 h-16 mb-4", iconClassName)}>
              <Icon className="w-full h-full transition-transform transform hover:scale-125" />
            </div>
        )}
        {/* 展示标题 */}
        <h3 className="text-lg font-semibold text-neutral-900 dark:text-white mb-2">
          {title}
        </h3>
        {/* 展示描述 */}
        {description && (
            <p className="text-neutral-600 dark:text-neutral-400 mb-6 max-w-sm">
              {description}
            </p>
        )}
        {/* 如果有action按钮，则展示 */}
        {action && (
            <Button
                onClick={action.onClick}
                className={buttonClassName}
            >
              {action.label}
            </Button>
        )}
      </div>
  )
}

export default EmptyState
