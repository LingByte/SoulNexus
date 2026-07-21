import { useCallback, useEffect, useMemo, useState } from 'react'
import { Modal, Space, Switch, Tag, Typography } from '@arco-design/web-react'
import { Button, Card, Input, Select, TableEmpty } from '@/components/ui'
import { Loading } from '@/components/ui/loading'
import { useTranslation } from '@/i18n'
import { listAIProviderPools } from '@/api/platformAiPools'
import {
  deleteTenantAIPoolGrant,
  listTenantAIPoolGrants,
  upsertTenantAIPoolGrant,
  type TenantAIPoolGrantRow,
} from '@/api/tenantAiPoolGrants'
import { showAlert } from '@/utils/notification'
import { extractApiErrorMessage } from '@/utils/apiError'

type Props = { tenantId: number }

export default function TenantAiPoolGrants({ tenantId }: Props) {
  const { t } = useTranslation()
  const [grants, setGrants] = useState<TenantAIPoolGrantRow[]>([])
  const [pools, setPools] = useState<{ value: number; label: string }[]>([])
  const [loading, setLoading] = useState(false)
  const [addOpen, setAddOpen] = useState(false)
  const [poolId, setPoolId] = useState<number | undefined>()
  const [quotaLimit, setQuotaLimit] = useState('0')
  const [saving, setSaving] = useState(false)

  const grantedPoolIds = useMemo(() => new Set(grants.map((g) => g.poolId)), [grants])
  const poolOptions = useMemo(
    () => pools.filter((p) => !grantedPoolIds.has(p.value)),
    [pools, grantedPoolIds],
  )

  const load = useCallback(async () => {
    setLoading(true)
    try {
      const [gRes, pRes] = await Promise.all([listTenantAIPoolGrants(tenantId), listAIProviderPools()])
      if (gRes.code === 200 && gRes.data) setGrants(gRes.data.list || [])
      if (pRes.code === 200 && pRes.data) {
        setPools(
          (pRes.data.list || []).map((p) => ({
            value: p.id,
            label: `${p.name} (${p.modality}/${p.provider})`,
          })),
        )
      }
    } catch (e: unknown) {
      showAlert(extractApiErrorMessage(e, t('common.loadFailed')), 'error')
    } finally {
      setLoading(false)
    }
  }, [tenantId, t])

  useEffect(() => {
    void load()
  }, [load])

  const submitAdd = async () => {
    if (!poolId) {
      showAlert(t('tenantAiPoolGrants.poolRequired'), 'error')
      return
    }
    setSaving(true)
    try {
      const res = await upsertTenantAIPoolGrant(tenantId, {
        poolId,
        quotaLimit: Number(quotaLimit) || 0,
        enabled: true,
      })
      if (res.code === 200) {
        showAlert(t('common.saveSuccess'), 'success')
        setAddOpen(false)
        setPoolId(undefined)
        setQuotaLimit('0')
        void load()
      } else {
        showAlert(res.msg || t('common.saveFailed'), 'error')
      }
    } catch (e: unknown) {
      showAlert(extractApiErrorMessage(e, t('common.saveFailed')), 'error')
    } finally {
      setSaving(false)
    }
  }

  const onToggle = async (row: TenantAIPoolGrantRow, enabled: boolean) => {
    try {
      const res = await upsertTenantAIPoolGrant(tenantId, {
        poolId: row.poolId,
        quotaLimit: row.quotaLimit,
        enabled,
      })
      if (res.code === 200) void load()
      else showAlert(res.msg || t('common.saveFailed'), 'error')
    } catch (e: unknown) {
      showAlert(extractApiErrorMessage(e, t('common.saveFailed')), 'error')
    }
  }

  const onQuotaBlur = async (row: TenantAIPoolGrantRow, raw: string) => {
    const next = Number(raw) || 0
    if (next === row.quotaLimit) return
    try {
      const res = await upsertTenantAIPoolGrant(tenantId, {
        poolId: row.poolId,
        quotaLimit: next,
        enabled: row.enabled,
      })
      if (res.code === 200) void load()
      else showAlert(res.msg || t('common.saveFailed'), 'error')
    } catch (e: unknown) {
      showAlert(extractApiErrorMessage(e, t('common.saveFailed')), 'error')
    }
  }

  const onDelete = (row: TenantAIPoolGrantRow) => {
    Modal.confirm({
      title: t('common.confirmDelete'),
      onOk: async () => {
        const res = await deleteTenantAIPoolGrant(tenantId, row.id)
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
    <Card bordered={false} style={{ maxWidth: 960, marginTop: 24 }}>
      <div className="mb-3 flex flex-wrap items-center justify-between gap-2">
        <div>
          <Typography.Title heading={6} style={{ margin: 0 }}>
            {t('tenantAiPoolGrants.title')}
          </Typography.Title>
          <Typography.Paragraph type="secondary" style={{ marginBottom: 0, marginTop: 6, fontSize: 13 }}>
            {t('tenantAiPoolGrants.hint')}
          </Typography.Paragraph>
        </div>
        <Button type="primary" onClick={() => setAddOpen(true)} disabled={poolOptions.length === 0}>
          {t('tenantAiPoolGrants.assign')}
        </Button>
      </div>
      {loading ? (
        <Loading block />
      ) : grants.length === 0 ? (
        <TableEmpty description={t('tenantAiPoolGrants.empty')} />
      ) : (
        <table className="w-full text-sm">
          <thead>
            <tr className="border-b text-left text-muted-foreground">
              <th className="p-2">{t('common.name')}</th>
              <th className="p-2">Modality</th>
              <th className="p-2">{t('tenantAiPoolGrants.quota')}</th>
              <th className="p-2">{t('common.status')}</th>
              <th className="p-2 text-right">{t('common.actions')}</th>
            </tr>
          </thead>
          <tbody>
            {grants.map((g) => {
              const p = g.pool
              return (
                <tr key={g.id} className="border-b">
                  <td className="p-2">{p?.name ?? `#${g.poolId}`}</td>
                  <td className="p-2">
                    {p ? (
                      <Space size={4}>
                        <Tag size="small">{p.modality}</Tag>
                        <span className="font-mono text-xs">{p.provider}</span>
                      </Space>
                    ) : (
                      '—'
                    )}
                  </td>
                  <td className="p-2">
                    <span className="mr-2 text-xs text-muted-foreground">
                      {g.quotaUsed ?? 0}
                      {g.quotaLimit ? ` / ${g.quotaLimit}` : ' / ∞'}
                    </span>
                    <Input
                      size="small"
                      style={{ width: 88, display: 'inline-block' }}
                      defaultValue={String(g.quotaLimit ?? 0)}
                      onBlur={(e) => {
                        const v = (e.target as HTMLInputElement).value
                        void onQuotaBlur(g, v)
                      }}
                    />
                  </td>
                  <td className="p-2">
                    <Switch checked={g.enabled} onChange={(v) => void onToggle(g, v)} />
                  </td>
                  <td className="p-2 text-right">
                    <Button size="small" status="danger" onClick={() => onDelete(g)}>
                      {t('common.delete')}
                    </Button>
                  </td>
                </tr>
              )
            })}
          </tbody>
        </table>
      )}

      <Modal
        title={t('tenantAiPoolGrants.assign')}
        visible={addOpen}
        onCancel={() => setAddOpen(false)}
        onOk={() => void submitAdd()}
        confirmLoading={saving}
      >
        <Space direction="vertical" style={{ width: '100%' }}>
          <Select
            placeholder={t('tenantAiPoolGrants.selectPool')}
            options={poolOptions}
            value={poolId}
            onChange={(v) => setPoolId(v as number)}
          />
          <Input
            placeholder={t('tenantAiPoolGrants.quotaLimitHint')}
            value={quotaLimit}
            onChange={setQuotaLimit}
          />
        </Space>
      </Modal>
    </Card>
  )
}
