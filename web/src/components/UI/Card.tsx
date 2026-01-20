import { ReactNode } from 'react'
import { motion } from 'framer-motion'
import { cn } from '@/utils/cn.ts'
import { getCurrentTheme, createThemeAwareStyles } from '@/utils/themeAdapter.ts'

interface CardProps {
  children: ReactNode
  title?: string
  subtitle?: string
  actions?: ReactNode
  variant?: 'default' | 'outlined' | 'elevated' | 'filled' | 'glass'
  padding?: 'none' | 'sm' | 'md' | 'lg' | 'xl'
  className?: string
  hover?: boolean
  onClick?: () => void
  headerClassName?: string
  bodyClassName?: string
  footerClassName?: string
  footer?: ReactNode
  animation?: 'none' | 'fade' | 'slide' | 'scale' | 'flip'
  delay?: number
}

const Card = ({ 
  children, 
  title, 
  subtitle, 
  actions,
  variant = 'default',
  padding = 'md',
  className = "",
  hover = false,
  onClick,
  headerClassName,
  bodyClassName,
  footerClassName,
  footer,
  animation = 'fade',
  delay = 0
}: CardProps) => {
  // 获取当前主题
  const currentTheme = getCurrentTheme()
  const themeStyles = createThemeAwareStyles(currentTheme)
  
  const variantClasses = {
    default: themeStyles.card.default,
    outlined: 'bg-transparent border-2 border-gray-200 shadow-none hover:shadow-md',
    elevated: themeStyles.card.elevated,
    filled: themeStyles.card.filled,
    glass: themeStyles.card.glass
  }

  const paddingClasses = {
    none: '',
    sm: 'p-4',
    md: 'p-6',
    lg: 'p-8',
    xl: 'p-10'
  }

  const hoverClasses = hover ? 'hover:shadow-lg hover:-translate-y-1 transition-all duration-300' : ''
  const clickableClasses = onClick ? 'cursor-pointer' : ''

  const animationVariants = {
    none: {
      initial: { opacity: 1 },
      animate: { opacity: 1 },
      exit: { opacity: 0 }
    },
    fade: {
      initial: { opacity: 0, y: 20 },
      animate: { opacity: 1, y: 0 },
      exit: { opacity: 0, y: -20 }
    },
    slide: {
      initial: { opacity: 0, x: -50 },
      animate: { opacity: 1, x: 0 },
      exit: { opacity: 0, x: 50 }
    },
    scale: {
      initial: { opacity: 0, scale: 0.9 },
      animate: { opacity: 1, scale: 1 },
      exit: { opacity: 0, scale: 0.9 }
    },
    flip: {
      initial: { opacity: 0, rotateY: -90 },
      animate: { opacity: 1, rotateY: 0 },
      exit: { opacity: 0, rotateY: 90 }
    }
  }

  const currentAnimation = animationVariants[animation]

  return (
    <motion.div
      className={cn(
        'rounded-xl transition-all duration-300 group',
        variantClasses[variant],
        paddingClasses[padding],
        hoverClasses,
        clickableClasses,
        className
      )}
      onClick={onClick}
      initial={currentAnimation.initial}
      animate={currentAnimation.animate}
      exit={currentAnimation.exit}
      whileHover={hover ? { 
        y: -4, 
        scale: 1.02,
        transition: { duration: 0.2 }
      } : {}}
      whileTap={onClick ? { 
        scale: 0.98,
        transition: { duration: 0.1 }
      } : {}}
      transition={{ 
        duration: 0.3, 
        delay: delay,
        ease: "easeOut"
      }}
    >
      {/* Hover glow effect */}
      {hover && (
        <motion.div
          className="absolute inset-0 rounded-xl bg-gradient-to-r from-blue-500/10 to-purple-500/10 opacity-0 group-hover:opacity-100 transition-opacity duration-300"
          initial={{ opacity: 0 }}
          whileHover={{ opacity: 1 }}
        />
      )}

      {(title || subtitle || actions) && (
        <motion.div 
          className={cn('mb-6 relative z-10', headerClassName)}
          initial={{ opacity: 0, y: -10 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ duration: 0.3, delay: delay + 0.1 }}
        >
          <div className="flex items-start justify-between">
            <div className="flex-1 min-w-0">
              {title && (
                <motion.h3 
                  className="text-lg font-semibold text-gray-900 leading-tight group-hover:text-blue-600 transition-colors duration-300"
                  whileHover={{ scale: 1.02 }}
                  transition={{ duration: 0.2 }}
                >
                  {title}
                </motion.h3>
              )}
              {subtitle && (
                <motion.p 
                  className="mt-1 text-sm text-gray-600 leading-relaxed group-hover:text-gray-700 transition-colors duration-300"
                  initial={{ opacity: 0 }}
                  animate={{ opacity: 1 }}
                  transition={{ duration: 0.3, delay: delay + 0.2 }}
                >
                  {subtitle}
                </motion.p>
              )}
            </div>
            {actions && (
              <motion.div 
                className="ml-4 flex-shrink-0"
                initial={{ opacity: 0, scale: 0.8 }}
                animate={{ opacity: 1, scale: 1 }}
                transition={{ duration: 0.3, delay: delay + 0.3 }}
              >
                {actions}
              </motion.div>
            )}
          </div>
        </motion.div>
      )}
      
      <motion.div 
        className={cn('min-h-0 relative z-10', bodyClassName)}
        initial={{ opacity: 0, y: 10 }}
        animate={{ opacity: 1, y: 0 }}
        transition={{ duration: 0.3, delay: delay + 0.2 }}
      >
        {children}
      </motion.div>
      
      {footer && (
        <motion.div 
          className={cn('mt-6 pt-6 border-t border-gray-200 relative z-10', footerClassName)}
          initial={{ opacity: 0, y: 10 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ duration: 0.3, delay: delay + 0.4 }}
        >
          {footer}
        </motion.div>
      )}
    </motion.div>
  )
}

