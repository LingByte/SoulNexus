import { forwardRef, useId, useImperativeHandle, useRef, useState } from 'react'
import { motion, type HTMLMotionProps } from 'framer-motion'
import { clsx } from 'clsx'

type Size = 'sm' | 'md' | 'lg'

type BaseMotionTextareaProps = Omit<HTMLMotionProps<'textarea'>, 'size'>

interface TextareaProps extends BaseMotionTextareaProps {
  label?: string
  error?: string
  helperText?: string
  size?: Size
  wrapperClassName?: string
  textareaClassName?: string
  onValueChange?: (value: string) => void
}

const sizeMap: Record<Size, { px: string; py: string; text: string }> = {
  sm: { px: 'px-3', py: 'py-1.5', text: 'text-sm' },
  md: { px: 'px-3.5', py: 'py-2', text: 'text-sm' },
  lg: { px: 'px-4', py: 'py-2.5', text: 'text-base' },
}

const Textarea = forwardRef<HTMLTextAreaElement, TextareaProps>(function Textarea(
  {
    className,
    textareaClassName,
    wrapperClassName,
    label,
    error,
    helperText,
    value,
    defaultValue,
    onChange,
    onValueChange,
    size = 'md',
    disabled,
    readOnly,
    rows = 3,
    ...restProps
  },
  ref,
) {
  const generatedId = useId()
  const textareaId = restProps.id ?? `textarea-${generatedId}`
  const errorId = `${textareaId}-error`
  const helpId = `${textareaId}-help`

  const isControlled = value !== undefined
  const [inner, setInner] = useState(String(defaultValue ?? ''))
  const currentValue = String(isControlled ? value ?? '' : inner)

  const textareaRef = useRef<HTMLTextAreaElement>(null)
  useImperativeHandle(ref, () => textareaRef.current as HTMLTextAreaElement)

  const handleChange = (e: React.ChangeEvent<HTMLTextAreaElement>) => {
    const val = e.target.value
    if (!isControlled) setInner(val)
    onValueChange?.(val)
    onChange?.(e)
  }

  const sizeTokens = sizeMap[size]

  const describedBy =
    [error ? errorId : null, helperText ? helpId : null, restProps['aria-describedby'] ?? null]
      .filter(Boolean)
      .join(' ') || undefined

  return (
    <div className={clsx('w-full', wrapperClassName)}>
      {label && (
        <motion.label
          htmlFor={textareaId}
          initial={{ opacity: 0, y: -4 }}
          animate={{ opacity: 1, y: 0 }}
          className="mb-1.5 block text-sm font-medium text-foreground"
        >
          {label}
        </motion.label>
      )}

      <motion.textarea
        ref={textareaRef}
        id={textareaId}
        rows={rows}
        value={currentValue}
        onChange={handleChange}
        disabled={disabled}
        readOnly={readOnly}
        aria-invalid={!!error || undefined}
        aria-describedby={describedBy}
        className={clsx(
          'w-full resize-y rounded-md border bg-background text-foreground placeholder:text-muted-foreground',
          'border-input focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring',
          sizeTokens.px,
          sizeTokens.py,
          sizeTokens.text,
          disabled && 'cursor-not-allowed opacity-60',
          readOnly && 'opacity-90',
          error && 'border-destructive focus-visible:ring-destructive',
          textareaClassName,
          className,
        )}
        {...(restProps as HTMLMotionProps<'textarea'>)}
      />

      <div className="mt-1.5">
        {error ? (
          <motion.p
            id={errorId}
            initial={{ opacity: 0, y: -4 }}
            animate={{ opacity: 1, y: 0 }}
            className="text-sm text-destructive"
          >
            {error}
          </motion.p>
        ) : helperText ? (
          <motion.p
            id={helpId}
            initial={{ opacity: 0, y: -4 }}
            animate={{ opacity: 1, y: 0 }}
            className="text-sm text-muted-foreground"
          >
            {helperText}
          </motion.p>
        ) : null}
      </div>
    </div>
  )
})

Textarea.displayName = 'Textarea'

export default Textarea
