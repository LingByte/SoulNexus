import { ReactNode } from 'react'
import { cn } from '@/utils/cn.ts'

interface FormSectionProps {
  title?: string
  description?: string
  children: ReactNode
  className?: string
  collapsible?: boolean
  defaultCollapsed?: boolean
}

const FormSection = ({
  title,
  description,
  children,
  className
}: FormSectionProps) => {
  return (
    <div className={cn('space-y-6', className)}>
      {(title || description) && (
        <div className="border-b border-neutral-200 dark:border-neutral-700 pb-4">
          {title && (
            <h3 className="text-lg font-medium text-neutral-900 dark:text-white">
              {title}
            </h3>
          )}
          {description && (
            <p className="mt-1 text-sm text-neutral-600 dark:text-neutral-400">
              {description}
            </p>
          )}
        </div>
      )}
      
      <div className="space-y-4">
        {children}
      </div>
    </div>
  )
}

export default FormSection
