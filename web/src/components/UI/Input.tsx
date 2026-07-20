import { forwardRef, useId, type ReactNode } from 'react'
import { Input as ArcoInput } from '@arco-design/web-react'
import type { InputProps as ArcoInputProps, TextAreaProps as ArcoTextAreaProps } from '@arco-design/web-react'
import { cn } from '@/utils/utils.ts'

export type InputSize = 'sm' | 'md' | 'lg'
export type ArcoInputSize = 'mini' | 'small' | 'default' | 'large'
export type AnyInputSize = InputSize | ArcoInputSize

export type InputVariant = 'default' | 'filled' | 'outline' | 'underline' | 'wave'

export type InputStatus = 'error' | 'success' | 'warning'

export interface InputProps extends Omit<ArcoInputProps, 'size' | 'status'> {
  size?: AnyInputSize
  variant?: InputVariant
  status?: InputStatus
  block?: boolean
  prefix?: ReactNode
  suffix?: ReactNode
  allowClear?: boolean
  /** Floating label text — used with variant="wave" */
  label?: string
}

export interface TextAreaProps extends Omit<ArcoTextAreaProps, 'size' | 'status'> {
  size?: AnyInputSize
  variant?: InputVariant
  status?: InputStatus
  block?: boolean
}

function normalizeSize(size: AnyInputSize): InputSize {
  switch (size) {
    case 'mini':
    case 'small':
      return 'sm'
    case 'default':
    case 'md':
      return 'md'
    case 'large':
    case 'lg':
      return 'lg'
    default:
      return 'md'
  }
}

const sizeToArcoSize: Record<InputSize, ArcoInputProps['size']> = {
  sm: 'small',
  md: 'default',
  lg: 'large',
}

const sizeClassMap: Record<InputSize, string> = {
  sm: 'h-8 min-h-[32px] text-xs px-2.5',
  md: 'h-9 min-h-[36px] text-sm px-3',
  lg: 'h-10 min-h-[40px] text-base px-4',
}

const variantClassMap: Record<Exclude<InputVariant, 'wave'>, string> = {
  default: cn(
    'ui-input',
    'bg-[hsl(var(--background))] border border-[hsl(var(--border))]',
    'text-[hsl(var(--foreground))]',
    'placeholder:text-[hsl(var(--muted-foreground))]',
    'hover:border-[hsl(var(--primary)/0.4)]',
    'focus:border-[hsl(var(--primary))] focus:ring-1 focus:ring-[hsl(var(--primary)/0.2)]',
  ),
  filled: cn(
    'ui-input',
    'bg-[hsl(var(--secondary)/0.5)] border border-transparent',
    'text-[hsl(var(--foreground))]',
    'placeholder:text-[hsl(var(--muted-foreground))]',
    'hover:bg-[hsl(var(--secondary)/0.7)]',
    'focus:bg-[hsl(var(--background))] focus:border-[hsl(var(--primary))] focus:ring-1 focus:ring-[hsl(var(--primary)/0.2)]',
  ),
  outline: cn(
    'ui-input',
    'bg-transparent border border-[hsl(var(--border))]',
    'text-[hsl(var(--foreground))]',
    'placeholder:text-[hsl(var(--muted-foreground))]',
    'hover:border-[hsl(var(--primary)/0.4)]',
    'focus:border-[hsl(var(--primary))] focus:ring-1 focus:ring-[hsl(var(--primary)/0.2)]',
  ),
  underline: cn(
    'ui-input',
    'bg-transparent border-0 border-b border-[hsl(var(--border))]',
    'text-[hsl(var(--foreground))]',
    'placeholder:text-[hsl(var(--muted-foreground))]',
    'hover:border-[hsl(var(--primary)/0.4)]',
    'focus:border-[hsl(var(--primary))] focus:ring-0',
  ),
}

const statusClassMap: Record<InputStatus, string> = {
  error: 'border-red-500! focus:border-red-500! focus:ring-red-500/15',
  success: 'border-green-500! focus:border-green-500! focus:ring-green-500/15',
  warning: 'border-amber-500! focus:border-amber-500! focus:ring-amber-500/15',
}

const sharedBase = cn(
  'ui-input-base',
  'rounded-md transition-colors duration-150',
  'focus-visible:outline-none',
  'disabled:opacity-50 disabled:cursor-not-allowed disabled:hover:border-[hsl(var(--border))]',
  'read-only:opacity-80',
)

function resolveInputClasses(
  variant: Exclude<InputVariant, 'wave'>,
  size: InputSize,
  status?: InputStatus,
  block?: boolean,
  className?: string | string[],
) {
  return cn(
    sharedBase,
    variantClassMap[variant],
    sizeClassMap[size],
    status && statusClassMap[status],
    block && 'w-full',
    className,
  )
}

function WaveInputField({
  id,
  label,
  block,
  status,
  className,
  disabled,
  readOnly,
  value,
  defaultValue,
  onChange,
  onBlur,
  onFocus,
  placeholder,
  ...rest
}: InputProps & { id: string }) {
  const labelText = label ?? placeholder ?? ''
  const hasValue =
    (value != null && String(value).length > 0) ||
    (defaultValue != null && String(defaultValue).length > 0)

  return (
    <div className={cn('ui-wave-group', block !== false && 'w-full', className)}>
      <ArcoInput
        id={id}
        className={cn(
          'ui-wave-input',
          status === 'error' && 'ui-wave-input-error',
          status === 'success' && 'ui-wave-input-success',
          status === 'warning' && 'ui-wave-input-warning',
        )}
        disabled={disabled}
        readOnly={readOnly}
        value={value}
        defaultValue={defaultValue}
        onChange={onChange}
        onBlur={onBlur}
        onFocus={onFocus}
        placeholder=" "
        {...rest}
      />
      <span className="ui-wave-bar" aria-hidden />
      {labelText ? (
        <label
          htmlFor={id}
          className={cn('ui-wave-label', hasValue && 'ui-wave-label-active')}
        >
          {labelText.split('').map((char, index) => (
            <span
              key={`${char}-${index}`}
              className="ui-wave-label-char"
              style={{ ['--index' as string]: index }}
            >
              {char === ' ' ? '\u00a0' : char}
            </span>
          ))}
        </label>
      ) : null}
    </div>
  )
}

