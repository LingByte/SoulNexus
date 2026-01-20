// DatePicker.tsx
import { useState, useRef, useEffect } from 'react'
import { Calendar, ChevronLeft, ChevronRight } from 'lucide-react'
import { cn } from '@/utils/cn.ts'

interface DatePickerProps {
  value?: Date | null;              // 允许 null
  onChange: (date: Date | null) => void
  placeholder?: string
  disabled?: boolean
  className?: string
  label?: string
  error?: string
  minDate?: Date
  maxDate?: Date
}

const DatePicker = ({
                      value,
                      onChange,
                      placeholder = '选择日期',
                      disabled = false,
                      className,
                      label,
                      error,
                      minDate,
                      maxDate
                    }: DatePickerProps) => {
  const [isOpen, setIsOpen] = useState(false)
  // value 可能为 null，用 ?? 提供兜底
  const [currentMonth, setCurrentMonth] = useState<Date>(value ?? new Date())
  const datePickerRef = useRef<HTMLDivElement>(null)

  const today = new Date()
  const currentYear = currentMonth.getFullYear()
  const currentMonthIndex = currentMonth.getMonth()

  const monthNames = [
    '一月','二月','三月','四月','五月','六月',
    '七月','八月','九月','十月','十一月','十二月'
  ]
  const dayNames = ['日','一','二','三','四','五','六']

  const getDaysInMonth = (year: number, month: number) =>
      new Date(year, month + 1, 0).getDate()

  const getFirstDayOfMonth = (year: number, month: number) =>
      new Date(year, month, 1).getDay()

  const isDateDisabled = (date: Date) => {
    if (minDate && date < minDate) return true
    return !!(maxDate && date > maxDate);
  }

  const isDateSelected = (date: Date) => {
    if (!value) return false               // ✅ 支持 null
    return date.toDateString() === value.toDateString()
  }

  const isToday = (date: Date) =>
      date.toDateString() === today.toDateString()

  const handleDateSelect = (date: Date) => {
    if (!isDateDisabled(date)) {
      onChange(date)
      setIsOpen(false)
    }
  }

  const navigateMonth = (direction: 'prev' | 'next') => {
    setCurrentMonth(prev => {
      const newMonth = new Date(prev)
      newMonth.setMonth(prev.getMonth() + (direction === 'prev' ? -1 : 1))
      return newMonth
    })
  }

  const renderCalendar = () => {
    const daysInMonth = getDaysInMonth(currentYear, currentMonthIndex)
    const firstDay = getFirstDayOfMonth(currentYear, currentMonthIndex)
    const days: JSX.Element[] = []

    for (let i = 0; i < firstDay; i++) {
      days.push(<div key={`empty-${i}`} className="h-8" />)
    }

    for (let day = 1; day <= daysInMonth; day++) {
      const date = new Date(currentYear, currentMonthIndex, day)
      const disabled = isDateDisabled(date)
      const selected = isDateSelected(date)
      const isTodayDate = isToday(date)

      days.push(
          <button
              key={day}
              type="button"
              onClick={() => handleDateSelect(date)}
              disabled={disabled}
              className={cn(
                  'h-8 w-8 flex items-center justify-center text-sm rounded-full transition-colors',
                  disabled && 'opacity-50 cursor-not-allowed',
                  selected && 'bg-primary text-primary-foreground',
                  !selected && !disabled && 'hover:bg-neutral-100 dark:hover:bg-neutral-700',
                  isTodayDate && !selected && 'bg-neutral-100 dark:bg-neutral-700 font-medium'
              )}
          >
            {day}
          </button>
      )
    }

    return days
  }

  useEffect(() => {
    // ✅ 当外部 value 变化时，同步月份视图（选了新日期或被清空）
    if (value instanceof Date) {
      setCurrentMonth(value)
    }
  }, [value])

  useEffect(() => {
    const handleClickOutside = (event: MouseEvent) => {
      if (datePickerRef.current && !datePickerRef.current.contains(event.target as Node)) {
        setIsOpen(false)
      }
    }
    document.addEventListener('mousedown', handleClickOutside)
    return () => document.removeEventListener('mousedown', handleClickOutside)
  }, [])

  return (
      <div className={cn('relative', className)}>
        {label && (
            <label className="block text-sm font-medium text-neutral-700 dark:text-neutral-300 mb-2">
              {label}
            </label>
        )}

        <div ref={datePickerRef} className="relative">
          <button
              type="button"
              onClick={() => !disabled && setIsOpen(!isOpen)}
              disabled={disabled}
              className={cn(
                  'w-full px-3 py-2 text-left bg-white dark:bg-neutral-800 border border-neutral-300 dark:border-neutral-600 rounded-lg shadow-sm focus:outline-none focus:ring-2 focus:ring-primary focus:border-transparent transition-colors',
                  disabled && 'opacity-50 cursor-not-allowed',
                  error && 'border-red-500 focus:ring-red-500'
              )}
          >
            <div className="flex items-center justify-between">
            <span className={cn(
                'block truncate',
                !value && 'text-neutral-500 dark:text-neutral-400'
            )}>
              {value ? value.toLocaleDateString('zh-CN') : placeholder}
            </span>
              <Calendar className="w-4 h-4 text-neutral-400" />
            </div>
          </button>

          {isOpen && (
              <div className="absolute z-50 mt-1 bg-white dark:bg-neutral-800 border border-neutral-300 dark:border-neutral-600 rounded-lg shadow-lg p-4 w-80">
                {/* 月份导航 */}
                <div className="flex items-center justify-between mb-4">
                  <button
                      type="button"
                      onClick={() => navigateMonth('prev')}
                      className="p-1 hover:bg-neutral-100 dark:hover:bg-neutral-700 rounded transition-colors"
                  >
                    <ChevronLeft className="w-4 h-4" />
                  </button>
                  <h3 className="font-medium text-neutral-900 dark:text-white">
                    {currentYear}年 {monthNames[currentMonthIndex]}
                  </h3>
                  <button
                      type="button"
                      onClick={() => navigateMonth('next')}
                      className="p-1 hover:bg-neutral-100 dark:hover:bg-neutral-700 rounded transition-colors"
                  >
                    <ChevronRight className="w-4 h-4" />
                  </button>
                </div>

                {/* 星期标题 */}
                <div className="grid grid-cols-7 gap-1 mb-2">
                  {dayNames.map(day => (
                      <div key={day} className="h-8 flex items-center justify-center text-xs font-medium text-neutral-500 dark:text-neutral-400">
                        {day}
                      </div>
                  ))}
                </div>

                {/* 日期网格 */}
                <div className="grid grid-cols-7 gap-1">
                  {renderCalendar()}
                </div>

                {/* 操作按钮 */}
                <div className="flex justify-end gap-2 mt-4 pt-4 border-t border-neutral-200 dark:border-neutral-700">
                  <button
                      type="button"
                      onClick={() => { onChange(null); setIsOpen(false) }} // ✅ 允许 null
                      className="px-3 py-1 text-sm text-neutral-600 dark:text-neutral-400 hover:text-neutral-900 dark:hover:text-neutral-100 transition-colors"
                  >
                    清除
                  </button>
                  <button
                      type="button"
                      onClick={() => setIsOpen(false)}
                      className="px-3 py-1 text-sm bg-primary text-primary-foreground rounded hover:bg-primary/90 transition-colors"
                  >
                    确定
                  </button>
                </div>
              </div>
          )}
        </div>

        {error && (
            <p className="mt-1 text-sm text-red-600 dark:text-red-400">{error}</p>
        )}
      </div>
  )
}

export default DatePicker
