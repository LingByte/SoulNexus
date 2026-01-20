import React, { useState } from 'react'
import { WidgetConfig } from '@/types/overview'
import Card, { CardContent, CardHeader, CardTitle } from '@/components/UI/Card'
import { Play, Video as VideoIcon } from 'lucide-react'

interface VideoWidgetProps {
  config: WidgetConfig
  data?: {
    url?: string
    type?: 'youtube' | 'vimeo' | 'direct' | 'embed'
  }
}

const VideoWidget: React.FC<VideoWidgetProps> = ({ config, data }) => {
  const { title, props, style } = config
  const videoUrl = data?.url || props?.videoUrl || props?.url
  const videoType = data?.type || props?.type || 'direct'
  const [isPlaying, setIsPlaying] = useState(false)

  const getEmbedUrl = (url: string, type: string): string => {
    if (type === 'youtube') {
      const videoId = url.match(/(?:youtube\.com\/watch\?v=|youtu\.be\/)([^&\n?#]+)/)?.[1]
      return videoId ? `https://www.youtube.com/embed/${videoId}` : url
    }
    if (type === 'vimeo') {
      const videoId = url.match(/vimeo\.com\/(\d+)/)?.[1]
      return videoId ? `https://player.vimeo.com/video/${videoId}` : url
    }
    return url
  }

  if (!videoUrl) {
    return (
      <Card className="h-full" style={style}>
        <CardHeader>
          <CardTitle>{title}</CardTitle>
        </CardHeader>
        <CardContent className="h-[calc(100%-60px)] flex items-center justify-center">
          <div className="text-center text-muted-foreground">
            <VideoIcon className="w-12 h-12 mx-auto mb-2 opacity-50" />
            <p className="text-sm">未配置视频URL</p>
          </div>
        </CardContent>
      </Card>
    )
  }

  if (videoType === 'youtube' || videoType === 'vimeo') {
    return (
      <Card className="h-full" style={style}>
        <CardHeader>
          <CardTitle>{title}</CardTitle>
        </CardHeader>
        <CardContent className="h-[calc(100%-60px)] p-0 overflow-hidden">
          <div className="relative w-full h-full" style={{ paddingBottom: '56.25%' }}>
            <iframe
              src={getEmbedUrl(videoUrl, videoType)}
              className="absolute inset-0 w-full h-full"
              allow="accelerometer; autoplay; clipboard-write; encrypted-media; gyroscope; picture-in-picture"
              allowFullScreen
            />
          </div>
        </CardContent>
      </Card>
    )
  }

  return (
    <Card className="h-full" style={style}>
      <CardHeader>
        <CardTitle>{title}</CardTitle>
      </CardHeader>
      <CardContent className="h-[calc(100%-60px)] p-0 overflow-hidden relative">
        {!isPlaying ? (
          <div
            className="w-full h-full bg-muted flex items-center justify-center cursor-pointer group"
            onClick={() => setIsPlaying(true)}
          >
            <div className="text-center">
              <div className="w-16 h-16 mx-auto mb-4 bg-primary/20 rounded-full flex items-center justify-center group-hover:bg-primary/30 transition-colors">
                <Play className="w-8 h-8 text-primary ml-1" />
              </div>
              <p className="text-sm text-muted-foreground">点击播放视频</p>
            </div>
          </div>
        ) : (
          <video
            src={videoUrl}
            controls
            className="w-full h-full object-contain"
            autoPlay
          />
        )}
      </CardContent>
    </Card>
  )
}

export default VideoWidget

