import React, { memo } from 'react'
import { WidgetConfig } from '@/types/overview'
import StatCardWidget from './Widgets/StatCardWidget'
import ChartWidget from './Widgets/ChartWidget'
import ActivityFeedWidget from './Widgets/ActivityFeedWidget'
import CustomHtmlWidget from './Widgets/CustomHtmlWidget'
import HeroBannerWidget from './Widgets/HeroBannerWidget'
import FeatureGridWidget from './Widgets/FeatureGridWidget'
import ProgressRingWidget from './Widgets/ProgressRingWidget'
import MetricComparisonWidget from './Widgets/MetricComparisonWidget'
import TestimonialWidget from './Widgets/TestimonialWidget'
import TimelineWidget from './Widgets/TimelineWidget'
import TableWidget from './Widgets/TableWidget'
import ImageWidget from './Widgets/ImageWidget'
import VideoWidget from './Widgets/VideoWidget'
import IframeWidget from './Widgets/IframeWidget'
import MarkdownWidget from './Widgets/MarkdownWidget'

interface WidgetRendererProps {
  config: WidgetConfig
  data?: any
}

const WidgetRenderer: React.FC<WidgetRendererProps> = ({ config, data }) => {
  if (!config.visible) return null

  switch (config.type) {
    case 'stat-card':
      return <StatCardWidget config={config} data={data} />
    
    case 'chart-line':
    case 'chart-bar':
    case 'chart-pie':
    case 'chart-area':
    case 'chart-radar':
      return <ChartWidget config={config} data={data} />
    
    case 'activity-feed':
      return <ActivityFeedWidget config={config} data={data} />
    
    case 'custom-html':
      return <CustomHtmlWidget config={config} />
    
    case 'hero-banner':
      return <HeroBannerWidget config={config} />
    
    case 'feature-grid':
      return <FeatureGridWidget config={config} data={data} />
    
    case 'progress-ring':
      return <ProgressRingWidget config={config} data={data} />
    
    case 'metric-comparison':
      return <MetricComparisonWidget config={config} data={data} />
    
    case 'testimonial':
      return <TestimonialWidget config={config} data={data} />
    
    case 'timeline':
      return <TimelineWidget config={config} data={data} />
    
    case 'table':
      return <TableWidget config={config} data={data} />
    
    case 'image':
      return <ImageWidget config={config} data={data} />
    
    case 'video':
      return <VideoWidget config={config} data={data} />
    
    case 'iframe':
      return <IframeWidget config={config} data={data} />
    
    case 'markdown':
      return <MarkdownWidget config={config} data={data} />
    
    default:
      return (
        <div className="p-4 border rounded-lg">
          <p className="text-muted-foreground">未知的Widget类型: {config.type}</p>
        </div>
      )
  }
}

// 使用 memo 优化，避免不必要的重渲染
export default memo(WidgetRenderer, (prevProps, nextProps) => {
  // 自定义比较函数：如果 config.id 和 data 相同，则不重新渲染
  return (
    prevProps.config.id === nextProps.config.id &&
    prevProps.config.visible === nextProps.config.visible &&
    prevProps.data === nextProps.data
  )
})

