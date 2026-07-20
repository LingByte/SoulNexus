import { useCallback, useEffect, useState } from 'react'
import { Drawer, Form, Input, Modal, Pagination, Select, Switch, Table, Tag } from '@arco-design/web-react'
import { Button } from '@/components/ui'
import {
  createDialogChannel,
  deleteDialogChannel,
  listDialogChannelProviders,
  listDialogChannels,
  updateDialogChannel,
  type TenantDialogChannelRow,
} from '@/api/dialog'
import { listAssistants, type AssistantRow } from '@/api/assistants'
import { useTranslation } from '@/i18n'
import { showAlert } from '@/utils/notification'
import { extractApiErrorMessage } from '@/utils/apiError'

const FormItem = Form.Item

export default function DialogChannelsPanel({ embedded = false }: { embedded?: boolean }) {
  const { t } = useTranslation()
  const [loading, setLoading] = useState(true)
  const [rows, setRows] = useState<TenantDialogChannelRow[]>([])
  const [total, setTotal] = useState(0)
  const [page, setPage] = useState(1)
  const pageSize = 10
  const [providers, setProviders] = useState<string[]>(['wecom_app', 'wechat_oa'])
  const [assistants, setAssistants] = useState<AssistantRow[]>([])
  const [drawerOpen, setDrawerOpen] = useState(false)
  const [editing, setEditing] = useState<TenantDialogChannelRow | null>(null)
  const [saving, setSaving] = useState(false)
  const [form] = Form.useForm()
  const provider = Form.useWatch('provider', form) as string

  const load = useCallback(async () => {
    setLoading(true)
    try {
      const res = await listDialogChannels(page, pageSize)
      if (res.code !== 200) {
        showAlert(res.msg || t('profile.dialogChannelsLoadFailed'), 'error')
        return
      }
      setRows(res.data?.list ?? [])
      setTotal(res.data?.total ?? 0)
    } catch (e: unknown) {
      showAlert(extractApiErrorMessage(e, t('profile.dialogChannelsLoadFailed')), 'error')
    } finally {
      setLoading(false)
    }
  }, [page, t])

  useEffect(() => {
    void load()
  }, [load])

  useEffect(() => {
    void listDialogChannelProviders().then((res) => {
      if (res.code === 200 && res.data?.providers?.length) setProviders(res.data.providers)
    })
    void listAssistants(1, 100).then((res) => {
      if (res.code === 200) setAssistants(res.data?.list ?? [])
    })
  }, [])

  const openCreate = () => {
    setEditing(null)
    form.resetFields()
    form.setFieldsValue({ enabled: true, provider: providers[0] || 'wecom_app' })
    setDrawerOpen(true)
  }

  const openEdit = (row: TenantDialogChannelRow) => {
    setEditing(row)
    form.setFieldsValue({
      name: row.name,
      code: row.code,
      remark: row.remark || '',
      enabled: row.enabled,
      provider: row.provider,
      assistantId: row.assistantId,
    })
    setDrawerOpen(true)
  }

  const handleSave = async () => {
    try {
      const v = await form.validate()
      setSaving(true)
      const config: Record<string, unknown> = {}
      const put = (k: string, val: unknown) => {
        const s = String(val || '').trim()
        if (s) config[k] = s
      }
      if (String(v.provider) === 'wecom_app') {
        put('corpId', v.corpId)
        put('agentId', v.agentId)
        put('secret', v.secret)
        put('token', v.token)
        put('encodingAESKey', v.encodingAESKey)
      } else {
        put('appId', v.appId)
        put('appSecret', v.appSecret)
        put('token', v.token)
        put('encodingAESKey', v.encodingAESKey)
      }
      if (editing) {
        const body: {
          name?: string
          remark?: string
          enabled?: boolean
          assistantId?: string
          config?: Record<string, unknown>
        } = {
          name: String(v.name || '').trim(),
          remark: String(v.remark || '').trim(),
          enabled: Boolean(v.enabled),
          assistantId: String(v.assistantId || ''),
        }
        if (Object.keys(config).length) body.config = config
        const res = await updateDialogChannel(editing.id, body)
        if (res.code !== 200) {
          showAlert(res.msg || t('profile.dialogChannelsSaveFailed'), 'error')
          return
        }
      } else {
        const res = await createDialogChannel({
          provider: String(v.provider || ''),
          code: String(v.code || '').trim() || undefined,
          name: String(v.name || '').trim(),
          assistantId: String(v.assistantId || ''),
          remark: String(v.remark || '').trim(),
          enabled: Boolean(v.enabled),
          config,
        })
        if (res.code !== 200) {
          showAlert(res.msg || t('profile.dialogChannelsSaveFailed'), 'error')
          return
        }
      }
      showAlert(t('profile.dialogChannelsSaved'), 'success')
      setDrawerOpen(false)
      await load()
    } catch (e: unknown) {
      if (e && typeof e === 'object' && 'errorFields' in e) return
      showAlert(extractApiErrorMessage(e, t('profile.dialogChannelsSaveFailed')), 'error')
    } finally {
      setSaving(false)
    }
  }

  const handleDelete = (row: TenantDialogChannelRow) => {
    Modal.confirm({
      title: t('profile.dialogChannelsDeleteConfirm'),
      onOk: async () => {
        const res = await deleteDialogChannel(row.id)
        if (res.code !== 200) {
          showAlert(res.msg || t('profile.dialogChannelsDeleteFailed'), 'error')
          return
        }
        showAlert(t('profile.dialogChannelsDeleted'), 'success')
        await load()
      },
    })
  }

  return (
    <div>
      {!embedded && (
        <div className="mb-4 flex items-center justify-between">
          <div>
            <div className="text-base font-medium">{t('profile.tabDialogChannels')}</div>
            <div className="text-sm text-neutral-500">{t('profile.dialogChannelsDesc')}</div>
          </div>
          <Button type="primary" onClick={openCreate}>
            {t('common.create')}
          </Button>
        </div>
      )}
      {embedded && (
        <div className="mb-3 flex justify-end">
          <Button type="primary" onClick={openCreate}>
            {t('common.create')}
          </Button>
        </div>
      )}
      <Table
        loading={loading}
        rowKey="id"
        pagination={false}
        data={rows}
        columns={[
          { title: t('common.name'), dataIndex: 'name' },
          { title: 'Code', dataIndex: 'code' },
          {
            title: t('common.type'),
            dataIndex: 'provider',
            render: (p: string) => <Tag>{p}</Tag>,
          },
          { title: 'Assistant', dataIndex: 'assistantId' },
          {
            title: t('common.status'),
            dataIndex: 'enabled',
            render: (en: boolean) => (
              <Tag color={en ? 'green' : 'gray'}>{en ? t('common.enabled') : t('common.disabled')}</Tag>
            ),
          },
          {
            title: t('common.actions'),
            render: (_: unknown, row: TenantDialogChannelRow) => (
              <div className="flex gap-2">
                <Button size="mini" onClick={() => openEdit(row)}>
                  {t('common.edit')}
                </Button>
                <Button size="mini" status="danger" onClick={() => handleDelete(row)}>
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
        width={520}
        title={editing ? t('common.edit') : t('common.create')}
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
          <FormItem label={t('common.type')} field="provider" disabled={!!editing} rules={[{ required: true }]}>
            <Select options={providers.map((p) => ({ label: p, value: p }))} />
          </FormItem>
          <FormItem label={t('common.name')} field="name" rules={[{ required: true }]}>
            <Input />
          </FormItem>
          {!editing && (
            <FormItem label="Code" field="code">
              <Input placeholder="wecom_app" />
            </FormItem>
          )}
          <FormItem label="Assistant" field="assistantId" rules={[{ required: true }]}>
            <Select
              options={assistants.map((a) => ({
                label: a.name || String(a.id),
                value: String(a.id),
              }))}
            />
          </FormItem>
          {provider === 'wecom_app' ? (
            <>
              <FormItem label="CorpID" field="corpId" rules={editing ? [] : [{ required: true }]}>
                <Input />
              </FormItem>
              <FormItem label="AgentID" field="agentId" rules={editing ? [] : [{ required: true }]}>
                <Input />
              </FormItem>
              <FormItem label="Secret" field="secret" rules={editing ? [] : [{ required: true }]}>
                <Input.Password />
              </FormItem>
            </>
          ) : (
            <>
              <FormItem label="AppID" field="appId" rules={editing ? [] : [{ required: true }]}>
                <Input />
              </FormItem>
              <FormItem label="AppSecret" field="appSecret" rules={editing ? [] : [{ required: true }]}>
                <Input.Password />
              </FormItem>
            </>
          )}
          <FormItem label="Token" field="token" rules={editing ? [] : [{ required: true }]}>
            <Input />
          </FormItem>
          <FormItem label="EncodingAESKey" field="encodingAESKey" rules={editing ? [] : [{ required: true }]}>
            <Input />
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
