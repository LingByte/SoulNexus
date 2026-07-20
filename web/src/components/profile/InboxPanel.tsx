import { useCallback, useEffect, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { Drawer } from '@arco-design/web-react'
import { Bell, ExternalLink, MailOpen, Trash2 } from 'lucide-react'
import { useTranslation } from '@/i18n'
import {
  deleteInboxMessage,
  listInboxMessages,
  markAllInboxRead,
  markInboxRead,
  type InboxMessage,
} from '@/api/inbox'
import { showAlert } from '@/utils/notification'
import { Button, Empty } from '@/components/ui'
import dayjs from 'dayjs'
import { cn } from '@/utils/cn'

type Filter = 'all' | 'unread' | 'read'

function resolveActionHref(raw?: string): string | null {
  const u = (raw || '').trim()
  if (!u) return null
  if (u.startsWith('http://') || u.startsWith('https://') || u.startsWith('/')) return u
  return `/${u.replace(/^\/*/, '')}`
}

export default function InboxPanel() {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const [loading, setLoading] = useState(false)
  const [list, setList] = useState<InboxMessage[]>([])
  const [total, setTotal] = useState(0)
  const [totalUnread, setTotalUnread] = useState(0)
  const [page, setPage] = useState(1)
  const [filter, setFilter] = useState<Filter>('all')
  const [detailRow, setDetailRow] = useState<InboxMessage | null>(null)
  const pageSize = 10

  const load = useCallback(async () => {
    setLoading(true)
    try {
      const res = await listInboxMessages({ page, pageSize, filter: filter === 'all' ? undefined : filter })
      if (res.code !== 200) {
        showAlert(res.msg || t('inbox.loadFailed'), 'error')
        return
      }
      setList(res.data?.list || [])
      setTotal(Number(res.data?.total || 0))
      setTotalUnread(Number(res.data?.totalUnread || 0))
    } finally {
      setLoading(false)
    }
  }, [filter, page, pageSize, t])

  useEffect(() => {
    void load()
  }, [load])

  const openDetail = async (row: InboxMessage) => {
    setDetailRow(row)
    if (!row.read) {
      await markInboxRead(row.id)
      void load()
      window.dispatchEvent(new CustomEvent('inbox:updated'))
    }
  }

  const goAction = (row: InboxMessage) => {
    const href = resolveActionHref(row.action_url)
    if (!href) return
    if (href.startsWith('http://') || href.startsWith('https://')) {
      window.open(href, '_blank', 'noopener,noreferrer')
      return
    }
    setDetailRow(null)
    navigate(href)
  }

  const totalPages = Math.max(1, Math.ceil(total / pageSize))

  return (
    <div>
      <div className="mb-5 flex flex-wrap items-center justify-between gap-3">
        <div>
          <h2 className="text-base font-medium text-foreground">{t('inbox.title')}</h2>
          <p className="mt-1 text-sm text-muted-foreground">{t('inbox.description')}</p>
        </div>
        <Button type="outline" size="small" onClick={() => void markAllInboxRead().then(() => { void load(); window.dispatchEvent(new CustomEvent('inbox:updated')) })}>
          <MailOpen size={14} className="mr-1.5 inline" />
          {t('inbox.markAllRead')}
        </Button>
      </div>

      <div className="mb-4 flex flex-wrap items-center gap-2">
        {(['all', 'unread', 'read'] as const).map((f) => (
          <Button
            key={f}
            type={filter === f ? 'primary' : 'outline'}
            size="small"
            onClick={() => {
              setPage(1)
              setFilter(f)
            }}
          >
            {t(`inbox.filter.${f}`)}
            {f === 'unread' && totalUnread > 0 ? ` (${totalUnread})` : ''}
          </Button>
        ))}
      </div>

      {loading && list.length === 0 ? (
        <div className="rounded-xl border border-border bg-card py-16 text-center text-sm text-muted-foreground">
          {t('common.loading')}
        </div>
      ) : list.length === 0 ? (
        <div className="rounded-xl border border-dashed border-border bg-card">
          <Empty preset="no-message" description={t('inbox.empty')} />
        </div>
      ) : (
        <div className="overflow-hidden rounded-xl border border-border bg-card">
          {list.map((row) => (
            <div
              key={row.id}
              className={cn(
                'flex items-start gap-3 border-b border-border px-5 py-4 last:border-b-0 transition-colors',
                !row.read && 'bg-primary/[0.03]'
              )}
            >
              <button
                type="button"
                className="flex min-w-0 flex-1 items-start gap-4 text-left hover:opacity-90"
                onClick={() => void openDetail(row)}
              >
                <div
                  className={cn(
                    'mt-0.5 flex h-9 w-9 shrink-0 items-center justify-center rounded-lg',
                    row.read ? 'bg-muted text-muted-foreground' : 'bg-primary/10 text-primary'
                  )}
                >
                  <Bell size={18} strokeWidth={1.75} />
                </div>
                <div className="min-w-0 flex-1">
                  <div className="flex flex-wrap items-center gap-2">
                    <span className={cn('text-sm', row.read ? 'font-medium text-foreground' : 'font-semibold text-foreground')}>
                      {row.title}
                    </span>
                    {!row.read && (
                      <span className="rounded-full bg-primary px-2 py-0.5 text-[10px] font-medium text-primary-foreground">
                        {t('inbox.unread')}
                      </span>
                    )}
                    {row.action_url ? (
                      <span className="rounded-full bg-muted px-2 py-0.5 text-[10px] text-muted-foreground">
                        {row.action_label || t('inbox.openLink')}
                      </span>
                    ) : null}
                  </div>
                  <p className="mt-1 line-clamp-3 whitespace-pre-wrap text-sm text-muted-foreground">{row.content}</p>
                  <p className="mt-2 text-xs text-muted-foreground/80">
                    {dayjs(row.created_at).format('YYYY/MM/DD HH:mm:ss')}
                  </p>
                </div>
              </button>
              <Button
                size="small"
                type="outline"
                status="danger"
                className="shrink-0"
                onClick={() => void deleteInboxMessage(row.id).then(() => { void load(); window.dispatchEvent(new CustomEvent('inbox:updated')) })}
              >
                <Trash2 size={14} className="mr-1 inline" />
                {t('inbox.delete')}
              </Button>
            </div>
          ))}
        </div>
      )}

      {total > 0 && (
        <div className="mt-4 flex items-center justify-between gap-3 text-sm">
          <span className="text-muted-foreground">
            {t('common.total')}: {total}
          </span>
          <div className="flex items-center gap-2">
            <Button type="outline" size="small" disabled={page <= 1} onClick={() => setPage((p) => p - 1)}>
              {t('common.previous')}
            </Button>
            <span className="text-muted-foreground">
              {page} / {totalPages}
            </span>
            <Button type="outline" size="small" disabled={page >= totalPages} onClick={() => setPage((p) => p + 1)}>
              {t('common.next')}
            </Button>
          </div>
        </div>
      )}

      <Drawer
        width={520}
        title={detailRow?.title || t('inbox.title')}
        visible={!!detailRow}
        onCancel={() => setDetailRow(null)}
        footer={
          detailRow ? (
            <div className="flex flex-wrap justify-end gap-2">
              {resolveActionHref(detailRow.action_url) ? (
                <Button size="small" type="primary" onClick={() => goAction(detailRow)}>
                  <ExternalLink size={14} className="mr-1.5 inline" />
                  {detailRow.action_label || t('inbox.openLink')}
                </Button>
              ) : null}
              {!detailRow.read && (
                <Button size="small" type="outline" onClick={() => void markInboxRead(detailRow.id).then(() => { void load(); window.dispatchEvent(new CustomEvent('inbox:updated')); setDetailRow({ ...detailRow, read: true }) })}>
                  {t('inbox.markRead')}
                </Button>
              )}
              <Button
                size="small"
                type="outline"
                status="danger"
                onClick={() => void deleteInboxMessage(detailRow.id).then(() => { setDetailRow(null); void load(); window.dispatchEvent(new CustomEvent('inbox:updated')) })}
              >
                {t('inbox.delete')}
              </Button>
            </div>
          ) : null
        }
      >
        {detailRow && (
          <div className="space-y-4">
            <p className="text-xs text-muted-foreground">
              {dayjs(detailRow.created_at).format('YYYY/MM/DD HH:mm:ss')}
            </p>
            <div className="rounded-xl border border-border bg-muted/30 px-4 py-3">
              <p className="whitespace-pre-wrap text-sm leading-relaxed text-foreground">{detailRow.content}</p>
            </div>
            {resolveActionHref(detailRow.action_url) ? (
              <button
                type="button"
                className="inline-flex items-center gap-2 text-sm text-primary hover:underline"
                onClick={() => goAction(detailRow)}
              >
                <ExternalLink size={14} />
                {detailRow.action_label || t('inbox.openLink')}
                <span className="text-xs text-muted-foreground">{resolveActionHref(detailRow.action_url)}</span>
              </button>
            ) : null}
          </div>
        )}
      </Drawer>
    </div>
  )
}
