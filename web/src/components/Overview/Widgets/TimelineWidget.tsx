import React from 'react'
import { WidgetConfig } from '@/types/overview'
import Card, { CardContent, CardHeader, CardTitle } from '@/components/UI/Card'

interface TimelineItem {
  title: string
  description?: string
  time: string
  icon?: string
  color?: string
}

interface TimelineWidgetProps {
  config: WidgetConfig
  data?: TimelineItem[]
}

const TimelineWidget: React.FC<TimelineWidgetProps> = ({ config, data }) => {
  const { title, props, style } = config
  
  // 确保 items 是数组
  let items: TimelineItem[] = []
  if (Array.isArray(data)) {
    items = data
  } else if (Array.isArray(props?.items)) {
    items = props.items
  } else if (data && typeof data === 'object' && !Array.isArray(data)) {
    items = data.items || data.timeline || []
  } else {
    items = [
      { title: '事件1', description: '描述1', time: '2024-01-01' },
      { title: '事件2', description: '描述2', time: '2024-01-02' },
      { title: '事件3', description: '描述3', time: '2024-01-03' },
    ]
  }

  // 确保是数组
  if (!Array.isArray(items)) {
    items = []
  }

  const primaryColor = style?.textColor || '#6366f1'

  return (
    <Card className="h-full" style={style}>
      <CardHeader>
        <CardTitle>{title}</CardTitle>
      </CardHeader>
      <CardContent className="h-[calc(100%-60px)] overflow-y-auto">
        <div className="relative">
          {/* 时间线 */}
          <div
            className="absolute left-4 top-0 bottom-0 w-0.5"
            style={{ backgroundColor: primaryColor + '20' }}
          />
          <div className="space-y-6">
            {items.map((item, index) => (
              <div key={index} className="relative flex gap-4">
                {/* 时间点 */}
                <div className="relative z-10 flex-shrink-0">
                  <div
                    className="w-8 h-8 rounded-full border-2 flex items-center justify-center"
                    style={{
                      backgroundColor: style?.backgroundColor || '#fff',
                      borderColor: item.color || primaryColor,
                    }}
                  >
                    {item.icon ? (
                      <span className="text-xs">{item.icon}</span>
                    ) : (
                      <div
                        className="w-3 h-3 rounded-full"
                        style={{ backgroundColor: item.color || primaryColor }}
                      />
                    )}
                  </div>
                </div>
                {/* 内容 */}
                <div className="flex-1 pb-6">
                  <div className="text-sm font-medium mb-1">{item.title}</div>
                  {item.description && (
                    <div className="text-xs text-muted-foreground mb-2">
                      {item.description}
                    </div>
                  )}
                  <div className="text-xs text-muted-foreground">{item.time}</div>
                </div>
              </div>
            ))}
          </div>
        </div>
      </CardContent>
    </Card>
  )
}

export default TimelineWidget

