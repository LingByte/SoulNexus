import { useState, useEffect, useRef } from 'react'
import { motion } from 'framer-motion'
import { cn } from '@/utils/cn.ts'

interface ImageOptimizerProps {
  src: string
  alt: string
  width?: number
  height?: number
  className?: string
  quality?: number
  format?: 'webp' | 'avif' | 'jpeg' | 'png'
  sizes?: string
  placeholder?: 'blur' | 'empty'
  blurDataURL?: string
  priority?: boolean
  onLoad?: () => void
  onError?: () => void
}

const ImageOptimizer = ({
  src,
  alt,
  width,
  height,
  className = "",
  quality = 75,
  format = 'webp',
  sizes,
  placeholder = 'blur',
  blurDataURL,
  priority = false,
  onLoad,
  onError
}: ImageOptimizerProps) => {
  const [isLoaded, setIsLoaded] = useState(false)
  const [hasError, setHasError] = useState(false)
  const [currentSrc, setCurrentSrc] = useState('')
  const imgRef = useRef<HTMLImageElement>(null)

  // 生成优化后的图片URL
  const generateOptimizedSrc = (originalSrc: string) => {
    // 这里可以集成图片优化服务，如 Cloudinary, ImageKit 等
    // 或者使用 Next.js 的 Image 组件
    const params = new URLSearchParams({
      w: width?.toString() || '',
      h: height?.toString() || '',
      q: quality.toString(),
      f: format,
      ...(sizes && { sizes })
    })

    // 示例：假设使用 Cloudinary
    if (originalSrc.includes('cloudinary.com')) {
      return `${originalSrc}?${params.toString()}`
    }

    // 示例：假设使用自定义图片优化服务
    if (originalSrc.includes('your-cdn.com')) {
      return `${originalSrc}?${params.toString()}`
    }

    // 默认返回原图
    return originalSrc
  }

  // 生成模糊占位符
  const generateBlurDataURL = () => {
    if (blurDataURL) return blurDataURL

    // 生成简单的模糊占位符
    const canvas = document.createElement('canvas')
    canvas.width = 40
    canvas.height = 40
    const ctx = canvas.getContext('2d')
    
    if (ctx) {
      ctx.fillStyle = '#f3f4f6'
      ctx.fillRect(0, 0, 40, 40)
      
      // 添加简单的渐变
      const gradient = ctx.createLinearGradient(0, 0, 40, 40)
      gradient.addColorStop(0, '#e5e7eb')
      gradient.addColorStop(1, '#d1d5db')
      ctx.fillStyle = gradient
      ctx.fillRect(0, 0, 40, 40)
    }

    return canvas.toDataURL('image/jpeg', 0.1)
  }

  useEffect(() => {
    const optimizedSrc = generateOptimizedSrc(src)
    setCurrentSrc(optimizedSrc)
  }, [src, width, height, quality, format, sizes])

  const handleLoad = () => {
    setIsLoaded(true)
    onLoad?.()
  }

  const handleError = () => {
    setHasError(true)
    onError?.()
  }

  return (
    <div className={cn('relative overflow-hidden', className)}>
      {/* 模糊占位符 */}
      {placeholder === 'blur' && !isLoaded && !hasError && (
        <div
          className="absolute inset-0 bg-gray-200 animate-pulse"
          style={{
            backgroundImage: `url(${generateBlurDataURL()})`,
            backgroundSize: 'cover',
            backgroundPosition: 'center',
            filter: 'blur(8px)',
            transform: 'scale(1.1)'
          }}
        />
      )}

      {/* 错误状态 */}
      {hasError && (
        <div className="absolute inset-0 bg-gray-100 flex items-center justify-center">
          <div className="text-center text-gray-400">
            <div className="text-2xl mb-2">📷</div>
            <div className="text-sm">图片加载失败</div>
          </div>
        </div>
      )}

      {/* 实际图片 */}
      {currentSrc && !hasError && (
        <motion.img
          ref={imgRef}
          src={currentSrc}
          alt={alt}
          width={width}
          height={height}
          sizes={sizes}
          loading={priority ? 'eager' : 'lazy'}
          decoding="async"
          className={cn(
            'w-full h-full object-cover transition-opacity duration-300',
            isLoaded ? 'opacity-100' : 'opacity-0'
          )}
          onLoad={handleLoad}
          onError={handleError}
          initial={{ opacity: 0 }}
          animate={{ opacity: isLoaded ? 1 : 0 }}
          transition={{ duration: 0.3 }}
        />
      )}
    </div>
  )
}

export default ImageOptimizer
