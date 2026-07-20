import type { ReactNode } from 'react'
import { cn } from '@/lib/utils'

export function Field({
  label,
  hint,
  htmlFor,
  children,
  className,
}: {
  label: string
  hint?: string
  htmlFor?: string
  children: ReactNode
  className?: string
}) {
  return (
    <div className={cn('space-y-1.5', className)}>
      <label htmlFor={htmlFor} className="text-[11px] font-medium text-foreground/70">
        {label}
      </label>
      {children}
      {hint ? <p className="text-[11px] text-muted-foreground leading-relaxed">{hint}</p> : null}
    </div>
  )
}
