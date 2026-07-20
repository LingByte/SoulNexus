import { Bot } from 'lucide-react'
import { cn } from '@/utils/cn'

export function assistantAvatarSrc(row?: { avatarUrl?: string; avatar?: string }) {
  return (row?.avatarUrl || row?.avatar || '').trim()
}

export type AssistantAvatarProps = {
  src?: string
  name?: string
  size?: 'sm' | 'md' | 'lg'
  className?: string
  rounded?: 'full' | 'xl' | 'lg'
}

const sizeMap = {
  sm: 'h-8 w-8',
  md: 'h-10 w-10',
  lg: 'h-16 w-16',
}

const iconSizeMap = {
  sm: 16,
  md: 20,
  lg: 28,
}

const roundedMap = {
  full: 'rounded-full',
  xl: 'rounded-xl',
  lg: 'rounded-lg',
}

export default function AssistantAvatar({
  src,
  name,
  size = 'md',
  className,
  rounded = 'xl',
}: AssistantAvatarProps) {
  const url = src?.trim()
  return (
    <div
      className={cn(
        'flex shrink-0 items-center justify-center overflow-hidden border border-border bg-primary/8 text-primary',
        sizeMap[size],
        roundedMap[rounded],
        className,
      )}
      title={name}
    >
      {url ? (
        <img src={url} alt={name || 'assistant'} className="h-full w-full object-cover" />
      ) : (
        <Bot size={iconSizeMap[size]} strokeWidth={1.75} />
      )}
    </div>
  )
}
