import React from 'react'
import { WidgetConfig } from '@/types/overview'
import Card, { CardContent, CardHeader, CardTitle } from '@/components/UI/Card'

interface ProgressRingWidgetProps {
  config: WidgetConfig
  data?: {
    value: number
    max?: number
    label?: string
    color?: string
  }
}

const ProgressRingWidget: React.FC<ProgressRingWidgetProps> = ({ config, data }) => {
  const { title, props, style } = config
  const value = data?.value ?? props?.value ?? 0
  const max = data?.max ?? props?.max ?? 100
  const percentage = Math.min(Math.max((value / max) * 100, 0), 100)
  const color = data?.color || props?.color || style?.textColor || '#6366f1'
  const size = props?.size || 120
  const strokeWidth = props?.strokeWidth || 12
  const radius = (size - strokeWidth) / 2
  const circumference = 2 * Math.PI * radius
  const offset = circumference - (percentage / 100) * circumference

  return (
    <Card className="h-full" style={style}>
      <CardHeader>
        <CardTitle>{title}</CardTitle>
      </CardHeader>
      <CardContent className="flex flex-col items-center justify-center h-[calc(100%-60px)]">
        <div className="relative" style={{ width: size, height: size }}>
          <svg
            width={size}
            height={size}
            className="transform -rotate-90"
          >
            {/* 背景圆环 */}
            <circle
              cx={size / 2}
              cy={size / 2}
              r={radius}
              fill="none"
              stroke="currentColor"
              strokeWidth={strokeWidth}
              className="text-muted opacity-20"
            />
            {/* 进度圆环 */}
            <circle
              cx={size / 2}
              cy={size / 2}
              r={radius}
              fill="none"
              stroke={color}
              strokeWidth={strokeWidth}
              strokeDasharray={circumference}
              strokeDashoffset={offset}
              strokeLinecap="round"
              className="transition-all duration-500"
            />
          </svg>
          {/* 中心文字 */}
          <div className="absolute inset-0 flex flex-col items-center justify-center">
            <div className="text-2xl font-bold" style={{ color }}>
              {Math.round(percentage)}%
            </div>
            {data?.label && (
              <div className="text-xs text-muted-foreground mt-1">
                {data.label}
              </div>
            )}
          </div>
        </div>
        <div className="mt-4 text-center">
          <div className="text-sm text-muted-foreground">
            {value} / {max}
          </div>
        </div>
      </CardContent>
    </Card>
  )
}

export default ProgressRingWidget

