import { ReactNode } from 'react'
import { cn } from '@/utils/cn.ts'

interface FormFieldProps {
  label?: string
  error?: string
  hint?: string
  required?: boolean
  children: ReactNode
  className?: string
  labelClassName?: string
}

const FormField = ({
  label,
  error,
  hint,
  required = false,
  children,
  className,
  labelClassName
}: FormFieldProps) => {
  return (
    <div className={cn('space-y-2', className)}>
      {label && (
        <label className={cn(
          'block text-sm font-medium text-neutral-700 dark:text-neutral-300',
          labelClassName
        )}>
          {label}
          {required && (
            <span className="text-red-500 ml-1">*</span>
          )}
        </label>
      )}
      
      {children}
      
      {hint && !error && (
        <p className="text-sm text-neutral-500 dark:text-neutral-400">
          {hint}
        </p>
      )}
      
      {error && (
        <p className="text-sm text-red-600 dark:text-red-400">
          {error}
        </p>
      )}
    </div>
  )
}

export default FormField
