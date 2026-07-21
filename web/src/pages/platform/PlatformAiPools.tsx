import { useCallback, useEffect, useState } from 'react'
import { Modal, Space, Tag } from '@arco-design/web-react'
import { Button, Card, TableEmpty } from '@/components/ui'
import { Loading } from '@/components/ui/loading'
import BaseLayout from '@/components/Layout/BaseLayout'
import { useTranslation } from '@/i18n'
import { deleteAIProviderPool, listAIProviderPools, type AIProviderPoolRow } from '@/api/platformAiPools'
import AiProviderPoolDrawer from '@/components/platform/AiProviderPoolDrawer'
import { showAlert } from '@/utils/notification'
import { extractApiErrorMessage } from '@/utils/apiError'
import { ruleFor } from '@/constants/tenantAiConfigRules'

export default function PlatformAiPools() {
  const { t } = useTranslation()
  const [rows, setRows] = useState<AIProviderPoolRow[]>([])
  const [loading, setLoading] = useState(false)
  const [drawerMode, setDrawerMode] = useState<'create' | 'edit' | null>(null)
  const [editing, setEditing] = useState<AIProviderPoolRow | null>(null)

  const load = useCallback(async () => {
    setLoading(true)
    try {
      const res = await listAIProviderPools()
      if (res.code === 200 && res.data) setRows(res.data.list || [])
    } catch (e: unknown) {
      showAlert(extractApiErrorMessage(e, t('common.loadFailed')), 'error')
    } finally {
      setLoading(false)
    }
  }, [t])

  useEffect(() => {
    void load()
  }, [load])

  const providerDisplay = (row: AIProviderPoolRow) => {
    const tab = row.modality === 'realtime' ? 'realtime' : row.modality === 'llm' ? 'llm' : row.modality === 'asr' ? 'asr' : 'tts'
    const label = ruleFor(tab as 'asr' | 'tts' | 'llm' | 'realtime', row.provider)?.label
    return label ? `${label} (${row.provider})` : row.provider
  }

  const onDelete = (row: AIProviderPoolRow) => {
    Modal.confirm({
      title: t('common.confirmDelete'),
      onOk: async () => {
        const res = await deleteAIProviderPool(row.id)
        if (res.code === 200) {
          showAlert(t('common.deleteSuccess'), 'success')
          void load()
        } else {
          showAlert(res.msg || t('common.deleteFailed'), 'error')
        }
      },
    })
  }

  return (
    <BaseLayout title={t('platformAiPools.title')} description={t('platformAiPools.description')}>
      <Card bordered={false}>
        <div className="mb-4 flex justify-end">
          <Button
            type="primary"
            onClick={() => {
              setEditing(null)
              setDrawerMode('create')
            }}
          >
            {t('platformAiPools.create')}
          </Button>
        </div>
        {loading ? (
          <Loading block />
        ) : rows.length === 0 ? (
          <TableEmpty description={t('common.noData')} />
        ) : (
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b text-left text-muted-foreground">
                <th className="p-2">{t('common.name')}</th>
                <th className="p-2">{t('platformAiPools.modality')}</th>
                <th className="p-2">{t('tenantAiConfig.providerLabel')}</th>
                <th className="p-2">{t('platformAiPools.voiceIds')}</th>
                <th className="p-2">{t('tenantAiPoolGrants.quota')}</th>
                <th className="p-2 text-right">{t('common.actions')}</th>
              </tr>
            </thead>
            <tbody>
              {rows.map((r) => (
                <tr key={r.id} className="border-b">
                  <td className="p-2">{r.name}</td>
                  <td className="p-2">
                    <Tag>{t(`platformAiPools.modality_${r.modality}`)}</Tag>
                  </td>
                  <td className="p-2 text-xs">{providerDisplay(r)}</td>
                  <td className="p-2 text-xs">{(r.voiceIds || []).join(', ') || '—'}</td>
                  <td className="p-2 text-xs">
                    {r.quotaLimit ? `${r.quotaUsed ?? 0}/${r.quotaLimit}` : '∞'}
                  </td>
                  <td className="p-2 text-right">
                    <Space>
                      <Button
                        size="small"
                        onClick={() => {
                          setEditing(r)
                          setDrawerMode('edit')
                        }}
                      >
                        {t('platformAiPools.edit')}
                      </Button>
                      <Button size="small" status="danger" onClick={() => onDelete(r)}>
                        {t('common.delete')}
                      </Button>
                    </Space>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </Card>

      <AiProviderPoolDrawer
        visible={drawerMode !== null}
        mode={drawerMode === 'edit' ? 'edit' : 'create'}
        row={editing}
        onClose={() => {
          setDrawerMode(null)
          setEditing(null)
        }}
        onSaved={() => void load()}
      />
    </BaseLayout>
  )
}
