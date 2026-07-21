import { useCallback, useEffect, useState } from 'react'
import { Modal, Space, Tag, Typography } from '@arco-design/web-react'
import { Button, Input, Card, TableEmpty } from '@/components/ui'
import { Loading } from '@/components/ui/loading'
import { IconCopy } from '@arco-design/web-react/icon'
import {
  createTenantCredentialPlatform,
  listTenantCredentialsPlatform,
} from '@/api/tenantCredentials'
import type { CredentialCreateResult, CredentialRow } from '@/api/credentials'
import { showAlert } from '@/utils/notification'
import { extractApiErrorMessage } from '@/utils/apiError'
import { useTranslation } from '@/i18n'

export default function TenantPlatformApiKeys({ tenantId }: { tenantId: string }) {
  const { t } = useTranslation()
  const [rows, setRows] = useState<CredentialRow[]>([])
  const [loading, setLoading] = useState(false)
  const [creating, setCreating] = useState(false)
  const [name, setName] = useState('')
  const [reveal, setReveal] = useState<CredentialCreateResult | null>(null)

  const load = useCallback(async () => {
    setLoading(true)
    try {
      const res = await listTenantCredentialsPlatform(tenantId, 1, 50)
      if (res.code === 200 && res.data) {
        const list = res.data.list || []
        setRows(list.filter((r) => r.kind === 'platform_bundle' || r.usesTenantAi))
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

  const onCreate = async () => {
    setCreating(true)
    try {
      const res = await createTenantCredentialPlatform(tenantId, {
        name: name.trim() || undefined,
      })
      if (res.code === 200 && res.data) {
        setReveal(res.data)
        setName('')
        void load()
      } else {
        showAlert(res.msg || t('common.createFailed'), 'error')
      }
    } catch (e: unknown) {
      showAlert(extractApiErrorMessage(e, t('common.createFailed')), 'error')
    } finally {
      setCreating(false)
    }
  }

  return (
    <Card bordered={false} style={{ marginTop: 24 }}>
      <Typography.Title heading={6} style={{ marginTop: 0 }}>
        {t('tenantAiConfig.platformApiKeysTitle')}
      </Typography.Title>
      <Typography.Paragraph type="secondary" style={{ fontSize: 13 }}>
        {t('tenantAiConfig.platformApiKeysHint')}
      </Typography.Paragraph>
      <Space wrap style={{ marginBottom: 16 }}>
        <Input
          placeholder={t('common.credentialNameExample')}
          style={{ width: 240 }}
          value={name}
          onChange={setName}
        />
        <Button type="primary" loading={creating} onClick={() => void onCreate()}>
          {t('tenantAiConfig.issuePlatformKey')}
        </Button>
      </Space>
      {loading ? (
        <Loading block />
      ) : rows.length === 0 ? (
        <TableEmpty description={t('tenantAiConfig.noPlatformKeys')} />
      ) : (
        <div style={{ overflowX: 'auto' }}>
          <table style={{ width: '100%', fontSize: 13 }}>
            <thead>
              <tr style={{ textAlign: 'left', borderBottom: '1px solid var(--color-border)' }}>
                <th style={{ padding: 8 }}>{t('common.name')}</th>
                <th style={{ padding: 8 }}>API Key</th>
                <th style={{ padding: 8 }}>{t('common.status')}</th>
              </tr>
            </thead>
            <tbody>
              {rows.map((r) => (
                <tr key={r.id} style={{ borderBottom: '1px solid var(--color-border)' }}>
                  <td style={{ padding: 8 }}>{r.name}</td>
                  <td style={{ padding: 8, fontFamily: 'monospace', fontSize: 12 }}>
                    {r.apiKeyPrefix || r.accessKey}
                    <Tag color="arcoblue" style={{ marginLeft: 8 }}>
                      {t('credentialAi.kindPlatform')}
                    </Tag>
                  </td>
                  <td style={{ padding: 8 }}>{r.status}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}

      <Modal
        title={t('common.credentialCreated')}
        visible={!!reveal}
        onCancel={() => setReveal(null)}
        footer={
          <Button type="primary" onClick={() => setReveal(null)}>
            {t('common.iSaved')}
          </Button>
        }
      >
        {reveal ? (
          <Space direction="vertical" style={{ width: '100%' }}>
            <Typography.Text type="warning">{t('common.apiKeyWarning')}</Typography.Text>
            <Typography.Paragraph copyable style={{ fontFamily: 'monospace', fontSize: 12, wordBreak: 'break-all' }}>
              {reveal.apiKey}
            </Typography.Paragraph>
            <Button icon={<IconCopy />} onClick={() => void navigator.clipboard.writeText(reveal.apiKey)}>
              {t('common.copy')}
            </Button>
          </Space>
        ) : null}
      </Modal>
    </Card>
  )
}