const InputRoot = forwardRef<any, InputProps>(function Input(
  {
    size = 'md',
    variant = 'default',
    status,
    block = true,
    prefix,
    suffix,
    allowClear = false,
    label,
    className,
    disabled,
    readOnly,
    id: idProp,
    ...rest
  },
  ref,
) {
  const autoId = useId()
  const inputId = idProp ?? autoId

  if (variant === 'wave') {
    return (
      <WaveInputField
        id={inputId}
        label={label}
        block={block}
        status={status}
        className={className}
        disabled={disabled}
        readOnly={readOnly}
        {...rest}
      />
    )
  }

  const normalizedSize = normalizeSize(size)
  const hasAffix = prefix || suffix || allowClear
  const inputClassName = resolveInputClasses(variant, normalizedSize, status, !hasAffix && block)

  if (hasAffix) {
    return (
      <div
        className={cn(
          sharedBase,
          variantClassMap[variant],
          sizeClassMap[normalizedSize],
          status && statusClassMap[status],
          block && 'w-full',
          'flex items-center gap-1.5',
          disabled && 'opacity-50 cursor-not-allowed',
          className,
        )}
      >
        {prefix && (
          <span className="ui-input-prefix shrink-0 text-[hsl(var(--muted-foreground))] flex items-center">
            {prefix}
          </span>
        )}
        <ArcoInput
          ref={ref}
          id={inputId}
          size={sizeToArcoSize[normalizedSize]}
          className="flex-1 min-w-0 bg-transparent border-0 outline-none focus:ring-0 focus-visible:ring-0 p-0 h-auto"
          disabled={disabled}
          readOnly={readOnly}
          {...rest}
        />
        {suffix && (
          <span className="ui-input-suffix shrink-0 text-[hsl(var(--muted-foreground))] flex items-center text-xs">
            {suffix}
          </span>
        )}
      </div>
    )
  }

  return (
    <ArcoInput
      ref={ref}
      id={inputId}
      size={sizeToArcoSize[normalizedSize]}
      className={inputClassName}
      disabled={disabled}
      readOnly={readOnly}
      allowClear={allowClear}
      {...rest}
    />
  )
})

InputRoot.displayName = 'Input'

const Password = forwardRef<any, InputProps>(function Password(
  { size = 'md', variant = 'default', status, block = true, className, disabled, readOnly, ...rest },
  ref,
) {
  const normalizedSize = normalizeSize(size)

  return (
    <div
      className={cn(
        sharedBase,
        variantClassMap[variant === 'wave' ? 'default' : variant],
        sizeClassMap[normalizedSize],
        status && statusClassMap[status],
        block && 'w-full',
        'flex items-center gap-1.5',
        disabled && 'opacity-50 cursor-not-allowed',
        className,
      )}
    >
      <ArcoInput.Password
        ref={ref}
        size={sizeToArcoSize[normalizedSize]}
        className="flex-1 min-w-0"
        disabled={disabled}
        readOnly={readOnly}
        {...rest}
      />
    </div>
  )
})

Password.displayName = 'Input.Password'

const TextArea = forwardRef<any, TextAreaProps>(function TextArea(
  { size = 'md', variant = 'default', status, block = true, className, disabled, readOnly, ...rest },
  ref,
) {
  const normalizedSize = normalizeSize(size)
  const resolvedVariant = variant === 'wave' ? 'default' : variant

  return (
    <ArcoInput.TextArea
      ref={ref}
      className={cn(
        'ui-input-base',
        'rounded-md transition-colors duration-150',
        'resize-none',
        variantClassMap[resolvedVariant],
        sizeClassMap[normalizedSize],
        status && statusClassMap[status],
        block && 'w-full',
        'focus-visible:outline-none',
        'disabled:opacity-50 disabled:cursor-not-allowed',
        className,
      )}
      disabled={disabled}
      readOnly={readOnly}
      {...rest}
    />
  )
})

TextArea.displayName = 'Input.TextArea'

const Search = forwardRef<any, InputProps & { onSearch?: (value: string) => void }>(function Search(
  { size = 'md', variant = 'default', status, block = true, className, disabled, readOnly, ...rest },
  ref,
) {
  const normalizedSize = normalizeSize(size)
  const resolvedVariant = variant === 'wave' ? 'default' : variant

  return (
    <div
      className={cn(
        sharedBase,
        variantClassMap[resolvedVariant],
        sizeClassMap[normalizedSize],
        status && statusClassMap[status],
        block && 'w-full',
        'flex items-center gap-1.5',
        disabled && 'opacity-50 cursor-not-allowed',
        className,
      )}
    >
      <ArcoInput.Search
        ref={ref}
        size={sizeToArcoSize[normalizedSize]}
        className="flex-1 min-w-0"
        disabled={disabled}
        readOnly={readOnly}
        {...rest}
      />
    </div>
  )
})

Search.displayName = 'Input.Search'

type InputCompound = typeof InputRoot & {
  Password: typeof Password
  TextArea: typeof TextArea
  Search: typeof Search
}

export const Input = InputRoot as InputCompound
Input.Password = Password
Input.TextArea = TextArea
Input.Search = Search

export default Input
