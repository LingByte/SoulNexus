import React from 'react'
import { WidgetConfig } from '@/types/overview'

interface HeroBannerWidgetProps {
  config: WidgetConfig
}

const HeroBannerWidget: React.FC<HeroBannerWidgetProps> = ({ config }) => {
  const { props, style } = config
  const title = props?.title || '欢迎'
  const subtitle = props?.subtitle || '这里是副标题'
  const backgroundImage = props?.backgroundImage || ''
  const backgroundGradient = style?.gradient || style?.backgroundGradient

  const backgroundStyle: React.CSSProperties = {
    backgroundImage: backgroundImage 
      ? `url(${backgroundImage})` 
      : backgroundGradient 
        ? `linear-gradient(135deg, ${backgroundGradient})`
        : undefined,
    backgroundColor: style?.backgroundColor || 'transparent',
    backgroundSize: 'cover',
    backgroundPosition: 'center',
    ...style
  }

  return (
    <div 
      className="relative w-full h-full rounded-lg overflow-hidden flex items-center justify-center"
      style={backgroundStyle}
    >
      <div className="relative z-10 text-center px-6 py-8">
        <h1 
          className="text-4xl md:text-5xl font-bold mb-4"
          style={{ color: style?.textColor || '#ffffff' }}
        >
          {title}
        </h1>
        <p 
          className="text-lg md:text-xl opacity-90"
          style={{ color: style?.textColor || '#ffffff' }}
        >
          {subtitle}
        </p>
      </div>
      {backgroundImage && (
        <div className="absolute inset-0 bg-black/20" />
      )}
    </div>
  )
}

export default HeroBannerWidget

