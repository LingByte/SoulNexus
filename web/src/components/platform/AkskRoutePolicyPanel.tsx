import { useCallback, useEffect, useState } from 'react'
import { Card, Space, Switch, Tag, Typography } from '@arco-design/web-react'
import { Button } from '@/components/ui'
import { useTranslation } from '@/i18n'
import {
  getAkskRouteCatalog,
  getAkskRoutePolicy,
  updateAkskRoutePolicy,
  type AkskRouteCatalogGroup,
} from '@/api/akskRoutePolicy'
import AkskRoutePicker from '@/components/platform/AkskRoutePicker'
import { showAlert } from '@/utils/notification'
import { extractApiErrorMessage } from '@/utils/apiError'

export default function AkskRoutePolicyPanel() {
  const { t } = useTranslation()
  const [enabled, setEnabled] = useState(false)
  const [selectedIds, setSelectedIds] = useState<string[]>([])
  const [groups, setGroups] = useState<AkskRouteCatalogGroup[]>([])
  const [total, setTotal] = useState(0)
  const [loading, setLoading] = useState(true)
  const [saving, setSaving] = useState(false)

  const load = useCallback(async () => {
    setLoading(true)
    try {
      const [catalogRes, policyRes] = await Promise.all([getAkskRouteCatalog(), getAkskRoutePolicy()])
      if (catalogRes.code === 200 && catalogRes.data) {
        setGroups(catalogRes.data.groups || [])
        setTotal(catalogRes.data.total || 0)
      }
      if (policyRes.code === 200 && policyRes.data) {
        setEnabled(!!policyRes.data.enabled)
        setSelectedIds(policyRes.data.routeIds || [])
      }
    } catch (e: unknown) {
      showAlert(extractApiErrorMessage(e, t('common.loadFailed')), 'error')
    } finally {
      setLoading(false)
    }
  }, [t])

  useEffect(() => {
    void load()
  }, [load])

  const save = async () => {
    if (enabled && selectedIds.length === 0) {
      showAlert(t('akskRoutePolicy.needSelection'), 'error')
      return
    }
    setSaving(true)
    try {
      const res = await updateAkskRoutePolicy({ enabled, routeIds: selectedIds })
      if (res.code !== 200) {
        showAlert(res.msg || t('common.saveFailed'), 'error')
        return
      }
      showAlert(t('common.saveSuccess'), 'success')
      await load()
    } catch (e: unknown) {
      showAlert(extractApiErrorMessage(e, t('common.saveFailed')), 'error')
    } finally {
      setSaving(false)
    }
  }

  return (
    <Card bordered={false} className="mb-4">
      <div className="flex flex-wrap items-start justify-between gap-3">
        <div>
          <Typography.Title heading={6} style={{ margin: 0 }}>
            {t('akskRoutePolicy.title')}
          </Typography.Title>
          <Typography.Paragraph type="secondary" style={{ margin: '6px 0 0' }}>
            {t('akskRoutePolicy.descCatalog', { total: String(total) })}
          </Typography.Paragraph>
        </div>
        <Space>
          <span style={{ color: 'var(--color-text-2)', fontSize: 13 }}>{t('akskRoutePolicy.enabled')}</span>
          <Switch checked={enabled} onChange={setEnabled} />
          <Button type="primary" loading={saving} disabled={loading} onClick={() => void save()}>
            {t('common.save')}
          </Button>
        </Space>
      </div>

      <div className="mt-4">
        <AkskRoutePicker
          groups={groups}
          selectedIds={selectedIds}
          onChange={setSelectedIds}
          disabled={loading || !enabled}
          emptyHint={t('akskRoutePolicy.catalogEmpty')}
        />
      </div>

      <div className="mt-3">
        <Tag color={enabled ? 'green' : 'orangered'}>
          {enabled ? t('akskRoutePolicy.statusOpen') : t('akskRoutePolicy.statusClosed')}
        </Tag>
        <Tag style={{ marginLeft: 8 }}>{selectedIds.length} selected</Tag>
      </div>
    </Card>
  )
}
