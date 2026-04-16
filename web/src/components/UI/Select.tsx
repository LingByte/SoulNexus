import React, { createContext, useContext, useEffect, useLayoutEffect, useMemo, useRef, useState } from 'react'
import { ChevronDown, Search } from 'lucide-react'
import { cn } from '@/utils/cn.ts'
import { createPortal } from 'react-dom'

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

type SelectContextValue = {
  value: string
  onValueChange: (value: string) => void
  isOpen: boolean
  setIsOpen: (open: boolean) => void
  disabled: boolean
  searchTerm: string
  setSearchTerm: (value: string) => void
  registerItemLabel: (value: string, label: string) => void
  getItemLabel: (value: string) => string
  setPortalElement: (el: HTMLDivElement | null) => void
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
  const [searchTerm, setSearchTerm] = useState('')
  const rootRef = useRef<HTMLDivElement>(null)
  const portalRef = useRef<HTMLDivElement | null>(null)

  useEffect(() => {
    const handleClickOutside = (event: MouseEvent) => {
      if (!rootRef.current) return
      const target = event.target as Node
      const clickedInsideRoot = rootRef.current.contains(target)
      const clickedInsidePortal = portalRef.current?.contains(target) ?? false
      if (!clickedInsideRoot && !clickedInsidePortal) {
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
        setSearchTerm('')
      },
      isOpen,
      setIsOpen,
      disabled,
      searchTerm,
      setSearchTerm,
      registerItemLabel,
      getItemLabel: (itemValue: string) => labels[itemValue] ?? itemValue,
      setPortalElement: (el: HTMLDivElement | null) => {
        portalRef.current = el
      },
    }),
    [value, onValueChange, isOpen, disabled, labels, searchTerm],
  )

  return (
    <SelectContext.Provider value={ctxValue}>
      <div ref={rootRef} className={cn('relative', className)}>
        {children}
      </div>
    </SelectContext.Provider>
  )
}

const SelectTrigger: React.FC<SelectTriggerProps> = ({ children, className = '', selectedValue }) => {
  const { isOpen, setIsOpen, disabled } = useSelectContext()

  return (
    <button
      type="button"
      onClick={() => !disabled && setIsOpen(!isOpen)}
      disabled={disabled}
      className={cn(
        'flex h-10 w-full items-center justify-between rounded-lg border border-gray-200 bg-white px-3 py-2 text-sm font-medium text-gray-900 shadow-sm transition-all duration-200 hover:border-gray-300 hover:shadow-md focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-offset-0 disabled:cursor-not-allowed disabled:opacity-50 disabled:hover:border-gray-200 disabled:hover:shadow-sm dark:border-gray-700 dark:bg-gray-900 dark:text-gray-100',
        isOpen && 'border-blue-500 ring-2 ring-blue-500 ring-offset-0 shadow-md',
        className,
      )}
    >
      {selectedValue ? <span className="truncate">{selectedValue}</span> : children}
      <ChevronDown className={cn('h-4 w-4 text-gray-500 transition-all duration-200', isOpen && 'rotate-180 text-gray-700')} />
    </button>
  )
}

const SelectValue: React.FC<SelectValueProps> = ({ placeholder = '请选择', children }) => {
  const { value, getItemLabel } = useSelectContext()
  const displayValue = value ? getItemLabel(value) : children || placeholder
  return <span className={cn('truncate', !value && 'text-gray-500')}>{displayValue}</span>
}

