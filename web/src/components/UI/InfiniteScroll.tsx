import { useEffect, useRef, useCallback } from 'react'
import { cn } from '@/utils/cn.ts'

interface InfiniteScrollProps {
  children: React.ReactNode
  hasMore: boolean
  loadMore: () => void
  threshold?: number
  className?: string
  loadingComponent?: React.ReactNode
  endMessage?: React.ReactNode
}

const InfiniteScroll = ({
  children,
  hasMore,
  loadMore,
  threshold = 100,
  className,
  loadingComponent,
  endMessage
}: InfiniteScrollProps) => {
  const observerRef = useRef<IntersectionObserver | null>(null)
  const loadingRef = useRef<HTMLDivElement>(null)

  const handleObserver = useCallback(
    (entries: IntersectionObserverEntry[]) => {
      const [target] = entries
      if (target.isIntersecting && hasMore) {
        loadMore()
      }
    },
    [hasMore, loadMore]
  )

  useEffect(() => {
    const element = loadingRef.current
    if (!element) return

    observerRef.current = new IntersectionObserver(handleObserver, {
      threshold: 0.1,
      rootMargin: `${threshold}px`
    })

    observerRef.current.observe(element)

    return () => {
      if (observerRef.current) {
        observerRef.current.disconnect()
      }
    }
  }, [handleObserver, threshold])

  return (
    <div className={cn('w-full', className)}>
      {children}
      
      <div ref={loadingRef} className="flex justify-center py-4">
        {hasMore ? (
          loadingComponent || (
            <div className="flex items-center gap-2 text-neutral-500 dark:text-neutral-400">
              <div className="animate-spin rounded-full h-4 w-4 border-b-2 border-primary"></div>
              <span>加载中...</span>
            </div>
          )
        ) : (
          endMessage || (
            <div className="text-neutral-500 dark:text-neutral-400 text-sm">
              没有更多内容了
            </div>
          )
        )}
      </div>
    </div>
  )
}

export default InfiniteScroll
