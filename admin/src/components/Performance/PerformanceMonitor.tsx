import { useEffect, useState, useRef } from 'react'
import { motion } from 'framer-motion'
import { cn } from '@/utils/cn.ts'

interface PerformanceMetrics {
  fcp: number // First Contentful Paint
  lcp: number // Largest Contentful Paint
  fid: number // First Input Delay
  cls: number // Cumulative Layout Shift
  ttfb: number // Time to First Byte
  fmp: number // First Meaningful Paint
  tti: number // Time to Interactive
}

interface PerformanceMonitorProps {
  onMetricsUpdate?: (metrics: PerformanceMetrics) => void
  className?: string
  showMetrics?: boolean
  threshold?: {
    fcp: number
    lcp: number
    fid: number
    cls: number
  }
}

const PerformanceMonitor = ({
  onMetricsUpdate,
  className = "",
  showMetrics = false,
}: PerformanceMonitorProps) => {
  const [metrics, setMetrics] = useState<PerformanceMetrics>({
    fcp: 0,
    lcp: 0,
    fid: 0,
    cls: 0,
    ttfb: 0,
    fmp: 0,
    tti: 0
  })
  const [isVisible, setIsVisible] = useState(showMetrics)
  const observerRef = useRef<PerformanceObserver | null>(null)

  // 获取性能指标
  const getPerformanceMetrics = () => {
    const navigation = performance.getEntriesByType('navigation')[0] as PerformanceNavigationTiming
    const paintEntries = performance.getEntriesByType('paint')
    
    // FCP (First Contentful Paint)
    const fcpEntry = paintEntries.find(entry => entry.name === 'first-contentful-paint')
    const fcp = fcpEntry ? fcpEntry.startTime : 0

    // LCP (Largest Contentful Paint)
    const lcpEntries = performance.getEntriesByType('largest-contentful-paint')
    const lcp = lcpEntries.length > 0 ? lcpEntries[lcpEntries.length - 1].startTime : 0

    // TTFB (Time to First Byte)
    const ttfb = navigation ? navigation.responseStart - navigation.requestStart : 0

    // FMP (First Meaningful Paint) - 近似值
    const fmp = fcp * 1.2

    // TTI (Time to Interactive) - 近似值
    const tti = lcp + 1000

    return {
      fcp,
      lcp,
      fid: 0, // 需要用户交互才能测量
      cls: 0, // 需要布局变化才能测量
      ttfb,
      fmp,
      tti
    }
  }

  // 设置性能观察器
  useEffect(() => {
    if (typeof window === 'undefined') return

    // 观察 LCP
    if ('PerformanceObserver' in window) {
      try {
        observerRef.current = new PerformanceObserver((list) => {
          const entries = list.getEntries()
          const lastEntry = entries[entries.length - 1] as PerformanceEntry
          
          setMetrics(prev => ({
            ...prev,
            lcp: lastEntry.startTime
          }))
        })
        
        observerRef.current.observe({ entryTypes: ['largest-contentful-paint'] })
      } catch (error) {
        console.warn('PerformanceObserver not supported:', error)
      }
    }

    // 初始指标
    const initialMetrics = getPerformanceMetrics()
    setMetrics(initialMetrics)
    onMetricsUpdate?.(initialMetrics)

    // 定期更新指标
    const interval = setInterval(() => {
      const currentMetrics = getPerformanceMetrics()
      setMetrics(currentMetrics)
      onMetricsUpdate?.(currentMetrics)
    }, 1000)

    return () => {
      if (observerRef.current) {
        observerRef.current.disconnect()
      }
      clearInterval(interval)
    }
  }, [onMetricsUpdate])

  // 获取性能等级
  const getPerformanceGrade = (metric: keyof PerformanceMetrics, value: number) => {
    const thresholds = {
      fcp: { good: 1800, poor: 3000 },
      lcp: { good: 2500, poor: 4000 },
      fid: { good: 100, poor: 300 },
      cls: { good: 0.1, poor: 0.25 },
      ttfb: { good: 800, poor: 1800 },
      fmp: { good: 2000, poor: 3000 },
      tti: { good: 3800, poor: 7300 }
    }

    const threshold = thresholds[metric]
    if (!threshold) return 'unknown'

    if (value <= threshold.good) return 'good'
    if (value <= threshold.poor) return 'needs-improvement'
    return 'poor'
  }

  // 获取性能等级颜色
  const getGradeColor = (grade: string) => {
    switch (grade) {
      case 'good': return 'text-green-600'
      case 'needs-improvement': return 'text-yellow-600'
      case 'poor': return 'text-red-600'
      default: return 'text-gray-600'
    }
  }

  // 格式化数值
  const formatValue = (value: number, unit: string = 'ms') => {
    if (value === 0) return 'N/A'
    return `${Math.round(value)}${unit}`
  }

  if (!isVisible) return null

  return (
    <motion.div
      className={cn(
        'fixed bottom-4 right-4 bg-white border border-gray-200 rounded-lg shadow-lg p-4 z-50 max-w-sm',
        className
      )}
      initial={{ opacity: 0, y: 20 }}
      animate={{ opacity: 1, y: 0 }}
      transition={{ duration: 0.3 }}
    >
      <div className="flex items-center justify-between mb-3">
        <h3 className="text-sm font-semibold text-gray-900">性能监控</h3>
        <button
          onClick={() => setIsVisible(false)}
          className="text-gray-400 hover:text-gray-600"
        >
          ✕
        </button>
      </div>

      <div className="space-y-2 text-xs">
        {Object.entries(metrics).map(([key, value]) => {
          const grade = getPerformanceGrade(key as keyof PerformanceMetrics, value)
          const color = getGradeColor(grade)
          const unit = key === 'cls' ? '' : 'ms'
          
          return (
            <div key={key} className="flex justify-between items-center">
              <span className="text-gray-600 uppercase">{key}:</span>
              <span className={cn('font-mono', color)}>
                {formatValue(value, unit)}
              </span>
            </div>
          )
        })}
      </div>

      <div className="mt-3 pt-3 border-t border-gray-200">
        <div className="text-xs text-gray-500">
          <div>FCP: 首次内容绘制</div>
          <div>LCP: 最大内容绘制</div>
          <div>TTFB: 首字节时间</div>
        </div>
      </div>
    </motion.div>
  )
}

export default PerformanceMonitor