const SelectContent: React.FC<SelectContentProps> = ({
  children,
  className = '',
  searchable = false,
  searchPlaceholder = '搜索选项...',
  emptyText = '暂无匹配项',
}) => {
  const { isOpen, searchTerm, setSearchTerm } = useSelectContext()
  const { setPortalElement } = useSelectContext()
  const [menuStyle, setMenuStyle] = useState<React.CSSProperties>({})
  const [menuWidth, setMenuWidth] = useState<number>(0)
  const anchorRef = useRef<HTMLDivElement>(null)

  const childArray = React.Children.toArray(children)
  const visibleChildren = searchable
    ? childArray.filter((child) => {
        if (!React.isValidElement(child)) return true
        const childValue = String((child.props as { value?: string }).value ?? '')
        const childText = String((child.props as { children?: React.ReactNode }).children ?? childValue)
        return childText.toLowerCase().includes(searchTerm.toLowerCase()) || childValue.toLowerCase().includes(searchTerm.toLowerCase())
      })
    : childArray

  useLayoutEffect(() => {
    if (!isOpen) return

    const updatePosition = () => {
      const triggerEl = anchorRef.current?.parentElement?.querySelector('button')
      if (!triggerEl) return
      const rect = triggerEl.getBoundingClientRect()
      const gap = 8
      const viewportHeight = window.innerHeight
      const preferredHeight = 280
      const spaceBelow = viewportHeight - rect.bottom - gap
      const spaceAbove = rect.top - gap
      const placeTop = spaceBelow < 180 && spaceAbove > spaceBelow
      const maxHeight = Math.max(140, Math.min(preferredHeight, placeTop ? spaceAbove - 8 : spaceBelow - 8))

      setMenuWidth(rect.width)
      setMenuStyle({
        position: 'fixed',
        left: rect.left,
        top: placeTop ? Math.max(8, rect.top - gap - maxHeight) : rect.bottom + gap,
        maxHeight,
      })
    }

    updatePosition()
    window.addEventListener('resize', updatePosition)
    window.addEventListener('scroll', updatePosition, true)
    return () => {
      window.removeEventListener('resize', updatePosition)
      window.removeEventListener('scroll', updatePosition, true)
    }
  }, [isOpen])

  if (!isOpen) {
    return <div ref={anchorRef} />
  }

  return (
    <>
      <div ref={anchorRef} />
      {createPortal(
        <div
          ref={setPortalElement}
          style={{ ...menuStyle, width: menuWidth || undefined }}
          className={cn(
            'z-[100000] rounded-lg border border-gray-200 bg-white py-1.5 shadow-xl ring-1 ring-black/5 dark:border-gray-700 dark:bg-gray-900',
            className,
          )}
        >
          {searchable && (
            <div className="sticky top-0 z-10 bg-white px-2 pb-2 pt-1 dark:bg-gray-900">
              <div className="relative">
                <Search className="absolute left-2 top-1/2 h-3.5 w-3.5 -translate-y-1/2 text-gray-400" />
                <input
                  value={searchTerm}
                  onChange={(e) => setSearchTerm(e.target.value)}
                  placeholder={searchPlaceholder}
                  className="h-8 w-full rounded-md border border-gray-200 bg-white pl-7 pr-2 text-xs text-gray-900 outline-none focus:border-blue-400 dark:border-gray-700 dark:bg-gray-800 dark:text-gray-100"
                />
              </div>
            </div>
          )}
          <div className="overflow-y-auto" style={{ maxHeight: (menuStyle.maxHeight as number) ? (menuStyle.maxHeight as number) - (searchable ? 44 : 0) : 220 }}>
            {visibleChildren.length > 0 ? visibleChildren : <div className="px-3 py-2 text-xs text-gray-500">{emptyText}</div>}
          </div>
        </div>,
        document.body,
      )}
    </>
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
        'relative flex w-full cursor-pointer select-none items-center rounded-md px-3 py-2 text-sm font-medium text-gray-700 outline-none transition-colors duration-150 hover:bg-gray-100 focus:bg-gray-100 active:bg-gray-200 dark:text-gray-200 dark:hover:bg-gray-800 dark:focus:bg-gray-800',
        isSelected && 'text-blue-600 dark:text-blue-400',
        className,
      )}
    >
      <span className="flex-1 text-left">{children}</span>
    </button>
  )
}

export { Select, SelectTrigger, SelectContent, SelectItem, SelectValue }
