import { useState, useRef } from 'react'
import { motion } from 'framer-motion'
import { cn } from '@/utils/cn'

interface ChartData {
  label: string
  value: number
  color?: string
}

interface AdvancedChartProps {
  data: ChartData[]
  type?: 'bar' | 'line' | 'pie' | 'area'
  animated?: boolean
  interactive?: boolean
  className?: string
  height?: number
  showLegend?: boolean
  showTooltip?: boolean
}

const AdvancedChart = ({
  data,
  type = 'bar',
  animated = true,
  interactive = true,
  className = '',
  height = 300,
  showLegend = true,
  showTooltip = true
}: AdvancedChartProps) => {
  const [tooltip, setTooltip] = useState<{ x: number; y: number; data: ChartData } | null>(null)
  const svgRef = useRef<SVGSVGElement>(null)

  const maxValue = Math.max(...data.map(d => d.value))
  const colors = data.map((d, i) => d.color || `hsl(${(i * 137.5) % 360}, 70%, 50%)`)

  const handleMouseEnter = (index: number, event: React.MouseEvent) => {
    if (!interactive) return
    if (showTooltip) {
      const rect = svgRef.current?.getBoundingClientRect()
      if (rect) {
        setTooltip({
          x: event.clientX - rect.left,
          y: event.clientY - rect.top,
          data: data[index]
        })
      }
    }
  }

  const handleMouseLeave = () => {
    if (!interactive) return
    setTooltip(null)
  }

  const renderBarChart = () => {
    const barWidth = 100 / data.length
    return data.map((item, index) => {
      const barHeight = (item.value / maxValue) * 80
      const x = index * barWidth
      const y = 90 - barHeight

      return (
        <motion.rect
          key={index}
          x={`${x}%`}
          y={`${y}%`}
          width={`${barWidth * 0.8}%`}
          height={`${barHeight}%`}
          fill={colors[index]}
          rx="4"
          initial={animated ? { scaleY: 0 } : {}}
          animate={{ scaleY: 1 }}
          transition={{ duration: 0.6, delay: index * 0.1 }}
          whileHover={interactive ? { scaleY: 1.05, scaleX: 1.05 } : {}}
          onMouseEnter={(e) => handleMouseEnter(index, e)}
          onMouseLeave={handleMouseLeave}
          className="cursor-pointer transition-all duration-200"
        />
      )
    })
  }

  const renderLineChart = () => {
    const points = data.map((item, index) => {
      const x = (index / (data.length - 1)) * 80 + 10
      const y = 90 - (item.value / maxValue) * 80
      return `${x},${y}`
    }).join(' ')

    return (
      <motion.polyline
        points={points}
        fill="none"
        stroke={colors[0]}
        strokeWidth="3"
        strokeLinecap="round"
        strokeLinejoin="round"
        initial={animated ? { pathLength: 0 } : {}}
        animate={{ pathLength: 1 }}
        transition={{ duration: 1, ease: "easeInOut" }}
      />
    )
  }

  const renderPieChart = () => {
    let cumulativePercentage = 0
    const radius = 35
    const centerX = 50
    const centerY = 50

    return data.map((item, index) => {
      const percentage = (item.value / data.reduce((sum, d) => sum + d.value, 0)) * 100
      const startAngle = (cumulativePercentage / 100) * 360
      const endAngle = ((cumulativePercentage + percentage) / 100) * 360
      
      const x1 = centerX + radius * Math.cos((startAngle - 90) * Math.PI / 180)
      const y1 = centerY + radius * Math.sin((startAngle - 90) * Math.PI / 180)
      const x2 = centerX + radius * Math.cos((endAngle - 90) * Math.PI / 180)
      const y2 = centerY + radius * Math.sin((endAngle - 90) * Math.PI / 180)
      
      const largeArcFlag = percentage > 50 ? 1 : 0
      const pathData = [
        `M ${centerX} ${centerY}`,
        `L ${x1} ${y1}`,
        `A ${radius} ${radius} 0 ${largeArcFlag} 1 ${x2} ${y2}`,
        'Z'
      ].join(' ')

      cumulativePercentage += percentage

      return (
        <motion.path
          key={index}
          d={pathData}
          fill={colors[index]}
          initial={animated ? { scale: 0 } : {}}
          animate={{ scale: 1 }}
          transition={{ duration: 0.5, delay: index * 0.1 }}
          whileHover={interactive ? { scale: 1.1 } : {}}
          onMouseEnter={(e) => handleMouseEnter(index, e)}
          onMouseLeave={handleMouseLeave}
          className="cursor-pointer transition-all duration-200"
        />
      )
    })
  }

  const renderChart = () => {
    switch (type) {
      case 'bar':
        return renderBarChart()
      case 'line':
        return renderLineChart()
      case 'pie':
        return renderPieChart()
      default:
        return renderBarChart()
    }
  }

  return (
    <div className={cn('relative', className)}>
      <svg
        ref={svgRef}
        width="100%"
        height={height}
        viewBox="0 0 100 100"
        className="overflow-visible"
      >
        {renderChart()}
        
        {/* 网格线 */}
        <defs>
          <pattern id="grid" width="10" height="10" patternUnits="userSpaceOnUse">
            <path d="M 10 0 L 0 0 0 10" fill="none" stroke="#e5e7eb" strokeWidth="0.5"/>
          </pattern>
        </defs>
        <rect width="100" height="100" fill="url(#grid)" opacity="0.3"/>
      </svg>

      {/* 图例 */}
      {showLegend && (
        <div className="flex flex-wrap gap-2 mt-4">
          {data.map((item, index) => (
            <div key={index} className="flex items-center gap-2">
              <div
                className="w-3 h-3 rounded-full"
                style={{ backgroundColor: colors[index] }}
              />
              <span className="text-sm text-gray-600">{item.label}</span>
            </div>
          ))}
        </div>
      )}

      {/* 工具提示 */}
      {tooltip && showTooltip && (
        <motion.div
          className="absolute bg-gray-900 text-white px-3 py-2 rounded-lg text-sm pointer-events-none z-10"
          style={{
            left: tooltip.x + 10,
            top: tooltip.y - 10,
          }}
          initial={{ opacity: 0, scale: 0.8 }}
          animate={{ opacity: 1, scale: 1 }}
          exit={{ opacity: 0, scale: 0.8 }}
        >
          <div className="font-semibold">{tooltip.data.label}</div>
          <div>值: {tooltip.data.value}</div>
        </motion.div>
      )}
    </div>
  )
}

export default AdvancedChart
