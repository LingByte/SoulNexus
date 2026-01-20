import { ReactNode } from 'react'
import { motion } from 'framer-motion'
import { cn } from '@/utils/cn.ts'
import { getCurrentTheme, getThemeClasses } from '@/utils/themeAdapter.ts'

interface IconTextProps {
  icon: ReactNode
  children: ReactNode
  size?: 'xs' | 'sm' | 'md' | 'lg' | 'xl'
  variant?: 'default' | 'primary' | 'secondary' | 'success' | 'warning' | 'error' | 'muted'
  direction?: 'horizontal' | 'vertical'
  spacing?: 'tight' | 'normal' | 'loose'
  className?: string
  iconClassName?: string
  textClassName?: string
  onClick?: () => void
  animation?: 'none' | 'fade' | 'slide' | 'scale' | 'bounce' | 'pulse' | 'glow'
  delay?: number
}

const IconText = ({
  icon,
  children,
  size = 'md',
  variant = 'default',
  direction = 'horizontal',
  spacing = 'normal',
  className,
  iconClassName,
  textClassName,
  onClick,
  animation = 'fade',
  delay = 0
}: IconTextProps) => {
  const sizeClasses = {
    xs: {
      icon: 'w-3 h-3',
      text: 'text-xs',
      gap: 'gap-1'
    },
    sm: {
      icon: 'w-4 h-4',
      text: 'text-sm',
      gap: 'gap-1.5'
    },
    md: {
      icon: 'w-5 h-5',
      text: 'text-base',
      gap: 'gap-2'
    },
    lg: {
      icon: 'w-6 h-6',
      text: 'text-lg',
      gap: 'gap-2.5'
    },
    xl: {
      icon: 'w-8 h-8',
      text: 'text-xl',
      gap: 'gap-3'
    }
  }

  // 获取当前主题
  const currentTheme = getCurrentTheme()
  const themeClasses = getThemeClasses(currentTheme, 'primary')
  
  const variantClasses = {
    default: {
      icon: 'text-gray-600 group-hover:text-gray-800',
      text: 'text-gray-900 group-hover:text-gray-700'
    },
    primary: {
      icon: `text-${themeClasses.ring} group-hover:text-${themeClasses.ring}`,
      text: `text-${themeClasses.ring} group-hover:text-${themeClasses.ring}`
    },
    secondary: {
      icon: 'text-gray-500 group-hover:text-gray-700',
      text: 'text-gray-700 group-hover:text-gray-900'
    },
    success: {
      icon: 'text-green-600 group-hover:text-green-800',
      text: 'text-green-900 group-hover:text-green-700'
    },
    warning: {
      icon: 'text-yellow-600 group-hover:text-yellow-800',
      text: 'text-yellow-900 group-hover:text-yellow-700'
    },
    error: {
      icon: 'text-red-600 group-hover:text-red-800',
      text: 'text-red-900 group-hover:text-red-700'
    },
    muted: {
      icon: 'text-gray-400 group-hover:text-gray-600',
      text: 'text-gray-500 group-hover:text-gray-700'
    }
  }

  const spacingClasses = {
    tight: 'gap-1',
    normal: sizeClasses[size].gap,
    loose: 'gap-4'
  }

  const directionClasses = {
    horizontal: 'flex-row items-center',
    vertical: 'flex-col items-center'
  }

  const animationVariants = {
    none: {
      initial: { opacity: 1 },
      animate: { opacity: 1 },
      hover: {},
      tap: {}
    },
    fade: {
      initial: { opacity: 0, y: 10 },
      animate: { opacity: 1, y: 0 },
      hover: { y: -2 },
      tap: { scale: 0.98 }
    },
    slide: {
      initial: { opacity: 0, x: direction === 'horizontal' ? -20 : 0, y: direction === 'vertical' ? -20 : 0 },
      animate: { opacity: 1, x: 0, y: 0 },
      hover: { 
        x: direction === 'horizontal' ? -2 : 0, 
        y: direction === 'vertical' ? -2 : 0 
      },
      tap: { scale: 0.95 }
    },
    scale: {
      initial: { opacity: 0, scale: 0.8 },
      animate: { opacity: 1, scale: 1 },
      hover: { scale: 1.05 },
      tap: { scale: 0.95 }
    },
    bounce: {
      initial: { opacity: 0, scale: 0.8, y: -10 },
      animate: { opacity: 1, scale: 1, y: 0 },
      hover: { 
        scale: 1.1, 
        y: -3,
        transition: { type: "spring", stiffness: 400, damping: 10 }
      },
      tap: { scale: 0.9 }
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
    glow: {
      initial: { opacity: 0, scale: 0.8 },
      animate: { opacity: 1, scale: 1 },
      hover: { 
        scale: 1.05,
        filter: 'drop-shadow(0 0 8px rgba(59, 130, 246, 0.3))'
      },
      tap: { scale: 0.95 }
    }
  }

  const currentSize = sizeClasses[size]
  const currentVariant = variantClasses[variant]
  const currentAnimation = animationVariants[animation] || animationVariants.fade

  return (
    <motion.div
      className={cn(
        'inline-flex group transition-all duration-200',
        directionClasses[direction],
        spacing === 'normal' ? currentSize.gap : spacingClasses[spacing],
        onClick && 'cursor-pointer',
        className
      )}
      onClick={onClick}
      initial={currentAnimation.initial}
      animate={currentAnimation.animate}
      whileHover={currentAnimation.hover}
      whileTap={currentAnimation.tap}
      transition={{ 
        duration: 0.3, 
        delay: delay,
        ease: "easeOut"
      }}
    >
      <motion.span 
        className={cn(
          'flex-shrink-0 transition-colors duration-200',
          currentSize.icon,
          currentVariant.icon,
          iconClassName
        )}
        whileHover={{ 
          scale: 1.1,
          rotate: direction === 'vertical' ? 360 : 0
        }}
        transition={{ duration: 0.2 }}
      >
        {icon}
      </motion.span>
      <motion.span 
        className={cn(
          'flex-shrink-0 transition-colors duration-200',
          currentSize.text,
          currentVariant.text,
          textClassName
        )}
        initial={{ opacity: 0 }}
        animate={{ opacity: 1 }}
        transition={{ duration: 0.2, delay: delay + 0.1 }}
      >
        {children}
      </motion.span>
    </motion.div>
  )
}

export default IconText