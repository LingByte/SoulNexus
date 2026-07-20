import type { ReactNode } from 'react'
import { cn } from '@/utils/utils.ts'

export type EmptyPreset = 'no-data' | 'no-permission' | 'no-message' | '404' | '500'

const PRESET_IMAGES: Record<EmptyPreset, string> = {
  'no-data': '/无数据.svg',
  'no-permission': '/无权限.svg',
  'no-message': '/无新消息.svg',
  '404': '/404.svg',
  '500': '/500.svg',
}

export interface EmptyProps {
  /** Built-in illustration from `public/` */
  preset?: EmptyPreset
  /** Custom SVG path under `public/` (e.g. `/无数据.svg`) */
  image?: string
  description?: ReactNode
  /** Action area — buttons, links, etc. */
  children?: ReactNode
  className?: string
  imageClassName?: string
}

export function Empty({
  preset = 'no-data',
  image,
  description,
  children,
  className,
  imageClassName,
}: EmptyProps) {
  const src = image ?? PRESET_IMAGES[preset]

  return (
    <div
      className={cn(
        'ui-empty flex flex-col items-center justify-center py-12 px-4 text-center',
        className,
      )}
    >
      <img
        src={src}
        alt=""
        className={cn('ui-empty-image mb-6 h-36 w-36 object-contain select-none', imageClassName)}
        draggable={false}
      />
      {description ? (
        <p className="mb-2 max-w-md text-sm text-[hsl(var(--muted-foreground))]">{description}</p>
      ) : null}
      {children ? <div className="mt-4 flex flex-wrap items-center justify-center gap-2">{children}</div> : null}
    </div>
  )
}

/** Compact empty state for Arco Table `noDataElement` or table `<td colSpan>`. */
export function TableEmpty({
  description = '暂无数据',
  preset = 'no-data',
  className,
}: {
  description?: ReactNode
  preset?: EmptyPreset
  className?: string
}) {
  return (
    <Empty
      preset={preset}
      description={description}
      className={cn('py-8', className)}
      imageClassName="h-20 w-20 mb-2"
    />
  )
}

export default Empty
