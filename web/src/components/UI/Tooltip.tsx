import { useState, useRef, useEffect } from 'react'
import { createPortal } from 'react-dom'
import { cn } from '@/utils/cn.ts'

interface TooltipProps {
  content: React.ReactNode
  children: React.ReactNode
  placement?: 'top' | 'bottom' | 'left' | 'right'
  trigger?: 'hover' | 'click' | 'focus'
  delay?: number
  className?: string
  disabled?: boolean
}

const Tooltip = ({
  content,
  children,
  placement = 'top',
  trigger = 'hover',
  delay = 200,
  className,
  disabled = false
}: TooltipProps) => {
  const [isVisible, setIsVisible] = useState(false)
  const [position, setPosition] = useState({ x: 0, y: 0 })
  const triggerRef = useRef<HTMLDivElement>(null)
  const tooltipRef = useRef<HTMLDivElement>(null)
  const timeoutRef = useRef<NodeJS.Timeout>()

  const showTooltip = () => {
    if (disabled) return
    clearTimeout(timeoutRef.current)
    timeoutRef.current = setTimeout(() => {
      setIsVisible(true)
      updatePosition()
    }, delay)
  }

  const hideTooltip = () => {
    clearTimeout(timeoutRef.current)
    setIsVisible(false)
  }

  const updatePosition = () => {
    if (!triggerRef.current || !tooltipRef.current) return

    const triggerRect = triggerRef.current.getBoundingClientRect()
    const tooltipRect = tooltipRef.current.getBoundingClientRect()
    const scrollX = window.pageXOffset || document.documentElement.scrollLeft
    const scrollY = window.pageYOffset || document.documentElement.scrollTop

    let x = 0
    let y = 0

    switch (placement) {
      case 'top':
        x = triggerRect.left + scrollX + triggerRect.width / 2 - tooltipRect.width / 2
        y = triggerRect.top + scrollY - tooltipRect.height - 8
        break
      case 'bottom':
        x = triggerRect.left + scrollX + triggerRect.width / 2 - tooltipRect.width / 2
        y = triggerRect.bottom + scrollY + 8
        break
      case 'left':
        x = triggerRect.left + scrollX - tooltipRect.width - 8
        y = triggerRect.top + scrollY + triggerRect.height / 2 - tooltipRect.height / 2
        break
      case 'right':
        x = triggerRect.right + scrollX + 8
        y = triggerRect.top + scrollY + triggerRect.height / 2 - tooltipRect.height / 2
        break
    }

    // 边界检查
    const viewportWidth = window.innerWidth
    const viewportHeight = window.innerHeight

    if (x < 8) x = 8
    if (x + tooltipRect.width > viewportWidth - 8) {
      x = viewportWidth - tooltipRect.width - 8
    }
    if (y < 8) y = 8
    if (y + tooltipRect.height > viewportHeight - 8) {
      y = viewportHeight - tooltipRect.height - 8
    }

    setPosition({ x, y })
  }

  useEffect(() => {
    if (isVisible) {
      updatePosition()
      const handleResize = () => updatePosition()
      const handleScroll = () => updatePosition()
      
      window.addEventListener('resize', handleResize)
      window.addEventListener('scroll', handleScroll, true)
      
      return () => {
        window.removeEventListener('resize', handleResize)
        window.removeEventListener('scroll', handleScroll, true)
      }
    }
  }, [isVisible, placement])

  const handleMouseEnter = () => {
    if (trigger === 'hover') showTooltip()
  }

  const handleMouseLeave = () => {
    if (trigger === 'hover') hideTooltip()
  }

  const handleClick = () => {
    if (trigger === 'click') {
      setIsVisible(!isVisible)
      if (!isVisible) updatePosition()
    }
  }

  const handleFocus = () => {
    if (trigger === 'focus') showTooltip()
  }

  const handleBlur = () => {
    if (trigger === 'focus') hideTooltip()
  }

  const getArrowClasses = () => {
    const baseClasses = 'absolute w-2 h-2 bg-neutral-900 dark:bg-neutral-100 transform rotate-45'
    
    switch (placement) {
      case 'top':
        return `${baseClasses} -bottom-1 left-1/2 -translate-x-1/2`
      case 'bottom':
        return `${baseClasses} -top-1 left-1/2 -translate-x-1/2`
      case 'left':
        return `${baseClasses} -right-1 top-1/2 -translate-y-1/2`
      case 'right':
        return `${baseClasses} -left-1 top-1/2 -translate-y-1/2`
      default:
        return baseClasses
    }
  }

  return (
    <>
      <div
        ref={triggerRef}
        onMouseEnter={handleMouseEnter}
        onMouseLeave={handleMouseLeave}
        onClick={handleClick}
        onFocus={handleFocus}
        onBlur={handleBlur}
        className="inline-block"
      >
        {children}
      </div>

      {isVisible && createPortal(
        <div
          ref={tooltipRef}
          className={cn(
            'fixed z-50 px-2 py-1 text-sm text-white bg-neutral-900 dark:bg-neutral-100 dark:text-neutral-900 rounded shadow-lg pointer-events-none',
            className
          )}
          style={{
            left: position.x,
            top: position.y,
          }}
        >
          {content}
          <div className={getArrowClasses()} />
        </div>,
        document.body
      )}
    </>
  )
}

export default Tooltip
