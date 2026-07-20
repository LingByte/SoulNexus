import { Tag, Typography } from '@arco-design/web-react'
import { IconDelete, IconEdit } from '@arco-design/web-react/icon'
import { MessageCircle } from 'lucide-react'
import { Button } from '@/components/ui'
import type { JSTemplateRow } from '@/api/jsTemplates'
import { jsTemplateAvatarSrc } from '@/components/js-template/JSTemplateAvatar'
import { useTranslation } from '@/i18n'
import { cn } from '@/utils/cn'

export type JSTemplateWidgetCardProps = {
  row: JSTemplateRow
  onEdit: () => void
  onDelete: () => void
  className?: string
}

function PreviewAvatar({ src, name, iconSize = 14 }: { src?: string; name: string; iconSize?: number }) {
  if (src) {
    return <img src={src} alt={name} className="h-full w-full object-cover" />
  }
  return (
    <div className="flex h-full w-full items-center justify-center bg-violet-600 text-white">
      <MessageCircle size={iconSize} strokeWidth={2} />
    </div>
  )
}

export default function JSTemplateWidgetCard({ row, onEdit, onDelete, className }: JSTemplateWidgetCardProps) {
  const { t } = useTranslation()
  const avatar = jsTemplateAvatarSrc(row)
  const isActive = row.status === 'active'
  const displayName = row.name?.trim() || t('jsTemplate.unnamed')

  return (
    <article
      className={cn(
        'group flex max-w-[220px] flex-col overflow-hidden rounded-xl border border-border bg-card shadow-sm transition-all duration-200 hover:-translate-y-0.5 hover:shadow-md',
        className,
      )}
    >
      <div
        className="relative aspect-[5/4] cursor-pointer overflow-hidden bg-[#f4f6f9] dark:bg-zinc-900/80"
        onClick={onEdit}
        role="button"
        tabIndex={0}
        onKeyDown={(e) => {
          if (e.key === 'Enter' || e.key === ' ') onEdit()
        }}
      >
        <div className="pointer-events-none select-none px-2.5 pb-10 pt-2.5">
          <div className="mb-1.5 flex items-center gap-1.5">
            <div className="h-1.5 w-1.5 rounded-full bg-slate-300 dark:bg-zinc-600" />
            <div className="h-1.5 flex-1 max-w-[50%] rounded-full bg-slate-200 dark:bg-zinc-700" />
          </div>
          <div className="space-y-1">
            <div className="h-1 w-full rounded-full bg-slate-200/90 dark:bg-zinc-700/90" />
            <div className="h-1 w-[80%] rounded-full bg-slate-200/80 dark:bg-zinc-700/80" />
          </div>
          <div className="mt-2 h-8 rounded-md border border-dashed border-slate-300/70 bg-white/40 dark:border-zinc-600 dark:bg-zinc-800/40" />
        </div>

        <div className="absolute bottom-9 right-2 w-[96px] overflow-hidden rounded-lg border border-black/5 bg-white shadow-md dark:border-white/10 dark:bg-zinc-800">
          <div className="flex items-center gap-1 bg-gradient-to-r from-violet-600 to-violet-500 px-1.5 py-1">
            <div className="h-5 w-5 shrink-0 overflow-hidden rounded-full border border-white/30">
              <PreviewAvatar src={avatar} name={displayName} iconSize={10} />
            </div>
            <span className="truncate text-[9px] font-medium text-white">{displayName}</span>
          </div>
          <div className="space-y-1 p-1.5">
            <div className="line-clamp-2 rounded-md rounded-tl-sm bg-slate-100 px-1.5 py-1 text-[8px] leading-tight text-slate-600 dark:bg-zinc-700 dark:text-zinc-200">
              {t('jsTemplate.cardGreeting')}
            </div>
            <div className="h-4 rounded border border-slate-200 bg-white dark:border-zinc-600 dark:bg-zinc-900/50" />
          </div>
        </div>

        <div className="absolute bottom-2 right-2">
          <div className="relative h-8 w-8 overflow-hidden rounded-full border border-white shadow-md ring-1 ring-violet-500/25">
            <PreviewAvatar src={avatar} name={displayName} iconSize={12} />
          </div>
        </div>

        <div className="absolute left-2 top-2 scale-[0.85] origin-top-left">
          <Tag size="small" color={isActive ? 'green' : 'gray'} className="!m-0 !text-[9px] !leading-none">
            {isActive ? t('jsTemplate.statusActive') : t('jsTemplate.statusDraft')}
          </Tag>
        </div>
      </div>

      <div className="flex flex-col gap-1 border-t border-border px-2.5 py-2">
        <h3 className="mb-0 truncate text-xs font-semibold text-foreground">{displayName}</h3>
        {row.usage?.trim() ? (
          <p className="mb-0 line-clamp-1 text-[10px] leading-snug text-muted-foreground">{row.usage.trim()}</p>
        ) : null}
        <div className="flex items-center justify-between gap-1 pt-0.5">
          <Typography.Text type="secondary" className="!text-[9px] font-mono truncate">
            {row.jsSourceId}
          </Typography.Text>
          <div className="flex shrink-0">
            <Button type="text" size="mini" icon={<IconEdit />} onClick={onEdit} />
            <Button type="text" size="mini" status="danger" icon={<IconDelete />} onClick={onDelete} />
          </div>
        </div>
      </div>
    </article>
  )
}
