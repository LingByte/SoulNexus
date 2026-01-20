import React, { useState, useEffect, useCallback, useMemo, Suspense, lazy } from 'react'
import { motion, AnimatePresence } from 'framer-motion'
import Card from '../UI/Card'
import Button from '../UI/Button'
import Badge from '../UI/Badge'
import IconText from '../UI/IconText'
import {
  Zap,
  Database,
  Image,
  Code,
  Network,
  Monitor,
  Cpu,
  MemoryStick,
  HardDrive,
  Wifi,
  Clock,
  TrendingUp,
  CheckCircle
} from 'lucide-react'

// 懒加载组件
const HeavyChart = lazy(() => import('./HeavyChart'))
const DataTable = lazy(() => import('./DataTable'))

interface PerformanceMetrics {
  bundleSize: number
  loadTime: number
  renderTime: number
  memoryUsage: number
  networkRequests: number
  cacheHitRate: number
}

const PerformanceOptimizer = () => {
  const [metrics, setMetrics] = useState<PerformanceMetrics>({
    bundleSize: 0,
    loadTime: 0,
    renderTime: 0,
    memoryUsage: 0,
    networkRequests: 0,
    cacheHitRate: 0
  })
  
  const [isOptimizing, setIsOptimizing] = useState(false)
  const [optimizationResults, setOptimizationResults] = useState<string[]>([])

  // 模拟性能监控
  const measurePerformance = useCallback(() => {
    const startTime = performance.now()
    
    // 模拟性能测量
    setTimeout(() => {
      const endTime = performance.now()
      const renderTime = endTime - startTime
      
      setMetrics(prev => ({
        ...prev,
        bundleSize: Math.floor(Math.random() * 500) + 200, // KB
        loadTime: Math.floor(Math.random() * 2000) + 500, // ms
        renderTime: Math.floor(renderTime),
        memoryUsage: Math.floor(Math.random() * 50) + 10, // MB
        networkRequests: Math.floor(Math.random() * 20) + 5,
        cacheHitRate: Math.floor(Math.random() * 30) + 70 // %
      }))
    }, 100)
  }, [])

  // 性能优化建议
  const optimizationSuggestions = useMemo(() => [
    {
      id: 'code-splitting',
      title: '代码分割',
      description: '将大型组件拆分为更小的块，按需加载',
      impact: 'high',
      effort: 'medium',
      icon: <Code className="w-5 h-5" />
    },
    {
      id: 'image-optimization',
      title: '图片优化',
      description: '使用 WebP 格式和懒加载减少图片大小',
      impact: 'high',
      effort: 'low',
      icon: <Image className="w-5 h-5" />
    },
    {
      id: 'bundle-optimization',
      title: '包优化',
      description: '移除未使用的代码，压缩 JavaScript',
      impact: 'medium',
      effort: 'low',
      icon: <Database className="w-5 h-5" />
    },
    {
      id: 'caching',
      title: '缓存策略',
      description: '实现 Service Worker 和 HTTP 缓存',
      impact: 'high',
      effort: 'medium',
      icon: <HardDrive className="w-5 h-5" />
    },
    {
      id: 'network-optimization',
      title: '网络优化',
      description: '使用 CDN 和 HTTP/2 加速资源加载',
      impact: 'medium',
      effort: 'high',
      icon: <Network className="w-5 h-5" />
    }
  ], [])

  // 执行优化
  const runOptimization = useCallback(async () => {
    setIsOptimizing(true)
    setOptimizationResults([])
    
    const results: string[] = []
    
    // 模拟优化过程
    for (let i = 0; i < optimizationSuggestions.length; i++) {
      await new Promise(resolve => setTimeout(resolve, 1000))
      results.push(`✅ ${optimizationSuggestions[i].title} 优化完成`)
      setOptimizationResults([...results])
    }
    
    setIsOptimizing(false)
  }, [optimizationSuggestions])

  // 性能指标组件
  const MetricCard = ({ title, value, unit, icon, color = 'primary' }: {
    title: string
    value: number
    unit: string
    icon: React.ReactNode
    color?: 'primary' | 'success' | 'warning' | 'danger'
  }) => {
    const colorClasses = {
      primary: 'text-blue-600 bg-blue-50',
      success: 'text-green-600 bg-green-50',
      warning: 'text-yellow-600 bg-yellow-50',
      danger: 'text-red-600 bg-red-50'
    }

    return (
      <motion.div
        initial={{ opacity: 0, y: 20 }}
        animate={{ opacity: 1, y: 0 }}
        className="p-4 rounded-lg border bg-white"
      >
        <div className="flex items-center justify-between mb-2">
          <div className={`p-2 rounded-lg ${colorClasses[color]}`}>
            {icon}
          </div>
          <Badge variant={color === 'danger' ? 'error' : color === 'warning' ? 'warning' : 'secondary'}>
            {value}{unit}
          </Badge>
        </div>
        <h3 className="font-medium text-gray-900">{title}</h3>
      </motion.div>
    )
  }

  useEffect(() => {
    measurePerformance()
    const interval = setInterval(measurePerformance, 5000)
    return () => clearInterval(interval)
  }, [measurePerformance])

  return (
    <div className="space-y-6">
      {/* 性能指标 */}
      <Card title="实时性能指标" variant="outlined">
        <div className="grid grid-cols-2 md:grid-cols-3 lg:grid-cols-6 gap-4">
          <MetricCard
            title="包大小"
            value={metrics.bundleSize}
            unit="KB"
            icon={<Database className="w-5 h-5" />}
            color={metrics.bundleSize > 300 ? 'warning' : 'success'}
          />
          <MetricCard
            title="加载时间"
            value={metrics.loadTime}
            unit="ms"
            icon={<Clock className="w-5 h-5" />}
            color={metrics.loadTime > 1500 ? 'warning' : 'success'}
          />
          <MetricCard
            title="渲染时间"
            value={metrics.renderTime}
            unit="ms"
            icon={<Cpu className="w-5 h-5" />}
            color={metrics.renderTime > 16 ? 'warning' : 'success'}
          />
          <MetricCard
            title="内存使用"
            value={metrics.memoryUsage}
            unit="MB"
            icon={<MemoryStick className="w-5 h-5" />}
            color={metrics.memoryUsage > 30 ? 'warning' : 'success'}
          />
          <MetricCard
            title="网络请求"
            value={metrics.networkRequests}
            unit="个"
            icon={<Wifi className="w-5 h-5" />}
            color={metrics.networkRequests > 15 ? 'warning' : 'success'}
          />
          <MetricCard
            title="缓存命中率"
            value={metrics.cacheHitRate}
            unit="%"
            icon={<TrendingUp className="w-5 h-5" />}
            color={metrics.cacheHitRate < 80 ? 'warning' : 'success'}
          />
        </div>
      </Card>

      {/* 优化建议 */}
      <Card title="性能优化建议" variant="outlined">
        <div className="space-y-4">
          {optimizationSuggestions.map((suggestion, index) => (
            <motion.div
              key={suggestion.id}
              initial={{ opacity: 0, x: -20 }}
              animate={{ opacity: 1, x: 0 }}
              transition={{ delay: index * 0.1 }}
              className="flex items-center justify-between p-4 rounded-lg border bg-gray-50"
            >
              <div className="flex items-center space-x-3">
                <div className="p-2 rounded-lg bg-blue-100 text-blue-600">
                  {suggestion.icon}
                </div>
                <div>
                  <h3 className="font-medium text-gray-900">{suggestion.title}</h3>
                  <p className="text-sm text-gray-600">{suggestion.description}</p>
                </div>
              </div>
              <div className="flex items-center space-x-2">
                <Badge
                  variant={suggestion.impact === 'danger' ? 'error' : suggestion.impact === 'medium' ? 'warning' : 'secondary'}
                >
                  {suggestion.impact === 'high' ? '高影响' : suggestion.impact === 'medium' ? '中影响' : '低影响'}
                </Badge>
                <Badge variant="outline">
                  {suggestion.effort === 'high' ? '高难度' : suggestion.effort === 'medium' ? '中难度' : '低难度'}
                </Badge>
              </div>
            </motion.div>
          ))}
        </div>
      </Card>

      {/* 优化执行 */}
      <Card title="执行优化" variant="outlined">
        <div className="space-y-4">
          <div className="flex items-center justify-between">
            <div>
              <h3 className="font-medium text-gray-900">自动优化</h3>
              <p className="text-sm text-gray-600">一键执行所有推荐的性能优化</p>
            </div>
            <Button
              onClick={runOptimization}
              disabled={isOptimizing}
              loading={isOptimizing}
              variant="primary"
            >
              {isOptimizing ? '优化中...' : '开始优化'}
            </Button>
          </div>

          <AnimatePresence>
            {optimizationResults.length > 0 && (
              <motion.div
                initial={{ opacity: 0, height: 0 }}
                animate={{ opacity: 1, height: 'auto' }}
                exit={{ opacity: 0, height: 0 }}
                className="space-y-2"
              >
                <h4 className="font-medium text-gray-900">优化进度</h4>
                {optimizationResults.map((result, index) => (
                  <motion.div
                    key={index}
                    initial={{ opacity: 0, x: -20 }}
                    animate={{ opacity: 1, x: 0 }}
                    transition={{ delay: index * 0.1 }}
                    className="flex items-center space-x-2 text-sm text-green-600"
                  >
                    <CheckCircle className="w-4 h-4" />
                    <span>{result}</span>
                  </motion.div>
                ))}
              </motion.div>
            )}
          </AnimatePresence>
        </div>
      </Card>

      {/* 懒加载演示 */}
      <Card title="懒加载演示" variant="outlined">
        <div className="space-y-4">
          <p className="text-gray-600">
            下面的组件使用懒加载，只有在需要时才会加载，减少初始包大小。
          </p>
          
          <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
            <div className="p-4 border rounded-lg">
              <h4 className="font-medium mb-2">图表组件</h4>
              <Suspense fallback={
                <div className="flex items-center justify-center h-32 bg-gray-100 rounded">
                  <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary"></div>
                </div>
              }>
                <HeavyChart />
              </Suspense>
            </div>
            
            <div className="p-4 border rounded-lg">
              <h4 className="font-medium mb-2">数据表格</h4>
              <Suspense fallback={
                <div className="flex items-center justify-center h-32 bg-gray-100 rounded">
                  <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary"></div>
                </div>
              }>
                <DataTable />
              </Suspense>
            </div>
          </div>
        </div>
      </Card>

      {/* 性能提示 */}
      <Card title="性能优化提示" variant="outlined">
        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
          <div className="space-y-3">
            <IconText
              icon={<Zap className="w-5 h-5" />}
              variant="success"
              size="sm"
            >
              使用 React.memo 避免不必要的重渲染
            </IconText>
            <IconText
              icon={<Image className="w-5 h-5" />}
              variant="success"
              size="sm"
            >
              实现图片懒加载和 WebP 格式
            </IconText>
            <IconText
              icon={<Code className="w-5 h-5" />}
              variant="success"
              size="sm"
            >
              使用代码分割减少初始包大小
            </IconText>
          </div>
          <div className="space-y-3">
            <IconText
              icon={<Database className="w-5 h-5" />}
              variant="warning"
              size="sm"
            >
              避免在渲染函数中创建新对象
            </IconText>
            <IconText
              icon={<Network className="w-5 h-5" />}
              variant="warning"
              size="sm"
            >
              使用防抖和节流优化事件处理
            </IconText>
            <IconText
              icon={<Monitor className="w-5 h-5" />}
              variant="warning"
              size="sm"
            >
              监控 Core Web Vitals 指标
            </IconText>
          </div>
        </div>
      </Card>
    </div>
  )
}

export default PerformanceOptimizer
