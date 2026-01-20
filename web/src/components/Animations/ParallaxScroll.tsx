import { ReactNode, useRef } from 'react'
import { motion, useScroll, useTransform } from 'framer-motion'

interface ParallaxScrollProps {
  children: ReactNode
  speed?: number
  direction?: 'up' | 'down'
  className?: string
}

const ParallaxScroll = ({ 
  children, 
  direction = 'up',
  className 
}: ParallaxScrollProps) => {
  const ref = useRef<HTMLDivElement>(null)
  const { scrollYProgress } = useScroll({
    target: ref,
    offset: ['start end', 'end start'],
  })

  const y = useTransform(
    scrollYProgress,
    [0, 1],
    direction === 'up' ? [100, -100] : [-100, 100]
  )

  return (
    <div ref={ref} className={className}>
      <motion.div style={{ y }}>
        {children}
      </motion.div>
    </div>
  )
}

export default ParallaxScroll
