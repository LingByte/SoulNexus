import { useCallback, useEffect, useState } from 'react'
import { Drawer, Form, Input, Modal, Pagination, Select, Switch, Table, Tag } from '@arco-design/web-react'
import { Button } from '@/components/ui'
import {
  createIMChannel,
  deleteIMChannel,
  listIMChannels,
  listIMProviders,
  testIMChannel,
  updateIMChannel,
  type TenantIMChannelRow,
} from '@/api/imChannels'
import { useTranslation } from '@/i18n'
import { showAlert } from '@/utils/notification'
import { extractApiErrorMessage } from '@/utils/apiError'

const FormItem = Form.Item

export default function IMChannelsPanel({ embedded = false }: { embedded?: boolean }) {
  const { t } = useTranslation()
  const [loading, setLoading] = useState(true)
  const [rows, setRows] = useState<TenantIMChannelRow[]>([])
  const [total, setTotal] = useState(0)
  const [page, setPage] = useState(1)
  const pageSize = 10
  const [providers, setProviders] = useState<string[]>(['wecom', 'feishu'])
  const [drawerOpen, setDrawerOpen] = useState(false)
  const [editing, setEditing] = useState<TenantIMChannelRow | null>(null)
  const [saving, setSaving] = useState(false)
  const [form] = Form.useForm()

  const load = useCallback(async () => {
    setLoading(true)
    try {
      const res = await listIMChannels(page, pageSize)
      if (res.code !== 200) {
        showAlert(res.msg || t('profile.imChannelsLoadFailed'), 'error')
        return
      }
      setRows(res.data?.list ?? [])
      setTotal(res.data?.total ?? 0)
    } catch (e: unknown) {
      showAlert(extractApiErrorMessage(e, t('profile.imChannelsLoadFailed')), 'error')
    } finally {
      setLoading(false)
    }
  }, [page, t])

  useEffect(() => {
    void load()
  }, [load])

  useEffect(() => {
    void listIMProviders().then((res) => {
      if (res.code === 200 && res.data?.providers?.length) setProviders(res.data.providers)
    })
  }, [])

  const openCreate = () => {
    setEditing(null)
    form.resetFields()
    form.setFieldsValue({ enabled: true, provider: providers[0] || 'wecom' })
    setDrawerOpen(true)
  }

  const openEdit = (row: TenantIMChannelRow) => {
    setEditing(row)
    form.setFieldsValue({
      name: row.name,
      remark: row.remark || '',
      enabled: row.enabled,
      provider: row.provider,
      webhookUrl: '',
      secret: '',
    })
    setDrawerOpen(true)
  }

  const handleSave = async () => {
    try {
      const v = await form.validate()
      setSaving(true)
      const config: Record<string, unknown> = {
        webhookUrl: String(v.webhookUrl || '').trim(),
      }
      if (String(v.secret || '').trim()) config.secret = String(v.secret).trim()
      if (editing) {
        const body: { name?: string; remark?: string; enabled?: boolean; config?: Record<string, unknown> } = {
          name: String(v.name || '').trim(),
          remark: String(v.remark || '').trim(),
          enabled: Boolean(v.enabled),
        }
        if (config.webhookUrl || config.secret) body.config = config
        const res = await updateIMChannel(editing.id, body)
        if (res.code !== 200) {
          showAlert(res.msg || t('profile.imChannelsSaveFailed'), 'error')
          return
        }
      } else {
        const res = await createIMChannel({
          provider: String(v.provider || ''),
          name: String(v.name || '').trim(),
          remark: String(v.remark || '').trim(),
          enabled: Boolean(v.enabled),
          config,
        })
        if (res.code !== 200) {
          showAlert(res.msg || t('profile.imChannelsSaveFailed'), 'error')
          return
        }
      }
      showAlert(t('profile.imChannelsSaved'), 'success')
      setDrawerOpen(false)
      await load()
    } catch (e: unknown) {
      if (e && typeof e === 'object' && 'errorFields' in e) return
      showAlert(extractApiErrorMessage(e, t('profile.imChannelsSaveFailed')), 'error')
    } finally {
      setSaving(false)
    }
  }

  return (
    <div>
      <div className="mb-3 flex items-center justify-between gap-3">
        <div>
          {!embedded ? (
            <>
              <div className="text-base font-medium">{t('profile.imChannelsTitle')}</div>
              <div className="text-sm text-neutral-500">{t('profile.imChannelsDesc')}</div>
            </>
          ) : (
            <div className="text-sm text-neutral-500">{t('profile.imChannelsDesc')}</div>
          )}
        </div>
        <Button type="primary" onClick={openCreate}>{t('common.create')}</Button>
      </div>
      <Table
        loading={loading}
        data={rows}
        rowKey="id"
        pagination={false}
        columns={[
          { title: t('common.name'), dataIndex: 'name' },
          {
            title: t('profile.imProvider'),
            dataIndex: 'provider',
            width: 100,
            render: (v: string) => <Tag>{v === 'wecom' ? t('profile.imWecom') : t('profile.imFeishu')}</Tag>,
          },
          {
            title: t('common.status'),
            dataIndex: 'enabled',
            width: 90,
            render: (v: boolean) => (v ? <Tag color="green">{t('common.enabled')}</Tag> : <Tag>{t('common.disabled')}</Tag>),
          },
          {
            title: t('common.actions'),
            width: 220,
            render: (_: unknown, row: TenantIMChannelRow) => (
              <div className="flex gap-2">
                <Button size="mini" onClick={() => openEdit(row)}>{t('common.edit')}</Button>
                <Button
                  size="mini"
                  onClick={async () => {
                    const res = await testIMChannel(row.id)
                    showAlert(res.code === 200 ? t('profile.imChannelsTestOk') : res.msg || t('profile.imChannelsTestFailed'), res.code === 200 ? 'success' : 'error')
                  }}
                >
                  {t('common.test')}
                </Button>
                <Button
                  size="mini"
                  status="danger"
                  onClick={() => {
                    Modal.confirm({
                      title: t('common.confirmDelete'),
                      onOk: async () => {
                        const res = await deleteIMChannel(row.id)
                        if (res.code === 200) {
                          showAlert(t('common.deleted'), 'success')
                          await load()
                        } else showAlert(res.msg || t('common.deleteFailed'), 'error')
                      },
                    })
                  }}
                >
                  {t('common.delete')}
                </Button>
              </div>
            ),
          },
        ]}
      />
      <div className="mt-3 flex justify-end">
        <Pagination current={page} pageSize={pageSize} total={total} onChange={setPage} />
      </div>

      <Drawer
        width={480}
        title={editing ? t('profile.imChannelsEdit') : t('profile.imChannelsCreate')}
        visible={drawerOpen}
        onCancel={() => setDrawerOpen(false)}
        unmountOnExit
        footer={
          <div className="flex justify-end gap-2">
            <Button onClick={() => setDrawerOpen(false)}>{t('common.cancel')}</Button>
            <Button type="primary" loading={saving} onClick={() => void handleSave()}>
              {t('common.save')}
            </Button>
          </div>
        }
      >
        <Form form={form} layout="vertical">
          {!editing ? (
            <FormItem label={t('profile.imProvider')} field="provider" rules={[{ required: true }]}>
              <Select
                options={providers.map((p) => ({
                  value: p,
                  label: p === 'wecom' ? t('profile.imWecom') : t('profile.imFeishu'),
                }))}
              />
            </FormItem>
          ) : null}
          <FormItem label={t('common.name')} field="name" rules={[{ required: true }]}>
            <Input />
          </FormItem>
          <FormItem
            label="Webhook URL"
            field="webhookUrl"
            rules={editing ? [] : [{ required: true, message: 'Webhook URL required' }]}
          >
            <Input placeholder={editing ? t('profile.imWebhookKeepHint') : 'https://…'} />
          </FormItem>
          <FormItem label={t('profile.imSecret')} field="secret">
            <Input.Password placeholder={editing ? t('profile.imSecretKeepHint') : ''} />
          </FormItem>
          <FormItem label={t('common.remark')} field="remark">
            <Input />
          </FormItem>
          <FormItem label={t('common.enabled')} field="enabled" triggerPropName="checked">
            <Switch />
          </FormItem>
        </Form>
      </Drawer>
    </div>
  )
}
