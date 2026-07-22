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
  open,
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
        <div className="mt-0.5 flex h-8 w-8 shrink-0 items-center justify-center rounded-xl bg-[#18181B]/[0.06] text-[#18181B]/80">
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
            'mt-1 h-4 w-4 shrink-0 text-muted-foreground transition-transform',
            open && 'rotate-180',
          )}
        />
      ) : null}
    </div>
  )

  return (
    <section
      className={cn(
        'rounded-xl border border-black/[0.06] bg-white shadow-[0_1px_2px_rgba(0,0,0,0.04)]',
        className,
      )}
    >
      {collapsible ? (
        <button
          type="button"
          className="flex w-full items-start px-4 py-3.5 text-left hover:bg-black/[0.015] rounded-xl transition-colors"
          onClick={() => onOpenChange?.(!open)}
        >
          {header}
        </button>
      ) : (
        <div className="px-4 pt-4 pb-1">{header}</div>
      )}
      {(!collapsible || open) && children ? (
        <div className="space-y-3.5 px-4 pb-4 pt-2">{children}</div>
      ) : null}
    </section>
  )
}
