import type { ReactNode } from 'react'
import { ChevronDown } from 'lucide-react'
import { cn } from '@/lib/utils'

export function SectionCard({
  icon,
  title,
  description,
  children,
  className,
  collapsible,
  open = false,
  onOpenChange,
}: {
  icon?: ReactNode
  title: string
  description?: string
  children?: ReactNode
  className?: string
  collapsible?: boolean
  open?: boolean
  onOpenChange?: (open: boolean) => void
}) {
  const header = (
    <div className="flex items-start gap-3">
      {icon ? (
        <div className="mt-0.5 flex h-8 w-8 shrink-0 items-center justify-center rounded-lg bg-primary/10 text-primary">
          {icon}
        </div>
      ) : null}
      <div className="min-w-0 flex-1">
        <h3 className="text-sm font-semibold tracking-tight text-foreground">{title}</h3>
        {description ? (
          <p className="mt-0.5 text-xs text-muted-foreground leading-relaxed">{description}</p>
        ) : null}
      </div>
      {collapsible ? (
        <ChevronDown
          className={cn(
            'mt-1 h-4 w-4 shrink-0 text-muted-foreground transition-transform duration-300 ease-out',
            open && 'rotate-180',
          )}
        />
      ) : null}
    </div>
  )

  const body = children ? (
    <div
      className={cn(
        'grid transition-[grid-template-rows] duration-300 ease-in-out motion-reduce:transition-none',
        collapsible ? (open ? 'grid-rows-[1fr]' : 'grid-rows-[0fr]') : 'grid-rows-[1fr]',
      )}
    >
      <div className="overflow-hidden">
        <div className="space-y-3.5 px-4 pb-4 pt-2">{children}</div>
      </div>
    </div>
  ) : null

  return (
    <section
      className={cn(
        'rounded-xl border border-border bg-card shadow-sm',
        className,
      )}
    >
      {collapsible ? (
        <button
          type="button"
          className="flex w-full items-start px-4 py-3.5 text-left hover:bg-muted/50 rounded-xl transition-colors"
          onClick={() => onOpenChange?.(!open)}
          aria-expanded={open}
        >
          {header}
        </button>
      ) : (
        <div className="px-4 pt-4 pb-1">{header}</div>
      )}
      {body}
    </section>
  )
}
