import { ReactNode } from 'react'
import { cn } from '@/utils/cn.ts'

interface PageHeaderProps {
  title: string
  actions?: ReactNode
  className?: string
}

const PageHeader = ({
  title,
  actions,
  className
}: PageHeaderProps) => {
  return (
    <header className={cn(
      'flex shrink-0 items-center gap-2 border-b border-border px-4 py-2',
      className
    )}>
      <div className="min-w-0 flex-1">
        <h1 className="truncate text-sm font-semibold">
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
