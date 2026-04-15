import { useState, useRef, useEffect } from 'react'
import { createPortal } from 'react-dom'
import { X } from 'lucide-react'
import { cn } from '@/utils/cn.ts'

interface PopoverProps {
  children: React.ReactNode
  content: React.ReactNode
  open?: boolean
  onOpenChange?: (open: boolean) => void
  placement?: 'top' | 'bottom' | 'left' | 'right'
  trigger?: 'click' | 'hover'
  className?: string
  contentClassName?: string
  showCloseButton?: boolean
  closeOnClickOutside?: boolean
}

const Popover = ({
  children,
  content,
  open: controlledOpen,
  onOpenChange,
  placement = 'bottom',
  trigger = 'click',
  className,
  contentClassName,
  showCloseButton = false,
  closeOnClickOutside = true
}: PopoverProps) => {
  const [isOpen, setIsOpen] = useState(false)
  const [position, setPosition] = useState({ x: 0, y: 0 })
  const triggerRef = useRef<HTMLDivElement>(null)
  const contentRef = useRef<HTMLDivElement>(null)

  const isControlled = controlledOpen !== undefined
  const open = isControlled ? controlledOpen : isOpen

  const updatePosition = () => {
    if (!triggerRef.current || !contentRef.current) return

    const triggerRect = triggerRef.current.getBoundingClientRect()
    const contentRect = contentRef.current.getBoundingClientRect()
    const scrollX = window.pageXOffset || document.documentElement.scrollLeft
    const scrollY = window.pageYOffset || document.documentElement.scrollTop

    let x = 0
    let y = 0

    switch (placement) {
      case 'top':
        x = triggerRect.left + scrollX + triggerRect.width / 2 - contentRect.width / 2
        y = triggerRect.top + scrollY - contentRect.height - 8
        break
      case 'bottom':
        x = triggerRect.left + scrollX + triggerRect.width / 2 - contentRect.width / 2
        y = triggerRect.bottom + scrollY + 8
        break
      case 'left':
        x = triggerRect.left + scrollX - contentRect.width - 8
        y = triggerRect.top + scrollY + triggerRect.height / 2 - contentRect.height / 2
        break
      case 'right':
        x = triggerRect.right + scrollX + 8
        y = triggerRect.top + scrollY + triggerRect.height / 2 - contentRect.height / 2
        break
    }

    // 边界检查
    const viewportWidth = window.innerWidth
    const viewportHeight = window.innerHeight

    if (x < 8) x = 8
    if (x + contentRect.width > viewportWidth - 8) {
      x = viewportWidth - contentRect.width - 8
    }
    if (y < 8) y = 8
    if (y + contentRect.height > viewportHeight - 8) {
      y = viewportHeight - contentRect.height - 8
    }

    setPosition({ x, y })
  }

  const handleOpen = () => {
    if (!isControlled) {
      setIsOpen(true)
    }
    onOpenChange?.(true)
  }

  const handleClose = () => {
    if (!isControlled) {
      setIsOpen(false)
    }
    onOpenChange?.(false)
  }

  const handleTriggerClick = () => {
    if (trigger === 'click') {
      open ? handleClose() : handleOpen()
    }
  }

  const handleTriggerMouseEnter = () => {
    if (trigger === 'hover') {
      handleOpen()
    }
  }

  const handleTriggerMouseLeave = () => {
    if (trigger === 'hover') {
      handleClose()
    }
  }

  useEffect(() => {
    if (open) {
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
  }, [open, placement])

  useEffect(() => {
    if (closeOnClickOutside && open) {
      const handleClickOutside = (event: MouseEvent) => {
        if (
          triggerRef.current &&
          contentRef.current &&
          !triggerRef.current.contains(event.target as Node) &&
          !contentRef.current.contains(event.target as Node)
        ) {
          handleClose()
        }
      }

      document.addEventListener('mousedown', handleClickOutside)
      return () => document.removeEventListener('mousedown', handleClickOutside)
    }
  }, [open, closeOnClickOutside])

  const getArrowClasses = () => {
    const baseClasses = 'absolute w-2 h-2 bg-white dark:bg-neutral-800 border border-neutral-200 dark:border-neutral-700 transform rotate-45'
    
    switch (placement) {
      case 'top':
        return `${baseClasses} -bottom-1 left-1/2 -translate-x-1/2 border-t-0 border-l-0`
      case 'bottom':
        return `${baseClasses} -top-1 left-1/2 -translate-x-1/2 border-b-0 border-r-0`
      case 'left':
        return `${baseClasses} -right-1 top-1/2 -translate-y-1/2 border-l-0 border-b-0`
      case 'right':
        return `${baseClasses} -left-1 top-1/2 -translate-y-1/2 border-r-0 border-t-0`
      default:
        return baseClasses
    }
  }

  return (
    <>
      <div
        ref={triggerRef}
        onClick={handleTriggerClick}
        onMouseEnter={handleTriggerMouseEnter}
        onMouseLeave={handleTriggerMouseLeave}
        className={cn('inline-block', className)}
      >
        {children}
      </div>

      {open && createPortal(
        <div
          ref={contentRef}
          className={cn(
            'fixed z-50 bg-white dark:bg-neutral-800 border border-neutral-200 dark:border-neutral-700 rounded-lg shadow-lg',
            contentClassName
          )}
          style={{
            left: position.x,
            top: position.y,
          }}
        >
          <div className="relative">
            {showCloseButton && (
              <button
                onClick={handleClose}
                className="absolute top-2 right-2 p-1 hover:bg-neutral-100 dark:hover:bg-neutral-700 rounded transition-colors"
              >
                <X className="w-4 h-4" />
              </button>
            )}
            <div className="p-4">
              {content}
            </div>
            <div className={getArrowClasses()} />
          </div>
        </div>,
        document.body
      )}
    </>
  )
}

export default Popover
