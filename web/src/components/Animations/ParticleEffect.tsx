import { useEffect, useRef, useState } from 'react'
import { motion } from 'framer-motion'

interface Particle {
  id: number
  x: number
  y: number
  vx: number
  vy: number
  size: number
  color: string
  life: number
  maxLife: number
}

interface ParticleEffectProps {
  count?: number
  colors?: string[]
  size?: { min: number; max: number }
  speed?: { min: number; max: number }
  life?: { min: number; max: number }
  gravity?: number
  className?: string
  interactive?: boolean
}

export const ParticleEffect = ({
  count = 50,
  colors = ['#3b82f6', '#8b5cf6', '#06b6d4', '#10b981', '#f59e0b'],
  size = { min: 2, max: 8 },
  speed = { min: 0.5, max: 2 },
  life = { min: 100, max: 200 },
  gravity = 0.1,
  className = '',
  interactive = true
}: ParticleEffectProps) => {
  const canvasRef = useRef<HTMLCanvasElement>(null)
  const particlesRef = useRef<Particle[]>([])
  const animationRef = useRef<number>()
  const [mouse, setMouse] = useState({ x: 0, y: 0 })

  useEffect(() => {
    const canvas = canvasRef.current
    if (!canvas) return

    const ctx = canvas.getContext('2d')
    if (!ctx) return

    const resizeCanvas = () => {
      canvas.width = canvas.offsetWidth
      canvas.height = canvas.offsetHeight
    }

    resizeCanvas()
    window.addEventListener('resize', resizeCanvas)

    // 创建粒子
    const createParticle = (x: number, y: number): Particle => ({
      id: Math.random(),
      x,
      y,
      vx: (Math.random() - 0.5) * (speed.max - speed.min) + speed.min,
      vy: (Math.random() - 0.5) * (speed.max - speed.min) + speed.min,
      size: Math.random() * (size.max - size.min) + size.min,
      color: colors[Math.floor(Math.random() * colors.length)],
      life: Math.random() * (life.max - life.min) + life.min,
      maxLife: Math.random() * (life.max - life.min) + life.min
    })

    // 初始化粒子
    const initParticles = () => {
      particlesRef.current = []
      for (let i = 0; i < count; i++) {
        particlesRef.current.push(
          createParticle(
            Math.random() * canvas.width,
            Math.random() * canvas.height
          )
        )
      }
    }

    initParticles()

    // 更新粒子
    const updateParticles = () => {
      particlesRef.current = particlesRef.current
        .map(particle => {
          // 更新位置
          particle.x += particle.vx
          particle.y += particle.vy
          particle.vy += gravity
          particle.life -= 1

          // 边界检测
          if (particle.x < 0 || particle.x > canvas.width) {
            particle.vx *= -0.8
            particle.x = Math.max(0, Math.min(canvas.width, particle.x))
          }
          if (particle.y < 0 || particle.y > canvas.height) {
            particle.vy *= -0.8
            particle.y = Math.max(0, Math.min(canvas.height, particle.y))
          }

          return particle
        })
        .filter(particle => particle.life > 0)

      // 补充粒子
      while (particlesRef.current.length < count) {
        particlesRef.current.push(
          createParticle(
            Math.random() * canvas.width,
            Math.random() * canvas.height
          )
        )
      }
    }

    // 渲染粒子
    const renderParticles = () => {
      ctx.clearRect(0, 0, canvas.width, canvas.height)
      
      particlesRef.current.forEach(particle => {
        const alpha = particle.life / particle.maxLife
        ctx.save()
        ctx.globalAlpha = alpha
        ctx.fillStyle = particle.color
        ctx.beginPath()
        ctx.arc(particle.x, particle.y, particle.size, 0, Math.PI * 2)
        ctx.fill()
        ctx.restore()
      })
    }

    // 动画循环
    const animate = () => {
      updateParticles()
      renderParticles()
      animationRef.current = requestAnimationFrame(animate)
    }

    animate()

    // 鼠标交互
    const handleMouseMove = (e: MouseEvent) => {
      const rect = canvas.getBoundingClientRect()
      setMouse({
        x: e.clientX - rect.left,
        y: e.clientY - rect.top
      })

      if (interactive) {
        // 在鼠标位置创建新粒子
        for (let i = 0; i < 3; i++) {
          particlesRef.current.push(
            createParticle(mouse.x, mouse.y)
          )
        }
      }
    }

    if (interactive) {
      canvas.addEventListener('mousemove', handleMouseMove)
    }

    return () => {
      window.removeEventListener('resize', resizeCanvas)
      if (interactive) {
        canvas.removeEventListener('mousemove', handleMouseMove)
      }
      if (animationRef.current) {
        cancelAnimationFrame(animationRef.current)
      }
    }
  }, [count, colors, size, speed, life, gravity, interactive, mouse.x, mouse.y])

  return (
    <motion.canvas
      ref={canvasRef}
      className={`absolute inset-0 pointer-events-none ${className}`}
      initial={{ opacity: 0 }}
      animate={{ opacity: 1 }}
      transition={{ duration: 1 }}
    />
  )
}

export default ParticleEffect
