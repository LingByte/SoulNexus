import { forwardRef } from 'react'
import { Input as ArcoInput } from '@arco-design/web-react'
import { cn } from '@/utils/cn.ts'

interface TextareaProps {
  label?: string
  error?: string | boolean
  helperText?: string
  size?: 'small' | 'default' | 'large'
  wrapperClassName?: string
  onValueChange?: (value: string) => void
  value?: string
  onChange?: (value: string) => void
  disabled?: boolean
  readOnly?: boolean
  rows?: number
  placeholder?: string
  className?: string
  [key: string]: any
}

const Textarea = forwardRef<any, TextareaProps>(function Textarea(
  {
    className,
    wrapperClassName,
    label,
    error,
    helperText,
    value,
    onChange,
    onValueChange,
    size = 'default',
    disabled,
    readOnly,
    rows = 3,
    placeholder,
  },
  ref,
) {
  const PURPLE_PRIMARY = '#6d28d9'

  const handleChange = (val: string) => {
    onValueChange?.(val)
  }

  return (
    <div className={cn('w-full', wrapperClassName)}>
      {label && (
        <label className="mb-2 block text-sm font-medium text-foreground">
          {label}
        </label>
      )}

      <ArcoInput.TextArea
        ref={ref}
        value={value}
        onChange={(val: string) => {
          handleChange(val)
          onChange?.(val)
        }}
        disabled={disabled}
        readOnly={readOnly}
        rows={rows}
        placeholder={placeholder}
        className={cn('w-full', className)}
        status={error ? 'error' : undefined}
        style={{
          '--color-primary': PURPLE_PRIMARY,
        } as React.CSSProperties}
      />

      <div className="mt-1.5">
        {error ? (
          <p className="text-sm text-destructive">
            {error}
          </p>
        ) : helperText ? (
          <p className="text-sm text-muted-foreground">
            {helperText}
          </p>
        ) : null}
      </div>
    </div>
  )
})

Textarea.displayName = 'Textarea'

export default Textarea
