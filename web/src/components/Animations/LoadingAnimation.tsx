import { useState, useEffect } from 'react'
import { motion } from 'framer-motion'

interface LoadingAnimationProps {
  type?: 'spinner' | 'dots' | 'pulse' | 'wave' | 'bounce' | 'progress'
  size?: 'sm' | 'md' | 'lg'
  color?: string
  className?: string
  showPercentage?: boolean
  duration?: number // 进度条完成的持续时间（毫秒）
}

export const LoadingAnimation = ({
  type = 'spinner',
  size = 'md',
  color = '#3b82f6',
  className = '',
  showPercentage = false,
  duration = 2000
}: LoadingAnimationProps) => {
  const [progress, setProgress] = useState(0)
  const sizeMap = {
    sm: 'w-4 h-4',
    md: 'w-8 h-8',
    lg: 'w-12 h-12'
  }

  const progressBarWidthMap = {
    sm: 'w-32',
    md: 'w-48',
    lg: 'w-64'
  }

  const progressBarHeightMap = {
    sm: 'h-1',
    md: 'h-2',
    lg: 'h-3'
  }

  // 进度条动画效果
  useEffect(() => {
    if (type === 'progress' || showPercentage) {
      setProgress(0)
      const interval = setInterval(() => {
        setProgress((prev) => {
          if (prev >= 100) {
            // 到达100%后重新开始
            return 0
          }
          // 使用非线性增长，开始快，后面慢
          const increment = Math.max(0.5, (100 - prev) / 20)
          return Math.min(100, prev + increment)
        })
      }, duration / 100)

      return () => clearInterval(interval)
    }
  }, [type, showPercentage, duration])

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

  const renderProgress = () => (
    <div className="flex flex-col items-center gap-3">
      {/* 进度条 */}
      <div className={`${progressBarWidthMap[size]} ${progressBarHeightMap[size]} bg-gray-200 dark:bg-gray-700 rounded-full overflow-hidden`}>
        <motion.div
          className="h-full rounded-full"
          style={{ 
            backgroundColor: color,
            background: `linear-gradient(90deg, ${color}, ${color}dd)`
          }}
          initial={{ width: '0%' }}
          animate={{ width: `${progress}%` }}
          transition={{ duration: 0.3, ease: 'easeOut' }}
        />
      </div>
      
      {/* 百分比文字 */}
      <motion.div
        className="text-sm font-semibold tabular-nums"
        style={{ color }}
        initial={{ opacity: 0 }}
        animate={{ opacity: 1 }}
        transition={{ duration: 0.3 }}
      >
        {Math.round(progress)}%
      </motion.div>
      
      {/* 加载点动画 */}
      <div className="flex gap-1">
        {[0, 1, 2].map((index) => (
          <motion.div
            key={index}
            className="w-1.5 h-1.5 rounded-full"
            style={{ backgroundColor: color }}
            animate={{
              scale: [1, 1.3, 1],
              opacity: [0.5, 1, 0.5]
            }}
            transition={{
              duration: 1,
              repeat: Infinity,
              delay: index * 0.2,
              ease: 'easeInOut'
            }}
          />
        ))}
      </div>
    </div>
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
      case 'progress':
        return renderProgress()
      default:
        return renderSpinner()
    }
  }

  return (
    <div className={`flex items-center justify-center ${className}`}>
      {renderAnimation()}
      {showPercentage && type !== 'progress' && (
        <motion.div
          className="ml-3 text-sm font-semibold tabular-nums"
          style={{ color }}
          initial={{ opacity: 0 }}
          animate={{ opacity: 1 }}
        >
          {Math.round(progress)}%
        </motion.div>
      )}
    </div>
  )
}

export default LoadingAnimation
