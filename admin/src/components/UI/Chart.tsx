import { useEffect, useRef } from 'react'
import { cn } from '@/utils/cn.ts'

interface ChartData {
  label: string
  value: number
  color?: string
}

interface ChartProps {
  data: ChartData[]
  type?: 'bar' | 'line' | 'pie' | 'doughnut'
  width?: number
  height?: number
  className?: string
  showLegend?: boolean
  showValues?: boolean
}

const Chart = ({
  data,
  type = 'bar',
  width = 400,
  height = 300,
  className,
  showLegend = true,
  showValues = true
}: ChartProps) => {
  const canvasRef = useRef<HTMLCanvasElement>(null)

  const colors = [
    '#3b82f6', '#ef4444', '#10b981', '#f59e0b', '#8b5cf6',
    '#06b6d4', '#84cc16', '#f97316', '#ec4899', '#6366f1'
  ]

  useEffect(() => {
    const canvas = canvasRef.current
    if (!canvas || !data.length) return

    const ctx = canvas.getContext('2d')
    if (!ctx) return

    // 设置画布尺寸
    canvas.width = width
    canvas.height = height

    // 清除画布
    ctx.clearRect(0, 0, width, height)

    const maxValue = Math.max(...data.map(d => d.value))
    const padding = 40
    const chartWidth = width - padding * 2
    const chartHeight = height - padding * 2

    if (type === 'bar') {
      drawBarChart(ctx, data, padding, chartWidth, chartHeight, maxValue)
    } else if (type === 'line') {
      drawLineChart(ctx, data, padding, chartWidth, chartHeight, maxValue)
    } else if (type === 'pie' || type === 'doughnut') {
      drawPieChart(ctx, data, width / 2, height / 2, Math.min(width, height) / 2 - padding, type === 'doughnut')
    }
  }, [data, type, width, height])

  const drawBarChart = (
    ctx: CanvasRenderingContext2D,
    data: ChartData[],
    padding: number,
    chartWidth: number,
    chartHeight: number,
    maxValue: number
  ) => {
    const barWidth = chartWidth / data.length * 0.8
    const barSpacing = chartWidth / data.length * 0.2

    data.forEach((item, index) => {
      const barHeight = (item.value / maxValue) * chartHeight
      const x = padding + index * (barWidth + barSpacing) + barSpacing / 2
      const y = padding + chartHeight - barHeight

      // 绘制柱子
      ctx.fillStyle = item.color || colors[index % colors.length]
      ctx.fillRect(x, y, barWidth, barHeight)

      // 绘制数值
      if (showValues) {
        ctx.fillStyle = '#374151'
        ctx.font = '12px sans-serif'
        ctx.textAlign = 'center'
        ctx.fillText(item.value.toString(), x + barWidth / 2, y - 5)
      }

      // 绘制标签
      ctx.fillStyle = '#6b7280'
      ctx.font = '12px sans-serif'
      ctx.textAlign = 'center'
      ctx.fillText(item.label, x + barWidth / 2, padding + chartHeight + 20)
    })
  }

  const drawLineChart = (
    ctx: CanvasRenderingContext2D,
    data: ChartData[],
    padding: number,
    chartWidth: number,
    chartHeight: number,
    maxValue: number
  ) => {
    const pointSpacing = chartWidth / (data.length - 1)
    const points = data.map((item, index) => ({
      x: padding + index * pointSpacing,
      y: padding + chartHeight - (item.value / maxValue) * chartHeight
    }))

    // 绘制线条
    ctx.strokeStyle = colors[0]
    ctx.lineWidth = 2
    ctx.beginPath()
    points.forEach((point, index) => {
      if (index === 0) {
        ctx.moveTo(point.x, point.y)
      } else {
        ctx.lineTo(point.x, point.y)
      }
    })
    ctx.stroke()

    // 绘制点
    points.forEach((point, index) => {
      ctx.fillStyle = colors[0]
      ctx.beginPath()
      ctx.arc(point.x, point.y, 4, 0, 2 * Math.PI)
      ctx.fill()

      // 绘制数值
      if (showValues) {
        ctx.fillStyle = '#374151'
        ctx.font = '12px sans-serif'
        ctx.textAlign = 'center'
        ctx.fillText(data[index].value.toString(), point.x, point.y - 10)
      }
    })

    // 绘制标签
    data.forEach((item, index) => {
      ctx.fillStyle = '#6b7280'
      ctx.font = '12px sans-serif'
      ctx.textAlign = 'center'
      ctx.fillText(item.label, padding + index * pointSpacing, padding + chartHeight + 20)
    })
  }

  const drawPieChart = (
    ctx: CanvasRenderingContext2D,
    data: ChartData[],
    centerX: number,
    centerY: number,
    radius: number,
    isDoughnut: boolean
  ) => {
    const total = data.reduce((sum, item) => sum + item.value, 0)
    let currentAngle = -Math.PI / 2

    data.forEach((item, index) => {
      const sliceAngle = (item.value / total) * 2 * Math.PI
      const color = item.color || colors[index % colors.length]

      // 绘制扇形
      ctx.fillStyle = color
      ctx.beginPath()
      ctx.moveTo(centerX, centerY)
      ctx.arc(centerX, centerY, radius, currentAngle, currentAngle + sliceAngle)
      ctx.closePath()
      ctx.fill()

      // 绘制边框
      ctx.strokeStyle = '#ffffff'
      ctx.lineWidth = 2
      ctx.stroke()

      // 如果是环形图，绘制内圆
      if (isDoughnut) {
        ctx.fillStyle = '#ffffff'
        ctx.beginPath()
        ctx.arc(centerX, centerY, radius * 0.6, 0, 2 * Math.PI)
        ctx.fill()
      }

      currentAngle += sliceAngle
    })
  }

  return (
    <div className={cn('relative', className)}>
      <canvas
        ref={canvasRef}
        className="max-w-full h-auto"
        style={{ width: `${width}px`, height: `${height}px` }}
      />
      
      {showLegend && (type === 'pie' || type === 'doughnut') && (
        <div className="mt-4 flex flex-wrap gap-4 justify-center">
          {data.map((item, index) => (
            <div key={item.label} className="flex items-center gap-2">
              <div
                className="w-3 h-3 rounded-full"
                style={{ backgroundColor: item.color || colors[index % colors.length] }}
              />
              <span className="text-sm text-neutral-600 dark:text-neutral-400">
                {item.label} ({item.value})
              </span>
            </div>
          ))}
        </div>
      )}
    </div>
  )
}

export default Chart
