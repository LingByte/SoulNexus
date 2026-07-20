import type { ReactNode, HTMLAttributes } from 'react'
import { Card as ArcoCard } from '@arco-design/web-react'
import type { CardProps as ArcoCardProps } from '@arco-design/web-react'
import { cn } from '@/utils/utils.ts'
import { Loading } from './loading'

export type CardVariant = 'default' | 'elevated'

export interface CardProps extends ArcoCardProps {
  variant?: CardVariant
  /** Icon slot for elevated cards */
  icon?: ReactNode
  className?: string
  children?: ReactNode
}

export function Card({
  variant = 'default',
  icon,
  className,
  children,
  bordered = false,
  loading,
  style,
  ...rest
}: CardProps) {
  const content = loading ? <Loading block /> : children

  if (variant === 'elevated') {
    const divProps = rest as HTMLAttributes<HTMLDivElement>
    return (
      <div className={cn('ui-card-elevated-root', className)} style={style} {...divProps}>
        <div className="ui-card-elevated">
          <div className="ui-card-elevated-layer ui-card-elevated-layer--1" aria-hidden />
          <div className="ui-card-elevated-layer ui-card-elevated-layer--2" aria-hidden />
          <div className="ui-card-elevated-content">
            {loading ? (
              <Loading block />
            ) : (
              <>
                {icon ? <div className="ui-card-elevated-icon">{icon}</div> : null}
                <div className="ui-card-elevated-body">{content}</div>
              </>
            )}
          </div>
        </div>
      </div>
    )
  }

  return (
    <ArcoCard bordered={bordered} loading={false} className={cn('ui-card', className)} style={style} {...rest}>
      {content}
    </ArcoCard>
  )
}

export default Card