const CardHeader = ({ children, className }: { children: ReactNode; className?: string }) => {
  return (
    <motion.div 
      className={cn('mb-6', className)}
      initial={{ opacity: 0, y: -10 }}
      animate={{ opacity: 1, y: 0 }}
      transition={{ duration: 0.3 }}
    >
      {children}
    </motion.div>
  )
}

const CardTitle = ({ children, className }: { children: ReactNode; className?: string }) => {
  return (
    <motion.h3 
      className={cn('text-lg font-semibold text-gray-900 leading-tight', className)}
      whileHover={{ scale: 1.02 }}
      transition={{ duration: 0.2 }}
    >
      {children}
    </motion.h3>
  )
}

const CardDescription = ({ children, className }: { children: ReactNode; className?: string }) => {
  return (
    <motion.p 
      className={cn('mt-1 text-sm text-gray-600 leading-relaxed', className)}
      initial={{ opacity: 0 }}
      animate={{ opacity: 1 }}
      transition={{ duration: 0.3, delay: 0.1 }}
    >
      {children}
    </motion.p>
  )
}

const CardContent = ({ children, className }: { children: ReactNode; className?: string }) => {
  return (
    <motion.div 
      className={cn('min-h-0', className)}
      initial={{ opacity: 0, y: 10 }}
      animate={{ opacity: 1, y: 0 }}
      transition={{ duration: 0.3, delay: 0.2 }}
    >
      {children}
    </motion.div>
  )
}

const CardFooter = ({ children, className }: { children: ReactNode; className?: string }) => {
  return (
    <motion.div 
      className={cn('mt-6 pt-6 border-t border-gray-200', className)}
      initial={{ opacity: 0, y: 10 }}
      animate={{ opacity: 1, y: 0 }}
      transition={{ duration: 0.3, delay: 0.4 }}
    >
      {children}
    </motion.div>
  )
}

export default Card
export { CardHeader, CardTitle, CardDescription, CardContent, CardFooter }