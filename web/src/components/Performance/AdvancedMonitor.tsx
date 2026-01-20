import React, { useState, useEffect, useCallback, useMemo } from 'react'
import { motion, AnimatePresence } from 'framer-motion'
import Card from '../UI/Card'
import Button from '../UI/Button'
import Badge from '../UI/Badge'
import { 
  Activity, 
  TrendingUp, 
  TrendingDown, 
  AlertCircle,
  CheckCircle,
  Clock,
  Network,
  Cpu,
  MemoryStick,
} from 'lucide-react'

interface PerformanceData {
  timestamp: number
  fps: number
  memory: number
  cpu: number
  network: number
  renderTime: number
}

interface Alert {
  id: string
  type: 'warning' | 'error' | 'info'
  message: string
  timestamp: number
}

const AdvancedMonitor = () => {
  const [performanceData, setPerformanceData] = useState<PerformanceData[]>([])
  const [alerts, setAlerts] = useState<Alert[]>([])
  const [isMonitoring, setIsMonitoring] = useState(false)
  const [currentMetrics, setCurrentMetrics] = useState<PerformanceData | null>(null)

  // 生成模拟性能数据
  const generatePerformanceData = useCallback((): PerformanceData => {
    const baseFPS = 60
    const baseMemory = 50
    const baseCPU = 30
    const baseNetwork = 10
    const baseRenderTime = 16

    // 添加一些随机波动
    const fps = Math.max(30, baseFPS + (Math.random() - 0.5) * 20)
    const memory = Math.max(10, baseMemory + (Math.random() - 0.5) * 20)
    const cpu = Math.max(5, baseCPU + (Math.random() - 0.5) * 20)
    const network = Math.max(0, baseNetwork + (Math.random() - 0.5) * 10)
    const renderTime = Math.max(8, baseRenderTime + (Math.random() - 0.5) * 10)

    return {
      timestamp: Date.now(),
      fps: Math.round(fps),
      memory: Math.round(memory),
      cpu: Math.round(cpu),
      network: Math.round(network),
      renderTime: Math.round(renderTime)
    }
  }, [])

  // 检查性能阈值并生成警报
  const checkPerformanceThresholds = useCallback((data: PerformanceData) => {
    const newAlerts: Alert[] = []

    if (data.fps < 30) {
      newAlerts.push({
        id: `fps-${data.timestamp}`,
        type: 'error',
        message: `FPS 过低: ${data.fps} (建议 > 30)`,
        timestamp: data.timestamp
      })
    } else if (data.fps < 45) {
      newAlerts.push({
        id: `fps-warning-${data.timestamp}`,
        type: 'warning',
        message: `FPS 警告: ${data.fps} (建议 > 45)`,
        timestamp: data.timestamp
      })
    }

    if (data.memory > 80) {
      newAlerts.push({
        id: `memory-${data.timestamp}`,
        type: 'error',
        message: `内存使用过高: ${data.memory}% (建议 < 80%)`,
        timestamp: data.timestamp
      })
    } else if (data.memory > 60) {
      newAlerts.push({
        id: `memory-warning-${data.timestamp}`,
        type: 'warning',
        message: `内存使用警告: ${data.memory}% (建议 < 60%)`,
        timestamp: data.timestamp
      })
    }

    if (data.cpu > 70) {
      newAlerts.push({
        id: `cpu-${data.timestamp}`,
        type: 'error',
        message: `CPU 使用过高: ${data.cpu}% (建议 < 70%)`,
        timestamp: data.timestamp
      })
    }

    if (data.renderTime > 33) {
      newAlerts.push({
        id: `render-${data.timestamp}`,
        type: 'warning',
        message: `渲染时间过长: ${data.renderTime}ms (建议 < 33ms)`,
        timestamp: data.timestamp
      })
    }

    if (newAlerts.length > 0) {
      setAlerts(prev => [...prev, ...newAlerts].slice(-10)) // 只保留最近10个警报
    }
  }, [])

  // 开始/停止监控
  const toggleMonitoring = useCallback(() => {
    if (isMonitoring) {
      setIsMonitoring(false)
      setPerformanceData([])
      setAlerts([])
    } else {
      setIsMonitoring(true)
      setPerformanceData([])
      setAlerts([])
    }
  }, [isMonitoring])

  // 监控循环
  useEffect(() => {
    if (!isMonitoring) return

    const interval = setInterval(() => {
      const newData = generatePerformanceData()
      setCurrentMetrics(newData)
      setPerformanceData(prev => [...prev.slice(-29), newData]) // 保留最近30个数据点
      checkPerformanceThresholds(newData)
    }, 1000)

    return () => clearInterval(interval)
  }, [isMonitoring, generatePerformanceData, checkPerformanceThresholds])

  // 计算性能趋势
  const performanceTrends = useMemo(() => {
    if (performanceData.length < 2) return null

    const latest = performanceData[performanceData.length - 1]
    const previous = performanceData[performanceData.length - 2]

    return {
      fps: latest.fps - previous.fps,
      memory: latest.memory - previous.memory,
      cpu: latest.cpu - previous.cpu,
      network: latest.network - previous.network,
      renderTime: latest.renderTime - previous.renderTime
    }
  }, [performanceData])

  // 性能指标组件
  const MetricCard = ({ 
    title, 
    value, 
    unit, 
    icon, 
    trend, 
    threshold, 
    color = 'primary' 
  }: {
    title: string
    value: number
    unit: string
    icon: React.ReactNode
    trend?: number
    threshold?: { warning: number; error: number }
    color?: 'primary' | 'success' | 'warning' | 'danger'
  }) => {
    const getColor = () => {
      if (threshold) {
        if (value >= threshold.error) return 'danger'
        if (value >= threshold.warning) return 'warning'
      }
      return color
    }

    const currentColor = getColor()
    const colorClasses = {
      primary: 'text-blue-600 bg-blue-50',
      success: 'text-green-600 bg-green-50',
      warning: 'text-yellow-600 bg-yellow-50',
      danger: 'text-red-600 bg-red-50'
    }

    return (
      <motion.div
        initial={{ opacity: 0, scale: 0.9 }}
        animate={{ opacity: 1, scale: 1 }}
        className="p-4 rounded-lg border bg-white"
      >
        <div className="flex items-center justify-between mb-2">
          <div className={`p-2 rounded-lg ${colorClasses[currentColor]}`}>
            {icon}
          </div>
          <div className="flex items-center space-x-2">
            {trend !== undefined && (
              <div className="flex items-center">
                {trend > 0 ? (
                  <TrendingUp className="w-4 h-4 text-red-500" />
                ) : trend < 0 ? (
                  <TrendingDown className="w-4 h-4 text-green-500" />
                ) : null}
              </div>
            )}
            <Badge variant={currentColor === 'danger' ? 'error' : currentColor === 'warning' ? 'warning' : 'secondary'}>
              {value}{unit}
            </Badge>
          </div>
        </div>
        <h3 className="font-medium text-gray-900">{title}</h3>
        {trend !== undefined && (
          <p className="text-xs text-gray-500 mt-1">
            {trend > 0 ? `+${trend}` : trend < 0 ? `${trend}` : '0'} 变化
          </p>
        )}
      </motion.div>
    )
  }

  // 性能图表组件
  const PerformanceChart = ({ data, color = 'blue' }: { data: number[], color?: string }) => {
    if (data.length === 0) return null

    const maxValue = Math.max(...data)
    const minValue = Math.min(...data)
    const range = maxValue - minValue || 1

    return (
      <div className="h-20 flex items-end space-x-1">
        {data.map((value, index) => {
          const height = ((value - minValue) / range) * 100
          return (
            <motion.div
              key={index}
              initial={{ height: 0 }}
              animate={{ height: `${height}%` }}
              className={`bg-${color}-500 rounded-t flex-1 opacity-70`}
            />
          )
        })}
      </div>
    )
  }

  return (
    <div className="space-y-6">
      {/* 控制面板 */}
      <Card title="性能监控控制台" variant="outlined">
        <div className="flex items-center justify-between">
          <div className="flex items-center space-x-4">
            <div className={`w-3 h-3 rounded-full ${isMonitoring ? 'bg-green-500' : 'bg-gray-400'}`} />
            <span className="font-medium">
              {isMonitoring ? '监控中...' : '已停止'}
            </span>
            {isMonitoring && (
              <Badge variant="success">
                {performanceData.length} 个数据点
              </Badge>
            )}
          </div>
          <Button
            onClick={toggleMonitoring}
            variant={isMonitoring ? 'destructive' : 'primary'}
          >
            {isMonitoring ? '停止监控' : '开始监控'}
          </Button>
        </div>
      </Card>

      {/* 当前指标 */}
      {currentMetrics && (
        <Card title="实时性能指标" variant="outlined">
          <div className="grid grid-cols-2 md:grid-cols-3 lg:grid-cols-5 gap-4">
            <MetricCard
              title="FPS"
              value={currentMetrics.fps}
              unit=""
              icon={<Activity className="w-5 h-5" />}
              trend={performanceTrends?.fps}
              threshold={{ warning: 45, error: 30 }}
            />
            <MetricCard
              title="内存使用"
              value={currentMetrics.memory}
              unit="%"
              icon={<MemoryStick className="w-5 h-5" />}
              trend={performanceTrends?.memory}
              threshold={{ warning: 60, error: 80 }}
            />
            <MetricCard
              title="CPU 使用"
              value={currentMetrics.cpu}
              unit="%"
              icon={<Cpu className="w-5 h-5" />}
              trend={performanceTrends?.cpu}
              threshold={{ warning: 50, error: 70 }}
            />
            <MetricCard
              title="网络请求"
              value={currentMetrics.network}
              unit="个"
              icon={<Network className="w-5 h-5" />}
              trend={performanceTrends?.network}
            />
            <MetricCard
              title="渲染时间"
              value={currentMetrics.renderTime}
              unit="ms"
              icon={<Clock className="w-5 h-5" />}
              trend={performanceTrends?.renderTime}
              threshold={{ warning: 20, error: 33 }}
            />
          </div>
        </Card>
      )}

      {/* 性能趋势图 */}
      {performanceData.length > 0 && (
        <Card title="性能趋势" variant="outlined">
          <div className="space-y-4">
            <div>
              <h4 className="font-medium mb-2">FPS 趋势</h4>
              <PerformanceChart 
                data={performanceData.map(d => d.fps)} 
                color="blue" 
              />
            </div>
            <div>
              <h4 className="font-medium mb-2">内存使用趋势</h4>
              <PerformanceChart 
                data={performanceData.map(d => d.memory)} 
                color="green" 
              />
            </div>
            <div>
              <h4 className="font-medium mb-2">CPU 使用趋势</h4>
              <PerformanceChart 
                data={performanceData.map(d => d.cpu)} 
                color="red" 
              />
            </div>
          </div>
        </Card>
      )}

      {/* 警报面板 */}
      <AnimatePresence>
        {alerts.length > 0 && (
          <Card title="性能警报" variant="outlined">
            <div className="space-y-2">
              {alerts.map((alert) => (
                <motion.div
                  key={alert.id}
                  initial={{ opacity: 0, x: -20 }}
                  animate={{ opacity: 1, x: 0 }}
                  exit={{ opacity: 0, x: 20 }}
                  className={`flex items-center space-x-3 p-3 rounded-lg ${
                    alert.type === 'error' 
                      ? 'bg-red-50 border border-red-200' 
                      : alert.type === 'warning'
                      ? 'bg-yellow-50 border border-yellow-200'
                      : 'bg-blue-50 border border-blue-200'
                  }`}
                >
                  {alert.type === 'error' ? (
                    <AlertCircle className="w-5 h-5 text-red-500" />
                  ) : alert.type === 'warning' ? (
                    <AlertCircle className="w-5 h-5 text-yellow-500" />
                  ) : (
                    <CheckCircle className="w-5 h-5 text-blue-500" />
                  )}
                  <div className="flex-1">
                    <p className="text-sm font-medium text-gray-900">{alert.message}</p>
                    <p className="text-xs text-gray-500">
                      {new Date(alert.timestamp).toLocaleTimeString()}
                    </p>
                  </div>
                </motion.div>
              ))}
            </div>
          </Card>
        )}
      </AnimatePresence>
    </div>
  )
}

export default AdvancedMonitor
