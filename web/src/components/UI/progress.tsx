import { Progress as ArcoProgress } from '@arco-design/web-react'
import type { ProgressProps as ArcoProgressProps } from '@arco-design/web-react'
import { cn } from '@/utils/utils.ts'

export type ProgressSize = 'mini' | 'sm' | 'md' | 'lg'
export type ArcoProgressSize = 'mini' | 'small' | 'default' | 'large'
export type AnyProgressSize = ProgressSize | ArcoProgressSize

export type ProgressStatus = 'success' | 'error' | 'normal' | 'warning'
export type ProgressType = 'line' | 'circle' | 'steps'

export interface ProgressProps extends Omit<ArcoProgressProps, 'size' | 'status' | 'type'> {
  size?: AnyProgressSize
  status?: ProgressStatus
  type?: ProgressType
  /** Hide percentage / status text */
  showText?: boolean
  /** Animate line progress (type=line only) */
  animation?: boolean
  className?: string
}

function normalizeSize(size: AnyProgressSize): ArcoProgressProps['size'] {
  switch (size) {
    case 'mini':
      return 'mini'
    case 'sm':
    case 'small':
      return 'small'
    case 'md':
    case 'default':
      return 'default'
    case 'lg':
    case 'large':
      return 'large'
    default:
      return 'default'
  }
}

const statusClassMap: Record<ProgressStatus, string> = {
  success: 'ui-progress--success',
  error: 'ui-progress--error',
  warning: 'ui-progress--warning',
  normal: 'ui-progress--normal',
}

export function Progress({
  size = 'md',
  status,
  type = 'line',
  showText = true,
  animation,
  className,
  percent = 0,
  color,
  strokeWidth,
  trailColor,
  steps,
  buffer,
  bufferColor,
  width,
  formatText,
  style,
  ...rest
}: ProgressProps) {
  const arcoSize = normalizeSize(size)
  const isLine = type === 'line'

  return (
    <ArcoProgress
      className={cn('ui-progress', status && statusClassMap[status], className)}
      size={arcoSize}
      type={type}
      status={status}
      percent={percent}
      showText={showText}
      animation={animation ?? (isLine && status === 'normal')}
      color={color ?? (status ? undefined : 'hsl(var(--primary))')}
      strokeWidth={strokeWidth}
      trailColor={trailColor ?? 'hsl(var(--muted) / 0.55)'}
      steps={steps}
      buffer={buffer}
      bufferColor={bufferColor}
      width={width}
      formatText={formatText}
      style={style}
      {...rest}
    />
  )
}

export default Progress
