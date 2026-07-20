import { useCallback, useEffect, useMemo, useState } from 'react'
import { motion } from 'framer-motion'
import {
  Search,
  Download,
  Star,
  Package,
  Code,
  Zap,
  Bell,
  Wrench,
  Grid3X3,
  Settings,
  Eye,
  Upload,
} from 'lucide-react'
import {
  Card,
  Input,
  Modal,
  Select,
  Spin,
  Tag,
  Typography,
} from '@arco-design/web-react'
import { IconDelete, IconRefresh, IconUpload } from '@arco-design/web-react/icon'
import BaseLayout from '@/components/Layout/BaseLayout'
import { Button, Empty } from '@/components/ui'
import workflowService from '@/api/workflow'
import {
  workflowPluginService,
  type PublishWorkflowAsPluginRequest,
  type WorkflowPlugin,
  type WorkflowPluginCategory,
} from '@/api/workflowPlugin'
import { useTranslation } from '@/i18n'
import { showAlert } from '@/utils/notification'

const CATEGORIES: WorkflowPluginCategory[] = [
  'data_processing',
  'api_integration',
  'ai_service',
  'notification',
  'utility',
  'business',
  'custom',
]

const categoryIcons: Record<WorkflowPluginCategory, typeof Package> = {
  data_processing: Grid3X3,
  api_integration: Code,
  ai_service: Zap,
  notification: Bell,
  utility: Wrench,
  business: Package,
  custom: Settings,
}

const categoryColors: Record<WorkflowPluginCategory, string> = {
  data_processing: '#10b981',
  api_integration: '#3b82f6',
  ai_service: '#8b5cf6',
  notification: '#f59e0b',
  utility: '#64748b',
  business: '#6366f1',
  custom: '#ec4899',
}

