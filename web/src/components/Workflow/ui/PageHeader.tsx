import type { ReactNode } from 'react'
import { Link } from 'react-router-dom'
import { ArrowLeft } from 'lucide-react'
import { cn } from '@/utils/utils'

interface PageHeaderProps {
  title: string
  subtitle?: string
  backTo?: string
  actions?: ReactNode
  className?: string
}

export default function PageHeader({ title, subtitle, backTo, actions, className }: PageHeaderProps) {
  return (
    <header
      className={cn(
        'flex shrink-0 items-center gap-3 border-b border-[hsl(var(--border))] px-4 py-2.5',
        className,
      )}
    >
      {backTo ? (
        <Link
          to={backTo}
          className="flex h-7 w-7 shrink-0 items-center justify-center rounded-md text-gray-500 transition-colors hover:bg-gray-100 hover:text-gray-900 dark:text-gray-400 dark:hover:bg-gray-800 dark:hover:text-gray-100"
        >
          <ArrowLeft className="h-4 w-4" />
        </Link>
      ) : null}
      <div className="min-w-0 flex-1">
        <h1 className="truncate text-sm font-semibold">{title}</h1>
        {subtitle ? (
          <p className="mt-0.5 truncate text-xs text-gray-500 dark:text-gray-400">{subtitle}</p>
        ) : null}
      </div>
      {actions ? <div className="flex shrink-0 items-center gap-2">{actions}</div> : null}
    </header>
  )
}
