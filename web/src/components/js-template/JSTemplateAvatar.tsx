import { MessageSquare } from 'lucide-react'
import { cn } from '@/utils/cn'

export function jsTemplateAvatarSrc(row?: { avatarUrl?: string }) {
  return (row?.avatarUrl || '').trim()
}

export type JSTemplateAvatarProps = {
  src?: string
  name?: string
  size?: 'sm' | 'md' | 'lg' | 'xl'
  className?: string
}

const sizeMap = {
  sm: 'h-9 w-9',
  md: 'h-11 w-11',
  lg: 'h-14 w-14',
  xl: 'h-20 w-20',
}

const iconSizeMap = {
  sm: 18,
  md: 22,
  lg: 28,
  xl: 36,
}

export default function JSTemplateAvatar({ src, name, size = 'md', className }: JSTemplateAvatarProps) {
  const url = src?.trim()
  return (
    <div
      className={cn(
        'relative flex shrink-0 items-center justify-center overflow-hidden rounded-2xl border-2 border-white/60 bg-gradient-to-br from-violet-500/15 to-cyan-500/20 shadow-inner',
        sizeMap[size],
        className,
      )}
      title={name}
    >
      {url ? (
        <img src={url} alt={name || 'widget'} className="h-full w-full object-cover" />
      ) : (
        <MessageSquare size={iconSizeMap[size]} className="text-violet-600/80 dark:text-violet-300" strokeWidth={1.6} />
      )}
    </div>
  )
}
