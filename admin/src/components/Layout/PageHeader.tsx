import { ReactNode } from 'react'
import { cn } from '@/utils/cn.ts'

interface PageHeaderProps {
  title?: ReactNode
  subtitle?: ReactNode
  description?: ReactNode
  children?: ReactNode
  actions?: ReactNode
  className?: string
  breadcrumbs?: Array<{
    label: string
    href?: string
  }>
}

const PageHeader = ({
  title,
  subtitle,
  description,
  children,
  actions,
  className,
  breadcrumbs,
}: PageHeaderProps) => {
  const subtitleContent = description ?? subtitle
  const actionsContent = actions ?? children
  return (
    <div className={cn('mb-8', className)}>
      {/* 面包屑导航 */}
      {breadcrumbs && breadcrumbs.length > 0 && (
        <nav className="flex mb-4" aria-label="Breadcrumb">
          <ol className="inline-flex items-center space-x-1 md:space-x-3">
            {breadcrumbs.map((crumb, index) => (
              <li key={index} className="inline-flex items-center">
                {index > 0 && (
                  <svg
                    className="w-6 h-6 text-neutral-400"
                    fill="currentColor"
                    viewBox="0 0 20 20"
                  >
                    <path
                      fillRule="evenodd"
                      d="M7.293 14.707a1 1 0 010-1.414L10.586 10 7.293 6.707a1 1 0 011.414-1.414l4 4a1 1 0 010 1.414l-4 4a1 1 0 01-1.414 0z"
                      clipRule="evenodd"
                    />
                  </svg>
                )}
                {crumb.href ? (
                  <a
                    href={crumb.href}
                    className="text-sm font-medium text-neutral-500 hover:text-neutral-700 dark:text-neutral-400 dark:hover:text-neutral-200 transition-colors"
                  >
                    {crumb.label}
                  </a>
                ) : (
                  <span className="text-sm font-medium text-neutral-500 dark:text-neutral-400">
                    {crumb.label}
                  </span>
                )}
              </li>
            ))}
          </ol>
        </nav>
      )}

      {/* 页面标题和副标题 */}
      {(title || subtitleContent || actionsContent) && (
        <div className="flex flex-col gap-4 sm:flex-row sm:items-start sm:justify-between sm:gap-6">
          <div className="min-w-0 flex-1">
            {title && (
              <h1 className="text-xl font-semibold text-[var(--color-text-1)] sm:text-2xl">
                {title}
              </h1>
            )}
            {subtitleContent && (
              <p className="mt-1 break-all text-sm text-[var(--color-text-3)]">
                {subtitleContent}
              </p>
            )}
          </div>
          {actionsContent && (
            <div className="flex shrink-0 flex-wrap items-center gap-2 sm:justify-end">{actionsContent}</div>
          )}
        </div>
      )}
    </div>
  )
}

export default PageHeader
