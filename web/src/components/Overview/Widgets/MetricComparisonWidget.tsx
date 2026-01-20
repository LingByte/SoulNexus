import React from 'react'
import { WidgetConfig } from '@/types/overview'
import Card, { CardContent, CardHeader, CardTitle } from '@/components/UI/Card'
import { TrendingUp, TrendingDown, Minus } from 'lucide-react'

interface MetricItem {
  label: string
  value: number
  previousValue?: number
  unit?: string
  trend?: 'up' | 'down' | 'neutral'
}

interface MetricComparisonWidgetProps {
  config: WidgetConfig
  data?: MetricItem[]
}

const MetricComparisonWidget: React.FC<MetricComparisonWidgetProps> = ({ config, data }) => {
  const { title, props, style } = config
  
  // 确保 metrics 是数组
  let metrics: MetricItem[] = []
  if (Array.isArray(data)) {
    metrics = data
  } else if (Array.isArray(props?.metrics)) {
    metrics = props.metrics
  } else if (data && typeof data === 'object' && !Array.isArray(data)) {
    // 如果 data 是对象，尝试提取数组
    metrics = data.metrics || data.items || []
  } else {
    metrics = [
      { label: '指标1', value: 100, previousValue: 80, trend: 'up' },
      { label: '指标2', value: 50, previousValue: 60, trend: 'down' },
      { label: '指标3', value: 75, previousValue: 75, trend: 'neutral' },
    ]
  }

  // 确保是数组
  if (!Array.isArray(metrics)) {
    metrics = []
  }

  const calculateChange = (current: number, previous?: number): { percent: number; trend: 'up' | 'down' | 'neutral' } => {
    if (!previous || previous === 0) {
      return { percent: 0, trend: 'neutral' }
    }
    const percent = ((current - previous) / previous) * 100
    return {
      percent: Math.abs(percent),
      trend: percent > 0 ? 'up' : percent < 0 ? 'down' : 'neutral'
    }
  }

  return (
    <Card className="h-full" style={style}>
      <CardHeader>
        <CardTitle>{title}</CardTitle>
      </CardHeader>
      <CardContent className="h-[calc(100%-60px)] overflow-y-auto">
        <div className="space-y-4">
          {metrics.map((metric, index) => {
            const change = calculateChange(metric.value, metric.previousValue)
            return (
              <div key={index} className="flex items-center justify-between p-3 border rounded-lg hover:bg-muted/50 transition-colors">
                <div className="flex-1">
                  <div className="text-sm font-medium mb-1">{metric.label}</div>
                  <div className="text-2xl font-bold" style={{ color: style?.textColor }}>
                    {metric.value.toLocaleString()}
                    {metric.unit && <span className="text-sm font-normal text-muted-foreground ml-1">{metric.unit}</span>}
                  </div>
                </div>
                {metric.previousValue !== undefined && (
                  <div className="flex flex-col items-end">
                    <div className={`flex items-center gap-1 text-sm ${
                      change.trend === 'up' ? 'text-green-500' :
                      change.trend === 'down' ? 'text-red-500' :
                      'text-muted-foreground'
                    }`}>
                      {change.trend === 'up' && <TrendingUp className="w-4 h-4" />}
                      {change.trend === 'down' && <TrendingDown className="w-4 h-4" />}
                      {change.trend === 'neutral' && <Minus className="w-4 h-4" />}
                      <span>{change.percent.toFixed(1)}%</span>
                    </div>
                    <div className="text-xs text-muted-foreground mt-1">
                      上期: {metric.previousValue.toLocaleString()}
                    </div>
                  </div>
                )}
              </div>
            )
          })}
        </div>
      </CardContent>
    </Card>
  )
}

export default MetricComparisonWidget

