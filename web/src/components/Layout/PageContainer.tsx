import { ReactNode } from 'react'
import { cn } from '@/utils/cn.ts'

interface PageContainerProps {
  children: ReactNode
  className?: string
  maxWidth?: 'sm' | 'md' | 'lg' | 'xl' | '2xl' | 'full'
  padding?: 'none' | 'sm' | 'md' | 'lg'
  fullHeight?: boolean
  scrollable?: boolean
}

const PageContainer = ({
  children,
  className,
  maxWidth = 'xl',
  padding = 'md',
  fullHeight = false,
  scrollable = false
}: PageContainerProps) => {
  const maxWidthClasses = {
    sm: 'max-w-2xl',
    md: 'max-w-4xl',
    lg: 'max-w-6xl',
    xl: 'max-w-7xl',
    '2xl': 'max-w-8xl',
    full: 'max-w-full'
  }

  const paddingClasses = {
    none: '',
    sm: 'px-4 py-6 sm:px-6',
    md: 'px-4 py-6 sm:px-6 lg:px-8',
    lg: 'px-6 py-8 sm:px-8 lg:px-10'
  }

  return (
    <div className={cn(
      'mx-auto w-full',
      maxWidthClasses[maxWidth],
      paddingClasses[padding],
      fullHeight && 'h-full flex flex-col',
      scrollable && 'overflow-y-auto',
      className
    )}>
      {children}
    </div>
  )
}

export default PageContainer
