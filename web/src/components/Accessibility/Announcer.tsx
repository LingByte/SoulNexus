import { useEffect, useState } from 'react'
import { motion } from 'framer-motion'
import { cn } from '@/utils/cn.ts'

interface AnnouncerProps {
  message: string
  priority?: 'polite' | 'assertive'
  className?: string
  animation?: 'none' | 'fade' | 'slide' | 'scale'
  delay?: number
  duration?: number
}

const Announcer = ({
  message,
  priority = 'polite',
  className = "",
  animation = 'fade',
  delay = 0,
  duration = 3000
}: AnnouncerProps) => {
  const [isVisible, setIsVisible] = useState(false)
  const [currentMessage, setCurrentMessage] = useState('')

  useEffect(() => {
    if (!message) return

    setCurrentMessage(message)
    setIsVisible(true)

    // 自动隐藏
    const timer = setTimeout(() => {
      setIsVisible(false)
    }, duration)

    return () => clearTimeout(timer)
  }, [message, duration])

  const animationVariants = {
    none: {
      initial: { opacity: 1 },
      animate: { opacity: 1 },
      exit: { opacity: 0 }
    },
    fade: {
      initial: { opacity: 0 },
      animate: { opacity: 1 },
      exit: { opacity: 0 }
    },
    slide: {
      initial: { opacity: 0, y: 20 },
      animate: { opacity: 1, y: 0 },
      exit: { opacity: 0, y: -20 }
    },
    scale: {
      initial: { opacity: 0, scale: 0.9 },
      animate: { opacity: 1, scale: 1 },
      exit: { opacity: 0, scale: 0.9 }
    }
  }

  const currentAnimation = animationVariants[animation]

  if (!isVisible || !currentMessage) return null

  return (
    <motion.div
      className={cn(
        'sr-only',
        className
      )}
      initial={currentAnimation.initial}
      animate={currentAnimation.animate}
      exit={currentAnimation.exit}
      transition={{
        duration: 0.3,
        delay: delay,
        ease: "easeOut"
      }}
      role="status"
      aria-live={priority}
      aria-atomic="true"
    >
      {currentMessage}
    </motion.div>
  )
}

// 全局公告器Hook
export const useAnnouncer = () => {
  const [message, setMessage] = useState('')
  const [priority, setPriority] = useState<'polite' | 'assertive'>('polite')

  const announce = (newMessage: string, newPriority: 'polite' | 'assertive' = 'polite') => {
    setMessage(newMessage)
    setPriority(newPriority)
  }

  const clear = () => {
    setMessage('')
  }

  return {
    message,
    priority,
    announce,
    clear
  }
}

export default Announcer
