import { useCallback, useEffect, useState } from 'react'
import { Space, Typography } from '@arco-design/web-react'
import { Button, Link, Select, Card } from '@/components/ui'
import { Loading } from '@/components/ui/loading'
import { IconLeft } from '@arco-design/web-react/icon'
import { useNavigate, useParams } from 'react-router-dom'
import { useTranslation } from '@/i18n'
import BaseLayout from '@/components/Layout/BaseLayout'
import { getTenant, updateTenantPlatform } from '@/api/tenants'
import TenantPlatformApiKeys from '@/components/credentials/TenantPlatformApiKeys'
import TenantAiPoolGrants from '@/components/tenant/TenantAiPoolGrants'
import { showAlert } from '@/utils/notification'

export default function TenantAiConfig() {
  const { tenantId } = useParams<{ tenantId: string }>()
  const navigate = useNavigate()
  const { t } = useTranslation()
  const [tenantName, setTenantName] = useState('')
  const [loading, setLoading] = useState(true)
  const [saving, setSaving] = useState(false)
  const [voiceMode, setVoiceMode] = useState<'pipeline' | 'realtime'>('pipeline')

  const load = useCallback(async () => {
    if (!tenantId) return
    setLoading(true)
    try {
      const r = await getTenant(tenantId)
      if (r.code !== 200 || !r.data?.tenant) {
        showAlert(r.msg || t('tenantAiConfig.loadFailed'), 'error')
        navigate('/tenant-management', { replace: true })
        return
      }
      const tenant = r.data.tenant
      setTenantName(tenant.name)
      setVoiceMode(tenant.voiceMode === 'realtime' ? 'realtime' : 'pipeline')
    } finally {
      setLoading(false)
    }
  }, [navigate, tenantId, t])

  useEffect(() => {
    void load()
  }, [load])

  const onSaveVoiceMode = async () => {
    if (!tenantId) return
    setSaving(true)
    try {
      const r = await updateTenantPlatform(tenantId, { voiceMode })
      if (r.code !== 200) {
        showAlert(r.msg || t('tenantAiConfig.saveFailed'), 'error')
        return
      }
      showAlert(t('tenantAiConfig.saveSuccess'), 'success')
    } finally {
      setSaving(false)
    }
  }

  const title = tenantName ? `${t('tenantAiConfig.title')} — ${tenantName}` : t('tenantAiConfig.title')

  return (
    <BaseLayout
      title={title}
      description={t('tenantAiConfig.descriptionTransit')}
      actions={
        <Link to="/tenant-management">
          <Button icon={<IconLeft />}>{t('tenantAiConfig.backToList')}</Button>
        </Link>
      }
    >
      <Card bordered={false} style={{ maxWidth: 960 }}>
        {loading ? (
          <Loading block tip={t('tenantAiConfig.loadingConfig')} />
        ) : (
          <Space direction="vertical" size={16} style={{ width: '100%' }}>
            <Typography.Paragraph type="secondary" style={{ marginBottom: 0, fontSize: 13 }}>
              {t('tenantAiConfig.transitHint')}
            </Typography.Paragraph>
            <div>
              <Typography.Text style={{ display: 'block', marginBottom: 8 }}>
                <strong>{t('tenantAiConfig.voiceMode')}</strong>
              </Typography.Text>
              <Select
                style={{ width: 360 }}
                value={voiceMode}
                options={[
                  { value: 'pipeline', label: t('tenantAiConfig.pipelineLabel') },
                  { value: 'realtime', label: t('tenantAiConfig.realtimeLabel') },
                ]}
                onChange={(v) => setVoiceMode(v as 'pipeline' | 'realtime')}
              />
              <Typography.Paragraph type="secondary" style={{ marginBottom: 0, marginTop: 8, fontSize: 13 }}>
                {voiceMode === 'realtime' ? t('tenantAiConfig.realtimeHint') : t('tenantAiConfig.pipelineHint')}
              </Typography.Paragraph>
              <Button
                type="outline"
                size="small"
                className="mt-3"
                loading={saving}
                onClick={() => void onSaveVoiceMode()}
              >
                {t('tenantAiConfig.saveVoiceMode')}
              </Button>
            </div>
          </Space>
        )}
      </Card>
      {tenantId ? <TenantAiPoolGrants tenantId={Number(tenantId)} /> : null}
      {tenantId ? <TenantPlatformApiKeys tenantId={tenantId} /> : null}
    </BaseLayout>
  )
}
