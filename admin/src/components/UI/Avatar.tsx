import { cn } from '@/utils/cn.ts'

interface AvatarProps {
  src?: string
  alt?: string
  fallback?: string
  size?: 'sm' | 'md' | 'lg' | 'xl'
  className?: string
  onClick?: () => void
}

// 颜色计算函数：基于字符生成背景色
const getColorFromText = (text: string): string => {
  let hash = 0;
  for (let i = 0; i < text.length; i++) {
    hash = text.charCodeAt(i) + ((hash << 5) - hash);
  }

  // 生成颜色的哈希值
  const color = `hsl(${hash % 360}, 70%, 60%)`;  // 生成基于哈希值的颜色
  return color;
}

const Avatar = ({
                  src,
                  alt = 'Avatar',
                  fallback = 'U',
                  size = 'md',
                  className,
                  onClick
                }: AvatarProps) => {
  const sizeClasses = {
    sm: 'w-8 h-8 text-xs',
    md: 'w-10 h-10 text-sm',
    lg: 'w-12 h-12 text-base',
    xl: 'w-16 h-16 text-lg',
  }

  const avatarText = size === 'sm' ? fallback.charAt(0).toUpperCase() : fallback.substring(0, 2).toUpperCase();

  return (
      <div
          className={cn(
              'relative inline-flex items-center justify-center rounded-full text-primary-foreground overflow-hidden',
              sizeClasses[size],
              onClick && 'cursor-pointer hover:opacity-80 transition-opacity',
              className
          )}
          style={{ backgroundColor: src ? 'transparent' : getColorFromText(fallback) }} // 设置动态背景颜色
          onClick={onClick}
      >
        {src ? (
            <img
                src={src}
                alt={alt}
                className="w-full h-full object-cover"
            />
        ) : (
            <span className="font-medium">
          {avatarText}
        </span>
        )}
      </div>
  )
}

export default Avatar
