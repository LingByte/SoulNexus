import type { ReactNode } from 'react'
import { Tooltip as ArcoTooltip } from '@arco-design/web-react'
import type { TooltipProps as ArcoTooltipProps } from '@arco-design/web-react'
import { cn } from '@/utils/utils.ts'

export type TooltipVariant = 'default' | 'hint'
export type HintPosition = 1 | 2 | 3 | 4

export interface TooltipProps extends Omit<ArcoTooltipProps, 'content'> {
  variant?: TooltipVariant
  content?: ReactNode
  /** Hint dot label (hint variant) */
  hintLabel?: ReactNode
  hintPosition?: HintPosition
  children?: ReactNode
  className?: string
}

export function Tooltip({
  variant = 'default',
  content,
  hintLabel,
  hintPosition = 4,
  children,
  className,
  trigger = 'hover',
  position = 'top',
  ...rest
}: TooltipProps) {
  if (variant === 'hint') {
    return (
      <ArcoTooltip
        trigger={trigger}
        position={position}
        content={content}
        className={cn('ui-hint-tooltip-popup', className)}
        triggerProps={{
          popupStyle: {
            boxShadow: 'var(--ui-hint-popup-shadow)',
            borderRadius: 8,
            border: '1px solid hsl(var(--border) / 0.65)',
          },
          ...(rest.triggerProps ?? {}),
        }}
        {...rest}
      >
        <span
          className={cn('ui-hint-tooltip inline-flex cursor-pointer', className)}
          data-position={hintPosition}
        >
          <span className="ui-hint-ripple ui-hint-ripple--outer" aria-hidden />
          <span className="ui-hint-ripple ui-hint-ripple--inner" aria-hidden />
          <span className="ui-hint-radius" aria-hidden />
          <span className="ui-hint-dot">{hintLabel ?? children ?? 'Tip'}</span>
        </span>
      </ArcoTooltip>
    )
  }

  return (
    <ArcoTooltip trigger={trigger} position={position} content={content} className={className} {...rest}>
      {children}
    </ArcoTooltip>
  )
}

export default Tooltip
