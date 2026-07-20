import { useCallback, useEffect, useState } from 'react'
import { Alert, Modal, Tag } from '@arco-design/web-react'
import { BrainCircuit, Plus, Pencil, RefreshCw, Trash2 } from 'lucide-react'
import { useNavigate } from 'react-router-dom'
import BaseLayout from '@/components/Layout/BaseLayout'
import { Button, DataList } from '@/components/ui'
import type { DataListColumn } from '@/components/ui'
import { useTranslation } from '@/i18n'
import { createNluModel, deleteNluModel, fetchNluConfig, listNluModels, type TenantNluModel } from '@/api/nluModels'
import { showAlert } from '@/utils/notification'
import { extractApiErrorMessage } from '@/utils/apiError'

const statusColor: Record<string, string> = { draft: 'gray', training: 'blue', ready: 'green', failed: 'red' }

export default function NluModelsPage() {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const [deployEnabled, setDeployEnabled] = useState(false)
  const [cfgOk, setCfgOk] = useState(false)
  const [loading, setLoading] = useState(false)
  const [models, setModels] = useState<TenantNluModel[]>([])

  const load = useCallback(async () => {
    setLoading(true)
    try {
      const c = await fetchNluConfig(); setDeployEnabled(Boolean(c.deployEnabled)); setCfgOk(Boolean(c.deployEnabled && c.platformReady))
      const rows = await listNluModels(); setModels(rows)
    } catch (e: unknown) { showAlert(t('nluLab.statusFailed'), 'error', extractApiErrorMessage(e)) }
    finally { setLoading(false) }
  }, [t])

  useEffect(() => { void load() }, [load])

  const handleCreate = async () => {
    try { const row = await createNluModel({ name: t('nluLab.newModelName') }); navigate(`/nlu-models/${String(row.id)}`) }
    catch (e: unknown) { showAlert(t('nluLab.createFailed'), 'error', extractApiErrorMessage(e)) }
  }

  const handleDelete = (row: TenantNluModel) => {
    Modal.confirm({ title: t('nluLab.deleteConfirm', { name: row.name }), onOk: async () => { await deleteNluModel(row.id); await load() } })
  }

  const columns: DataListColumn<Record<string, unknown>>[] = [
    {
      key: 'icon', width: 40,
      render: () => (
        <div className="flex h-9 w-9 shrink-0 items-center justify-center rounded-full bg-purple-50 text-purple-600">
          <BrainCircuit size={18} strokeWidth={1.75} />
        </div>
      ),
    },
    {
      key: 'info', title: t('nluLab.colName'),
      render: (_, r) => (
        <div className="min-w-0 flex-1">
          <div className="truncate text-sm font-medium text-neutral-900">{String(r.name || '—')}</div>
          <div className="mt-0.5 text-xs text-neutral-500">{t('nluLab.colClasses')}: {String((r.numClasses as number) > 0 ? r.numClasses : (r.spec as Record<string, unknown>)?.intents ? String((r.spec as Record<string, unknown>).intents).split(',').length : 0)}</div>
        </div>
      ),
    },
    {
      key: 'status', title: t('nluLab.colStatus'), width: 100,
      render: (_, r) => <Tag color={statusColor[String(r.status)] || 'gray'} className="!rounded-full">{String(r.status)}</Tag>,
    },
    {
      key: 'actions', width: 140, align: 'right',
      render: (_, r) => (
        <div className="flex items-center justify-end gap-1">
          <Button size="mini" icon={<Pencil size={12} />} onClick={() => navigate(`/nlu-models/${String(r.id)}`)}>{t('common.edit')}</Button>
          <Button size="mini" status="danger" icon={<Trash2 size={12} />} onClick={() => handleDelete(r as unknown as TenantNluModel)}>{t('common.delete')}</Button>
        </div>
      ),
    },
  ]

  return (
    <BaseLayout title={t('nluLab.title')} description={t('nluLab.tenantDescription')}>
      <div className="space-y-4">
        {!deployEnabled ? (
          <Alert type="warning" showIcon content={t('nluLab.envDisabledHint')} />
        ) : !cfgOk ? (
          <Alert type="warning" showIcon content={t('nluLab.platformNotReady')} />
        ) : null}
        <DataList
          data={models as unknown as (TenantNluModel & Record<string, unknown>)[]}
          columns={columns}
          loading={loading}
          rowKey="id"
          emptyText={t('nluLab.emptyModels')}
          header={
            <div className="flex items-center justify-between gap-3">
              <span className="text-sm font-medium text-neutral-900">{t('nluLab.modelsTitle')}</span>
              <div className="flex items-center gap-2">
                <Button type="outline" icon={<RefreshCw size={14} />} loading={loading} onClick={() => void load()}>{t('common.refresh')}</Button>
                <Button type="primary" icon={<Plus size={14} />} disabled={!cfgOk} onClick={() => void handleCreate()}>{t('nluLab.createModel')}</Button>
              </div>
            </div>
          }
        />
      </div>
    </BaseLayout>
  )
}
