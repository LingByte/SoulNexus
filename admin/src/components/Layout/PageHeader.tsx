import { ReactNode } from 'react'
import { cn } from '@/utils/cn.ts'

interface PageHeaderProps {
  title: string
  subtitle?: string
  children?: ReactNode
  className?: string
  breadcrumbs?: Array<{
    label: string
    href?: string
  }>
}

const PageHeader = ({
  title,
  subtitle,
  children,
  className,
  breadcrumbs
}: PageHeaderProps) => {
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
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-3xl font-bold text-neutral-900 dark:text-white">
            {title}
          </h1>
          {subtitle && (
            <p className="mt-2 text-lg text-neutral-600 dark:text-neutral-400">
              {subtitle}
            </p>
          )}
        </div>
        
        {/* 右侧操作按钮 */}
        {children && (
          <div className="flex items-center gap-3">
            {children}
          </div>
        )}
      </div>
    </div>
  )
}

export default PageHeader
