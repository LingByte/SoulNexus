import React, { forwardRef } from 'react'
import { Select as ArcoSelect } from '@arco-design/web-react'
import type { SelectProps as ArcoSelectProps } from '@arco-design/web-react'
import { cn } from '@/utils/utils.ts'

export type SelectSize = 'sm' | 'md' | 'lg'
export type ArcoSelectSize = 'mini' | 'small' | 'default' | 'large'
export type AnySelectSize = SelectSize | ArcoSelectSize

export interface SelectOption {
  label: React.ReactNode
  value: string | number
  disabled?: boolean
}

export type SelectVariant = 'default' | 'filled' | 'outline'

/** Minimum option count before search is enabled by default */
export const SELECT_SEARCH_THRESHOLD = 6

export interface SelectProps
  extends Omit<ArcoSelectProps, 'size' | 'options' | 'onChange' | 'showSearch' | 'filterOption'> {
  size?: AnySelectSize
  options?: SelectOption[]
  style?: React.CSSProperties
  block?: boolean
  /** Override auto search; default on when options.length >= SELECT_SEARCH_THRESHOLD */
  showSearch?: boolean
  filterOption?: (inputValue: string, option: SelectOption) => boolean
  onChange?: (value: any, option: any) => void
}

function normalizeSelectSize(size: AnySelectSize): SelectSize {
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

const sizeToArcoSize: Record<SelectSize, ArcoSelectProps['size']> = {
  sm: 'small',
  md: 'default',
  lg: 'large',
}

function optionSearchText(option: unknown): string {
  if (option == null) return ''
  if (typeof option === 'string' || typeof option === 'number') return String(option)
  if (typeof option === 'object') {
    const o = option as Record<string, unknown>
    if (o.label != null) return String(o.label)
    const props = o.props as Record<string, unknown> | undefined
    if (props?.children != null) return String(props.children)
    if (props?.value != null) return String(props.value)
    if (o.value != null) return String(o.value)
  }
  return ''
}

function defaultFilterOption(inputValue: string, option: unknown): boolean {
  const q = inputValue.trim().toLowerCase()
  if (!q) return true
  return optionSearchText(option).toLowerCase().includes(q)
}

const SelectRoot = forwardRef<any, SelectProps>(function Select(
  {
    size = 'md',
    options,
    block = true,
    className,
    style,
    onChange,
    placeholder,
    allowClear,
    showSearch: showSearchProp,
    filterOption,
    children,
    ...rest
  },
  ref,
) {
  const normalizedSize = normalizeSelectSize(size)
  const rootClasses = cn(block && 'w-full', className)
  const optionCount = options?.length ?? 0
  const showSearch = showSearchProp ?? optionCount >= SELECT_SEARCH_THRESHOLD

  return (
    <ArcoSelect
      ref={ref}
      size={sizeToArcoSize[normalizedSize]}
      placeholder={placeholder ?? '请选择'}
      allowClear={allowClear !== undefined ? allowClear : true}
      showSearch={showSearch}
      filterOption={
        filterOption
          ? (inputValue: string, option: any) => filterOption(inputValue, option as SelectOption)
          : defaultFilterOption
      }
      options={options?.map((opt) => ({
        label: opt.label,
        value: opt.value,
        disabled: opt.disabled,
      }))}
      className={rootClasses}
      style={style}
      onChange={onChange}
      {...rest}
    >
      {children}
    </ArcoSelect>
  )
})

SelectRoot.displayName = 'Select'

type SelectCompound = typeof SelectRoot & {
  Option: typeof ArcoSelect.Option
  OptGroup: typeof ArcoSelect.OptGroup
}

export const Select = SelectRoot as SelectCompound
Select.Option = ArcoSelect.Option
Select.OptGroup = ArcoSelect.OptGroup

export default Select
