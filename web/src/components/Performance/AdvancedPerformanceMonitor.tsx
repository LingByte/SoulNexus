import {useEffect, useState, useRef, useCallback} from 'react'
import {motion, AnimatePresence} from 'framer-motion'
import { BarChart3 } from 'lucide-react'
import {cn} from '@/utils/cn.ts'

interface PerformanceMetrics {
    fcp: number
    lcp: number
    fid: number
    cls: number
    ttfb: number
    fmp: number
    tti: number
    memory?: {
        usedJSHeapSize: number
        totalJSHeapSize: number
        jsHeapSizeLimit: number
    }
    network?: {
        effectiveType: string
        downlink: number
        rtt: number
    }
}

interface AdvancedPerformanceMonitorProps {
    onMetricsUpdate?: (metrics: PerformanceMetrics) => void
    className?: string
    showMetrics?: boolean
    showMemory?: boolean
    showNetwork?: boolean
    threshold?: {
        fcp: number
        lcp: number
        fid: number
        cls: number
    }
    autoHide?: boolean
    hideDelay?: number
}

const AdvancedPerformanceMonitor = ({
                                        onMetricsUpdate,
                                        className = "",
                                        showMetrics = false,
                                        showMemory = false,
                                        showNetwork = false,
                                        autoHide = true,
                                        hideDelay = 5000
                                    }: AdvancedPerformanceMonitorProps) => {
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
    const [isMinimized, setIsMinimized] = useState(false)
    const observerRef = useRef<PerformanceObserver | null>(null)
    const timeoutRef = useRef<NodeJS.Timeout | null>(null)

    // 获取内存信息
    const getMemoryInfo = useCallback((): PerformanceMetrics['memory'] => {
        if ('memory' in performance) {
            const memory = (performance as any).memory
            return {
                usedJSHeapSize: memory.usedJSHeapSize,
                totalJSHeapSize: memory.totalJSHeapSize,
                jsHeapSizeLimit: memory.jsHeapSizeLimit
            }
        }
        return undefined
    }, [])

    // 获取网络信息
    const getNetworkInfo = useCallback((): PerformanceMetrics['network'] => {
        if ('connection' in navigator) {
            const connection = (navigator as any).connection
            return {
                effectiveType: connection.effectiveType || 'unknown',
                downlink: connection.downlink || 0,
                rtt: connection.rtt || 0
            }
        }
        return undefined
    }, [])

    // 获取性能指标
    const getPerformanceMetrics = useCallback((): PerformanceMetrics => {
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
            tti,
            memory: showMemory ? getMemoryInfo() : undefined,
            network: showNetwork ? getNetworkInfo() : undefined
        }
    }, [showMemory, showNetwork, getMemoryInfo, getNetworkInfo])

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

                observerRef.current.observe({entryTypes: ['largest-contentful-paint']})
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
        }, 2000)

        return () => {
            if (observerRef.current) {
                observerRef.current.disconnect()
            }
            clearInterval(interval)
        }
    }, [onMetricsUpdate, getPerformanceMetrics])

    // 自动隐藏
    useEffect(() => {
        if (autoHide && isVisible) {
            if (timeoutRef.current) {
                clearTimeout(timeoutRef.current)
            }
            timeoutRef.current = setTimeout(() => {
                setIsVisible(false)
            }, hideDelay)
        }

        return () => {
            if (timeoutRef.current) {
                clearTimeout(timeoutRef.current)
            }
        }
    }, [autoHide, isVisible, hideDelay])

    // 在组件顶部添加类型定义
    type CoreWebVitalMetric = 'fcp' | 'lcp' | 'fid' | 'cls' | 'ttfb' | 'fmp' | 'tti';


    // 修改 getPerformanceGrade 函数
    const getPerformanceGrade = (metric: CoreWebVitalMetric, value: number) => {
        const thresholds = {
            fcp: {good: 1800, poor: 3000},
            lcp: {good: 2500, poor: 4000},
            fid: {good: 100, poor: 300},
            cls: {good: 0.1, poor: 0.25},
            ttfb: {good: 800, poor: 1800},
            fmp: {good: 2000, poor: 3000},
            tti: {good: 3800, poor: 7300}
        }

        const threshold = thresholds[metric]
        if (!threshold) return 'unknown'

        if (value <= threshold.good) return 'good'
        if (value <= threshold.poor) return 'needs-improvement'
        return 'poor'
    }

    // 获取性能等级颜色和图标
    const getGradeInfo = (grade: string) => {
        switch (grade) {
            case 'good':
                return {color: 'text-green-600', icon: '✅', bg: 'bg-green-50'}
            case 'needs-improvement':
                return {color: 'text-yellow-600', icon: '⚠️', bg: 'bg-yellow-50'}
            case 'poor':
                return {color: 'text-red-600', icon: '❌', bg: 'bg-red-50'}
            default:
                return {color: 'text-gray-600', icon: '❓', bg: 'bg-gray-50'}
        }
    }

    // 格式化数值
    const formatValue = (value: number, unit: string = 'ms') => {
        if (value === 0) return 'N/A'
        if (unit === 'bytes') {
            return `${(value / 1024 / 1024).toFixed(2)}MB`
        }
        return `${Math.round(value)}${unit}`
    }

    // 格式化内存使用率
    const formatMemoryUsage = (used: number, total: number) => {
        const percentage = (used / total) * 100
        return `${percentage.toFixed(1)}%`
    }

    if (!isVisible) return null

    return (
        <AnimatePresence>
            <motion.div
                className={cn(
                    'fixed bottom-4 right-4 bg-white border border-gray-200 rounded-xl shadow-2xl z-50 max-w-sm',
                    isMinimized ? 'w-12 h-12' : 'w-80',
                    className
                )}
                initial={{opacity: 0, y: 20, scale: 0.9}}
                animate={{opacity: 1, y: 0, scale: 1}}
                exit={{opacity: 0, y: 20, scale: 0.9}}
                transition={{duration: 0.3, ease: "easeOut"}}
            >
                {isMinimized ? (
                    <button
                        onClick={() => setIsMinimized(false)}
                        className="w-full h-full flex items-center justify-center text-gray-600 hover:text-gray-800 transition-colors"
                    >
                        <BarChart3 className="w-5 h-5 text-gray-600 hover:text-gray-800 transition-colors" />
                    </button>
                ) : (
                    <div className="p-4">
                        <div className="flex items-center justify-between mb-4">
                            <h3 className="text-sm font-bold text-gray-900 flex items-center gap-2">
                                性能监控
                            </h3>
                            <div className="flex items-center gap-1">
                                <button
                                    onClick={() => setIsMinimized(true)}
                                    className="text-gray-400 hover:text-gray-600 p-1"
                                    title="最小化"
                                >
                                    ➖
                                </button>
                                <button
                                    onClick={() => setIsVisible(false)}
                                    className="text-gray-400 hover:text-gray-600 p-1"
                                    title="关闭"
                                >
                                    ✕
                                </button>
                            </div>
                        </div>

                        <div className="space-y-3 text-xs">
                            {/* Core Web Vitals */}
                            <div className="space-y-2">
                                <h4 className="text-xs font-semibold text-gray-700 uppercase tracking-wide">Core Web Vitals</h4>
                                {
                                    Object.entries(metrics).map(([key, value]) => {
                                        // 排除 memory 和 network 属性
                                        if (key === 'memory' || key === 'network') return null;

                                        if (typeof value !== 'number') return null

                                        const grade = getPerformanceGrade(key as CoreWebVitalMetric, value)
                                        const {color, icon, bg} = getGradeInfo(grade)
                                        const unit = key === 'cls' ? '' : 'ms'

                                        return (
                                            <div key={key}
                                                 className={cn('flex justify-between items-center p-2 rounded-lg', bg)}>
                                                <span className="text-gray-600 uppercase font-medium">{key}:</span>
                                                <div className="flex items-center gap-2">
          <span className={cn('font-mono font-bold', color)}>
            {formatValue(value, unit)}
          </span>
                                                    <span className="text-lg">{icon}</span>
                                                </div>
                                            </div>
                                        )
                                    })
                                }
                            </div>

                            {/* 内存使用情况 */}
                            {showMemory && metrics.memory && (
                                <div className="space-y-2">
                                    <h4 className="text-xs font-semibold text-gray-700 uppercase tracking-wide">内存使用</h4>
                                    <div className="space-y-1">
                                        <div className="flex justify-between items-center p-2 bg-blue-50 rounded-lg">
                                            <span className="text-gray-600">已使用:</span>
                                            <span className="font-mono font-bold text-blue-600">
                        {formatValue(metrics.memory.usedJSHeapSize, 'bytes')}
                      </span>
                                        </div>
                                        <div className="flex justify-between items-center p-2 bg-blue-50 rounded-lg">
                                            <span className="text-gray-600">使用率:</span>
                                            <span className="font-mono font-bold text-blue-600">
                        {formatMemoryUsage(metrics.memory.usedJSHeapSize, metrics.memory.jsHeapSizeLimit)}
                      </span>
                                        </div>
                                    </div>
                                </div>
                            )}

                            {/* 网络信息 */}
                            {showNetwork && metrics.network && (
                                <div className="space-y-2">
                                    <h4 className="text-xs font-semibold text-gray-700 uppercase tracking-wide">网络状态</h4>
                                    <div className="space-y-1">
                                        <div className="flex justify-between items-center p-2 bg-purple-50 rounded-lg">
                                            <span className="text-gray-600">连接类型:</span>
                                            <span className="font-mono font-bold text-purple-600">
                        {metrics.network.effectiveType}
                      </span>
                                        </div>
                                        <div className="flex justify-between items-center p-2 bg-purple-50 rounded-lg">
                                            <span className="text-gray-600">下载速度:</span>
                                            <span className="font-mono font-bold text-purple-600">
                        {metrics.network.downlink}Mbps
                      </span>
                                        </div>
                                    </div>
                                </div>
                            )}
                        </div>
                    </div>
                )}
            </motion.div>
        </AnimatePresence>
    )
}

export default AdvancedPerformanceMonitor
