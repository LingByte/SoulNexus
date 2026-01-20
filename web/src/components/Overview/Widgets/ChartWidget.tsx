import React from 'react'
import { CardContent, CardHeader, CardTitle } from '@/components/UI/Card'
import { LineChart, Line, BarChart, Bar, PieChart, Pie, Cell, XAxis, YAxis, CartesianGrid, Tooltip, Legend, ResponsiveContainer } from 'recharts'
import { WidgetConfig } from '@/types/overview'

interface ChartWidgetProps {
  config: WidgetConfig
  data?: any[]
}

const COLORS = ['#6366f1', '#8b5cf6', '#ec4899', '#f59e0b', '#10b981', '#3b82f6']

const ChartWidget: React.FC<ChartWidgetProps> = ({ config, data = [] }) => {
  const { type, title, props, style } = config
  const chartType = type.replace('chart-', '') as 'line' | 'bar' | 'pie' | 'area' | 'radar'

  // 确保data是数组
  const chartData = Array.isArray(data) ? data : (Array.isArray(props?.data) ? props.data : [])

  const renderChart = () => {
    switch (chartType) {
      case 'line':
        return (
          <ResponsiveContainer width="100%" height="100%">
            <LineChart data={chartData}>
              <CartesianGrid strokeDasharray="3 3" />
              <XAxis dataKey={props?.xAxisKey || 'name'} />
              <YAxis />
              <Tooltip />
              <Legend />
              {(props?.dataKeys || ['value']).map((key: string, index: number) => (
                <Line
                  key={key}
                  type="monotone"
                  dataKey={key}
                  stroke={COLORS[index % COLORS.length]}
                  strokeWidth={2}
                />
              ))}
            </LineChart>
          </ResponsiveContainer>
        )
      
      case 'bar':
        return (
          <ResponsiveContainer width="100%" height="100%">
            <BarChart data={chartData}>
              <CartesianGrid strokeDasharray="3 3" />
              <XAxis dataKey={props?.xAxisKey || 'name'} />
              <YAxis />
              <Tooltip />
              <Legend />
              {(props?.dataKeys || ['value']).map((key: string, index: number) => (
                <Bar
                  key={key}
                  dataKey={key}
                  fill={COLORS[index % COLORS.length]}
                />
              ))}
            </BarChart>
          </ResponsiveContainer>
        )
      
      case 'area':
        return (
          <ResponsiveContainer width="100%" height="100%">
            <LineChart data={chartData}>
              <CartesianGrid strokeDasharray="3 3" />
              <XAxis dataKey={props?.xAxisKey || 'name'} />
              <YAxis />
              <Tooltip />
              <Legend />
              {(props?.dataKeys || ['value']).map((key: string, index: number) => (
                <Line
                  key={key}
                  type="monotone"
                  dataKey={key}
                  stroke={COLORS[index % COLORS.length]}
                  fill={COLORS[index % COLORS.length]}
                  fillOpacity={0.6}
                  strokeWidth={2}
                />
              ))}
            </LineChart>
          </ResponsiveContainer>
        )
      
      case 'pie':
        return (
          <ResponsiveContainer width="100%" height="100%">
            <PieChart>
              <Pie
                data={chartData}
                cx="50%"
                cy="50%"
                labelLine={false}
                label={props?.showLabel ? ({ name, percent }: any) => `${name} ${(percent * 100).toFixed(0)}%` : false}
                outerRadius={80}
                fill="#8884d8"
                dataKey={props?.dataKey || 'value'}
              >
                {chartData.map((_, index) => (
                  <Cell key={`cell-${index}`} fill={COLORS[index % COLORS.length]} />
                ))}
              </Pie>
              <Tooltip />
              <Legend />
            </PieChart>
          </ResponsiveContainer>
        )
      
      case 'radar':
        // Radar chart需要特殊处理，暂时用Line chart代替
        return (
          <ResponsiveContainer width="100%" height="100%">
            <LineChart data={chartData}>
              <CartesianGrid strokeDasharray="3 3" />
              <XAxis dataKey={props?.xAxisKey || 'name'} />
              <YAxis />
              <Tooltip />
              <Legend />
              {(props?.dataKeys || ['value']).map((key: string, index: number) => (
                <Line
                  key={key}
                  type="monotone"
                  dataKey={key}
                  stroke={COLORS[index % COLORS.length]}
                  strokeWidth={2}
                />
              ))}
            </LineChart>
          </ResponsiveContainer>
        )
      
      default:
        return (
          <div className="flex items-center justify-center h-full text-muted-foreground">
            <p>不支持的图表类型: {chartType}</p>
          </div>
        )
    }
  }

  // 处理 padding（支持字符串或对象格式）
  const getPadding = () => {
    if (!style?.padding) return undefined
    if (typeof style.padding === 'object') {
      return `${style.padding.top || 0}px ${style.padding.right || 0}px ${style.padding.bottom || 0}px ${style.padding.left || 0}px`
    }
    return style.padding
  }

  const cardStyle: React.CSSProperties = {
    ...style,
    padding: getPadding(),
  }

  return (
    <div 
      className="h-full rounded-lg border bg-card text-card-foreground shadow-sm"
      style={cardStyle}
    >
      <CardHeader>
        <CardTitle>{title}</CardTitle>
      </CardHeader>
      <CardContent className="h-[calc(100%-60px)]">
        {chartData.length > 0 ? (
          renderChart()
        ) : (
          <div className="flex items-center justify-center h-full text-muted-foreground">
            <div className="text-center">
              <p className="text-sm mb-2">暂无数据</p>
              <p className="text-xs opacity-50">请配置数据键或提供数据</p>
            </div>
          </div>
        )}
      </CardContent>
    </div>
  )
}

export default ChartWidget

