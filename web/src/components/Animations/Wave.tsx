import { motion } from 'framer-motion'
import { ReactNode } from 'react'

interface WaveProps {
  children: ReactNode
  className?: string
  amplitude?: number
  frequency?: number
  speed?: number
  direction?: 'horizontal' | 'vertical'
}

export const Wave = ({
  children,
  className = '',
  amplitude = 10, speed = 1,
  direction = 'horizontal'
}: WaveProps) => {
  const waveVariants = {
    animate: {
      [direction === 'horizontal' ? 'x' : 'y']: [
        0,
        amplitude,
        -amplitude,
        amplitude,
        -amplitude,
        0
      ],
      transition: {
        duration: 2 / speed,
        repeat: Infinity,
        ease: 'easeInOut'
      }
    }
  }

  return (
    <motion.div
      className={className}
      variants={waveVariants}
      animate="animate"
    >
      {children}
    </motion.div>
  )
}

export default Wave
