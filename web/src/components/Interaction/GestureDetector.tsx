import { useRef, useEffect, useState, useCallback } from 'react'
import { motion } from 'framer-motion'

interface GestureDetectorProps {
  children: React.ReactNode
  onSwipe?: (direction: 'up' | 'down' | 'left' | 'right', velocity: number) => void
  onPinch?: (scale: number, velocity: number) => void
  onRotate?: (rotation: number, velocity: number) => void
  onTap?: (position: { x: number; y: number }) => void
  onDoubleTap?: (position: { x: number; y: number }) => void
  onLongPress?: (position: { x: number; y: number }) => void
  onPan?: (position: { x: number; y: number }, velocity: { x: number; y: number }) => void
  onPanStart?: (position: { x: number; y: number }) => void
  onPanEnd?: (position: { x: number; y: number }) => void
  swipeThreshold?: number
  pinchThreshold?: number
  rotateThreshold?: number
  longPressDelay?: number
  doubleTapDelay?: number
  className?: string
  disabled?: boolean
}

interface TouchPoint {
  id: number
  x: number
  y: number
  startX: number
  startY: number
  startTime: number
}

const GestureDetector = ({
  children,
  onSwipe,
  onPinch,
  onRotate,
  onTap,
  onDoubleTap,
  onLongPress,
  onPan,
  onPanStart,
  onPanEnd,
  swipeThreshold = 50,
  pinchThreshold = 0.1,
  rotateThreshold = 10,
  longPressDelay = 500,
  doubleTapDelay = 300,
  className = '',
  disabled = false
}: GestureDetectorProps) => {
  const containerRef = useRef<HTMLDivElement>(null)
  const touchesRef = useRef<Map<number, TouchPoint>>(new Map())
  const lastTapRef = useRef<{ time: number; x: number; y: number } | null>(null)
  const longPressTimerRef = useRef<NodeJS.Timeout | null>(null)
  const [isGesturing, setIsGesturing] = useState(false)

  // 计算两点之间的距离
  const getDistance = (p1: TouchPoint, p2: TouchPoint) => {
    const dx = p2.x - p1.x
    const dy = p2.y - p1.y
    return Math.sqrt(dx * dx + dy * dy)
  }

  // 计算两点之间的角度
  const getAngle = (p1: TouchPoint, p2: TouchPoint) => {
    return Math.atan2(p2.y - p1.y, p2.x - p1.x) * (180 / Math.PI)
  }

  // 计算中心点
  const getCenter = (touches: TouchPoint[]) => {
    if (touches.length === 0) return { x: 0, y: 0 }
    
    const sum = touches.reduce(
      (acc, touch) => ({ x: acc.x + touch.x, y: acc.y + touch.y }),
      { x: 0, y: 0 }
    )
    
    return {
      x: sum.x / touches.length,
      y: sum.y / touches.length
    }
  }

  // 处理触摸开始
  const handleTouchStart = useCallback((e: React.TouchEvent) => {
    if (disabled) return

    const touches = Array.from(e.touches)
    const currentTime = Date.now()

    touches.forEach(touch => {
      const touchPoint: TouchPoint = {
        id: touch.identifier,
        x: touch.clientX,
        y: touch.clientY,
        startX: touch.clientX,
        startY: touch.clientY,
        startTime: currentTime
      }
      touchesRef.current.set(touch.identifier, touchPoint)
    })

    // 长按检测
    if (touches.length === 1) {
      const touch = touches[0]
      longPressTimerRef.current = setTimeout(() => {
        onLongPress?.({ x: touch.clientX, y: touch.clientY })
      }, longPressDelay)
    }

    setIsGesturing(true)
    onPanStart?.(getCenter(Array.from(touchesRef.current.values())))
  }, [disabled, onLongPress, onPanStart, longPressDelay])

  // 处理触摸移动
  const handleTouchMove = useCallback((e: React.TouchEvent) => {
    if (disabled) return

    // 清除长按定时器
    if (longPressTimerRef.current) {
      clearTimeout(longPressTimerRef.current)
      longPressTimerRef.current = null
    }

    const touches = Array.from(e.touches)
    const currentTime = Date.now()

    // 更新触摸点位置
    touches.forEach(touch => {
      const existingTouch = touchesRef.current.get(touch.identifier)
      if (existingTouch) {
        existingTouch.x = touch.clientX
        existingTouch.y = touch.clientY
      }
    })

    const touchPoints = Array.from(touchesRef.current.values())
    const center = getCenter(touchPoints)

    // 处理平移
    if (touchPoints.length === 1) {
      const touch = touchPoints[0]
      const velocity = {
        x: (touch.x - touch.startX) / (currentTime - touch.startTime),
        y: (touch.y - touch.startY) / (currentTime - touch.startTime)
      }
      onPan?.(center, velocity)
    }

    // 处理双指手势
    if (touchPoints.length === 2) {
      const [touch1, touch2] = touchPoints
      const currentDistance = getDistance(touch1, touch2)
      const startDistance = getDistance(
        { ...touch1, x: touch1.startX, y: touch1.startY },
        { ...touch2, x: touch2.startX, y: touch2.startY }
      )

      // 缩放手势
      const scale = currentDistance / startDistance
      if (Math.abs(scale - 1) > pinchThreshold) {
        const velocity = (scale - 1) / (currentTime - Math.min(touch1.startTime, touch2.startTime))
        onPinch?.(scale, velocity)
      }

      // 旋转手势
      const currentAngle = getAngle(touch1, touch2)
      const startAngle = getAngle(
        { ...touch1, x: touch1.startX, y: touch1.startY },
        { ...touch2, x: touch2.startX, y: touch2.startY }
      )
      const rotation = currentAngle - startAngle

      if (Math.abs(rotation) > rotateThreshold) {
        const velocity = rotation / (currentTime - Math.min(touch1.startTime, touch2.startTime))
        onRotate?.(rotation, velocity)
      }
    }
  }, [disabled, onPan, onPinch, onRotate, pinchThreshold, rotateThreshold])

  // 处理触摸结束
  const handleTouchEnd = useCallback((e: React.TouchEvent) => {
    if (disabled) return

    // 清除长按定时器
    if (longPressTimerRef.current) {
      clearTimeout(longPressTimerRef.current)
      longPressTimerRef.current = null
    }

    const touches = Array.from(e.touches)
    const currentTime = Date.now()

    // 移除结束的触摸点
    const endedTouches: TouchPoint[] = []
    touchesRef.current.forEach((touch, id) => {
      if (!touches.find(t => t.identifier === id)) {
        endedTouches.push(touch)
        touchesRef.current.delete(id)
      }
    })

    // 处理点击和滑动
    if (endedTouches.length === 1) {
      const touch = endedTouches[0]
      const deltaX = touch.x - touch.startX
      const deltaY = touch.y - touch.startY
      const deltaTime = currentTime - touch.startTime
      const distance = Math.sqrt(deltaX * deltaX + deltaY * deltaY)
      const velocity = distance / deltaTime

      // 判断是否为滑动
      if (distance > swipeThreshold) {
        let direction: 'up' | 'down' | 'left' | 'right'
        
        if (Math.abs(deltaX) > Math.abs(deltaY)) {
          direction = deltaX > 0 ? 'right' : 'left'
        } else {
          direction = deltaY > 0 ? 'down' : 'up'
        }
        
        onSwipe?.(direction, velocity)
      } else if (deltaTime < 300) {
        // 判断是否为双击
        if (lastTapRef.current) {
          const timeDiff = currentTime - lastTapRef.current.time
          const distanceDiff = Math.sqrt(
            Math.pow(touch.x - lastTapRef.current.x, 2) + 
            Math.pow(touch.y - lastTapRef.current.y, 2)
          )
          
          if (timeDiff < doubleTapDelay && distanceDiff < 50) {
            onDoubleTap?.({ x: touch.x, y: touch.y })
            lastTapRef.current = null
            return
          }
        }
        
        // 单击
        onTap?.({ x: touch.x, y: touch.y })
        lastTapRef.current = { time: currentTime, x: touch.x, y: touch.y }
      }
    }

    onPanEnd?.(getCenter(Array.from(touchesRef.current.values())))
    setIsGesturing(false)
  }, [disabled, onSwipe, onTap, onDoubleTap, onPanEnd, swipeThreshold, doubleTapDelay])

  // 处理鼠标事件（桌面端）
  const handleMouseDown = useCallback((e: React.MouseEvent) => {
    if (disabled) return

    const touchPoint: TouchPoint = {
      id: -1,
      x: e.clientX,
      y: e.clientY,
      startX: e.clientX,
      startY: e.clientY,
      startTime: Date.now()
    }
    touchesRef.current.set(-1, touchPoint)

    setIsGesturing(true)
    onPanStart?.({ x: e.clientX, y: e.clientY })
  }, [disabled, onPanStart])

  const handleMouseMove = useCallback((e: React.MouseEvent) => {
    if (disabled || !isGesturing) return

    const touch = touchesRef.current.get(-1)
    if (touch) {
      touch.x = e.clientX
      touch.y = e.clientY
      
      const velocity = {
        x: (touch.x - touch.startX) / (Date.now() - touch.startTime),
        y: (touch.y - touch.startY) / (Date.now() - touch.startTime)
      }
      
      onPan?.({ x: e.clientX, y: e.clientY }, velocity)
    }
  }, [disabled, isGesturing, onPan])

  const handleMouseUp = useCallback((e: React.MouseEvent) => {
    if (disabled) return

    const touch = touchesRef.current.get(-1)
    if (touch) {
      const deltaX = touch.x - touch.startX
      const deltaY = touch.y - touch.startY
      const distance = Math.sqrt(deltaX * deltaX + deltaY * deltaY)
      const velocity = distance / (Date.now() - touch.startTime)

      if (distance > swipeThreshold) {
        let direction: 'up' | 'down' | 'left' | 'right'
        
        if (Math.abs(deltaX) > Math.abs(deltaY)) {
          direction = deltaX > 0 ? 'right' : 'left'
        } else {
          direction = deltaY > 0 ? 'down' : 'up'
        }
        
        onSwipe?.(direction, velocity)
      } else {
        onTap?.({ x: e.clientX, y: e.clientY })
      }
    }

    touchesRef.current.delete(-1)
    onPanEnd?.({ x: e.clientX, y: e.clientY })
    setIsGesturing(false)
  }, [disabled, onSwipe, onTap, onPanEnd, swipeThreshold])

  // 清理定时器
  useEffect(() => {
    return () => {
      if (longPressTimerRef.current) {
        clearTimeout(longPressTimerRef.current)
      }
    }
  }, [])

  return (
    <motion.div
      ref={containerRef}
      className={`relative ${className}`}
      onTouchStart={handleTouchStart}
      onTouchMove={handleTouchMove}
      onTouchEnd={handleTouchEnd}
      onMouseDown={handleMouseDown}
      onMouseMove={handleMouseMove}
      onMouseUp={handleMouseUp}
      onMouseLeave={() => {
        touchesRef.current.clear()
        setIsGesturing(false)
        if (longPressTimerRef.current) {
          clearTimeout(longPressTimerRef.current)
          longPressTimerRef.current = null
        }
      }}
      animate={{
        scale: isGesturing ? 0.98 : 1,
      }}
      transition={{
        duration: 0.1,
        ease: 'easeOut'
      }}
    >
      {children}
    </motion.div>
  )
}

export default GestureDetector
