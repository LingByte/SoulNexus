import type { ReactNode } from 'react'
import { Pagination } from '@arco-design/web-react'
import { cn } from '@/utils/cn'
import { Loading } from './loading'
import { Empty } from './empty'

export interface DataListColumn<T> {
  key: string
  title?: string
  width?: number | string
  minWidth?: number
  align?: 'left' | 'center' | 'right'
  render?: (value: unknown, row: T, index: number) => ReactNode
  className?: string
}

export interface DataListProps<T extends Record<string, unknown>> {
  data: T[]
  columns: DataListColumn<T>[]
  loading?: boolean
  emptyText?: string
  rowKey: string | ((row: T) => string | number)
  className?: string
  rowClassName?: string | ((row: T, index: number) => string)
  pagination?: {
    current: number
    pageSize: number
    total: number
    onChange: (page: number) => void
  } | null
  renderRow?: (row: T, index: number) => ReactNode
  header?: ReactNode
  footer?: ReactNode
  size?: 'sm' | 'md' | 'lg'
  bordered?: boolean
  onRowClick?: (row: T, index: number) => void
  hideColumnHeader?: boolean
}

function getRowKey<T extends Record<string, unknown>>(row: T, rowKey: string | ((row: T) => string | number), index: number): string | number {
  if (typeof rowKey === 'function') return rowKey(row)
  return (row[rowKey] as string | number) ?? index
}

function getCellValue<T extends Record<string, unknown>>(row: T, key: string): unknown {
  return row[key]
}

function ColumnHeaderRow<T extends Record<string, unknown>>({ columns, size }: { columns: DataListColumn<T>[]; size: 'sm' | 'md' | 'lg' }) {
  const sizeClasses = { sm: 'px-3 py-2', md: 'px-4 py-3', lg: 'px-5 py-4' }
  return (
    <div className={cn('flex items-center gap-4 border-b border-neutral-100 bg-neutral-50/80', sizeClasses[size])}>
      {columns.map((col) => (
        <div
          key={col.key}
          className={cn(
            'min-w-0 shrink-0 overflow-hidden text-xs font-medium text-neutral-500',
            col.align === 'center' && 'text-center',
            col.align === 'right' && 'text-right',
            col.className,
          )}
          style={{ width: col.width, minWidth: col.minWidth, flex: col.width ? undefined : 1 }}
        >
          <span className="block truncate">{col.title || ''}</span>
        </div>
      ))}
    </div>
  )
}

export function DataList<T extends Record<string, unknown>>({
  data,
  columns,
  loading = false,
  emptyText = '暂无数据',
  rowKey,
  className,
  rowClassName,
  pagination,
  renderRow,
  header,
  footer,
  size = 'md',
  bordered = true,
  onRowClick,
  hideColumnHeader,
}: DataListProps<T>) {
  const sizeClasses = { sm: 'px-3 py-2', md: 'px-4 py-3', lg: 'px-5 py-4' }
  const hasTitles = columns.some((c) => c.title)
  const showHeader = hasTitles && !hideColumnHeader

  const content = loading ? (
    <div className="py-12"><Loading block /></div>
  ) : data.length === 0 ? (
    <Empty description={emptyText} className="py-10" imageClassName="h-20 w-20 mb-2" />
  ) : (
    <>
      {showHeader && <ColumnHeaderRow columns={columns} size={size} />}
      <div className="divide-y divide-neutral-100">
        {data.map((row, index) => {
          const key = getRowKey(row, rowKey, index)
          const rowCls = typeof rowClassName === 'function' ? rowClassName(row, index) : rowClassName
          if (renderRow) {
            return (
              <div key={key} className={cn('transition-colors', onRowClick && 'cursor-pointer hover:bg-neutral-50', rowCls)} onClick={() => onRowClick?.(row, index)}>
                {renderRow(row, index)}
              </div>
            )
          }
          return (
            <div key={key} className={cn('flex items-center gap-4', sizeClasses[size], 'transition-colors', onRowClick && 'cursor-pointer hover:bg-neutral-50', rowCls)} onClick={() => onRowClick?.(row, index)}>
              {columns.map((col) => (
                <div
                  key={col.key}
                  className={cn(
                    'min-w-0 shrink-0 overflow-hidden',
                    col.align === 'center' && 'text-center',
                    col.align === 'right' && 'text-right',
                    col.className,
                  )}
                  style={{ width: col.width, minWidth: col.minWidth, flex: col.width ? undefined : 1 }}
                >
                  {col.render
                    ? col.render(getCellValue(row, col.key), row, index)
                    : <span className="block truncate text-sm text-neutral-900">{String(getCellValue(row, col.key) ?? '—')}</span>
                  }
                </div>
              ))}
            </div>
          )
        })}
      </div>
    </>
  )

  return (
    <div className={cn('w-full min-w-0 overflow-hidden rounded-xl border border-border bg-card', className)}>
      {header && <div className="border-b border-neutral-100 px-4 py-3">{header}</div>}
      <div className="w-full overflow-x-auto">
        {bordered ? content : <div className="divide-y-0">{content}</div>}
      </div>
      {pagination && pagination.total > pagination.pageSize && (
        <div className="flex justify-end border-t border-neutral-100 px-4 py-3">
          <Pagination total={pagination.total} current={pagination.current} pageSize={pagination.pageSize} onChange={pagination.onChange} size="small" />
        </div>
      )}
      {footer && <div className="border-t border-neutral-100 px-4 py-3">{footer}</div>}
    </div>
  )
}

export default DataList
