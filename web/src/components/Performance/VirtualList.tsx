import { ReactNode, useMemo, useRef, useEffect, useState } from 'react'
import { motion } from 'framer-motion'
import { cn } from '@/utils/cn.ts'

interface VirtualListProps<T> {
  items: T[]
  itemHeight: number
  containerHeight: number
  renderItem: (item: T, index: number) => ReactNode
  className?: string
  itemClassName?: string
  overscan?: number
  animation?: 'none' | 'fade' | 'slide' | 'scale'
  delay?: number
  onScroll?: (scrollTop: number) => void
  onVisibleItemsChange?: (visibleItems: T[]) => void
}

const VirtualList = <T,>({
  items,
  itemHeight,
  containerHeight,
  renderItem,
  className = "",
  itemClassName = "",
  overscan = 5,
  animation = 'fade',
  delay = 0,
  onScroll,
  onVisibleItemsChange
}: VirtualListProps<T>) => {
  const [scrollTop, setScrollTop] = useState(0)
  const containerRef = useRef<HTMLDivElement>(null)

  // 计算可见范围
  const visibleRange = useMemo(() => {
    const startIndex = Math.floor(scrollTop / itemHeight)
    const endIndex = Math.min(
      startIndex + Math.ceil(containerHeight / itemHeight) + overscan,
      items.length - 1
    )
    
    return {
      start: Math.max(0, startIndex - overscan),
      end: endIndex
    }
  }, [scrollTop, itemHeight, containerHeight, overscan, items.length])

  // 可见项目
  const visibleItems = useMemo(() => {
    return items.slice(visibleRange.start, visibleRange.end + 1)
  }, [items, visibleRange])

  // 总高度
  const totalHeight = items.length * itemHeight

  // 偏移量
  const offsetY = visibleRange.start * itemHeight

  // 处理滚动
  const handleScroll = (e: React.UIEvent<HTMLDivElement>) => {
    const scrollTop = e.currentTarget.scrollTop
    setScrollTop(scrollTop)
    onScroll?.(scrollTop)
  }

  // 通知可见项目变化
  useEffect(() => {
    onVisibleItemsChange?.(visibleItems)
  }, [visibleItems, onVisibleItemsChange])

  const animationVariants = {
    none: {
      initial: { opacity: 1 },
      animate: { opacity: 1 }
    },
    fade: {
      initial: { opacity: 0 },
      animate: { opacity: 1 }
    },
    slide: {
      initial: { opacity: 0, y: 20 },
      animate: { opacity: 1, y: 0 }
    },
    scale: {
      initial: { opacity: 0, scale: 0.9 },
      animate: { opacity: 1, scale: 1 }
    }
  }

  const currentAnimation = animationVariants[animation]

  return (
    <div
      ref={containerRef}
      className={cn('overflow-auto', className)}
      style={{ height: containerHeight }}
      onScroll={handleScroll}
    >
      <div style={{ height: totalHeight, position: 'relative' }}>
        <div
          style={{
            transform: `translateY(${offsetY}px)`,
            position: 'absolute',
            top: 0,
            left: 0,
            right: 0
          }}
        >
          {visibleItems.map((item, index) => {
            const actualIndex = visibleRange.start + index
            return (
              <motion.div
                key={actualIndex}
                className={cn('w-full', itemClassName)}
                style={{ height: itemHeight }}
                initial={currentAnimation.initial}
                animate={currentAnimation.animate}
                transition={{
                  duration: 0.3,
                  delay: delay + (index * 0.05),
                  ease: "easeOut"
                }}
              >
                {renderItem(item, actualIndex)}
              </motion.div>
            )
          })}
        </div>
      </div>
    </div>
  )
}

export default VirtualList
