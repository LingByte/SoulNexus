import { useCallback, useEffect, useMemo, useState } from 'react'
import { Download, Package, Search } from 'lucide-react'
import { Modal, Tag, Typography } from '@arco-design/web-react'
import { IconRefresh } from '@arco-design/web-react/icon'
import BaseLayout from '@/components/Layout/BaseLayout'
import { Button, Card, Empty, Input, Loading, Select } from '@/components/ui'
import widgetMarketService, { type WidgetMarketItem } from '@/api/widgetMarket'
import { useTranslation } from '@/i18n'
import { showAlert } from '@/utils/notification'

const CATEGORIES = ['all', 'desktop_pet', 'chat_widget', 'live2d', 'utility', 'custom'] as const
const PAGE_SIZE = 24

export default function WidgetMarketPage() {
  const { t } = useTranslation()
  const [loading, setLoading] = useState(false)
  const [items, setItems] = useState<WidgetMarketItem[]>([])
  const [total, setTotal] = useState(0)
  const [page, setPage] = useState(1)
  const [keyword, setKeyword] = useState('')
  const [searchDebounce, setSearchDebounce] = useState('')
  const [category, setCategory] = useState<string>('all')
  const [detail, setDetail] = useState<WidgetMarketItem | null>(null)

  useEffect(() => {
    const timer = setTimeout(() => setSearchDebounce(keyword), 300)
    return () => clearTimeout(timer)
  }, [keyword])

  useEffect(() => {
    setPage(1)
  }, [searchDebounce, category])

  const categoryOptions = useMemo(
    () =>
      CATEGORIES.map((c) => ({
        value: c,
        label: t(`widgetMarket.categories.${c}`),
      })),
    [t],
  )

  const load = useCallback(async () => {
    setLoading(true)
    try {
      const res = await widgetMarketService.listPublished({
        page,
        pageSize: PAGE_SIZE,
        keyword: searchDebounce,
        category,
        useTenantApi: true,
      })
      if (res.code === 200 && res.data) {
        setItems(res.data.list || [])
        setTotal(res.data.total || 0)
      } else {
        showAlert(res.msg || t('common.loadFailed'), 'error')
      }
    } catch (e: unknown) {
      showAlert((e as { msg?: string })?.msg || t('common.loadFailed'), 'error')
    } finally {
      setLoading(false)
    }
  }, [category, page, searchDebounce, t])

  useEffect(() => {
    void load()
  }, [load])

  const totalPages = Math.max(1, Math.ceil(total / PAGE_SIZE))

  const onDownload = async (item: WidgetMarketItem) => {
    try {
      await widgetMarketService.trackDownload(item.id)
      showAlert(t('widgetMarket.downloadTracked'), 'success')
    } catch {
      /* optional metric */
    }
    const embed = `${window.location.origin}/api${item.embedPath}`
    await navigator.clipboard.writeText(`jsSourceId: ${item.jsSourceId}\nembed: ${embed}`)
    showAlert(t('widgetMarket.copiedEmbed'), 'success')
  }

  const renderCard = (item: WidgetMarketItem) => (
    <Card
      key={item.id}
      hoverable
      className="flex h-full cursor-pointer flex-col"
      onClick={() => setDetail(item)}
    >
      <div className="mb-3 flex items-start gap-3">
        {item.avatarUrl ? (
          <img src={item.avatarUrl} alt="" className="h-12 w-12 shrink-0 rounded-lg border object-cover" />
        ) : (
          <div className="flex h-12 w-12 shrink-0 items-center justify-center rounded-lg bg-muted">
            <Package className="h-6 w-6 text-muted-foreground" />
          </div>
        )}
        <div className="min-w-0 flex-1">
          <Typography.Title heading={6} className="!mb-0 truncate">
            {item.displayName || item.name}
          </Typography.Title>
          <Typography.Text type="secondary" className="!text-xs font-mono truncate !block">
            {item.jsSourceId}
          </Typography.Text>
          <Tag size="small" className="!mt-2">
            {t(`widgetMarket.categories.${item.category}`, item.category)}
          </Tag>
        </div>
      </div>
      {item.description ? (
        <Typography.Paragraph type="secondary" className="!mb-3 line-clamp-2 !text-sm">
          {item.description}
        </Typography.Paragraph>
      ) : (
        <Typography.Paragraph type="secondary" className="!mb-3 !text-sm">
          —
        </Typography.Paragraph>
      )}
      <div className="mt-auto flex items-center justify-between gap-2 pt-1">
        <Typography.Text type="secondary" className="!text-xs">
          {t('widgetMarket.installs', { count: item.installCount ?? 0 })}
        </Typography.Text>
        <Button
          type="primary"
          size="sm"
          icon={<Download className="h-3.5 w-3.5" />}
          onClick={(e) => {
            e.stopPropagation()
            void onDownload(item)
          }}
        >
          {t('widgetMarket.useWidget')}
        </Button>
      </div>
    </Card>
  )

  return (
    <BaseLayout title={t('pages.widgetMarket.title')} description={t('pages.widgetMarket.description')}>
      <div className="mb-4 flex flex-wrap items-center gap-2">
        <Input
          allowClear
          block={false}
          prefix={<Search className="h-4 w-4 text-muted-foreground" />}
          placeholder={t('widgetMarket.searchPlaceholder')}
          value={keyword}
          onChange={setKeyword}
          style={{ width: 240 }}
        />
        <Select
          block={false}
          allowClear={false}
          value={category}
          onChange={(v) => setCategory(String(v ?? 'all'))}
          options={categoryOptions}
          style={{ width: 160 }}
        />
        <div className="flex-1" />
        <Button type="outline" icon={<IconRefresh />} onClick={() => void load()}>
          {t('common.refresh')}
        </Button>
      </div>

      {loading && items.length === 0 ? (
        <Loading block tip={t('common.loading')} />
      ) : items.length === 0 ? (
        <Empty preset="no-data" description={t('widgetMarket.empty')} />
      ) : (
        <div className="grid grid-cols-1 gap-4 md:grid-cols-2 lg:grid-cols-3">{items.map(renderCard)}</div>
      )}

      {totalPages > 1 ? (
        <div className="mt-6 flex items-center justify-center gap-3">
          <Button type="outline" size="sm" disabled={page <= 1} onClick={() => setPage((p) => Math.max(1, p - 1))}>
            {t('common.previous')}
          </Button>
          <Typography.Text type="secondary">
            {page} / {totalPages}
          </Typography.Text>
          <Button type="outline" size="sm" disabled={page >= totalPages} onClick={() => setPage((p) => p + 1)}>
            {t('common.next')}
          </Button>
        </div>
      ) : null}

      <Modal
        visible={!!detail}
        title={detail?.displayName}
        onCancel={() => setDetail(null)}
        footer={
          detail ? (
            <Button type="primary" onClick={() => void onDownload(detail)}>
              {t('widgetMarket.useWidget')}
            </Button>
          ) : null
        }
        style={{ width: 520 }}
      >
        {detail ? (
          <div className="space-y-2 text-sm">
            <p>
              <span className="text-muted-foreground">jsSourceId：</span>
              <code className="font-mono">{detail.jsSourceId}</code>
            </p>
            <Typography.Paragraph type="secondary" className="!mb-0 whitespace-pre-wrap">
              {detail.description || '—'}
            </Typography.Paragraph>
            <Typography.Text type="secondary" className="!text-xs break-all">
              /api{detail.embedPath}
            </Typography.Text>
          </div>
        ) : null}
      </Modal>
    </BaseLayout>
  )
}
