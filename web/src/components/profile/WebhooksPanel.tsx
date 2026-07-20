import { useCallback, useEffect, useMemo, useState } from 'react'
import { Drawer, Form, Input, Modal, Pagination, Select, Switch, Table, Tag } from '@arco-design/web-react'
import { Button } from '@/components/ui'
import {
  createWebhook,
  deleteWebhook,
  listWebhookEvents,
  listWebhooks,
  testWebhook,
  updateWebhook,
  type TenantWebhookRow,
} from '@/api/webhooks'
import { useTranslation } from '@/i18n'
import { showAlert } from '@/utils/notification'
import { extractApiErrorMessage } from '@/utils/apiError'

const FormItem = Form.Item

function parseEvents(raw: unknown): string[] {
  if (Array.isArray(raw)) return raw.map(String)
  if (typeof raw === 'string') {
    try {
      const v = JSON.parse(raw)
      return Array.isArray(v) ? v.map(String) : []
    } catch {
      return []
    }
  }
  return []
}

export default function WebhooksPanel({ embedded = false }: { embedded?: boolean }) {
  const { t } = useTranslation()
  const [loading, setLoading] = useState(true)
  const [rows, setRows] = useState<TenantWebhookRow[]>([])
  const [total, setTotal] = useState(0)
  const [page, setPage] = useState(1)
  const pageSize = 10
  const [eventsCatalog, setEventsCatalog] = useState<string[]>([])
  const [drawerOpen, setDrawerOpen] = useState(false)
  const [editing, setEditing] = useState<TenantWebhookRow | null>(null)
  const [saving, setSaving] = useState(false)
  const [form] = Form.useForm()

  const load = useCallback(async () => {
    setLoading(true)
    try {
      const res = await listWebhooks(page, pageSize)
      if (res.code !== 200) {
        showAlert(res.msg || t('profile.webhooksLoadFailed'), 'error')
        return
      }
      const list = (res.data?.list ?? []).map((r) => ({ ...r, events: parseEvents(r.events) }))
      setRows(list)
      setTotal(res.data?.total ?? 0)
    } catch (e: unknown) {
      showAlert(extractApiErrorMessage(e, t('profile.webhooksLoadFailed')), 'error')
    } finally {
      setLoading(false)
    }
  }, [page, t])

  useEffect(() => {
    void load()
  }, [load])

  useEffect(() => {
    void listWebhookEvents().then((res) => {
      if (res.code === 200 && res.data?.events) {
        setEventsCatalog(res.data.events)
      }
    })
  }, [])

  const eventOptions = useMemo(
    () => eventsCatalog.map((e) => ({ label: e, value: e })),
    [eventsCatalog],
  )

  const openCreate = () => {
    setEditing(null)
    form.resetFields()
    form.setFieldsValue({ enabled: true, events: eventsCatalog, formatType: 'json' })
    setDrawerOpen(true)
  }

  const openEdit = (row: TenantWebhookRow) => {
    setEditing(row)
    form.setFieldsValue({
      name: row.name,
      url: row.url,
      secret: row.secret || '',
      events: parseEvents(row.events),
      enabled: row.enabled,
      formatType: 'json',
    })
    setDrawerOpen(true)
  }

  const handleSave = async () => {
    try {
      const v = await form.validate()
      setSaving(true)
      const body = {
        name: String(v.name || '').trim(),
        url: String(v.url || '').trim(),
        secret: String(v.secret || '').trim() || undefined,
        events: (v.events as string[]) || [],
        enabled: Boolean(v.enabled),
      }
      const res = editing ? await updateWebhook(editing.id, body) : await createWebhook(body)
      if (res.code !== 200) {
        showAlert(res.msg || t('profile.webhooksSaveFailed'), 'error')
        return
      }
      showAlert(t('profile.webhooksSaved'), 'success')
      setDrawerOpen(false)
      await load()
    } catch (e: unknown) {
      if (e && typeof e === 'object' && 'errorFields' in e) return
      showAlert(extractApiErrorMessage(e, t('profile.webhooksSaveFailed')), 'error')
    } finally {
      setSaving(false)
    }
  }

  const handleDelete = (row: TenantWebhookRow) => {
    Modal.confirm({
      title: t('profile.webhooksDeleteConfirm'),
      onOk: async () => {
        const res = await deleteWebhook(row.id)
        if (res.code !== 200) {
          showAlert(res.msg || t('profile.webhooksDeleteFailed'), 'error')
          return
        }
        showAlert(t('profile.webhooksDeleted'), 'success')
        await load()
      },
    })
  }

  const handleTest = async (row: TenantWebhookRow) => {
    const res = await testWebhook(row.id)
    if (res.code !== 200) {
      showAlert(res.msg || t('profile.webhooksTestFailed'), 'error')
      return
    }
    showAlert(t('profile.webhooksTestSent'), 'success')
  }

  const columns = [
    { title: t('profile.webhookName'), dataIndex: 'name' },
    {
      title: t('profile.webhookFormat'),
      width: 100,
      render: () => 'JSON',
    },
    {
      title: t('profile.webhookUrl'),
      dataIndex: 'url',
      ellipsis: true,
    },
    {
      title: t('profile.webhookEnabled'),
      dataIndex: 'enabled',
      width: 90,
      render: (v: boolean) =>
        v ? <Tag color="green">{t('profile.webhookEnabledOn')}</Tag> : <Tag>{t('profile.webhookEnabledOff')}</Tag>,
    },
    {
      title: t('profile.webhookActions'),
      width: 200,
      render: (_: unknown, row: TenantWebhookRow) => (
        <div className="flex flex-wrap gap-2">
          <Button size="mini" type="text" onClick={() => openEdit(row)}>
            {t('common.edit')}
          </Button>
          <Button size="mini" type="text" onClick={() => void handleTest(row)}>
            {t('profile.webhookTest')}
          </Button>
          <Button size="mini" type="text" status="danger" onClick={() => handleDelete(row)}>
            {t('common.delete')}
          </Button>
        </div>
      ),
    },
  ]

  return (
    <div className={embedded ? undefined : 'rounded-xl border border-border bg-card p-4'}>
      <div className="mb-4 flex items-center justify-between gap-3">
        <div>
          {!embedded ? (
            <>
              <div className="text-base font-semibold text-foreground">{t('profile.navWebhooks')}</div>
              <div className="mt-1 text-sm text-muted-foreground">{t('profile.webhooksDesc')}</div>
            </>
          ) : (
            <div className="text-sm text-muted-foreground">{t('profile.webhooksDesc')}</div>
          )}
        </div>
        <Button type="primary" onClick={openCreate}>
          {t('profile.webhookAdd')}
        </Button>
      </div>
      <Table rowKey="id" loading={loading} columns={columns} data={rows} pagination={false} size="small" />
      {total > pageSize && (
        <div className="mt-4 flex justify-end">
          <Pagination total={total} current={page} pageSize={pageSize} onChange={setPage} size="small" />
        </div>
      )}

      <Drawer
        width={480}
        title={editing ? t('profile.webhookEdit') : t('profile.webhookAdd')}
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
          <FormItem label={t('profile.webhookName')} field="name" rules={[{ required: true }]}>
            <Input />
          </FormItem>
          <FormItem label={t('profile.webhookFormat')} field="formatType">
            <Select disabled options={[{ label: 'JSON', value: 'json' }]} />
          </FormItem>
          <FormItem label={t('profile.webhookUrl')} field="url" rules={[{ required: true, type: 'url' }]}>
            <Input placeholder="https://example.com/hook" />
          </FormItem>
          <FormItem label={t('profile.webhookSecret')} field="secret">
            <Input.Password placeholder={t('profile.webhookSecretHint')} />
          </FormItem>
          <FormItem label={t('profile.webhookEvents')} field="events">
            <Select mode="multiple" options={eventOptions} allowClear />
          </FormItem>
          <FormItem label={t('profile.webhookEnabled')} field="enabled" triggerPropName="checked">
            <Switch />
          </FormItem>
        </Form>
      </Drawer>
    </div>
  )
}