export default function WorkflowPluginMarketPage() {
  const { t } = useTranslation()
  const [loading, setLoading] = useState(false)
  const [plugins, setPlugins] = useState<WorkflowPlugin[]>([])
  const [total, setTotal] = useState(0)
  const [page, setPage] = useState(1)
  const [keyword, setKeyword] = useState('')
  const [category, setCategory] = useState<WorkflowPluginCategory | ''>('')
  const [status, setStatus] = useState<'published' | 'draft' | ''>('published')
  const [viewMode, setViewMode] = useState<'grid' | 'list'>('grid')
  const [installed, setInstalled] = useState<Set<number>>(new Set())
  const [detail, setDetail] = useState<WorkflowPlugin | null>(null)
  const [detailLoading, setDetailLoading] = useState(false)
  const [installingId, setInstallingId] = useState<number | null>(null)
  const [publishOpen, setPublishOpen] = useState(false)
  const [workflows, setWorkflows] = useState<{ id: number; name: string }[]>([])
  const [workflowId, setWorkflowId] = useState<number | undefined>()
  const [pubDisplayName, setPubDisplayName] = useState('')
  const [pubDesc, setPubDesc] = useState('')
  const [pubCategory, setPubCategory] = useState<WorkflowPluginCategory>('utility')
  const [saving, setSaving] = useState(false)

  const loadInstalled = useCallback(async () => {
    try {
      const res = await workflowPluginService.listInstalledWorkflowPlugins()
      if (res.code === 200 && Array.isArray(res.data)) {
        setInstalled(new Set(res.data.map((x) => x.pluginId || x.plugin?.id).filter(Boolean) as number[]))
      }
    } catch {
      /* optional */
    }
  }, [])

  const load = useCallback(async () => {
    setLoading(true)
    try {
      const res = await workflowPluginService.listWorkflowPlugins({
        page,
        pageSize: 20,
        keyword: keyword.trim() || undefined,
        category: category || undefined,
        status: status || undefined,
      })
      if (res.code === 200 && res.data) {
        setPlugins(res.data.plugins || [])
        setTotal(res.data.total || 0)
      } else {
        showAlert(res.msg || t('common.loadFailed'), 'error')
      }
    } catch (e: unknown) {
      showAlert((e as { msg?: string })?.msg || t('common.loadFailed'), 'error')
    } finally {
      setLoading(false)
    }
  }, [category, keyword, page, status, t])

  useEffect(() => {
    void load()
  }, [load])

  useEffect(() => {
    void loadInstalled()
  }, [loadInstalled])

  const openPublish = async () => {
    setPublishOpen(true)
    setWorkflowId(undefined)
    setPubDisplayName('')
    setPubDesc('')
    try {
      const res = await workflowService.listDefinitions({})
      if (res.code === 200 && Array.isArray(res.data)) {
        setWorkflows(res.data.map((w) => ({ id: w.id, name: w.name })))
      } else {
        setWorkflows([])
      }
    } catch {
      setWorkflows([])
    }
  }

  const publish = async () => {
    if (!workflowId || !pubDisplayName.trim()) {
      showAlert(t('pluginMarket.publishRequired'), 'error')
      return
    }
    setSaving(true)
    try {
      const body: PublishWorkflowAsPluginRequest = {
        // 内部名称和 slug 由显示名称自动生成，用户无需重复填写。
        name: pubDisplayName.trim(),
        displayName: pubDisplayName.trim(),
        description: pubDesc.trim(),
        category: pubCategory,
        inputSchema: { parameters: [] },
        outputSchema: { parameters: [] },
      }
      const res = await workflowPluginService.publishWorkflowAsPlugin(workflowId, body)
      if (res.code !== 200) {
        showAlert(res.msg || t('common.saveFailed'), 'error')
        return
      }
      if (res.data?.id) {
        await workflowPluginService.publishWorkflowPlugin(res.data.id)
      }
      showAlert(t('pluginMarket.publishSuccess'), 'success')
      setPublishOpen(false)
      void load()
    } catch (e: unknown) {
      showAlert((e as { msg?: string })?.msg || t('common.saveFailed'), 'error')
    } finally {
      setSaving(false)
    }
  }

  const install = async (row: WorkflowPlugin) => {
    setInstallingId(row.id)
    try {
      const res = await workflowPluginService.installWorkflowPlugin(row.id)
      if (res.code !== 200) {
        showAlert(res.msg || t('pluginMarket.installFailed'), 'error')
        return
      }
      showAlert(t('pluginMarket.installSuccess'), 'success')
      void loadInstalled()
      void load()
    } catch (e: unknown) {
      showAlert((e as { msg?: string })?.msg || t('pluginMarket.installFailed'), 'error')
    } finally {
      setInstallingId(null)
    }
  }

  const remove = (row: WorkflowPlugin) => {
    Modal.confirm({
      title: t('common.confirmDelete'),
      content: row.displayName || row.name,
      onOk: async () => {
        const res = await workflowPluginService.deleteWorkflowPlugin(row.id)
        if (res.code !== 200) {
          showAlert(res.msg || t('common.deleteFailed'), 'error')
          return
        }
        showAlert(t('common.deleteSuccess'), 'success')
        void load()
      },
    })
  }

  const viewDetail = async (plugin: WorkflowPlugin) => {
    setDetail(plugin)
    setDetailLoading(true)
    try {
      const res = await workflowPluginService.getWorkflowPlugin(plugin.id)
      if (res.code === 200 && res.data) {
        setDetail(res.data)
      }
    } catch {
      /* keep list row snapshot */
    } finally {
      setDetailLoading(false)
    }
  }

  const totalPages = Math.max(1, Math.ceil(total / 20))
  const categoryOptions = useMemo(() => CATEGORIES.map((c) => ({ value: c, label: c })), [])

  const renderPluginCard = (plugin: WorkflowPlugin) => {
    const CategoryIcon = categoryIcons[plugin.category] || Package
    const isInstalled = installed.has(plugin.id)
    const iconColor = plugin.color || categoryColors[plugin.category] || '#6366f1'

    return (
      <Card key={plugin.id} hoverable className="flex h-full flex-col">
        <div className="mb-3 flex items-start justify-between gap-2">
          <div className="flex min-w-0 flex-1 items-center gap-3">
            <div className="rounded-xl p-3 shadow-md" style={{ backgroundColor: iconColor, color: '#fff' }}>
              <CategoryIcon className="h-6 w-6" />
            </div>
            <div className="min-w-0">
              <Typography.Title heading={6} className="!mb-0 truncate">
                {plugin.displayName}
              </Typography.Title>
              <Typography.Text type="secondary" className="!text-xs">
                {t('pluginMarket.author')}: {plugin.author || '—'}
              </Typography.Text>
            </div>
          </div>
          <Tag size="small">{plugin.category}</Tag>
        </div>
        <Typography.Paragraph type="secondary" className="!mb-3 line-clamp-3 !text-sm">
          {plugin.description || t('pluginMarket.noDescription')}
        </Typography.Paragraph>
        {plugin.tags?.length ? (
          <div className="mb-3 flex flex-wrap gap-1">
            {plugin.tags.slice(0, 3).map((tag) => (
              <Tag key={tag} size="small">
                {tag}
              </Tag>
            ))}
          </div>
        ) : null}
        <div className="mb-3 flex items-center justify-between text-xs text-[var(--color-text-3)]">
          <span>
            <Download className="mr-1 inline h-3.5 w-3.5" />
            {plugin.downloadCount || 0}
            <Star className="ml-3 mr-1 inline h-3.5 w-3.5" />
            {(plugin.rating || 0).toFixed(1)}
          </span>
          <span>v{plugin.version || '1.0.0'}</span>
        </div>
        <div className="mt-auto flex gap-2">
          <Button
            type="outline"
            className="flex-1"
            icon={<Eye className="h-3.5 w-3.5" />}
            loading={detailLoading && detail?.id === plugin.id}
            onClick={() => void viewDetail(plugin)}
          >
            {t('common.detail')}
          </Button>
          <Button
            type="primary"
            className="flex-1"
            icon={<Download className="h-3.5 w-3.5" />}
            disabled={isInstalled || plugin.status !== 'published'}
            loading={installingId === plugin.id}
            onClick={() => void install(plugin)}
          >
            {isInstalled ? t('pluginMarket.installed') : t('pluginMarket.install')}
          </Button>
          <Button type="text" status="danger" icon={<IconDelete />} onClick={() => remove(plugin)} />
        </div>
      </Card>
    )
  }

  return (
    <BaseLayout title={t('pages.pluginMarket.title')} description={t('pages.pluginMarket.description')}>
      <div className="mb-4 flex flex-wrap items-center gap-2">
        <Input
          allowClear
          prefix={<Search className="h-4 w-4 text-gray-400" />}
          placeholder={t('pluginMarket.searchPlaceholder')}
          value={keyword}
          onChange={setKeyword}
          style={{ width: 240 }}
        />
        <Select
          allowClear
          placeholder={t('pluginMarket.category')}
          value={category || undefined}
          onChange={(v) => {
            setCategory((v as WorkflowPluginCategory) || '')
            setPage(1)
          }}
          options={categoryOptions}
          style={{ width: 180 }}
        />
        <Select
          allowClear
          placeholder={t('common.status')}
          value={status || undefined}
          onChange={(v) => {
            setStatus((v as 'published' | 'draft') || '')
            setPage(1)
          }}
          options={[
            { value: 'published', label: 'published' },
            { value: 'draft', label: 'draft' },
          ]}
          style={{ width: 140 }}
        />
        <Select
          value={viewMode}
          onChange={setViewMode}
          options={[
            { value: 'grid', label: t('workflow.viewMode.grid') },
            { value: 'list', label: t('workflow.viewMode.list') },
          ]}
          style={{ width: 120 }}
        />
        <div className="flex-1" />
        <Button type="outline" icon={<IconRefresh />} onClick={() => void load()}>
          {t('common.refresh')}
        </Button>
        <Button type="primary" icon={<IconUpload />} onClick={() => void openPublish()}>
          {t('pluginMarket.publishFromWorkflow')}
        </Button>
      </div>

      {loading && plugins.length === 0 ? (
        <div className="flex justify-center py-16">
          <Spin tip={t('workflow.loading')} />
        </div>
      ) : plugins.length === 0 ? (
        <Empty preset="no-data" description={t('pluginMarket.emptyTitle')} />
      ) : viewMode === 'grid' ? (
        <div className="grid grid-cols-1 gap-4 md:grid-cols-2 lg:grid-cols-3">
          {plugins.map((plugin) => (
            <motion.div key={plugin.id} initial={{ opacity: 0, y: 12 }} animate={{ opacity: 1, y: 0 }}>
              {renderPluginCard(plugin)}
            </motion.div>
          ))}
        </div>
      ) : (
        <div className="space-y-3">{plugins.map(renderPluginCard)}</div>
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
        title={detail?.displayName}
        visible={!!detail}
        onCancel={() => {
          setDetail(null)
          setDetailLoading(false)
        }}
        footer={null}
        style={{ width: 640 }}
      >
        {detailLoading ? (
          <div className="flex justify-center py-12">
            <Spin tip={t('common.loading')} />
          </div>
        ) : detail ? (
          <div className="space-y-2 text-sm">
            <Typography.Paragraph>{detail.description || '—'}</Typography.Paragraph>
            <Typography.Text type="secondary">slug: {detail.slug}</Typography.Text>
            <br />
            <Typography.Text type="secondary">
              {t('pluginMarket.downloads')}: {detail.downloadCount ?? 0} · {t('pluginMarket.category')}: {detail.category}
            </Typography.Text>
          </div>
        ) : null}
      </Modal>

      <Modal
        title={t('pluginMarket.publishFromWorkflow')}
        visible={publishOpen}
        onCancel={() => setPublishOpen(false)}
        onOk={() => void publish()}
        confirmLoading={saving}
        style={{ width: 560 }}
        unmountOnExit
      >
        <div className="space-y-3">
          <div>
            <Typography.Text className="!text-xs">{t('pluginMarket.sourceWorkflow')}</Typography.Text>
            <Select
              placeholder={t('pluginMarket.selectWorkflow')}
              value={workflowId}
              onChange={(value) => {
                setWorkflowId(value)
                const selected = workflows.find((item) => item.id === value)
                if (selected && !pubDisplayName.trim()) {
                  setPubDisplayName(selected.name)
                }
              }}
              options={workflows.map((w) => ({ value: w.id, label: w.name }))}
              style={{ width: '100%' }}
            />
          </div>
          <div>
            <Typography.Text className="!text-xs">{t('pluginMarket.displayName')}</Typography.Text>
            <Input value={pubDisplayName} onChange={setPubDisplayName} placeholder="默认使用源工作流名称，可按需修改" />
          </div>
          <div>
            <Typography.Text className="!text-xs">{t('common.description')}</Typography.Text>
            <Input value={pubDesc} onChange={setPubDesc} />
          </div>
          <div>
            <Typography.Text className="!text-xs">{t('pluginMarket.category')}</Typography.Text>
            <Select value={pubCategory} onChange={setPubCategory} options={categoryOptions} style={{ width: '100%' }} />
          </div>
        </div>
      </Modal>
    </BaseLayout>
  )
}
