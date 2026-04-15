import React from 'react'
import { cn } from '@/utils/cn.ts'

interface SimpleTabsProps {
  value: string
  onValueChange: (value: string) => void
  children: React.ReactNode
  className?: string
}

interface SimpleTabsListProps {
  children: React.ReactNode
  className?: string
}

interface SimpleTabsTriggerProps {
  value: string
  children: React.ReactNode
  className?: string
}

interface SimpleTabsContentProps {
  value: string
  children: React.ReactNode
  className?: string
}

const SimpleTabs: React.FC<SimpleTabsProps> = ({
  value,
  onValueChange,
  children,
  className = ''
}) => {
  return (
    <div className={cn('w-full', className)}>
      {React.Children.map(children, (child) => {
        if (React.isValidElement(child)) {
          return React.cloneElement(child, {
            currentValue: value,
            onValueChange
          } as any)
        }
        return child
      })}
    </div>
  )
}

const SimpleTabsList: React.FC<SimpleTabsListProps & { currentValue?: string; onValueChange?: (value: string) => void }> = ({
  children,
  className = '',
  currentValue,
  onValueChange
}) => {
  return (
    <div className={cn(
      'inline-flex h-10 items-center justify-center rounded-md bg-gray-100 p-1 text-gray-500',
      className
    )}>
      {React.Children.map(children, (child) => {
        if (React.isValidElement(child)) {
          return React.cloneElement(child, {
            currentValue,
            onValueChange
          } as any)
        }
        return child
      })}
    </div>
  )
}

const SimpleTabsTrigger: React.FC<SimpleTabsTriggerProps & { currentValue?: string; onValueChange?: (value: string) => void }> = ({
  value,
  children,
  className = '',
  currentValue,
  onValueChange
}) => {
  const isSelected = currentValue === value

  return (
    <button
      type="button"
      onClick={() => onValueChange?.(value)}
      className={cn(
        'inline-flex items-center justify-center whitespace-nowrap rounded-sm px-3 py-1.5 text-sm font-medium ring-offset-white transition-all focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary focus-visible:ring-offset-2 disabled:pointer-events-none disabled:opacity-50',
        isSelected 
          ? 'bg-white text-gray-950 shadow-sm' 
          : 'text-gray-500 hover:text-gray-900',
        className
      )}
    >
      {children}
    </button>
  )
}

const SimpleTabsContent: React.FC<SimpleTabsContentProps & { currentValue?: string }> = ({
  value,
  children,
  className = '',
  currentValue
}) => {
  const isSelected = currentValue === value

  if (!isSelected) {
    return null
  }

  return (
    <div className={cn(
      'mt-2 ring-offset-white focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary focus-visible:ring-offset-2',
      className
    )}>
      {children}
    </div>
  )
}

export { 
  SimpleTabs as Tabs, 
  SimpleTabsList as TabsList, 
  SimpleTabsTrigger as TabsTrigger, 
  SimpleTabsContent as TabsContent 
}
