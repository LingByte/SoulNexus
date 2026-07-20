import type { LucideIcon } from 'lucide-react'
import { Button } from '@/components/ui'
import { cn } from '@/utils/utils'

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

export default function EmptyState({
  icon: Icon,
  title,
  description,
  action,
  className,
  iconClassName = 'text-neutral-400',
  buttonClassName = 'mt-4',
}: EmptyStateProps) {
  return (
    <div
      className={cn(
        'mx-auto flex max-w-sm flex-col items-center justify-center px-4 py-12 text-center',
        className,
      )}
    >
      {Icon ? (
        <div className={cn('mb-4 h-16 w-16', iconClassName)}>
          <Icon className="h-full w-full" />
        </div>
      ) : null}
      <h3 className="mb-2 text-lg font-semibold text-neutral-900 dark:text-white">{title}</h3>
      {description ? (
        <p className="mb-6 max-w-sm text-neutral-600 dark:text-neutral-400">{description}</p>
      ) : null}
      {action ? (
        <Button onClick={action.onClick} className={buttonClassName}>
          {action.label}
        </Button>
      ) : null}
    </div>
  )
}
