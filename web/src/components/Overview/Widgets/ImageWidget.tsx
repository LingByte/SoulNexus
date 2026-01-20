import React, { useState } from 'react'
import { WidgetConfig } from '@/types/overview'
import Card, { CardContent, CardHeader, CardTitle } from '@/components/UI/Card'
import { Image as ImageIcon, X } from 'lucide-react'

interface ImageWidgetProps {
  config: WidgetConfig
  data?: {
    url?: string
    alt?: string
  }
}

const ImageWidget: React.FC<ImageWidgetProps> = ({ config, data }) => {
  const { title, props, style } = config
  const imageUrl = data?.url || props?.imageUrl || props?.url
  const alt = data?.alt || props?.alt || title
  const [imageError, setImageError] = useState(false)
  const [isFullscreen, setIsFullscreen] = useState(false)

  if (!imageUrl) {
    return (
      <Card className="h-full" style={style}>
        <CardHeader>
          <CardTitle>{title}</CardTitle>
        </CardHeader>
        <CardContent className="h-[calc(100%-60px)] flex items-center justify-center">
          <div className="text-center text-muted-foreground">
            <ImageIcon className="w-12 h-12 mx-auto mb-2 opacity-50" />
            <p className="text-sm">未配置图片URL</p>
          </div>
        </CardContent>
      </Card>
    )
  }

  return (
    <>
      <Card className="h-full" style={style}>
        <CardHeader>
          <CardTitle>{title}</CardTitle>
        </CardHeader>
        <CardContent className="h-[calc(100%-60px)] p-0 overflow-hidden">
          <div className="relative w-full h-full group">
            {!imageError ? (
              <img
                src={imageUrl}
                alt={alt}
                className="w-full h-full object-cover cursor-pointer transition-transform hover:scale-105"
                onError={() => setImageError(true)}
                onClick={() => setIsFullscreen(true)}
              />
            ) : (
              <div className="w-full h-full flex items-center justify-center bg-muted">
                <div className="text-center text-muted-foreground">
                  <ImageIcon className="w-12 h-12 mx-auto mb-2 opacity-50" />
                  <p className="text-sm">图片加载失败</p>
                </div>
              </div>
            )}
            {!imageError && (
              <div className="absolute inset-0 bg-black/0 group-hover:bg-black/10 transition-colors flex items-center justify-center opacity-0 group-hover:opacity-100">
                <span className="text-white text-sm">点击查看大图</span>
              </div>
            )}
          </div>
        </CardContent>
      </Card>

      {/* 全屏预览 */}
      {isFullscreen && (
        <div
          className="fixed inset-0 z-50 bg-black/90 flex items-center justify-center p-4"
          onClick={() => setIsFullscreen(false)}
        >
          <button
            className="absolute top-4 right-4 text-white hover:bg-white/20 p-2 rounded-full transition-colors"
            onClick={() => setIsFullscreen(false)}
          >
            <X className="w-6 h-6" />
          </button>
          <img
            src={imageUrl}
            alt={alt}
            className="max-w-full max-h-full object-contain"
            onClick={(e) => e.stopPropagation()}
          />
        </div>
      )}
    </>
  )
}

export default ImageWidget

