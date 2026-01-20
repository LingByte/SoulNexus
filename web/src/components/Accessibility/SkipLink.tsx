import { ReactNode } from 'react'
import { motion } from 'framer-motion'
import { cn } from '@/utils/cn.ts'

interface SkipLinkProps {
  href: string
  children: ReactNode
  className?: string
  animation?: 'none' | 'fade' | 'slide' | 'scale'
  delay?: number
}

const SkipLink = ({
  href,
  children,
  className = "",
  animation = 'fade',
  delay = 0
}: SkipLinkProps) => {
  const animationVariants = {
    none: {
      initial: { opacity: 1 },
      animate: { opacity: 1 }
    },
    fade: {
      initial: { opacity: 0 },
      animate: { opacity: 1 }
    },
    slide: {
      initial: { opacity: 0, x: -20 },
      animate: { opacity: 1, x: 0 }
    },
    scale: {
      initial: { opacity: 0, scale: 0.9 },
      animate: { opacity: 1, scale: 1 }
    }
  }

  const currentAnimation = animationVariants[animation]

  return (
    <motion.a
      href={href}
      className={cn(
        'sr-only focus:not-sr-only focus:absolute focus:top-4 focus:left-4',
        'bg-blue-600 text-white px-4 py-2 rounded-md text-sm font-medium',
        'z-50 transition-all duration-200',
        'focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-offset-2',
        className
      )}
      initial={currentAnimation.initial}
      animate={currentAnimation.animate}
      transition={{
        duration: 0.3,
        delay: delay,
        ease: "easeOut"
      }}
      onFocus={(e) => {
        e.currentTarget.scrollIntoView({ behavior: 'smooth', block: 'center' })
      }}
    >
      {children}
    </motion.a>
  )
}

export default SkipLink
