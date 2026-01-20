import { ReactNode } from 'react'
import { motion } from 'framer-motion'
import { cn } from '@/utils/cn.ts'

interface BadgeProps {
  children: ReactNode
  variant?: 'default' | 'primary' | 'secondary' | 'success' | 'warning' | 'error' | 'outline' | 'muted'
  size?: 'xs' | 'sm' | 'md' | 'lg'
  shape?: 'rounded' | 'pill' | 'square'
  icon?: ReactNode
  className?: string
  onClick?: () => void
  animation?: 'none' | 'bounce' | 'pulse' | 'scale' | 'fade'
  delay?: number
}

const Badge = ({
  children,
  variant = 'default',
  size = 'sm',
  shape = 'rounded',
  icon,
  className,
  onClick,
  animation = 'scale',
  delay = 0
}: BadgeProps) => {
  const sizeClasses = {
    xs: 'px-1.5 py-0.5 text-xs',
    sm: 'px-2 py-1 text-xs',
    md: 'px-3 py-1.5 text-sm',
    lg: 'px-4 py-2 text-base'
  }

  const variantClasses = {
    default: 'bg-gray-100 text-gray-800 hover:bg-gray-200',
    primary: 'bg-blue-100 text-blue-800 hover:bg-blue-200',
    secondary: 'bg-gray-200 text-gray-800 hover:bg-gray-300',
    success: 'bg-green-100 text-green-800 hover:bg-green-200',
    warning: 'bg-yellow-100 text-yellow-800 hover:bg-yellow-200',
    error: 'bg-red-100 text-red-800 hover:bg-red-200',
    outline: 'border border-gray-300 bg-transparent text-gray-700 hover:bg-gray-50',
    muted: 'bg-gray-50 text-gray-500 hover:bg-gray-100'
  }

  const shapeClasses = {
    rounded: 'rounded-md',
    pill: 'rounded-full',
    square: 'rounded-none'
  }

  const iconSizeClasses = {
    xs: 'w-3 h-3',
    sm: 'w-3.5 h-3.5',
    md: 'w-4 h-4',
    lg: 'w-5 h-5'
  }

  const animationVariants = {
    none: {
      initial: { opacity: 1, scale: 1 },
      animate: { opacity: 1, scale: 1 },
      hover: { scale: 1 },
      tap: { scale: 1 }
    },
    bounce: {
      initial: { opacity: 0, scale: 0.8, y: -10 },
      animate: { opacity: 1, scale: 1, y: 0 },
      hover: { 
        scale: 1.1, 
        y: -2,
        transition: { type: "spring", stiffness: 400, damping: 10 }
      },
      tap: { scale: 0.95 }
    },
    pulse: {
      initial: { opacity: 0, scale: 0.8 },
      animate: { opacity: 1, scale: 1 },
      hover: { 
        scale: 1.05,
        boxShadow: "0 0 0 4px rgba(59, 130, 246, 0.1)"
      },
      tap: { scale: 0.95 }
    },
    scale: {
      initial: { opacity: 0, scale: 0.8 },
      animate: { opacity: 1, scale: 1 },
      hover: { scale: 1.05 },
      tap: { scale: 0.95 }
    },
    fade: {
      initial: { opacity: 0, y: 10 },
      animate: { opacity: 1, y: 0 },
      hover: { y: -2, scale: 1.02 },
      tap: { scale: 0.98 }
    }
  }

  const currentAnimation = animationVariants[animation]

  return (
    <motion.span
      className={cn(
        'inline-flex items-center font-medium transition-all duration-200 relative overflow-hidden',
        sizeClasses[size],
        variantClasses[variant],
        shapeClasses[shape],
        onClick && 'cursor-pointer',
        className
      )}
      onClick={onClick}
      initial={currentAnimation.initial}
      animate={currentAnimation.animate}
      whileHover={currentAnimation.hover}
      whileTap={currentAnimation.tap}
      transition={{ 
        duration: 0.2, 
        delay: delay,
        ease: "easeOut"
      }}
    >
      {/* Hover effect background */}
      <motion.div
        className="absolute inset-0 bg-white/20 rounded-inherit"
        initial={{ scale: 0, opacity: 0 }}
        whileHover={{ scale: 1, opacity: 1 }}
        transition={{ duration: 0.2 }}
      />
      
      <div className="relative flex items-center gap-1.5">
        {icon && (
          <motion.span 
            className={cn('flex-shrink-0', iconSizeClasses[size])}
            whileHover={{ rotate: 360 }}
            transition={{ duration: 0.3 }}
          >
            {icon}
          </motion.span>
        )}
        <motion.span
          initial={{ opacity: 0 }}
          animate={{ opacity: 1 }}
          transition={{ duration: 0.2, delay: delay + 0.1 }}
        >
          {children}
        </motion.span>
      </div>
    </motion.span>
  )
}

export default Badge