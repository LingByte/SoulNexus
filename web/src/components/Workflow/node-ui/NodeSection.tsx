import React from 'react'
import { cn } from '@/utils/utils'

export const NodeSection: React.FC<{
  title: string
  children: React.ReactNode
  className?: string
  extra?: React.ReactNode
}> = ({ title, children, className, extra }) => (
  <div className={cn('rounded-lg bg-[rgb(var(--primary-1))]/40 p-3', className)}>
    <div className="mb-2.5 flex items-center justify-between gap-2">
      <div className="flex items-center gap-2">
        <span className="h-3.5 w-0.5 rounded-full bg-[rgb(var(--primary-6))]" />
        <span className="text-xs font-semibold text-[var(--color-text-1)]">{title}</span>
      </div>
      {extra}
    </div>
    <div className="space-y-2.5">{children}</div>
  </div>
)

export const NodeField: React.FC<{
  label: string
  required?: boolean
  hint?: React.ReactNode
  children: React.ReactNode
  className?: string
}> = ({ label, required, hint, children, className }) => (
  <div className={cn('space-y-1.5', className)}>
    <div className="flex items-center gap-1.5">
      <span className="text-xs font-medium text-[var(--color-text-2)]">{label}</span>
      {required ? <span className="text-[10px] text-[rgb(var(--danger-6))]">*</span> : null}
      {hint ? <span className="text-[10px] text-[var(--color-text-3)]">{hint}</span> : null}
    </div>
    {children}
  </div>
)

export const NodeTag: React.FC<{ children: React.ReactNode; tone?: 'blue' | 'gray' | 'green' }> = ({
  children,
  tone = 'gray',
}) => (
  <span
    className={cn(
      'inline-flex items-center rounded px-1.5 py-0.5 text-[10px] font-medium',
      tone === 'blue' && 'bg-[rgb(var(--primary-1))] text-[rgb(var(--primary-6))]',
      tone === 'green' && 'bg-[rgb(var(--success-1))] text-[rgb(var(--success-6))]',
      tone === 'gray' && 'bg-[var(--color-fill-2)] text-[var(--color-text-3)]',
    )}
  >
    {children}
  </span>
)
