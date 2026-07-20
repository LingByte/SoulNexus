import type { ReactNode } from 'react'
import { Tag } from '@arco-design/web-react'
import { cn } from '@/utils/utils'

interface BadgeProps {
  children: ReactNode
  variant?: 'default' | 'primary' | 'secondary' | 'success' | 'warning' | 'error' | 'outline' | 'muted'
  size?: 'xs' | 'sm' | 'md' | 'lg'
  className?: string
  onClick?: () => void
}

const colorMap: Record<NonNullable<BadgeProps['variant']>, string | undefined> = {
  default: 'gray',
  primary: 'arcoblue',
  secondary: 'gray',
  success: 'green',
  warning: 'orangered',
  error: 'red',
  outline: undefined,
  muted: 'gray',
}

export default function Badge({
  children,
  variant = 'default',
  size = 'sm',
  className,
  onClick,
}: BadgeProps) {
  const tagSize = size === 'xs' || size === 'sm' ? 'small' : 'medium'
  const bordered = variant === 'outline'

  return (
    <Tag
      size={tagSize}
      color={colorMap[variant]}
      bordered={bordered}
      className={cn(
        variant === 'muted' && 'opacity-70',
        onClick && 'cursor-pointer',
        className,
      )}
      onClick={onClick}
    >
      {children}
    </Tag>
  )
}
