import { motion } from 'framer-motion'
import { ReactNode } from 'react'

interface PageTransitionProps {
  children: ReactNode
  className?: string
  variant?: 'fade' | 'slide' | 'scale' | 'flip' | 'glitch' | 'morph'
  duration?: number
}

const transitionVariants = {
  fade: {
    initial: { opacity: 0 },
    animate: { opacity: 1 },
    exit: { opacity: 0 }
  },
  slide: {
    initial: { opacity: 0, x: 100 },
    animate: { opacity: 1, x: 0 },
    exit: { opacity: 0, x: -100 }
  },
  scale: {
    initial: { opacity: 0, scale: 0.8 },
    animate: { opacity: 1, scale: 1 },
    exit: { opacity: 0, scale: 1.2 }
  },
  flip: {
    initial: { opacity: 0, rotateY: 90 },
    animate: { opacity: 1, rotateY: 0 },
    exit: { opacity: 0, rotateY: -90 }
  },
  glitch: {
    initial: { 
      opacity: 0, 
      x: -10,
      filter: 'hue-rotate(0deg) saturate(1)'
    },
    animate: { 
      opacity: 1, 
      x: 0,
      filter: 'hue-rotate(0deg) saturate(1)'
    },
    exit: { 
      opacity: 0, 
      x: 10,
      filter: 'hue-rotate(180deg) saturate(2)'
    }
  },
  morph: {
    initial: { 
      opacity: 0, 
      scale: 0.5,
      borderRadius: '50%',
      rotate: 180
    },
    animate: { 
      opacity: 1, 
      scale: 1,
      borderRadius: '0%',
      rotate: 0
    },
    exit: { 
      opacity: 0, 
      scale: 0.5,
      borderRadius: '50%',
      rotate: -180
    }
  }
}

export const PageTransition = ({
  children,
  className = '',
  variant = 'fade',
  duration = 0.5
}: PageTransitionProps) => {
  const variants = transitionVariants[variant]

  return (
    <motion.div
      className={className}
      initial="initial"
      animate="animate"
      exit="exit"
      variants={variants}
      transition={{
        duration,
        ease: variant === 'glitch' ? 'easeInOut' : 'easeOut'
      }}
    >
      {children}
    </motion.div>
  )
}

export default PageTransition
