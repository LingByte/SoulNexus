import { useState, useRef } from 'react'
import { motion } from 'framer-motion'
import { playHoverSound, playClickSound } from '@/utils/audioEffects.ts'

interface EnhancedMagneticButtonProps {
  children: React.ReactNode
  variant?: 'primary' | 'secondary' | 'outline' | 'ghost'
  size?: 'sm' | 'md' | 'lg' | 'xl'
  intensity?: number
  enableAudio?: boolean
  enableRipple?: boolean
  enableGlow?: boolean
  className?: string
  onClick?: () => void
}

const EnhancedMagneticButton = ({
  children,
  variant = 'primary',
  size = 'md',
  intensity = 0.3,
  enableAudio = true,
  enableRipple = true,
  enableGlow = true,
  className = '',
  onClick
}: EnhancedMagneticButtonProps) => {
  const [position, setPosition] = useState({ x: 0, y: 0 })
  const [isHovered, setIsHovered] = useState(false)
  const [ripples, setRipples] = useState<Array<{ id: number; x: number; y: number }>>([])
  const buttonRef = useRef<HTMLDivElement>(null)

  const getVariantClasses = () => {
    switch (variant) {
      case 'primary':
        return 'bg-gradient-to-r from-blue-500 to-purple-600 text-white shadow-lg hover:shadow-xl'
      case 'secondary':
        return 'bg-gradient-to-r from-gray-500 to-gray-700 text-white shadow-lg hover:shadow-xl'
      case 'outline':
        return 'border-2 border-blue-500 text-blue-500 hover:bg-blue-500 hover:text-white'
      case 'ghost':
        return 'text-gray-600 hover:bg-gray-100 dark:text-gray-300 dark:hover:bg-gray-800'
      default:
        return 'bg-gradient-to-r from-blue-500 to-purple-600 text-white shadow-lg hover:shadow-xl'
    }
  }

  const getSizeClasses = () => {
    switch (size) {
      case 'sm':
        return 'px-4 py-2 text-sm'
      case 'md':
        return 'px-6 py-3 text-base'
      case 'lg':
        return 'px-8 py-4 text-lg'
      case 'xl':
        return 'px-12 py-6 text-xl'
      default:
        return 'px-6 py-3 text-base'
    }
  }

  const handleMouseMove = (e: React.MouseEvent<HTMLDivElement>) => {
    if (!buttonRef.current) return

    const rect = buttonRef.current.getBoundingClientRect()
    const centerX = rect.left + rect.width / 2
    const centerY = rect.top + rect.height / 2

    const deltaX = (e.clientX - centerX) * intensity
    const deltaY = (e.clientY - centerY) * intensity

    setPosition({ x: deltaX, y: deltaY })
  }

  const handleMouseLeave = () => {
    setPosition({ x: 0, y: 0 })
    setIsHovered(false)
  }

  const handleMouseEnter = () => {
    setIsHovered(true)
    if (enableAudio) {
      playHoverSound()
    }
  }

  const handleClick = (e: React.MouseEvent<HTMLDivElement>) => {
    if (enableAudio) {
      playClickSound()
    }

    if (enableRipple) {
      const rect = buttonRef.current?.getBoundingClientRect()
      if (rect) {
        const ripple = {
          id: Date.now(),
          x: e.clientX - rect.left,
          y: e.clientY - rect.top
        }
        setRipples(prev => [...prev, ripple])
        
        // 移除涟漪效果
        setTimeout(() => {
          setRipples(prev => prev.filter(r => r.id !== ripple.id))
        }, 600)
      }
    }

    onClick?.()
  }

  return (
    <motion.div
      ref={buttonRef}
      className={`relative inline-block cursor-pointer select-none ${className}`}
      onMouseMove={handleMouseMove}
      onMouseLeave={handleMouseLeave}
      onMouseEnter={handleMouseEnter}
      onClick={handleClick}
      whileHover={{ scale: 1.05 }}
      whileTap={{ scale: 0.95 }}
      transition={{ duration: 0.2 }}
    >
      {/* 发光效果 */}
      {enableGlow && isHovered && (
        <motion.div
          className="absolute inset-0 rounded-lg bg-gradient-to-r from-blue-400 to-purple-500 opacity-20 blur-lg"
          initial={{ opacity: 0, scale: 0.8 }}
          animate={{ opacity: 0.3, scale: 1.2 }}
          exit={{ opacity: 0, scale: 0.8 }}
          transition={{ duration: 0.3 }}
        />
      )}

      {/* 主按钮 */}
      <motion.div
        className={`
          relative z-10 rounded-lg font-semibold transition-all duration-300
          ${getVariantClasses()}
          ${getSizeClasses()}
        `}
        animate={{
          x: position.x,
          y: position.y,
          rotateX: position.y * 0.1,
          rotateY: position.x * 0.1
        }}
        transition={{ duration: 0.1, ease: 'easeOut' }}
        style={{
          transformStyle: 'preserve-3d'
        }}
      >
        {/* 涟漪效果 */}
        {enableRipple && ripples.map(ripple => (
          <motion.div
            key={ripple.id}
            className="absolute bg-white/30 rounded-full pointer-events-none"
            initial={{ 
              x: ripple.x - 10, 
              y: ripple.y - 10, 
              width: 20, 
              height: 20,
              opacity: 1
            }}
            animate={{ 
              x: ripple.x - 50, 
              y: ripple.y - 50, 
              width: 100, 
              height: 100,
              opacity: 0
            }}
            transition={{ duration: 0.6, ease: 'easeOut' }}
          />
        ))}

        {/* 内容 */}
        <div className="relative z-10">
          {children}
        </div>

        {/* 3D 效果阴影 */}
        <motion.div
          className="absolute inset-0 rounded-lg bg-black/20"
          animate={{
            x: position.x * 0.5,
            y: position.y * 0.5 + 4,
            opacity: isHovered ? 0.3 : 0.1
          }}
          transition={{ duration: 0.1 }}
          style={{
            transform: 'translateZ(-1px)'
          }}
        />
      </motion.div>
    </motion.div>
  )
}

export default EnhancedMagneticButton
