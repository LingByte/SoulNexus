import { ReactNode, useEffect, useRef, useState, useCallback } from 'react'
import { motion } from 'framer-motion'
import { cn } from '@/utils/cn.ts'

interface InfiniteScrollProps<T> {
  items: T[]
  renderItem: (item: T, index: number) => ReactNode
  loadMore: () => Promise<void>
  hasMore: boolean
  loading?: boolean
  className?: string
  itemClassName?: string
  loadingComponent?: ReactNode
  endComponent?: ReactNode
  threshold?: number
  animation?: 'none' | 'fade' | 'slide' | 'scale' | 'stagger'
  delay?: number
  onLoadMore?: () => void
}

const InfiniteScroll = <T,>({
  items,
  renderItem,
  loadMore,
  hasMore,
  className = "",
  itemClassName = "",
  loadingComponent,
  endComponent,
  threshold = 100,
  animation = 'stagger',
  delay = 0,
  onLoadMore
}: InfiniteScrollProps<T>) => {
  const [isLoading, setIsLoading] = useState(false)
  const observerRef = useRef<HTMLDivElement>(null)
  const containerRef = useRef<HTMLDivElement>(null)

  // 加载更多数据
  const handleLoadMore = useCallback(async () => {
    if (isLoading || !hasMore) return

    setIsLoading(true)
    try {
      await loadMore()
      onLoadMore?.()
    } catch (error) {
      console.error('Failed to load more items:', error)
    } finally {
      setIsLoading(false)
    }
  }, [isLoading, hasMore, loadMore, onLoadMore])

  // Intersection Observer
  useEffect(() => {
    const observer = new IntersectionObserver(
      ([entry]) => {
        if (entry.isIntersecting && hasMore && !isLoading) {
          handleLoadMore()
        }
      },
      {
        threshold: 0.1,
        rootMargin: `${threshold}px`
      }
    )

    if (observerRef.current) {
      observer.observe(observerRef.current)
    }

    return () => observer.disconnect()
  }, [hasMore, isLoading, handleLoadMore, threshold])

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
      initial: { opacity: 0, y: 30 },
      animate: { opacity: 1, y: 0 }
    },
    scale: {
      initial: { opacity: 0, scale: 0.9 },
      animate: { opacity: 1, scale: 1 }
    },
    stagger: {
      initial: { opacity: 0, y: 30, scale: 0.9 },
      animate: { opacity: 1, y: 0, scale: 1 }
    }
  }

  const currentAnimation = animationVariants[animation]

  const defaultLoadingComponent = (
    <div className="flex items-center justify-center py-8">
      <motion.div
        className="flex items-center space-x-2"
        initial={{ opacity: 0 }}
        animate={{ opacity: 1 }}
        transition={{ duration: 0.3 }}
      >
        <div className="w-4 h-4 bg-blue-500 rounded-full animate-bounce"></div>
        <div className="w-4 h-4 bg-blue-500 rounded-full animate-bounce" style={{ animationDelay: '0.1s' }}></div>
        <div className="w-4 h-4 bg-blue-500 rounded-full animate-bounce" style={{ animationDelay: '0.2s' }}></div>
        <span className="ml-2 text-gray-600">加载中...</span>
      </motion.div>
    </div>
  )

  const defaultEndComponent = (
    <div className="flex items-center justify-center py-8">
      <div className="text-gray-500 text-sm">没有更多内容了</div>
    </div>
  )

  return (
    <div ref={containerRef} className={cn('w-full', className)}>
      {/* 项目列表 */}
      <div className="space-y-4">
        {items.map((item, index) => (
          <motion.div
            key={index}
            className={cn('w-full', itemClassName)}
            initial={currentAnimation.initial}
            animate={currentAnimation.animate}
            transition={{
              duration: 0.4,
              delay: delay + (index * 0.05),
              ease: "easeOut"
            }}
          >
            {renderItem(item, index)}
          </motion.div>
        ))}
      </div>

      {/* 加载更多触发器 */}
      {hasMore && (
        <div ref={observerRef} className="h-4">
          {isLoading && (loadingComponent || defaultLoadingComponent)}
        </div>
      )}

      {/* 结束组件 */}
      {!hasMore && !isLoading && (endComponent || defaultEndComponent)}
    </div>
  )
}

export default InfiniteScroll
