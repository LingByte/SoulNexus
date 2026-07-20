import { useCallback, useEffect, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { Drawer, Input, Modal, Space, Table, Tag, Typography } from '@arco-design/web-react'
import { IconDelete, IconEdit, IconList, IconApps, IconPlus, IconRefresh, IconSearch } from '@arco-design/web-react/icon'
import dayjs from 'dayjs'
import BaseLayout from '@/components/Layout/BaseLayout'
import { Button, Empty, TableEmpty } from '@/components/ui'
import { Loading } from '@/components/ui/loading'
import JSTemplateWidgetCard from '@/components/js-template/JSTemplateWidgetCard'
import JSTemplateAvatar, { jsTemplateAvatarSrc } from '@/components/js-template/JSTemplateAvatar'
import {
  deleteJSTemplate,
  isValidJSTemplateId,
  listJSTemplates,
  listJSTemplateUsage,
  type JSTemplateRow,
  type JSTemplateUsageRow,
} from '@/api/jsTemplates'
import { useTranslation } from '@/i18n'
import { showAlert } from '@/utils/notification'
import { cn } from '@/utils/cn'

void import('@monaco-editor/react')

type ViewMode = 'card' | 'list'

function PaginationButtons({
  page,
  pageSize,
  total,
  onPageChange,
}: {
  page: number
  pageSize: number
  total: number
  onPageChange: (p: number) => void
}) {
  const { t } = useTranslation()
  return (
    <Space>
      <Button size="small" disabled={page <= 1} onClick={() => onPageChange(Math.max(1, page - 1))}>
        {t('common.previous')}
      </Button>
      <Button size="small" disabled={page * pageSize >= total} onClick={() => onPageChange(page + 1)}>
        {t('common.next')}
      </Button>
    </Space>
  )
}

export default function JSTemplateManager() {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const [loading, setLoading] = useState(false)
  const [list, setList] = useState<JSTemplateRow[]>([])
  const [total, setTotal] = useState(0)
  const [page, setPage] = useState(1)
  const [keyword, setKeyword] = useState('')
  const [searchDebounce, setSearchDebounce] = useState('')
  const [viewMode, setViewMode] = useState<ViewMode>('card')
  const pageSize = 20
  const [usageOpen, setUsageOpen] = useState(false)
  const [usageLoading, setUsageLoading] = useState(false)
  const [usageList, setUsageList] = useState<JSTemplateUsageRow[]>([])
  const [usageTotal, setUsageTotal] = useState(0)
  const [usagePage, setUsagePage] = useState(1)
  const [usageFilter, setUsageFilter] = useState('')
  const usagePageSize = 20

  useEffect(() => {
    const timer = setTimeout(() => setSearchDebounce(keyword), 300)
    return () => clearTimeout(timer)
  }, [keyword])

  const load = useCallback(async () => {
    setLoading(true)
    try {
      const res = await listJSTemplates(page, pageSize, {
        name: searchDebounce || undefined,
      })
      if (res.code === 200 && res.data) {
        setList(res.data.list || [])
        setTotal(res.data.total || 0)
      } else {
        showAlert(res.msg || t('common.loadFailed'), 'error')
      }
    } catch (e: unknown) {
      showAlert((e as { msg?: string })?.msg || t('common.loadFailed'), 'error')
    } finally {
      setLoading(false)
    }
  }, [page, pageSize, searchDebounce, t])

  useEffect(() => {
    void load()
  }, [load])

  const loadUsage = useCallback(async () => {
    setUsageLoading(true)
    try {
      const res = await listJSTemplateUsage(usagePage, usagePageSize, {
        jsSourceId: usageFilter.trim() || undefined,
      })
      if (res.code === 200 && res.data) {
        setUsageList(res.data.list || [])
        setUsageTotal(res.data.total || 0)
      } else {
        showAlert(res.msg || t('common.loadFailed'), 'error')
      }
    } catch (e: unknown) {
      showAlert((e as { msg?: string })?.msg || t('common.loadFailed'), 'error')
    } finally {
      setUsageLoading(false)
    }
  }, [usagePage, usageFilter, t])

  useEffect(() => {
    if (usageOpen) void loadUsage()
  }, [usageOpen, loadUsage])

  const eventLabel = (event: string) => {
    if (event === 'load') return t('jsTemplate.usageEventLoad')
    if (event === 'session_start') return t('jsTemplate.usageEventSession')
    return event || '—'
  }

  const remove = (row: JSTemplateRow) => {
    Modal.confirm({
      title: t('common.confirmDelete'),
      content: row.name,
      onOk: async () => {
        const res = await deleteJSTemplate(row.id)
        if (res.code !== 200) {
          showAlert(res.msg || t('common.deleteFailed'), 'error')
          return
        }
        showAlert(t('common.deleteSuccess'), 'success')
        void load()
      },
    })
  }

  const goEdit = (id: string) => {
    if (!isValidJSTemplateId(id)) {
      navigate('/js-templates/new')
      return
    }
    navigate(`/js-templates/${id}/edit`)
  }

  const listColumns = [
    {
      title: t('jsTemplate.avatar'),
      dataIndex: 'avatarUrl',
      width: 72,
      render: (_: unknown, row: JSTemplateRow) => (
        <JSTemplateAvatar src={jsTemplateAvatarSrc(row)} name={row.name} size="sm" />
      ),
    },
    {
      title: t('jsTemplate.name'),
      dataIndex: 'name',
      ellipsis: true,
      render: (text: string, row: JSTemplateRow) => (
        <div className="min-w-0">
          <Typography.Text bold className="!block truncate">
            {text}
          </Typography.Text>
          {row.usage ? (
            <Typography.Text type="secondary" className="!text-xs !block truncate">
              {row.usage}
            </Typography.Text>
          ) : null}
        </div>
      ),
    },
    {
      title: t('jsTemplate.sourceId'),
      dataIndex: 'jsSourceId',
      width: 200,
      render: (text: string) => <Typography.Text code className="!text-xs">{text}</Typography.Text>,
    },
    {
      title: t('jsTemplate.status'),
      dataIndex: 'status',
      width: 100,
      render: (text: string) => (
        <Tag color={text === 'active' ? 'green' : 'gray'}>{text || 'draft'}</Tag>
      ),
    },
    {
      title: t('common.actions'),
      width: 120,
      render: (_: unknown, row: JSTemplateRow) => (
        <Space>
          <Button type="text" size="mini" icon={<IconEdit />} onClick={() => goEdit(row.id)} />
          <Button type="text" size="mini" status="danger" icon={<IconDelete />} onClick={() => remove(row)} />
        </Space>
      ),
    },
  ]

  return (
    <BaseLayout title={t('pages.jsTemplates.title')} description={t('pages.jsTemplates.description')}>
      <div className="mb-4 flex flex-wrap items-center justify-between gap-3">
        <Typography.Text type="secondary" className="!text-sm max-w-xl">
          {t('jsTemplate.hint')}
        </Typography.Text>
        <div className="flex flex-wrap items-center gap-2">
          <Input
            prefix={<IconSearch />}
            placeholder={t('common.searchPlaceholder')}
            allowClear
            value={keyword}
            onChange={setKeyword}
            style={{ width: 200 }}
          />
          <div className="flex items-center rounded-md border border-border bg-muted/40 p-0.5">
            <button
              type="button"
              onClick={() => setViewMode('card')}
              className={cn(
                'rounded px-2 py-1.5 transition-colors',
                viewMode === 'card' ? 'bg-card shadow-sm text-foreground' : 'text-muted-foreground hover:text-foreground',
              )}
              title={t('common.cardView')}
            >
              <IconApps style={{ fontSize: 16 }} />
            </button>
            <button
              type="button"
              onClick={() => setViewMode('list')}
              className={cn(
                'rounded px-2 py-1.5 transition-colors',
                viewMode === 'list' ? 'bg-card shadow-sm text-foreground' : 'text-muted-foreground hover:text-foreground',
              )}
              title={t('common.listView')}
            >
              <IconList style={{ fontSize: 16 }} />
            </button>
          </div>
          <Button
            type="outline"
            onClick={() => {
              setUsagePage(1)
              setUsageOpen(true)
            }}
          >
            {t('jsTemplate.usageLogs')}
          </Button>
          <Button type="outline" icon={<IconRefresh />} onClick={() => void load()}>
            {t('common.refresh')}
          </Button>
          <Button type="primary" icon={<IconPlus />} onClick={() => navigate('/js-templates/new')}>
            {t('jsTemplate.create')}
          </Button>
        </div>
      </div>

      {loading ? (
        <Loading block tip={t('common.loading')} />
      ) : list.length === 0 ? (
        <Empty preset="no-data" description={t('common.noData')}>
          <Button type="primary" icon={<IconPlus />} onClick={() => navigate('/js-templates/new')}>
            {t('jsTemplate.create')}
          </Button>
        </Empty>
      ) : viewMode === 'card' ? (
        <>
          <div className="grid grid-cols-[repeat(auto-fill,minmax(200px,220px))] justify-start gap-3">
            {list.map((row) => (
              <JSTemplateWidgetCard
                key={row.id}
                row={row}
                onEdit={() => goEdit(row.id)}
                onDelete={() => remove(row)}
              />
            ))}
          </div>
          <div className="mt-4 flex items-center justify-between text-sm">
            <span className="text-muted-foreground">{t('common.pageRecord', { count: total })}</span>
            <PaginationButtons page={page} pageSize={pageSize} total={total} onPageChange={setPage} />
          </div>
        </>
      ) : (
        <Table
          rowKey="id"
          loading={loading}
          data={list}
          columns={listColumns}
          noDataElement={<TableEmpty description={t('common.noData')} />}
          pagination={{
            current: page,
            total,
            pageSize,
            onChange: (p) => setPage(p),
          }}
        />
      )}

      <Drawer
        width={720}
        title={t('jsTemplate.usageLogsTitle')}
        visible={usageOpen}
        onCancel={() => setUsageOpen(false)}
        footer={null}
      >
        <Typography.Text type="secondary" className="!mb-3 !block !text-sm">
          {t('jsTemplate.usageLogsHint')}
        </Typography.Text>
        <div className="mb-3 flex flex-wrap items-center gap-2">
          <Input
            allowClear
            placeholder={t('jsTemplate.usageFilterSourceId')}
            value={usageFilter}
            onChange={setUsageFilter}
            style={{ width: 260 }}
            onPressEnter={() => {
              setUsagePage(1)
              void loadUsage()
            }}
          />
          <Button
            type="outline"
            icon={<IconSearch />}
            onClick={() => {
              setUsagePage(1)
              void loadUsage()
            }}
          >
            {t('common.search')}
          </Button>
          <Button type="outline" icon={<IconRefresh />} onClick={() => void loadUsage()}>
            {t('common.refresh')}
          </Button>
        </div>
        <Table
          rowKey="id"
          loading={usageLoading}
          data={usageList}
          noDataElement={<TableEmpty description={t('jsTemplate.usageEmpty')} />}
          columns={[
            {
              title: t('jsTemplate.usageCreatedAt'),
              dataIndex: 'createdAt',
              width: 160,
              render: (v: string) => (v ? dayjs(v).format('YYYY-MM-DD HH:mm:ss') : '—'),
            },
            {
              title: t('jsTemplate.sourceId'),
              dataIndex: 'jsSourceId',
              width: 180,
              render: (v: string) => <Typography.Text code className="!text-xs">{v}</Typography.Text>,
            },
            {
              title: t('jsTemplate.usageEvent'),
              dataIndex: 'event',
              width: 120,
              render: (v: string) => <Tag>{eventLabel(v)}</Tag>,
            },
            {
              title: t('jsTemplate.usageSessionId'),
              dataIndex: 'sessionId',
              ellipsis: true,
              render: (v: string) => v || '—',
            },
          ]}
          pagination={{
            current: usagePage,
            total: usageTotal,
            pageSize: usagePageSize,
            onChange: (p) => setUsagePage(p),
          }}
        />
      </Drawer>
    </BaseLayout>
  )
}
