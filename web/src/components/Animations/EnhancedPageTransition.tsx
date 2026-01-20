import { useEffect, useState } from 'react'
import { motion, AnimatePresence } from 'framer-motion'
import { playPageTransitionSound } from '@/utils/audioEffects.ts'

interface EnhancedPageTransitionProps {
  children: React.ReactNode
  variant?: 'fade' | 'slide' | 'scale' | 'flip' | 'glitch' | 'morph' | 'wave'
  duration?: number
  enableAudio?: boolean
  className?: string
}

const EnhancedPageTransition = ({
  children,
  variant = 'fade',
  duration = 0.5,
  enableAudio = true,
  className = ''
}: EnhancedPageTransitionProps) => {
  const [isVisible, setIsVisible] = useState(false)

  useEffect(() => {
    setIsVisible(true)
    if (enableAudio) {
      playPageTransitionSound()
    }
  }, [enableAudio])

  const getVariants = () => {
    const baseTransition = {
      duration,
      ease: [0.16, 1, 0.3, 1]
    }

    switch (variant) {
      case 'fade':
        return {
          initial: { opacity: 0 },
          animate: { opacity: 1 },
          exit: { opacity: 0 },
          transition: baseTransition
        }
      
      case 'slide':
        return {
          initial: { opacity: 0, x: -100 },
          animate: { opacity: 1, x: 0 },
          exit: { opacity: 0, x: 100 },
          transition: baseTransition
        }
      
      case 'scale':
        return {
          initial: { opacity: 0, scale: 0.8 },
          animate: { opacity: 1, scale: 1 },
          exit: { opacity: 0, scale: 1.2 },
          transition: baseTransition
        }
      
      case 'flip':
        return {
          initial: { opacity: 0, rotateY: -90 },
          animate: { opacity: 1, rotateY: 0 },
          exit: { opacity: 0, rotateY: 90 },
          transition: baseTransition
        }
      
      case 'glitch':
        return {
          initial: { 
            opacity: 0, 
            x: -20,
            filter: 'hue-rotate(0deg) saturate(1)'
          },
          animate: { 
            opacity: 1, 
            x: 0,
            filter: 'hue-rotate(0deg) saturate(1)',
            transition: {
              ...baseTransition,
              filter: {
                duration: 0.1,
                repeat: 3,
                repeatType: 'reverse'
              }
            }
          },
          exit: { 
            opacity: 0, 
            x: 20,
            filter: 'hue-rotate(180deg) saturate(2)'
          }
        }
      
      case 'morph':
        return {
          initial: { 
            opacity: 0, 
            scale: 0.5,
            borderRadius: '50%'
          },
          animate: { 
            opacity: 1, 
            scale: 1,
            borderRadius: '0%',
            transition: {
              ...baseTransition,
              borderRadius: {
                duration: duration * 0.8,
                ease: 'easeInOut'
              }
            }
          },
          exit: { 
            opacity: 0, 
            scale: 0.5,
            borderRadius: '50%'
          }
        }
      
      case 'wave':
        return {
          initial: { 
            opacity: 0, 
            y: 50,
            rotateX: -15
          },
          animate: { 
            opacity: 1, 
            y: 0,
            rotateX: 0,
            transition: {
              ...baseTransition,
              y: {
                duration: duration * 0.8,
                ease: 'easeOut'
              }
            }
          },
          exit: { 
            opacity: 0, 
            y: -50,
            rotateX: 15
          }
        }
      
      default:
        return {
          initial: { opacity: 0 },
          animate: { opacity: 1 },
          exit: { opacity: 0 },
          transition: baseTransition
        }
    }
  }

  const variants = getVariants()

  return (
    <AnimatePresence mode="wait">
      {isVisible && (
        <motion.div
          className={className}
          initial={variants.initial}
          animate={variants.animate}
          exit={variants.exit}
          transition={variants.transition}
        >
          {children}
        </motion.div>
      )}
    </AnimatePresence>
  )
}

export default EnhancedPageTransition
