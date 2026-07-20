import * as React from 'react'
import { cn } from '@/lib/utils'
import { Input } from '@/components/ui/input'

const PRESETS = ['#165DFF', '#0EA5E9', '#10B981', '#F59E0B', '#EF4444', '#8B5CF6', '#18181B', '#E3E3E6']

function normalizeHex(value: string): string {
  const v = value.trim()
  if (/^#[0-9A-Fa-f]{6}$/.test(v)) return v.toUpperCase()
  if (/^[0-9A-Fa-f]{6}$/.test(v)) return `#${v.toUpperCase()}`
  if (/^#[0-9A-Fa-f]{3}$/.test(v)) {
    const r = v[1]
    const g = v[2]
    const b = v[3]
    return `#${r}${r}${g}${g}${b}${b}`.toUpperCase()
  }
  return value
}

function toPickerValue(value: string): string {
  const n = normalizeHex(value)
  return /^#[0-9A-Fa-f]{6}$/.test(n) ? n : '#165DFF'
}

export type ColorPickerProps = {
  id?: string
  value: string
  onChange: (hex: string) => void
  className?: string
}

export function ColorPicker({ id, value, onChange, className }: ColorPickerProps) {
  const pickerValue = toPickerValue(value)
  const [text, setText] = React.useState(pickerValue)

  React.useEffect(() => {
    setText(toPickerValue(value))
  }, [value])

  const commitText = () => {
    const n = normalizeHex(text)
    if (/^#[0-9A-Fa-f]{6}$/.test(n)) {
      onChange(n)
      setText(n)
    } else {
      setText(pickerValue)
    }
  }

  return (
    <div className={cn('space-y-2.5', className)}>
      <div className="flex items-center gap-2.5">
        <label
          className={cn(
            'relative h-10 w-10 shrink-0 cursor-pointer overflow-hidden rounded-lg border border-black/10 shadow-sm',
            'ring-offset-background transition hover:ring-2 hover:ring-ring/25',
          )}
          style={{ backgroundColor: pickerValue }}
          title="选择颜色"
        >
          <input
            id={id}
            type="color"
            value={pickerValue}
            onChange={(e) => {
              const next = e.target.value.toUpperCase()
              onChange(next)
              setText(next)
            }}
            className="absolute inset-0 h-full w-full cursor-pointer opacity-0"
          />
        </label>
        <Input
          value={text}
          onChange={(e) => setText(e.target.value)}
          onBlur={commitText}
          onKeyDown={(e) => {
            if (e.key === 'Enter') {
              e.currentTarget.blur()
            }
          }}
          spellCheck={false}
          className="font-mono text-xs uppercase"
          placeholder="#165DFF"
          aria-label="主题色十六进制"
        />
      </div>
      <div className="flex flex-wrap gap-1.5">
        {PRESETS.map((c) => {
          const active = pickerValue.toUpperCase() === c.toUpperCase()
          return (
            <button
              key={c}
              type="button"
              title={c}
              aria-label={`预设 ${c}`}
              onClick={() => {
                onChange(c)
                setText(c)
              }}
              className={cn(
                'h-7 w-7 rounded-md border border-black/10 shadow-sm transition',
                'hover:scale-105 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring/40',
                active && 'ring-2 ring-offset-1 ring-foreground/30',
              )}
              style={{ backgroundColor: c }}
            />
          )
        })}
      </div>
    </div>
  )
}
