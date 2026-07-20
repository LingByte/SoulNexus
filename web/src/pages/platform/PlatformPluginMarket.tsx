import { useCallback, useEffect, useState } from 'react'
import { Modal as ArcoModal, Table, Tag } from '@arco-design/web-react'
import { IconDelete, IconRefresh } from '@arco-design/web-react/icon'
import BaseLayout from '@/components/Layout/BaseLayout'
import { Button, Card, Input, Select, TableEmpty } from '@/components/ui'
import { showAlert } from '@/utils/notification'
import {
  deleteAdminWorkflowPlugin,
  getWorkflowMarketStats,
  listAdminWorkflowPlugins,
  updateAdminWorkflowPluginStatus,
  type AdminWorkflowPlugin,
  type WorkflowMarketStats,
} from '@/api/platformWorkflowMarket'
import { useTranslation } from '@/i18n'

export default function PlatformPluginMarketPage() {
  const { t } = useTranslation()
  const [stats, setStats] = useState<WorkflowMarketStats | null>(null)
  const [loading, setLoading] = useState(false)
  const [list, setList] = useState<AdminWorkflowPlugin[]>([])
  const [total, setTotal] = useState(0)
  const [page, setPage] = useState(1)
  const [search, setSearch] = useState('')
  const [status, setStatus] = useState('')

  const loadStats = useCallback(async () => {
    try {
      const res = await getWorkflowMarketStats()
      if (res.code === 200) setStats(res.data || null)
    } catch {
      /* ignore */
    }
  }, [])

  const load = useCallback(async () => {
    setLoading(true)
    try {
      const res = await listAdminWorkflowPlugins({
        page,
        size: 20,
        search: search.trim() || undefined,
        status: status || undefined,
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
  }, [page, search, status, t])

  useEffect(() => {
    void loadStats()
  }, [loadStats])

  useEffect(() => {
    void load()
  }, [load])

  const setPluginStatus = async (row: AdminWorkflowPlugin, next: string) => {
    try {
      const res = await updateAdminWorkflowPluginStatus(row.id, next)
      if (res.code !== 200) {
        showAlert(res.msg || t('common.saveFailed'), 'error')
        return
      }
      showAlert(t('common.saveSuccess'), 'success')
      void load()
      void loadStats()
    } catch (e: unknown) {
      showAlert((e as { msg?: string })?.msg || t('common.saveFailed'), 'error')
    }
  }

  const remove = (row: AdminWorkflowPlugin) => {
    ArcoModal.confirm({
      title: t('common.confirmDelete'),
      content: row.displayName || row.name,
      onOk: async () => {
        const res = await deleteAdminWorkflowPlugin(row.id)
        if (res.code !== 200) {
          showAlert(res.msg || t('common.deleteFailed'), 'error')
          return
        }
        showAlert(t('common.deleteSuccess'), 'success')
        void load()
        void loadStats()
      },
    })
  }

  const cards = [
    { label: t('pluginMarketAdmin.pluginTotal'), value: stats?.pluginTotal ?? '—' },
    { label: t('pluginMarketAdmin.published'), value: stats?.pluginPublished ?? '—' },
    { label: t('pluginMarketAdmin.downloads'), value: stats?.downloadTotal ?? '—' },
    { label: t('pluginMarketAdmin.installs'), value: stats?.installTotal ?? '—' },
    { label: t('pluginMarketAdmin.workflows'), value: stats?.workflowTotal ?? '—' },
    { label: t('pluginMarketAdmin.nodePlugins'), value: stats?.nodePluginTotal ?? '—' },
  ]

  return (
    <BaseLayout title={t('pages.platformPluginMarket.title')} description={t('pages.platformPluginMarket.description')}>
      <div className="mb-4 grid grid-cols-2 gap-3 md:grid-cols-3 lg:grid-cols-6">
        {cards.map((c) => (
          <Card key={c.label} className="p-3">
            <div className="text-xs text-muted-foreground">{c.label}</div>
            <div className="mt-1 text-xl font-semibold tabular-nums">{c.value}</div>
          </Card>
        ))}
      </div>

      <Card className="shadow-sm">
        <div className="mb-4 flex flex-wrap items-end gap-3">
          <div>
            <div className="mb-1 text-xs text-muted-foreground">{t('common.search')}</div>
            <Input
              allowClear
              placeholder={t('pluginMarket.searchPlaceholder')}
              value={search}
              onChange={setSearch}
              style={{ width: 220 }}
            />
          </div>
          <div>
            <div className="mb-1 text-xs text-muted-foreground">{t('common.status')}</div>
            <Select
              allowClear
              placeholder={t('common.status')}
              value={status || undefined}
              onChange={(v) => setStatus(String(v || ''))}
              style={{ width: 140 }}
              options={[
                { value: 'published', label: 'published' },
                { value: 'draft', label: 'draft' },
                { value: 'archived', label: 'archived' },
              ]}
            />
          </div>
          <Button
            type="outline"
            icon={<IconRefresh />}
            onClick={() => {
              void load()
              void loadStats()
            }}
          >
            {t('common.refresh')}
          </Button>
        </div>

        <Table
          rowKey="id"
          loading={loading}
          data={list}
          noDataElement={<TableEmpty description={t('common.noData')} />}
          pagination={{ current: page, total, pageSize: 20, onChange: setPage }}
          columns={[
            { title: 'ID', dataIndex: 'id', width: 80 },
            { title: t('pluginMarket.displayName'), dataIndex: 'displayName' },
            { title: t('pluginMarket.category'), dataIndex: 'category', width: 140 },
            {
              title: t('common.status'),
              dataIndex: 'status',
              width: 110,
              render: (v: string) => <Tag>{v}</Tag>,
            },
            { title: t('pluginMarket.downloads'), dataIndex: 'downloadCount', width: 100 },
            { title: t('pluginMarket.author'), dataIndex: 'author', width: 120 },
            {
              title: t('common.actions'),
              width: 260,
              render: (_: unknown, row: AdminWorkflowPlugin) => (
                <div className="flex flex-wrap gap-1">
                  {row.status !== 'published' && (
                    <Button type="text" size="mini" onClick={() => void setPluginStatus(row, 'published')}>
                      {t('pluginMarketAdmin.publish')}
                    </Button>
                  )}
                  {row.status !== 'archived' && (
                    <Button type="text" size="mini" onClick={() => void setPluginStatus(row, 'archived')}>
                      {t('pluginMarketAdmin.archive')}
                    </Button>
                  )}
                  <Button type="text" size="mini" status="danger" icon={<IconDelete />} onClick={() => remove(row)} />
                </div>
              ),
            },
          ]}
        />

        {!!stats?.topDownloaded?.length && (
          <div className="mt-6">
            <div className="text-sm font-medium">{t('pluginMarketAdmin.topDownloaded')}</div>
            <ul className="mt-2 space-y-1 text-sm text-muted-foreground">
              {stats.topDownloaded.map((p) => (
                <li key={p.id}>
                  {p.displayName || p.name} · {p.downloadCount}
                </li>
              ))}
            </ul>
          </div>
        )}
      </Card>
    </BaseLayout>
  )
}
