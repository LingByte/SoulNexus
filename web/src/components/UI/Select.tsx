import React, { useMemo } from 'react'
import { Select as ArcoSelect, SelectProps as ArcoSelectProps } from '@arco-design/web-react'
import { cn } from '@/utils/cn.ts'

interface SelectProps extends Omit<ArcoSelectProps, 'onChange'> {
  value?: string
  onValueChange?: (value: string) => void
  children?: React.ReactNode
  disabled?: boolean
  className?: string
  searchable?: boolean
  searchPlaceholder?: string
  emptyText?: string
  options?: Array<{ label: string; value: string; disabled?: boolean }>
}

interface SelectTriggerProps {
  children: React.ReactNode
  className?: string
  selectedValue?: string
}

interface SelectContentProps {
  children: React.ReactNode
  className?: string
  searchable?: boolean
  searchPlaceholder?: string
  emptyText?: string
}

interface SelectItemProps {
  value: string
  children: React.ReactNode
  className?: string
}

interface SelectValueProps {
  placeholder?: string
  children?: React.ReactNode
}

const Select = React.forwardRef<any, SelectProps>(
  ({
    value,
    onValueChange,
    children,
    disabled = false,
    className = '',
    searchable = false,
    searchPlaceholder = '搜索选项...',
    emptyText = '无选项',
    options,
    ...props
  }, ref) => {
    const PURPLE_PRIMARY = '#6d28d9'

    // 如果传入 options，转换为 ArcoSelect 的格式
    const arcoOptions = useMemo(() => {
      if (!options) return undefined
      return options.map(opt => ({
        label: opt.label,
        value: opt.value,
        disabled: opt.disabled
      }))
    }, [options])

    return (
      <ArcoSelect
        ref={ref}
        value={value}
        onChange={(val: string) => onValueChange?.(val)}
        disabled={disabled}
        allowClear
        showSearch={searchable}
        filterOption={searchable}
        placeholder={searchPlaceholder}
        notFoundContent={emptyText}
        options={arcoOptions}
        className={cn('w-full', className)}
        style={{
          '--color-primary': PURPLE_PRIMARY,
        } as React.CSSProperties}
        {...props}
      >
        {!arcoOptions && children}
      </ArcoSelect>
    )
  }
)

Select.displayName = 'Select'

// 导出子组件接口以保持兼容性
export const SelectTrigger: React.FC<SelectTriggerProps> = ({ children, className }) => (
  <div className={cn('w-full', className)}>{children}</div>
)

export const SelectContent: React.FC<SelectContentProps> = ({ children }) => (
  <>{children}</>
)

export const SelectItem: React.FC<SelectItemProps> = ({ value, children }) => (
  <div data-value={value}>{children}</div>
)

export const SelectValue: React.FC<SelectValueProps> = ({ placeholder, children }) => (
  <span>{children || placeholder}</span>
)

export { Select }
export default Select
