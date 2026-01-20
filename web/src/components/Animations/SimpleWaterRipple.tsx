import { useRef, useState, useCallback } from 'react'
import { motion, AnimatePresence } from 'framer-motion'

interface SimpleWaterRippleProps {
  children: React.ReactNode
  className?: string
  intensity?: 'low' | 'medium' | 'high'
  color?: string
  duration?: number
}

interface Ripple {
  id: number
  x: number
  y: number
  size: number
}

export const SimpleWaterRipple = ({
  children,
  className = '',
  intensity = 'medium',
  color = 'rgba(59, 130, 246, 0.4)',
  duration = 800
}: SimpleWaterRippleProps) => {
  const [ripples, setRipples] = useState<Ripple[]>([])
  const containerRef = useRef<HTMLDivElement>(null)
  const rippleId = useRef(0)

  const intensityMap = {
    low: { maxSize: 80, count: 1 },
    medium: { maxSize: 120, count: 2 },
    high: { maxSize: 160, count: 3 }
  }

  const createRipple = useCallback((event: React.MouseEvent) => {
    if (!containerRef.current) return

    const rect = containerRef.current.getBoundingClientRect()
    const x = event.clientX - rect.left
    const y = event.clientY - rect.top

    const { maxSize, count } = intensityMap[intensity]

    const newRipples: Ripple[] = []
    for (let i = 0; i < count; i++) {
      newRipples.push({
        id: rippleId.current++,
        x: x + (Math.random() - 0.5) * 10,
        y: y + (Math.random() - 0.5) * 10,
        size: maxSize + Math.random() * 40
      })
    }

    setRipples(prev => [...prev, ...newRipples])

    // 自动清理涟漪
    setTimeout(() => {
      setRipples(prev => prev.filter(ripple => !newRipples.some(newRipple => newRipple.id === ripple.id)))
    }, duration)
  }, [intensity, duration])

  return (
    <div
      ref={containerRef}
      className={`relative overflow-hidden ${className}`}
      onMouseDown={createRipple}
    >
      {children}
      
      <AnimatePresence>
        {ripples.map(ripple => (
          <motion.div
            key={ripple.id}
            className="absolute pointer-events-none rounded-full"
            style={{
              left: ripple.x - ripple.size / 2,
              top: ripple.y - ripple.size / 2,
              width: ripple.size,
              height: ripple.size,
              background: `radial-gradient(circle, ${color} 0%, transparent 70%)`,
              boxShadow: `0 0 20px ${color}, inset 0 0 20px ${color}`
            }}
            initial={{ 
              scale: 0, 
              opacity: 0.8
            }}
            animate={{ 
              scale: 1,
              opacity: 0
            }}
            exit={{ 
              opacity: 0,
              scale: 1.2
            }}
            transition={{
              duration: duration / 1000,
              ease: [0.25, 0.46, 0.45, 0.94]
            }}
          />
        ))}
      </AnimatePresence>
    </div>
  )
}

export default SimpleWaterRipple
