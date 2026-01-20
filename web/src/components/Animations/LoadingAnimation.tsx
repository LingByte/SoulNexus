import { motion } from 'framer-motion'

interface LoadingAnimationProps {
  type?: 'spinner' | 'dots' | 'pulse' | 'wave' | 'bounce'
  size?: 'sm' | 'md' | 'lg'
  color?: string
  className?: string
}

export const LoadingAnimation = ({
  type = 'spinner',
  size = 'md',
  color = '#3b82f6',
  className = ''
}: LoadingAnimationProps) => {
  const sizeMap = {
    sm: 'w-4 h-4',
    md: 'w-8 h-8',
    lg: 'w-12 h-12'
  }

  const spinnerVariants = {
    animate: {
      rotate: 360,
      transition: {
        duration: 1,
        repeat: Infinity,
        ease: 'linear'
      }
    }
  }

  const dotsVariants = {
    animate: {
      scale: [1, 1.2, 1],
      transition: {
        duration: 0.6,
        repeat: Infinity,
        ease: 'easeInOut'
      }
    }
  }

  const pulseVariants = {
    animate: {
      scale: [1, 1.1, 1],
      opacity: [1, 0.7, 1],
      transition: {
        duration: 1.5,
        repeat: Infinity,
        ease: 'easeInOut'
      }
    }
  }

  const waveVariants = {
    animate: {
      y: [0, -10, 0],
      transition: {
        duration: 0.6,
        repeat: Infinity,
        ease: 'easeInOut'
      }
    }
  }

  const bounceVariants = {
    animate: {
      y: [0, -20, 0],
      transition: {
        duration: 0.6,
        repeat: Infinity,
        ease: 'easeInOut'
      }
    }
  }

  const renderSpinner = () => (
    <motion.div
      className={`${sizeMap[size]} border-2 border-gray-300 border-t-blue-500 rounded-full`}
      style={{ borderTopColor: color }}
      variants={spinnerVariants}
      animate="animate"
    />
  )

  const renderDots = () => (
    <div className="flex space-x-1">
      {[0, 1, 2].map((index) => (
        <motion.div
          key={index}
          className={`${sizeMap[size]} rounded-full`}
          style={{ backgroundColor: color }}
          variants={dotsVariants}
          animate="animate"
          transition={{ delay: index * 0.1 }}
        />
      ))}
    </div>
  )

  const renderPulse = () => (
    <motion.div
      className={`${sizeMap[size]} rounded-full`}
      style={{ backgroundColor: color }}
      variants={pulseVariants}
      animate="animate"
    />
  )

  const renderWave = () => (
    <div className="flex space-x-1">
      {[0, 1, 2, 3, 4].map((index) => (
        <motion.div
          key={index}
          className="w-1 rounded-full"
          style={{ 
            backgroundColor: color,
            height: size === 'sm' ? '16px' : size === 'md' ? '32px' : '48px'
          }}
          variants={waveVariants}
          animate="animate"
          transition={{ delay: index * 0.1 }}
        />
      ))}
    </div>
  )

  const renderBounce = () => (
    <motion.div
      className={`${sizeMap[size]} rounded-full`}
      style={{ backgroundColor: color }}
      variants={bounceVariants}
      animate="animate"
    />
  )

  const renderAnimation = () => {
    switch (type) {
      case 'spinner':
        return renderSpinner()
      case 'dots':
        return renderDots()
      case 'pulse':
        return renderPulse()
      case 'wave':
        return renderWave()
      case 'bounce':
        return renderBounce()
      default:
        return renderSpinner()
    }
  }

  return (
    <div className={`flex items-center justify-center ${className}`}>
      {renderAnimation()}
    </div>
  )
}

export default LoadingAnimation
