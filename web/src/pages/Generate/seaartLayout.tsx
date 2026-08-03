import { cn } from '@/utils/utils'

/** SeaArt-style chip / segmented control */
export function ChipGroup<T extends string>({
  label,
  options,
  value,
  onChange,
  renderLabel,
  disabled,
}: {
  label: string
  options: readonly T[]
  value: T
  onChange: (v: T) => void
  renderLabel?: (v: T) => string
  disabled?: boolean
}) {
  return (
    <div>
      <div className="mb-2 text-xs font-medium text-muted-foreground">{label}</div>
      <div className="flex flex-wrap gap-2">
        {options.map((opt) => {
          const active = value === opt
          return (
            <button
              key={opt}
              type="button"
              disabled={disabled}
              onClick={() => onChange(opt)}
              className={cn(
                'min-w-11 rounded-lg border px-3 py-1.5 text-xs transition disabled:opacity-50',
                active
                  ? 'border-foreground/80 bg-foreground text-background'
                  : 'border-border bg-muted/40 text-foreground hover:border-foreground/30 hover:bg-muted/70',
              )}
            >
              {renderLabel ? renderLabel(opt) : opt}
            </button>
          )
        })}
      </div>
    </div>
  )
}

/** Map SeaArt-style resolution + ratio → WxH for Seedream */
export function sizeFromResRatio(res: '1K' | '2K' | '4K', ratio: string): string {
  const base = res === '1K' ? 1024 : res === '2K' ? 1536 : 2048
  const [a, b] = ratio.split(':').map((n) => parseInt(n, 10))
  if (!a || !b) return `${base}x${base}`
  if (a >= b) {
    return `${base}x${Math.max(64, Math.round((base * b) / a))}`
  }
  return `${Math.max(64, Math.round((base * a) / b))}x${base}`
}
