import { useCallback, useEffect, useId, useRef, type InputHTMLAttributes, type ReactNode } from 'react'
import { cn } from '@/utils/utils.ts'

export type CheckboxColor = 'blue' | 'green' | 'purple' | 'red'
export type CheckboxSize = 'sm' | 'md' | 'lg'

export interface CheckboxProps extends Omit<InputHTMLAttributes<HTMLInputElement>, 'type' | 'onChange' | 'size'> {
  checked?: boolean
  defaultChecked?: boolean
  indeterminate?: boolean
  onChange?: (checked: boolean, e: React.ChangeEvent<HTMLInputElement>) => void
  disabled?: boolean
  color?: CheckboxColor
  size?: CheckboxSize
  className?: string
  children?: ReactNode
}

export function Checkbox({
  checked,
  defaultChecked,
  indeterminate = false,
  onChange,
  disabled,
  color = 'blue',
  size = 'md',
  className,
  children,
  id,
  ...rest
}: CheckboxProps) {
  const autoId = useId()
  const inputId = id ?? autoId
  const inputRef = useRef<HTMLInputElement>(null)

  useEffect(() => {
    if (inputRef.current) {
      inputRef.current.indeterminate = !!indeterminate
    }
  }, [indeterminate])

  const handleChange = useCallback(
    (e: React.ChangeEvent<HTMLInputElement>) => {
      onChange?.(e.target.checked, e)
    },
    [onChange],
  )

  const control = (
    <label
      className={cn(
        'ui-ios-checkbox',
        `ui-ios-checkbox--${color}`,
        `ui-ios-checkbox--${size}`,
        disabled && 'ui-ios-checkbox--disabled',
        className,
      )}
    >
      <input
        ref={inputRef}
        id={inputId}
        type="checkbox"
        checked={checked}
        defaultChecked={defaultChecked}
        disabled={disabled}
        onChange={handleChange}
        {...rest}
      />
      <span className="ui-ios-checkbox-wrapper" aria-hidden>
        <span className="ui-ios-checkbox-bg" />
        <svg fill="none" viewBox="0 0 24 24" className="ui-ios-checkbox-icon">
          <path
            strokeLinejoin="round"
            strokeLinecap="round"
            strokeWidth={3}
            stroke="currentColor"
            d={indeterminate ? 'M6 12H18' : 'M4 12L10 18L20 6'}
            className="ui-ios-checkbox-path"
          />
        </svg>
      </span>
    </label>
  )

  if (!children) return control

  return (
    <label htmlFor={inputId} className="ui-ios-checkbox-field">
      {control}
      <span className="ui-ios-checkbox-label">{children}</span>
    </label>
  )
}

export default Checkbox
