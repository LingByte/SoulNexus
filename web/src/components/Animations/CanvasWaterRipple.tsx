import { useRef, useEffect, useState } from 'react'

interface CanvasWaterRippleProps {
  children: React.ReactNode
  className?: string
  intensity?: 'low' | 'medium' | 'high'
  color?: string
}

interface Ripple {
  x: number
  y: number
  radius: number
  maxRadius: number
  speed: number
  opacity: number
  life: number
  maxLife: number
}

export const CanvasWaterRipple = ({
  children,
  className = '',
  intensity = 'medium',
  color = '#3b82f6'
}: CanvasWaterRippleProps) => {
  const canvasRef = useRef<HTMLCanvasElement>(null)
  const containerRef = useRef<HTMLDivElement>(null)
  const [ripples, setRipples] = useState<Ripple[]>([])
  const animationRef = useRef<number>()

  const intensityMap = {
    low: { maxRadius: 100, speed: 2, opacity: 0.3 },
    medium: { maxRadius: 150, speed: 3, opacity: 0.5 },
    high: { maxRadius: 200, speed: 4, opacity: 0.7 }
  }

  const createRipple = (x: number, y: number) => {
    const { maxRadius, speed, opacity } = intensityMap[intensity]
    
    const newRipple: Ripple = {
      x,
      y,
      radius: 0,
      maxRadius,
      speed,
      opacity,
      life: 0,
      maxLife: 60
    }

    setRipples(prev => [...prev, newRipple])
  }

  const handleMouseDown = (e: React.MouseEvent) => {
    if (!containerRef.current) return

    const rect = containerRef.current.getBoundingClientRect()
    const x = e.clientX - rect.left
    const y = e.clientY - rect.top

    createRipple(x, y)
  }

  useEffect(() => {
    const canvas = canvasRef.current
    if (!canvas) return

    const ctx = canvas.getContext('2d')
    if (!ctx) return

    const resizeCanvas = () => {
      if (!containerRef.current) return
      const rect = containerRef.current.getBoundingClientRect()
      canvas.width = rect.width
      canvas.height = rect.height
    }

    resizeCanvas()
    window.addEventListener('resize', resizeCanvas)

    const animate = () => {
      ctx.clearRect(0, 0, canvas.width, canvas.height)

      // 直接更新ripples状态，避免在useEffect中调用setState
      setRipples(prevRipples => {
        const updatedRipples = prevRipples.filter(ripple => {
          ripple.life++
          ripple.radius += ripple.speed
          ripple.opacity = ripple.opacity * (1 - ripple.life / ripple.maxLife)

          if (ripple.life >= ripple.maxLife) {
            return false
          }

          // 绘制涟漪
          ctx.save()
          ctx.globalAlpha = ripple.opacity
          ctx.strokeStyle = color
          ctx.lineWidth = 2
          ctx.beginPath()
          ctx.arc(ripple.x, ripple.y, ripple.radius, 0, Math.PI * 2)
          ctx.stroke()

          // 绘制内部涟漪
          if (ripple.radius > 20) {
            ctx.globalAlpha = ripple.opacity * 0.5
            ctx.lineWidth = 1
            ctx.beginPath()
            ctx.arc(ripple.x, ripple.y, ripple.radius * 0.7, 0, Math.PI * 2)
            ctx.stroke()
          }

          // 绘制中心点
          ctx.globalAlpha = ripple.opacity
          ctx.fillStyle = color
          ctx.beginPath()
          ctx.arc(ripple.x, ripple.y, 2, 0, Math.PI * 2)
          ctx.fill()

          ctx.restore()

          return true
        })

        // 继续动画循环
        if (updatedRipples.length > 0) {
          animationRef.current = requestAnimationFrame(animate)
        }

        return updatedRipples
      })
    }

    // 只在有涟漪时开始动画
    if (ripples.length > 0) {
      animate()
    }

    return () => {
      window.removeEventListener('resize', resizeCanvas)
      if (animationRef.current) {
        cancelAnimationFrame(animationRef.current)
      }
    }
  }, [ripples.length, color]) // 只依赖ripples.length和color

  return (
    <div
      ref={containerRef}
      className={`relative ${className}`}
      onMouseDown={handleMouseDown}
    >
      {children}
      <canvas
        ref={canvasRef}
        className="absolute inset-0 pointer-events-none"
        style={{ zIndex: 10 }}
      />
    </div>
  )
}

export default CanvasWaterRipple
