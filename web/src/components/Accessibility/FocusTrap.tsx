import { ReactNode, useEffect, useRef, useState } from 'react'
import { motion } from 'framer-motion'
import { cn } from '@/utils/cn.ts'

interface FocusTrapProps {
  children: ReactNode
  active?: boolean
  className?: string
  animation?: 'none' | 'fade' | 'scale' | 'slide'
  delay?: number
  onEscape?: () => void
  onFocusOut?: () => void
}

const FocusTrap = ({
  children,
  active = true,
  className = "",
  animation = 'fade',
  delay = 0,
  onEscape,
  onFocusOut
}: FocusTrapProps) => {
  const containerRef = useRef<HTMLDivElement>(null)
  const [isActive, setIsActive] = useState(active)

  // 获取所有可聚焦元素
  const getFocusableElements = () => {
    if (!containerRef.current) return []
    
    const focusableSelectors = [
      'button:not([disabled])',
      'input:not([disabled])',
      'select:not([disabled])',
      'textarea:not([disabled])',
      'a[href]',
      'area[href]',
      '[tabindex]:not([tabindex="-1"])',
      '[contenteditable="true"]'
    ].join(', ')

    return Array.from(containerRef.current.querySelectorAll(focusableSelectors))
  }

  // 处理键盘事件
  const handleKeyDown = (e: KeyboardEvent) => {
    if (!isActive) return

    if (e.key === 'Escape') {
      onEscape?.()
      return
    }

    if (e.key === 'Tab') {
      const focusableElements = getFocusableElements()
      if (focusableElements.length === 0) return

      const firstElement = focusableElements[0] as HTMLElement
      const lastElement = focusableElements[focusableElements.length - 1] as HTMLElement

      if (e.shiftKey) {
        // Shift + Tab
        if (document.activeElement === firstElement) {
          e.preventDefault()
          lastElement.focus()
        }
      } else {
        // Tab
        if (document.activeElement === lastElement) {
          e.preventDefault()
          firstElement.focus()
        }
      }
    }
  }

  // 处理焦点事件
  const handleFocusOut = (e: FocusEvent) => {
    if (!isActive) return

    const container = containerRef.current
    if (!container) return

    const relatedTarget = e.relatedTarget as HTMLElement
    if (relatedTarget && !container.contains(relatedTarget)) {
      onFocusOut?.()
    }
  }

  // 设置焦点陷阱
  useEffect(() => {
    if (!isActive) return

    const container = containerRef.current
    if (!container) return

    // 添加事件监听器
    document.addEventListener('keydown', handleKeyDown)
    container.addEventListener('focusout', handleFocusOut)

    // 聚焦到第一个可聚焦元素
    const focusableElements = getFocusableElements()
    if (focusableElements.length > 0) {
      (focusableElements[0] as HTMLElement).focus()
    }

    return () => {
      document.removeEventListener('keydown', handleKeyDown)
      container.removeEventListener('focusout', handleFocusOut)
    }
  }, [isActive, onEscape, onFocusOut])

  // 更新激活状态
  useEffect(() => {
    setIsActive(active)
  }, [active])

  const animationVariants = {
    none: {
      initial: { opacity: 1 },
      animate: { opacity: 1 }
    },
    fade: {
      initial: { opacity: 0 },
      animate: { opacity: 1 }
    },
    scale: {
      initial: { opacity: 0, scale: 0.9 },
      animate: { opacity: 1, scale: 1 }
    },
    slide: {
      initial: { opacity: 0, y: 20 },
      animate: { opacity: 1, y: 0 }
    }
  }

  const currentAnimation = animationVariants[animation]

  return (
    <motion.div
      ref={containerRef}
      className={cn('focus-trap', className)}
      initial={currentAnimation.initial}
      animate={currentAnimation.animate}
      transition={{
        duration: 0.3,
        delay: delay,
        ease: "easeOut"
      }}
      tabIndex={-1}
    >
      {children}
    </motion.div>
  )
}

export default FocusTrap
