import { useState, useEffect, useRef } from 'react'
import { ChevronLeft, ChevronRight, RotateCcw, RotateCw, BookOpen, Bookmark } from 'lucide-react'
import { motion, AnimatePresence } from 'framer-motion'
import { cn } from '@/utils/cn.ts'
import Button from '../UI/Button'

interface PageTurnerProps {
  currentPage: number
  totalPages: number
  onPageChange: (page: number) => void
  onBookmark?: (page: number) => void
  bookmarks?: number[]
  className?: string
  showPageNumbers?: boolean
  showProgress?: boolean
  autoSave?: boolean
  onAutoSave?: (page: number) => void
}

const PageTurner = ({
  currentPage,
  totalPages,
  onPageChange,
  onBookmark,
  bookmarks = [],
  className,
  showPageNumbers = true,
  showProgress = true,
  autoSave = false,
  onAutoSave
}: PageTurnerProps) => {
  const [isAnimating, setIsAnimating] = useState(false)
  const [direction, setDirection] = useState<'left' | 'right' | null>(null)
  // @ts-ignore
  const autoSaveRef = useRef<NodeJS.Timeout>()

  const progress = (currentPage / totalPages) * 100

  const handlePageChange = (newPage: number, dir: 'left' | 'right') => {
    if (newPage < 1 || newPage > totalPages || isAnimating) return
    
    setDirection(dir)
    setIsAnimating(true)
    onPageChange(newPage)
    
    // 自动保存
    if (autoSave && onAutoSave) {
      clearTimeout(autoSaveRef.current)
      autoSaveRef.current = setTimeout(() => {
        onAutoSave(newPage)
      }, 1000)
    }
    
    setTimeout(() => {
      setIsAnimating(false)
      setDirection(null)
    }, 300)
  }

  const goToPrevious = () => handlePageChange(currentPage - 1, 'left')
  const goToNext = () => handlePageChange(currentPage + 1, 'right')
  const goToFirst = () => handlePageChange(1, 'left')
  const goToLast = () => handlePageChange(totalPages, 'right')

  const handleBookmark = () => {
    if (onBookmark) {
      onBookmark(currentPage)
    }
  }

  const isBookmarked = bookmarks.includes(currentPage)

  // 键盘快捷键
  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.target instanceof HTMLInputElement || e.target instanceof HTMLTextAreaElement) return
      
      switch (e.key) {
        case 'ArrowLeft':
          e.preventDefault()
          goToPrevious()
          break
        case 'ArrowRight':
          e.preventDefault()
          goToNext()
          break
        case 'Home':
          e.preventDefault()
          goToFirst()
          break
        case 'End':
          e.preventDefault()
          goToLast()
          break
        case 'b':
          if (e.ctrlKey || e.metaKey) {
            e.preventDefault()
            handleBookmark()
          }
          break
      }
    }

    window.addEventListener('keydown', handleKeyDown)
    return () => window.removeEventListener('keydown', handleKeyDown)
  }, [currentPage, totalPages])

  return (
    <div className={cn('flex items-center justify-between p-4 bg-white dark:bg-neutral-800 border-t border-neutral-200 dark:border-neutral-700', className)}>
      {/* 左侧控制 */}
      <div className="flex items-center gap-2">
        <Button
          variant="outline"
          size="sm"
          onClick={goToFirst}
          disabled={currentPage === 1 || isAnimating}
          className="flex items-center gap-1"
        >
          <RotateCcw className="w-4 h-4" />
          首页
        </Button>
        
        <Button
          variant="outline"
          size="sm"
          onClick={goToPrevious}
          disabled={currentPage === 1 || isAnimating}
          className="flex items-center gap-1"
        >
          <ChevronLeft className="w-4 h-4" />
          上一页
        </Button>
      </div>

      {/* 中间信息 */}
      <div className="flex flex-col items-center gap-2">
        {showPageNumbers && (
          <div className="flex items-center gap-2">
            <span className="text-sm text-neutral-600 dark:text-neutral-400">
              第 {currentPage} 页 / 共 {totalPages} 页
            </span>
            {isBookmarked && (
              <Bookmark className="w-4 h-4 text-primary fill-current" />
            )}
          </div>
        )}

        {showProgress && (
          <div className="w-32 h-1 bg-neutral-200 dark:bg-neutral-700 rounded-full overflow-hidden">
            <motion.div
              className="h-full bg-primary rounded-full"
              initial={{ width: 0 }}
              animate={{ width: `${progress}%` }}
              transition={{ duration: 0.3 }}
            />
          </div>
        )}

        {/* 页面输入 */}
        <div className="flex items-center gap-2">
          <input
            type="number"
            min="1"
            max={totalPages}
            value={currentPage}
            onChange={(e) => {
              const page = parseInt(e.target.value)
              if (page >= 1 && page <= totalPages) {
                handlePageChange(page, page > currentPage ? 'right' : 'left')
              }
            }}
            className="w-16 px-2 py-1 text-sm text-center border border-neutral-300 dark:border-neutral-600 rounded bg-white dark:bg-neutral-800 text-neutral-900 dark:text-white"
            disabled={isAnimating}
          />
          <span className="text-xs text-neutral-500 dark:text-neutral-400">
            / {totalPages}
          </span>
        </div>
      </div>

      {/* 右侧控制 */}
      <div className="flex items-center gap-2">
        <Button
          variant="outline"
          size="sm"
          onClick={handleBookmark}
          className={cn(
            "flex items-center gap-1",
            isBookmarked && "bg-primary text-primary-foreground"
          )}
        >
          <Bookmark className="w-4 h-4" />
          {isBookmarked ? '已收藏' : '收藏'}
        </Button>
        
        <Button
          variant="outline"
          size="sm"
          onClick={goToNext}
          disabled={currentPage === totalPages || isAnimating}
          className="flex items-center gap-1"
        >
          下一页
          <ChevronRight className="w-4 h-4" />
        </Button>
        
        <Button
          variant="outline"
          size="sm"
          onClick={goToLast}
          disabled={currentPage === totalPages || isAnimating}
          className="flex items-center gap-1"
        >
          末页
          <RotateCw className="w-4 h-4" />
        </Button>
      </div>

      {/* 翻页动画 */}
      <AnimatePresence>
        {isAnimating && (
          <motion.div
            className="absolute inset-0 pointer-events-none"
            initial={{ opacity: 0, x: direction === 'left' ? -20 : 20 }}
            animate={{ opacity: 1, x: 0 }}
            exit={{ opacity: 0, x: direction === 'left' ? 20 : -20 }}
            transition={{ duration: 0.3 }}
          >
            <div className="w-full h-full bg-primary/10 flex items-center justify-center">
              <BookOpen className="w-8 h-8 text-primary animate-pulse" />
            </div>
          </motion.div>
        )}
      </AnimatePresence>
    </div>
  )
}

export default PageTurner
