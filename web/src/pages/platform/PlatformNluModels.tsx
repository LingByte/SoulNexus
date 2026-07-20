import { useCallback, useEffect, useState } from 'react'
import {
  Alert,
  Drawer,
  Modal,
  Tag,
} from '@arco-design/web-react'
import { BrainCircuit, Play, RefreshCw, Trash2, Zap } from 'lucide-react'
import BaseLayout from '@/components/Layout/BaseLayout'
import { Button, DataList, Input } from '@/components/ui'
import type { DataListColumn } from '@/components/ui'
import { NluLineListTextArea } from '@/components/nlu/NluLineListTextArea'
import { useTranslation } from '@/i18n'
import type { TenantNluIntentDef, TenantNluSpec } from '@/api/nluModels'
import {
  deletePlatformNluModel,
  fetchPlatformNluConfig,
  listPlatformNluModels,
  parsePlatformNluModel,
  trainPlatformNluModel,
  updatePlatformNluModel,
  type PlatformNluModel,
} from '@/api/platformNlu'
import { showAlert } from '@/utils/notification'
import { extractApiErrorMessage } from '@/utils/apiError'

const statusColor: Record<string, string> = {
  draft: 'gray',
  training: 'blue',
  ready: 'green',
  failed: 'red',
}

export default function PlatformNluModelsPage() {
  const { t } = useTranslation()

  const [deployEnabled, setDeployEnabled] = useState(false)
  const [cfgOk, setCfgOk] = useState(false)
  const [nluMode, setNluMode] = useState('embedding')
  const [maxIntents, setMaxIntents] = useState(64)
  const [loading, setLoading] = useState(false)
  const [page, setPage] = useState(1)
  const [total, setTotal] = useState(0)
  const [tenantFilter, setTenantFilter] = useState('')
  const [models, setModels] = useState<PlatformNluModel[]>([])
  const [active, setActive] = useState<PlatformNluModel | null>(null)
  const [drawerOpen, setDrawerOpen] = useState(false)
  const [spec, setSpec] = useState<TenantNluSpec | null>(null)
  const [parseText, setParseText] = useState('查订单')
  const [parseResult, setParseResult] = useState('')
  const [trainLoading, setTrainLoading] = useState(false)
  const [parseLoading, setParseLoading] = useState(false)

  const load = useCallback(async () => {
    setLoading(true)
    try {
      const c = await fetchPlatformNluConfig()
      setDeployEnabled(Boolean(c.deployEnabled))
      setCfgOk(Boolean(c.deployEnabled && c.platformReady))
      setNluMode(c.mode || 'embedding')
      setMaxIntents(c.maxIntents ?? 64)
      const res = await listPlatformNluModels({
        page,
        size: 20,
        tenantId: tenantFilter.trim() || undefined,
      })
      setModels(res.list ?? [])
      setTotal(res.total ?? 0)
    } catch (e: unknown) {
      showAlert(t('nluLab.statusFailed'), 'error', extractApiErrorMessage(e))
    } finally {
      setLoading(false)
    }
  }, [page, tenantFilter, t])

  useEffect(() => {
    void load()
  }, [load])

  const openModel = (row: PlatformNluModel) => {
    setActive(row)
    setSpec(JSON.parse(JSON.stringify(row.spec)) as TenantNluSpec)
    setParseResult('')
    setDrawerOpen(true)
  }

  const handleSave = async () => {
    if (!active || !spec) return
    try {
      await updatePlatformNluModel(active.id, { spec })
      showAlert(t('common.saveSuccess'), 'success')
      await load()
    } catch (e: unknown) {
      showAlert(t('common.saveFailed'), 'error', extractApiErrorMessage(e))
    }
  }

  const handleTrain = async () => {
    if (!active) return
    setTrainLoading(true)
    try {
      if (spec) await updatePlatformNluModel(active.id, { spec })
      const res = await trainPlatformNluModel(active.id)
      showAlert(res.message || t('nluLab.trainSuccess'), 'success')
      await load()
    } catch (e: unknown) {
      showAlert(t('nluLab.trainFailed'), 'error', extractApiErrorMessage(e))
    } finally {
      setTrainLoading(false)
    }
  }

  const handleParse = async () => {
    if (!active) return
    setParseLoading(true)
    setParseResult('')
    try {
      const res = await parsePlatformNluModel(active.id, parseText.trim())
      setParseResult(JSON.stringify(res, null, 2))
    } catch (e: unknown) {
      showAlert(t('nluLab.parseFailed'), 'error', extractApiErrorMessage(e))
    } finally {
      setParseLoading(false)
    }
  }

  const handleDelete = (row: PlatformNluModel) => {
    Modal.confirm({
      title: t('nluLab.deleteConfirm', { name: row.name }),
      onOk: async () => {
        await deletePlatformNluModel(row.id)
        await load()
        if (active?.id === row.id) setDrawerOpen(false)
      },
    })
  }

  const updateIntent = (idx: number, patch: Partial<TenantNluIntentDef>) => {
    if (!spec) return
    setSpec({ ...spec, intents: spec.intents.map((it, i) => (i === idx ? { ...it, ...patch } : it)) })
  }

  const addIntent = () => {
    if (!spec || spec.intents.length >= maxIntents) return
    setSpec({
      ...spec,
      intents: [
        ...spec.intents,
        { name: t('nluLab.newIntentName'), reply: t('nluLab.newIntentReply'), keywords: [], samples: [] },
      ],
    })
  }

  const removeIntent = (idx: number) => {
    if (!spec) return
    setSpec({ ...spec, intents: spec.intents.filter((_, i) => i !== idx) })
  }

  const columns: DataListColumn<Record<string, unknown>>[] = [
    {
      key: 'info',
      title: t('nluLab.colName'),
      render: (_, r) => (
        <div className="min-w-0 flex-1">
          <div className="truncate text-sm font-medium text-neutral-900">{String(r.name || '—')}</div>
          <div className="mt-0.5 text-xs text-neutral-500">{String(r.tenantName || r.tenantId || '')}</div>
        </div>
      ),
    },
    {
      key: 'status',
      title: t('nluLab.colStatus'),
      width: 100,
      render: (_, r) => (
        <Tag color={statusColor[String(r.status)] || 'gray'} className="!rounded-full">{String(r.status)}</Tag>
      ),
    },
    {
      key: 'classes',
      title: t('nluLab.colClasses'),
      width: 80,
      render: (_, r) => (
        <span className="text-sm text-neutral-700">{String((r.numClasses as number) > 0 ? r.numClasses : (r.spec as TenantNluSpec)?.intents?.length ?? 0)}</span>
      ),
    },
    {
      key: 'actions',
      width: 120,
     
      align: 'right',
      render: (_, r) => (
        <div className="flex items-center justify-end gap-1">
          <Button size="mini" onClick={() => openModel(r as unknown as PlatformNluModel)}>
            {t('common.edit')}
          </Button>
          <Button size="mini" status="danger" icon={<Trash2 size={12} />} onClick={() => handleDelete(r as unknown as PlatformNluModel)}>
            {t('common.delete')}
          </Button>
        </div>
      ),
    },
  ]

  return (
    <BaseLayout title={t('nluLab.platformTitle')} description={t('nluLab.platformDescription')}>
      <div className="space-y-4">
        {!deployEnabled ? (
          <Alert type="warning" showIcon content={t('nluLab.envDisabledHint')} />
        ) : !cfgOk ? (
          <Alert type="warning" showIcon content={t('nluLab.platformNotReady')} />
        ) : null}

        <DataList
          data={models as unknown as (PlatformNluModel & Record<string, unknown>)[]}
          columns={columns}
          loading={loading}
          rowKey="id"
          emptyText={t('nluLab.emptyModels')}
          pagination={{ current: page, pageSize: 20, total, onChange: (p) => setPage(p) }}
          header={
            <div className="flex flex-wrap items-center justify-between gap-2">
              <span className="text-sm font-medium text-neutral-900">{t('nluLab.modelsTitle')}</span>
              <div className="flex items-center gap-2">
                <Input
                  value={tenantFilter}
                  onChange={setTenantFilter}
                  placeholder={t('nluLab.tenantIdFilterPlaceholder')}
                  style={{ width: 200 }}
                />
                <Button type="outline" onClick={() => setPage(1)}>{t('common.search')}</Button>
                <Button type="outline" icon={<RefreshCw size={14} />} loading={loading} onClick={() => void load()}>
                  {t('common.refresh')}
                </Button>
              </div>
            </div>
          }
        />
      </div>

      <Drawer
        width={720}
        title={
          <span className="inline-flex items-center gap-2">
            <BrainCircuit size={16} />
            {active?.name}
            {active?.tenantName ? <Tag size="small">{active.tenantName}</Tag> : null}
          </span>
        }
        visible={drawerOpen}
        onCancel={() => setDrawerOpen(false)}
        footer={null}
      >
        {active && spec ? (
          <div className="space-y-4 pb-8">
            <Alert
              type="info"
              content={
                nluMode === 'embedding'
                  ? t('nluLab.embeddingIntentHint', { n: maxIntents })
                  : t('nluLab.intentCountHint', { n: active.numClasses || '?' })
              }
            />
            <div className="flex flex-wrap gap-2">
              <Button onClick={() => void handleSave()}>{t('common.save')}</Button>
              <Button type="outline" icon={<Zap size={14} />} loading={trainLoading} onClick={() => void handleTrain()}>
                {t('nluLab.train')}
              </Button>
            </div>
            <div>
              <div className="mb-2 flex items-center justify-between">
                <span className="text-sm font-medium text-neutral-900">{t('nluLab.intentsEditor')}</span>
                <Button size="small" type="outline" onClick={addIntent}>{t('nluLab.addIntent')}</Button>
              </div>
              <div className="space-y-3">
                {spec.intents.map((intent, idx) => (
                  <div key={`${intent.name}-${idx}`} className="rounded-xl border border-neutral-100 p-4">
                    <div className="mb-2 grid gap-2 sm:grid-cols-2">
                      <Input value={intent.name} onChange={(v) => updateIntent(idx, { name: v })} />
                      <Input value={intent.reply} onChange={(v) => updateIntent(idx, { reply: v })} />
                    </div>
                    <NluLineListTextArea
                      placeholder={t('nluLab.keywordsPlaceholder')}
                      value={intent.keywords ?? []}
                      onChange={(lines) => updateIntent(idx, { keywords: lines })}
                      minRows={2}
                      maxRows={4}
                    />
                    <NluLineListTextArea
                      className="mt-2"
                      placeholder={t('nluLab.samplesPlaceholder')}
                      value={intent.samples ?? []}
                      onChange={(lines) => updateIntent(idx, { samples: lines })}
                      minRows={2}
                      maxRows={6}
                    />
                    <Button className="mt-2" size="small" type="outline" status="danger" onClick={() => removeIntent(idx)}>
                      {t('nluLab.removeIntent')}
                    </Button>
                  </div>
                ))}
              </div>
            </div>
            <div>
              <span className="text-sm font-medium text-neutral-900">{t('nluLab.parseTitle')}</span>
              <Input.TextArea
                className="mt-2"
                value={parseText}
                onChange={setParseText}
                autoSize={{ minRows: 2, maxRows: 4 }}
              />
              <Button className="mt-2" icon={<Play size={14} />} loading={parseLoading} onClick={() => void handleParse()}>
                {t('nluLab.parse')}
              </Button>
              {parseResult ? (
                <pre className="mt-3 max-h-48 overflow-auto rounded-lg bg-neutral-50 p-3 text-xs">{parseResult}</pre>
              ) : null}
            </div>
          </div>
        ) : null}
      </Drawer>
    </BaseLayout>
  )
}
