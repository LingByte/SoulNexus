import { cn } from '@/lib/utils'

export type SegmentOption<T extends string> = {
  value: T
  label: string
  description?: string
}

export function SegmentedControl<T extends string>({
  value,
  options,
  onChange,
  className,
}: {
  value: T
  options: SegmentOption<T>[]
  onChange: (value: T) => void
  className?: string
}) {
  return (
    <div
      className={cn(
        'grid gap-1 rounded-xl bg-[rgb(232,232,235)]/90 p-1',
        options.length === 2 && 'grid-cols-2',
        options.length === 3 && 'grid-cols-3',
        options.length >= 4 && 'grid-cols-2 sm:grid-cols-4',
        className,
      )}
      role="radiogroup"
    >
      {options.map((opt) => {
        const active = value === opt.value
        return (
          <button
            key={opt.value}
            type="button"
            role="radio"
            aria-checked={active}
            onClick={() => onChange(opt.value)}
            className={cn(
              'rounded-lg px-3 py-2 text-left transition',
              'focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring/40',
              active
                ? 'bg-[#18181B] text-white shadow-sm'
                : 'text-muted-foreground hover:text-foreground hover:bg-white/60',
            )}
          >
            <div className="text-xs font-medium leading-none">{opt.label}</div>
            {opt.description ? (
              <div
                className={cn(
                  'mt-1 text-[10px] leading-snug',
                  active ? 'text-white/65' : 'text-muted-foreground',
                )}
              >
                {opt.description}
              </div>
            ) : null}
          </button>
        )
      })}
    </div>
  )
}
