import React, { createContext, useContext, useEffect, useMemo, useRef, useState } from 'react'
import { Check, ChevronDown } from 'lucide-react'
import { cn } from '@/utils/cn.ts'

interface SelectProps {
  value: string
  onValueChange: (value: string) => void
  children: React.ReactNode
  disabled?: boolean
  className?: string
}

interface SelectTriggerProps {
  children: React.ReactNode
  className?: string
  selectedValue?: string
}

interface SelectContentProps {
  children: React.ReactNode
  className?: string
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

type SelectContextValue = {
  value: string
  onValueChange: (value: string) => void
  isOpen: boolean
  setIsOpen: (open: boolean) => void
  disabled: boolean
  registerItemLabel: (value: string, label: string) => void
  getItemLabel: (value: string) => string
}

const SelectContext = createContext<SelectContextValue | null>(null)

const useSelectContext = () => {
  const ctx = useContext(SelectContext)
  if (!ctx) {
    throw new Error('Select sub-components must be used within Select')
  }
  return ctx
}

const Select: React.FC<SelectProps> = ({
  value,
  onValueChange,
  children,
  disabled = false,
  className = '',
}) => {
  const [isOpen, setIsOpen] = useState(false)
  const [labels, setLabels] = useState<Record<string, string>>({})
  const rootRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    const handleClickOutside = (event: MouseEvent) => {
      if (!rootRef.current) return
      if (!rootRef.current.contains(event.target as Node)) {
        setIsOpen(false)
      }
    }
    document.addEventListener('mousedown', handleClickOutside)
    return () => document.removeEventListener('mousedown', handleClickOutside)
  }, [])

  const registerItemLabel = (itemValue: string, label: string) => {
    setLabels((prev) => {
      if (prev[itemValue] === label) return prev
      return { ...prev, [itemValue]: label }
    })
  }

  const ctxValue = useMemo<SelectContextValue>(
    () => ({
      value,
      onValueChange: (nextValue: string) => {
        onValueChange(nextValue)
        setIsOpen(false)
      },
      isOpen,
      setIsOpen,
      disabled,
      registerItemLabel,
      getItemLabel: (itemValue: string) => labels[itemValue] ?? itemValue,
    }),
    [value, onValueChange, isOpen, disabled, labels],
  )

  return (
    <SelectContext.Provider value={ctxValue}>
      <div ref={rootRef} className={cn('relative', className)}>
        {children}
      </div>
    </SelectContext.Provider>
  )
}

const SelectTrigger: React.FC<SelectTriggerProps> = ({ children, className = '' }) => {
  const { isOpen, setIsOpen, disabled } = useSelectContext()

  return (
    <button
      type="button"
      onClick={() => !disabled && setIsOpen(!isOpen)}
      disabled={disabled}
      className={cn(
        'flex h-10 w-full items-center justify-between rounded-lg border border-gray-200 bg-white px-3 py-2 text-sm font-medium text-gray-900 shadow-sm transition-all duration-200 hover:border-gray-300 hover:shadow-md focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-offset-0 disabled:cursor-not-allowed disabled:opacity-50 disabled:hover:border-gray-200 disabled:hover:shadow-sm',
        isOpen && 'border-blue-500 ring-2 ring-blue-500 ring-offset-0 shadow-md',
        className,
      )}
    >
      {children}
      <ChevronDown className={cn('h-4 w-4 text-gray-500 transition-all duration-200', isOpen && 'rotate-180 text-gray-700')} />
    </button>
  )
}

const SelectValue: React.FC<SelectValueProps> = ({ placeholder = '请选择', children }) => {
  const { value, getItemLabel } = useSelectContext()
  const displayValue = value ? getItemLabel(value) : children || placeholder
  return <span className={cn('truncate', !value && 'text-gray-500')}>{displayValue}</span>
}

const SelectContent: React.FC<SelectContentProps> = ({ children, className = '' }) => {
  const { isOpen } = useSelectContext()
  if (!isOpen) return null

  return (
    <div
      className={cn(
        'absolute top-full left-0 right-0 z-[9999] mt-1.5 max-h-60 overflow-auto rounded-lg border border-gray-200 bg-white py-1.5 shadow-xl ring-1 ring-black/5',
        className,
      )}
    >
      {children}
    </div>
  )
}

const SelectItem: React.FC<SelectItemProps> = ({ value, children, className = '' }) => {
  const { value: selectedValue, onValueChange, registerItemLabel } = useSelectContext()
  const isSelected = value === selectedValue

  useEffect(() => {
    const text = typeof children === 'string' ? children : ''
    registerItemLabel(value, text || value)
  }, [children, value, registerItemLabel])

  return (
    <button
      type="button"
      onClick={() => onValueChange(value)}
      className={cn(
        'relative flex w-full cursor-pointer select-none items-center rounded-md py-2 pl-9 pr-3 text-sm font-medium outline-none transition-colors duration-150 hover:bg-gray-50 focus:bg-gray-50 active:bg-gray-100',
        isSelected && 'bg-blue-50 text-blue-900 hover:bg-blue-50 focus:bg-blue-50',
        className,
      )}
    >
      {isSelected && (
        <span className="absolute left-2.5 flex h-4 w-4 items-center justify-center">
          <Check className="h-4 w-4 text-blue-600" />
        </span>
      )}
      <span className={cn('flex-1 text-left', isSelected && 'text-blue-900')}>{children}</span>
    </button>
  )
}

export { Select, SelectTrigger, SelectContent, SelectItem, SelectValue }
