import React, { useEffect, useRef, useState } from 'react'
import { motion } from 'framer-motion'

interface LiquidCrystalBackgroundProps {
  enableMouseInteraction?: boolean
  enableRipple?: boolean
  enableAvoidance?: boolean
  intensity?: 'low' | 'medium' | 'high'
  color?: string
}

interface Ripple {
  id: number
  x: number
  y: number
  radius: number
  maxRadius: number
  life: number
  maxLife: number
  color: string
  opacity: number
}

interface Crystal {
  id: number
  x: number
  y: number
  size: number
  rotation: number
  rotationSpeed: number
  color: string
  opacity: number
  vx: number
  vy: number
}

const LiquidCrystalBackground: React.FC<LiquidCrystalBackgroundProps> = ({
  enableMouseInteraction = true,
  enableRipple = true,
  enableAvoidance = true,
  intensity = 'medium',
  color = '#00ff88'
}) => {
  const [ripples, setRipples] = useState<Ripple[]>([])
  const [crystals, setCrystals] = useState<Crystal[]>([])
  const [mousePosition, setMousePosition] = useState({ x: 0, y: 0 })
  const [isMouseOver, setIsMouseOver] = useState(false)
  const containerRef = useRef<HTMLDivElement>(null)
  const animationRef = useRef<number>()
  const rippleIdRef = useRef(0)
  const crystalIdRef = useRef(0)

  const intensitySettings = {
    low: { crystalCount: 8, rippleIntensity: 0.3, avoidanceStrength: 0.2 },
    medium: { crystalCount: 12, rippleIntensity: 0.5, avoidanceStrength: 0.4 },
    high: { crystalCount: 18, rippleIntensity: 0.8, avoidanceStrength: 0.6 }
  }

  const settings = intensitySettings[intensity]

  const createCrystal = (): Crystal => {
    const angle = Math.random() * Math.PI * 2
    const speed = 0.1 + Math.random() * 0.3
    const size = 1 + Math.random() * 2
    
    return {
      id: crystalIdRef.current++,
      x: Math.random() * (containerRef.current?.clientWidth || window.innerWidth),
      y: Math.random() * (containerRef.current?.clientHeight || window.innerHeight),
      size,
      rotation: Math.random() * Math.PI * 2,
      rotationSpeed: (Math.random() - 0.5) * 0.01,
      color: color,
      opacity: 0.05 + Math.random() * 0.1,
      vx: Math.cos(angle) * speed,
      vy: Math.sin(angle) * speed
    }
  }

  const createRipple = (x: number, y: number): Ripple => {
    return {
      id: rippleIdRef.current++,
      x,
      y,
      radius: 0,
      maxRadius: 80 + Math.random() * 60,
      life: 1,
      maxLife: 40 + Math.random() * 20,
      color: color,
      opacity: 0.1 + Math.random() * 0.15
    }
  }

  const handleMouseMove = (e: React.MouseEvent) => {
    if (!enableMouseInteraction) return
    
    const rect = containerRef.current?.getBoundingClientRect()
    if (!rect) return
    
    const x = e.clientX - rect.left
    const y = e.clientY - rect.top
    
    setMousePosition({ x, y })
    
    // 极低波纹频率
    if (enableRipple && Math.random() < 0.1) {
      setRipples(prev => [...prev, createRipple(x, y)])
    }
    
    // 极低水晶创建频率
    if (Math.random() < 0.02) {
      setCrystals(prev => [...prev, createCrystal()])
    }
  }

  const handleMouseEnter = () => {
    setIsMouseOver(true)
  }

  const handleMouseLeave = () => {
    setIsMouseOver(false)
  }

  const handleMouseClick = (e: React.MouseEvent) => {
    if (!enableMouseInteraction) return
    
    const rect = containerRef.current?.getBoundingClientRect()
    if (!rect) return
    
    const x = e.clientX - rect.left
    const y = e.clientY - rect.top
    
    // 创建爆炸效果
    for (let i = 0; i < 5; i++) {
      setTimeout(() => {
        setRipples(prev => [...prev, createRipple(x, y)])
      }, i * 100)
    }
    
    // 创建更多水晶
    for (let i = 0; i < 3; i++) {
      setCrystals(prev => [...prev, createCrystal()])
    }
  }

  const animateCrystals = () => {
    setCrystals(prev => 
      prev.map(crystal => {
        let newX = crystal.x + crystal.vx
        let newY = crystal.y + crystal.vy
        
        // 鼠标避让效果 - 增强版
        if (enableAvoidance && isMouseOver) {
          const dx = crystal.x - mousePosition.x
          const dy = crystal.y - mousePosition.y
          const distance = Math.sqrt(dx * dx + dy * dy)
          const avoidanceRadius = 150 // 增大避让半径
          
          if (distance < avoidanceRadius) {
            const force = (avoidanceRadius - distance) / avoidanceRadius
            const angle = Math.atan2(dy, dx)
            const pushForce = force * settings.avoidanceStrength * 4 // 增强推力
            newX += Math.cos(angle) * pushForce
            newY += Math.sin(angle) * pushForce
            
            // 添加旋转效果
            crystal.rotationSpeed = (Math.random() - 0.5) * 0.1 * force
          }
        }
        
        // 边界反弹
        const containerWidth = containerRef.current?.clientWidth || window.innerWidth
        const containerHeight = containerRef.current?.clientHeight || window.innerHeight
        
        if (newX < 0 || newX > containerWidth) {
          crystal.vx *= -1
          newX = crystal.x
        }
        if (newY < 0 || newY > containerHeight) {
          crystal.vy *= -1
          newY = crystal.y
        }
        
        return {
          ...crystal,
          x: newX,
          y: newY,
          rotation: crystal.rotation + crystal.rotationSpeed
        }
      })
    )
  }

  const animateRipples = () => {
    setRipples(prev => 
      prev
        .map(ripple => ({
          ...ripple,
          radius: ripple.radius + 2,
          life: ripple.life - 1,
          opacity: ripple.opacity * (ripple.life / ripple.maxLife)
        }))
        .filter(ripple => ripple.life > 0 && ripple.radius < ripple.maxRadius)
    )
  }

  useEffect(() => {
    // 初始化水晶
    const initialCrystals = Array.from({ length: settings.crystalCount }, createCrystal)
    setCrystals(initialCrystals)
  }, [settings.crystalCount])

  useEffect(() => {
    const animate = () => {
      animateCrystals()
      animateRipples()
      animationRef.current = requestAnimationFrame(animate)
    }
    animate()

    return () => {
      if (animationRef.current) {
        cancelAnimationFrame(animationRef.current)
      }
    }
  }, [isMouseOver, mousePosition, settings.avoidanceStrength])

  const renderCrystals = () => {
    return (
      <div className="absolute inset-0 pointer-events-none overflow-hidden">
        {crystals.map(crystal => (
          <motion.div
            key={crystal.id}
            className="absolute"
            style={{
              left: crystal.x,
              top: crystal.y,
              width: crystal.size,
              height: crystal.size,
              backgroundColor: crystal.color,
              opacity: crystal.opacity,
              boxShadow: `0 0 ${crystal.size * 2}px ${crystal.color}`,
              borderRadius: '50%'
            }}
            animate={{
              rotate: crystal.rotation,
              scale: [1, 1.1, 1],
              opacity: [crystal.opacity * 0.5, crystal.opacity, crystal.opacity * 0.5]
            }}
            transition={{
              duration: 3,
              repeat: Infinity,
              ease: "easeInOut"
            }}
          />
        ))}
      </div>
    )
  }

  const renderRipples = () => {
    if (!enableRipple || ripples.length === 0) return null

    return (
      <div className="absolute inset-0 pointer-events-none">
        {ripples.map(ripple => (
          <motion.div
            key={ripple.id}
            className="absolute rounded-full border"
            style={{
              left: ripple.x,
              top: ripple.y,
              width: ripple.radius,
              height: ripple.radius,
              borderColor: ripple.color,
              opacity: ripple.opacity,
              boxShadow: `0 0 ${ripple.radius / 3}px ${ripple.color}`,
              transform: 'translate(-50%, -50%)'
            }}
            animate={{
              scale: [0, 1, 1.2],
              opacity: [ripple.opacity, ripple.opacity * 0.5, 0]
            }}
            transition={{
              duration: 1,
              ease: "easeOut"
            }}
          />
        ))}
      </div>
    )
  }

  const renderLiquidCrystalGrid = () => {
    return (
      <div className="absolute inset-0">
        {/* 极简液晶网格 - 几乎看不见 */}
        <div 
          className="w-full h-full"
          style={{
            backgroundImage: `
              linear-gradient(${color}08 1px, transparent 1px),
              linear-gradient(90deg, ${color}08 1px, transparent 1px)
            `,
            backgroundSize: '30px 30px',
            animation: 'pulse 6s ease-in-out infinite'
          }}
        />
        
        {/* 极简扫描线 - 只有3条，几乎看不见 */}
        <div className="absolute inset-0">
          {Array.from({ length: 3 }, (_, i) => (
            <motion.div
              key={i}
              className="absolute w-full h-px"
              style={{
                background: `linear-gradient(90deg, transparent, ${color}15, transparent)`,
                top: `${(i + 1) * 33.33}%`,
                opacity: 0.05
              }}
              animate={{
                opacity: [0.02, 0.08, 0.02],
                scaleX: [0.95, 1.05, 0.95]
              }}
              transition={{
                duration: 8 + i * 2,
                repeat: Infinity,
                delay: i * 2
              }}
            />
          ))}
        </div>
        
        {/* 极简像素点 - 只有10个，几乎看不见 */}
        <div className="absolute inset-0">
          {Array.from({ length: 10 }, (_, i) => (
            <motion.div
              key={`pixel-${i}`}
              className="absolute w-0.5 h-0.5"
              style={{
                backgroundColor: color,
                left: `${(i * 10) % 100}%`,
                top: `${(i * 10) % 100}%`,
                opacity: 0.08
              }}
              animate={{
                opacity: [0.03, 0.12, 0.03],
                scale: [0.8, 1.1, 0.8]
              }}
              transition={{
                duration: 5 + (i % 2),
                repeat: Infinity,
                delay: i * 1
              }}
            />
          ))}
        </div>
      </div>
    )
  }

  return (
    <div
      ref={containerRef}
      className="fixed inset-0 bg-gradient-to-br from-black via-slate-900 to-black overflow-hidden cursor-none"
      onMouseMove={handleMouseMove}
      onMouseEnter={handleMouseEnter}
      onMouseLeave={handleMouseLeave}
      onClick={handleMouseClick}
    >
      {/* 液晶网格背景 */}
      {renderLiquidCrystalGrid()}
      
      {/* 水晶粒子 */}
      {renderCrystals()}
      
      {/* 波纹效果 */}
      {renderRipples()}
      
      {/* 极淡鼠标光晕 */}
      {isMouseOver && (
        <>
          {/* 主光晕 - 极淡 */}
          <motion.div
            className="absolute pointer-events-none"
            style={{
              left: mousePosition.x,
              top: mousePosition.y,
              width: 200,
              height: 200,
              background: `radial-gradient(circle, ${color}08 0%, ${color}04 30%, transparent 70%)`,
              borderRadius: '50%',
              transform: 'translate(-50%, -50%)',
              filter: 'blur(20px)'
            }}
            animate={{
              scale: [1, 1.1, 1],
              opacity: [0.1, 0.2, 0.1]
            }}
            transition={{
              duration: 3,
              repeat: Infinity,
              ease: "easeInOut"
            }}
          />
          
          {/* 核心光点 - 极淡 */}
          <motion.div
            className="absolute pointer-events-none"
            style={{
              left: mousePosition.x,
              top: mousePosition.y,
              width: 8,
              height: 8,
              backgroundColor: color,
              borderRadius: '50%',
              transform: 'translate(-50%, -50%)',
              opacity: 0.15
            }}
            animate={{
              scale: [1, 1.1, 1],
              opacity: [0.1, 0.2, 0.1]
            }}
            transition={{
              duration: 2,
              repeat: Infinity,
              ease: "easeInOut"
            }}
          />
        </>
      )}
      
      {/* 边缘光效 */}
      <div className="absolute inset-0">
        <div 
          className="absolute top-0 left-0 w-full h-px bg-gradient-to-r from-transparent via-current to-transparent"
          style={{ color }}
        />
        <div 
          className="absolute bottom-0 left-0 w-full h-px bg-gradient-to-r from-transparent via-current to-transparent"
          style={{ color }}
        />
        <div 
          className="absolute left-0 top-0 w-px h-full bg-gradient-to-b from-transparent via-current to-transparent"
          style={{ color }}
        />
        <div 
          className="absolute right-0 top-0 w-px h-full bg-gradient-to-b from-transparent via-current to-transparent"
          style={{ color }}
        />
      </div>
      
      {/* 极淡自定义鼠标指针 */}
      {isMouseOver && (
        <motion.div
          className="absolute pointer-events-none z-50"
          style={{
            left: mousePosition.x,
            top: mousePosition.y,
            width: 12,
            height: 12,
            background: `radial-gradient(circle, ${color}20 0%, transparent 70%)`,
            borderRadius: '50%',
            transform: 'translate(-50%, -50%)',
            border: `1px solid ${color}30`,
            opacity: 0.2
          }}
          animate={{
            scale: [1, 1.1, 1],
            rotate: [0, 180]
          }}
          transition={{
            duration: 4,
            repeat: Infinity,
            ease: "easeInOut"
          }}
        />
      )}
    </div>
  )
}

export default LiquidCrystalBackground
