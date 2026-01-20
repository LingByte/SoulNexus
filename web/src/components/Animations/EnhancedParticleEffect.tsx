import { useEffect, useRef, useState } from 'react'
import { motion } from 'framer-motion'
import { playParticleSound } from '@/utils/audioEffects.ts'

interface Particle {
  id: number
  x: number
  y: number
  vx: number
  vy: number
  life: number
  maxLife: number
  size: number
  color: string
  alpha: number
}

interface EnhancedParticleEffectProps {
  count?: number
  colors?: string[]
  interactive?: boolean
  className?: string
  enableAudio?: boolean
  intensity?: 'low' | 'medium' | 'high'
  shape?: 'circle' | 'square' | 'star' | 'heart'
}

const EnhancedParticleEffect = ({
  count = 50,
  colors = ['#3b82f6', '#8b5cf6', '#06b6d4', '#10b981', '#f59e0b'],
  interactive = true,
  className = '',
  enableAudio = true,
  intensity = 'medium',
  shape = 'circle'
}: EnhancedParticleEffectProps) => {
  const canvasRef = useRef<HTMLCanvasElement>(null)
  const animationRef = useRef<number>()
  const particlesRef = useRef<Particle[]>([])
  const mouseRef = useRef({ x: 0, y: 0 })
  const [isHovered, setIsHovered] = useState(false)

  const getIntensityMultiplier = () => {
    switch (intensity) {
      case 'low': return 0.5
      case 'medium': return 1
      case 'high': return 1.5
      default: return 1
    }
  }

  const createParticle = (x: number, y: number): Particle => {
    const multiplier = getIntensityMultiplier()
    return {
      id: Math.random(),
      x,
      y,
      vx: (Math.random() - 0.5) * 2 * multiplier,
      vy: (Math.random() - 0.5) * 2 * multiplier,
      life: 1,
      maxLife: Math.random() * 100 + 50,
      size: Math.random() * 4 + 1,
      color: colors[Math.floor(Math.random() * colors.length)],
      alpha: Math.random() * 0.8 + 0.2
    }
  }

  const drawParticle = (ctx: CanvasRenderingContext2D, particle: Particle) => {
    ctx.save()
    ctx.globalAlpha = particle.alpha * (particle.life / particle.maxLife)
    ctx.fillStyle = particle.color
    
    const size = particle.size * (particle.life / particle.maxLife)
    
    switch (shape) {
      case 'circle':
        ctx.beginPath()
        ctx.arc(particle.x, particle.y, size, 0, Math.PI * 2)
        ctx.fill()
        break
      case 'square':
        ctx.fillRect(particle.x - size/2, particle.y - size/2, size, size)
        break
      case 'star':
        drawStar(ctx, particle.x, particle.y, size)
        break
      case 'heart':
        drawHeart(ctx, particle.x, particle.y, size)
        break
    }
    ctx.restore()
  }

  const drawStar = (ctx: CanvasRenderingContext2D, x: number, y: number, size: number) => {
    const spikes = 5
    const outerRadius = size
    const innerRadius = size * 0.4
    
    ctx.beginPath()
    for (let i = 0; i < spikes * 2; i++) {
      const radius = i % 2 === 0 ? outerRadius : innerRadius
      const angle = (i * Math.PI) / spikes
      const px = x + Math.cos(angle) * radius
      const py = y + Math.sin(angle) * radius
      
      if (i === 0) {
        ctx.moveTo(px, py)
      } else {
        ctx.lineTo(px, py)
      }
    }
    ctx.closePath()
    ctx.fill()
  }

  const drawHeart = (ctx: CanvasRenderingContext2D, x: number, y: number, size: number) => {
    ctx.beginPath()
    const topCurveHeight = size * 0.3
    ctx.moveTo(x, y + topCurveHeight)
    ctx.bezierCurveTo(x, y, x - size/2, y, x - size/2, y + topCurveHeight)
    ctx.bezierCurveTo(x - size/2, y + (size * 0.7), x, y + (size * 0.7), x, y + size)
    ctx.bezierCurveTo(x, y + (size * 0.7), x + size/2, y + (size * 0.7), x + size/2, y + topCurveHeight)
    ctx.bezierCurveTo(x + size/2, y, x, y, x, y + topCurveHeight)
    ctx.closePath()
    ctx.fill()
  }

  const updateParticle = (particle: Particle, canvas: HTMLCanvasElement) => {
    particle.x += particle.vx
    particle.y += particle.vy
    particle.life--
    
    // 重力效果
    particle.vy += 0.02
    
    // 鼠标交互
    if (interactive && isHovered) {
      const dx = mouseRef.current.x - particle.x
      const dy = mouseRef.current.y - particle.y
      const distance = Math.sqrt(dx * dx + dy * dy)
      
      if (distance < 100) {
        const force = (100 - distance) / 100
        particle.vx += (dx / distance) * force * 0.1
        particle.vy += (dy / distance) * force * 0.1
      }
    }
    
    // 边界反弹
    if (particle.x < 0 || particle.x > canvas.width) {
      particle.vx *= -0.8
      particle.x = Math.max(0, Math.min(canvas.width, particle.x))
    }
    if (particle.y < 0 || particle.y > canvas.height) {
      particle.vy *= -0.8
      particle.y = Math.max(0, Math.min(canvas.height, particle.y))
    }
    
    // 摩擦力
    particle.vx *= 0.99
    particle.vy *= 0.99
  }

  const animate = () => {
    const canvas = canvasRef.current
    if (!canvas) return

    const ctx = canvas.getContext('2d')
    if (!ctx) return

    ctx.clearRect(0, 0, canvas.width, canvas.height)

    // 更新和绘制粒子
    particlesRef.current = particlesRef.current.filter(particle => {
      updateParticle(particle, canvas)
      if (particle.life > 0) {
        drawParticle(ctx, particle)
        return true
      }
      return false
    })

    // 添加新粒子
    if (particlesRef.current.length < count) {
      const newParticle = createParticle(
        Math.random() * canvas.width,
        Math.random() * canvas.height
      )
      particlesRef.current.push(newParticle)
    }

    animationRef.current = requestAnimationFrame(animate)
  }

  const handleMouseMove = (e: React.MouseEvent<HTMLCanvasElement>) => {
    const canvas = canvasRef.current
    if (!canvas) return

    const rect = canvas.getBoundingClientRect()
    mouseRef.current = {
      x: e.clientX - rect.left,
      y: e.clientY - rect.top
    }
  }

  const handleMouseEnter = () => {
    setIsHovered(true)
    if (enableAudio) {
      playParticleSound()
    }
  }

  const handleMouseLeave = () => {
    setIsHovered(false)
  }

  const handleClick = (e: React.MouseEvent<HTMLCanvasElement>) => {
    const canvas = canvasRef.current
    if (!canvas) return

    const rect = canvas.getBoundingClientRect()
    const x = e.clientX - rect.left
    const y = e.clientY - rect.top

    // 在点击位置创建爆炸效果
    for (let i = 0; i < 10; i++) {
      const particle = createParticle(x, y)
      particle.vx = (Math.random() - 0.5) * 10
      particle.vy = (Math.random() - 0.5) * 10
      particle.life = particle.maxLife = 30
      particlesRef.current.push(particle)
    }

    if (enableAudio) {
      playParticleSound()
    }
  }

  useEffect(() => {
    const canvas = canvasRef.current
    if (!canvas) return

    const resizeCanvas = () => {
      const rect = canvas.getBoundingClientRect()
      canvas.width = rect.width
      canvas.height = rect.height
    }

    resizeCanvas()
    window.addEventListener('resize', resizeCanvas)

    // 初始化粒子
    particlesRef.current = []
    for (let i = 0; i < count; i++) {
      particlesRef.current.push(createParticle(
        Math.random() * canvas.width,
        Math.random() * canvas.height
      ))
    }

    animate()

    return () => {
      window.removeEventListener('resize', resizeCanvas)
      if (animationRef.current) {
        cancelAnimationFrame(animationRef.current)
      }
    }
  }, [count, colors, intensity, shape])

  return (
    <motion.canvas
      ref={canvasRef}
      className={`absolute inset-0 w-full h-full ${className}`}
      onMouseMove={handleMouseMove}
      onMouseEnter={handleMouseEnter}
      onMouseLeave={handleMouseLeave}
      onClick={handleClick}
      initial={{ opacity: 0 }}
      animate={{ opacity: 1 }}
      transition={{ duration: 0.5 }}
    />
  )
}

export default EnhancedParticleEffect
