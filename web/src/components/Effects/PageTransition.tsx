import React, { useEffect, useState } from 'react'
import { motion, AnimatePresence } from 'framer-motion'

interface PageTransitionProps {
  children: React.ReactNode
  isVisible: boolean
  direction?: 'left' | 'right' | 'up' | 'down' | 'fade' | 'scale' | 'rotate'
  duration?: number
  delay?: number
}

const PageTransition: React.FC<PageTransitionProps> = ({
  children,
  isVisible,
  direction = 'fade',
  duration = 0.6,
  delay = 0
}) => {
  const [isLoaded, setIsLoaded] = useState(false)

  useEffect(() => {
    const timer = setTimeout(() => {
      setIsLoaded(true)
    }, delay)
    return () => clearTimeout(timer)
  }, [delay])

  const getVariants = () => {
    const baseVariants = {
      hidden: { opacity: 0 },
      visible: { 
        opacity: 1,
        transition: { duration, ease: "easeOut" }
      },
      exit: { 
        opacity: 0,
        transition: { duration: duration * 0.5, ease: "easeIn" }
      }
    }

    switch (direction) {
      case 'left':
        return {
          ...baseVariants,
          hidden: { ...baseVariants.hidden, x: -100 },
          visible: { ...baseVariants.visible, x: 0 },
          exit: { ...baseVariants.exit, x: -100 }
        }
      case 'right':
        return {
          ...baseVariants,
          hidden: { ...baseVariants.hidden, x: 100 },
          visible: { ...baseVariants.visible, x: 0 },
          exit: { ...baseVariants.exit, x: 100 }
        }
      case 'up':
        return {
          ...baseVariants,
          hidden: { ...baseVariants.hidden, y: -100 },
          visible: { ...baseVariants.visible, y: 0 },
          exit: { ...baseVariants.exit, y: -100 }
        }
      case 'down':
        return {
          ...baseVariants,
          hidden: { ...baseVariants.hidden, y: 100 },
          visible: { ...baseVariants.visible, y: 0 },
          exit: { ...baseVariants.exit, y: 100 }
        }
      case 'scale':
        return {
          ...baseVariants,
          hidden: { ...baseVariants.hidden, scale: 0.8 },
          visible: { ...baseVariants.visible, scale: 1 },
          exit: { ...baseVariants.exit, scale: 0.8 }
        }
      case 'rotate':
        return {
          ...baseVariants,
          hidden: { ...baseVariants.hidden, rotate: -180, scale: 0.8 },
          visible: { ...baseVariants.visible, rotate: 0, scale: 1 },
          exit: { ...baseVariants.exit, rotate: 180, scale: 0.8 }
        }
      default:
        return baseVariants
    }
  }

  return (
    <AnimatePresence mode="wait">
      {isVisible && isLoaded && (
        <motion.div
          variants={getVariants()}
          initial="hidden"
          animate="visible"
          exit="exit"
          className="w-full h-full"
        >
          {children}
        </motion.div>
      )}
    </AnimatePresence>
  )
}

export default PageTransition
