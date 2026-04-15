import { ChevronLeft, ChevronRight, MoreHorizontal } from 'lucide-react'
import { cn } from '@/utils/cn'
import Button from './Button'

interface PaginationProps {
  currentPage: number
  totalPages: number
  totalItems: number
  pageSize: number
  onPageChange: (page: number) => void
  showSizeChanger?: boolean
  onPageSizeChange?: (size: number) => void
  pageSizeOptions?: number[]
  showQuickJumper?: boolean
  showTotal?: boolean
  className?: string
}

const Pagination = ({
  currentPage,
  totalPages,
  totalItems,
  pageSize,
  onPageChange,
  showSizeChanger = false,
  onPageSizeChange,
  pageSizeOptions = [10, 20, 50, 100],
  showQuickJumper = false,
  showTotal = true,
  className
}: PaginationProps) => {
  const startItem = (currentPage - 1) * pageSize + 1
  const endItem = Math.min(currentPage * pageSize, totalItems)

  const getPageNumbers = () => {
    const pages: (number | string)[] = []
    const maxVisible = 7

    if (totalPages <= maxVisible) {
      for (let i = 1; i <= totalPages; i++) {
        pages.push(i)
      }
    } else {
      pages.push(1)

      if (currentPage > 4) {
        pages.push('...')
      }

      const start = Math.max(2, currentPage - 1)
      const end = Math.min(totalPages - 1, currentPage + 1)

      for (let i = start; i <= end; i++) {
        pages.push(i)
      }

      if (currentPage < totalPages - 3) {
        pages.push('...')
      }

      if (totalPages > 1) {
        pages.push(totalPages)
      }
    }

    return pages
  }

  if (totalPages <= 1) return null

  return (
    <div className={cn('flex items-center justify-between gap-4', className)}>
      {showTotal && (
        <div className="text-sm text-slate-600 dark:text-slate-400">
          显示 {startItem} 到 {endItem} 条，共 {totalItems} 条
        </div>
      )}

      <div className="flex items-center gap-2">
        {showSizeChanger && onPageSizeChange && (
          <div className="flex items-center gap-2">
            <span className="text-sm text-slate-600 dark:text-slate-400">每页</span>
            <select
              value={pageSize}
              onChange={(e) => onPageSizeChange(Number(e.target.value))}
              className="px-2 py-1 text-sm border border-slate-300 dark:border-slate-600 rounded bg-white dark:bg-slate-800 text-slate-900 dark:text-slate-100"
            >
              {pageSizeOptions.map(size => (
                <option key={size} value={size}>{size} 条/页</option>
              ))}
            </select>
          </div>
        )}

        {showQuickJumper && (
          <div className="flex items-center gap-2">
            <span className="text-sm text-slate-600 dark:text-slate-400">跳至</span>
            <input
              type="number"
              min={1}
              max={totalPages}
              className="w-16 px-2 py-1 text-sm border border-slate-300 dark:border-slate-600 rounded bg-white dark:bg-slate-800 text-slate-900 dark:text-slate-100"
              onKeyPress={(e) => {
                if (e.key === 'Enter') {
                  const page = Number((e.target as HTMLInputElement).value)
                  if (page >= 1 && page <= totalPages) {
                    onPageChange(page)
                  }
                }
              }}
            />
            <span className="text-sm text-slate-600 dark:text-slate-400">页</span>
          </div>
        )}

        <div className="flex items-center gap-1">
          <Button
            variant="outline"
            size="sm"
            onClick={() => onPageChange(currentPage - 1)}
            disabled={currentPage === 1}
            className="px-2"
          >
            <ChevronLeft className="w-4 h-4" />
          </Button>

          {getPageNumbers().map((page, index) => (
            <Button
              key={index}
              variant={page === currentPage ? 'default' : 'outline'}
              size="sm"
              onClick={() => typeof page === 'number' && onPageChange(page)}
              disabled={page === '...'}
              className="min-w-[32px] px-2"
            >
              {page === '...' ? <MoreHorizontal className="w-4 h-4" /> : page}
            </Button>
          ))}

          <Button
            variant="outline"
            size="sm"
            onClick={() => onPageChange(currentPage + 1)}
            disabled={currentPage === totalPages}
            className="px-2"
          >
            <ChevronRight className="w-4 h-4" />
          </Button>
        </div>
      </div>
    </div>
  )
}

export default Pagination