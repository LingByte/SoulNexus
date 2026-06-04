import { ReactNode } from 'react'
import { cn } from '@/utils/cn.ts'

interface PageHeaderProps {
  title: string
  icon?: ReactNode
  actions?: ReactNode
  className?: string
  border?: boolean
}

const PageHeader = ({
  title,
  icon,
  actions,
  className,
  border = true
}: PageHeaderProps) => {
  return (
    <header className={cn(
      'flex shrink-0 items-center gap-3 px-4 py-3',
      border && 'border-b border-border',
      className
    )}>
      {icon && (
        <div className="flex items-center justify-center text-muted-foreground">
          {icon}
        </div>
      )}
      <div className="min-w-0 flex-1">
        <h1 className="truncate text-base font-semibold text-foreground">
          {title}
        </h1>
      </div>
      {actions && (
        <div className="flex items-center gap-2 shrink-0">
          {actions}
        </div>
      )}
    </header>
  )
}

export default PageHeader
