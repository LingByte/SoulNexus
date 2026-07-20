import { useCallback, useEffect, useState } from 'react'
import {
  Drawer,
  Form,
  Modal,
  Space,
  Switch,
  Tag,
} from '@arco-design/web-react'
import { Shield, Plus, Pencil, Trash2 } from 'lucide-react'
import { Button, Input, Select, DataList } from '@/components/ui'
import type { DataListColumn } from '@/components/ui'
import BaseLayout from '@/components/Layout/BaseLayout'
import { useTranslation } from '@/i18n'
import {
  createSystemConfig,
  deleteSystemConfig,
  listSystemConfigs,
  updateSystemConfig,
  type SystemConfigRow,
} from '@/api/systemConfigs'
import { showAlert } from '@/utils/notification'
import { extractApiErrorMessage } from '@/utils/apiError'
import AkskRoutePicker from '@/components/platform/AkskRoutePicker'
import {
  getAkskRouteCatalog,
  getAkskRoutePolicy,
  updateAkskRoutePolicy,
  type AkskRouteCatalogGroup,
} from '@/api/akskRoutePolicy'

const formatOpts = [
  { label: 'text', value: 'text' },
  { label: 'json', value: 'json' },
  { label: 'yaml', value: 'yaml' },
  { label: 'int', value: 'int' },
  { label: 'float', value: 'float' },
  { label: 'bool', value: 'bool' },
]

function boolTag(v?: boolean, yes = '是', no = '否') {
  return v ? <Tag color="green">{yes}</Tag> : <Tag>{no}</Tag>
}

function ConfigKvForm({
  mode,
  fixedKey,
}: {
  mode: 'create' | 'edit'
  fixedKey?: string
}) {
  const { t } = useTranslation()

  return (
    <div className="space-y-4">
      <div className="flex items-center gap-3">
        <label className="w-20 shrink-0 text-right text-sm text-neutral-500">Key</label>
        {mode === 'create' ? (
          <Form.Item field="key" noStyle rules={[{ required: true, message: t('systemConfigs.keyRequired') }]}>
            <Input placeholder="TENANT_SELF_REGISTER" className="flex-1" />
          </Form.Item>
        ) : (
          <Input value={fixedKey || ''} disabled className="flex-1" />
        )}
      </div>
      <div className="flex items-center gap-3">
        <label className="w-20 shrink-0 text-right text-sm text-neutral-500">{t('systemConfigs.value')}</label>
        <Form.Item field="value" noStyle className="flex-1">
          <Input placeholder="true" className="flex-1" />
        </Form.Item>
      </div>
      <div className="flex items-center gap-3">
        <label className="w-20 shrink-0 text-right text-sm text-neutral-500">{t('systemConfigs.desc')}</label>
        <Form.Item field="desc" noStyle className="flex-1">
          <Input className="flex-1" />
        </Form.Item>
      </div>
      <div className="flex items-center gap-3">
        <label className="w-20 shrink-0 text-right text-sm text-neutral-500">{t('systemConfigs.format')}</label>
        <Form.Item field="format" noStyle className="flex-1">
          <Select options={formatOpts} style={{ width: 200 }} />
        </Form.Item>
      </div>
      <div className="flex items-center gap-3">
        <label className="w-20 shrink-0 text-right text-sm text-neutral-500">{t('systemConfigs.autoload')}</label>
        <Form.Item field="autoload" noStyle triggerPropName="checked">
          <Switch size="small" />
        </Form.Item>
      </div>
      <div className="flex items-center gap-3">
        <label className="w-20 shrink-0 text-right text-sm text-neutral-500">{t('systemConfigs.public')}</label>
        <Form.Item field="public" noStyle triggerPropName="checked">
          <Switch size="small" />
        </Form.Item>
      </div>
    </div>
  )
}

export default function SystemConfigs() {
  const { t } = useTranslation()
  const [rows, setRows] = useState<SystemConfigRow[]>([])
  const [total, setTotal] = useState(0)
  const [page, setPage] = useState(1)
  const [search, setSearch] = useState('')
  const [loading, setLoading] = useState(false)

  const [createOpen, setCreateOpen] = useState(false)
  const [createForm] = Form.useForm()
  const [creating, setCreating] = useState(false)

  const [editOpen, setEditOpen] = useState(false)
  const [editForm] = Form.useForm()
  const [editing, setEditing] = useState(false)
  const [editRow, setEditRow] = useState<SystemConfigRow | null>(null)

  const [delTarget, setDelTarget] = useState<SystemConfigRow | null>(null)
  const [delLoading, setDelLoading] = useState(false)

  const [routeOpen, setRouteOpen] = useState(false)
  const [routeEnabled, setRouteEnabled] = useState(false)
  const [routeSelectedIds, setRouteSelectedIds] = useState<string[]>([])
  const [routeGroups, setRouteGroups] = useState<AkskRouteCatalogGroup[]>([])
  const [routeLoading, setRouteLoading] = useState(true)
  const [routeSaving, setRouteSaving] = useState(false)

  const pageSize = 10

  const load = useCallback(async () => {
    setLoading(true)
    try {
      const res = await listSystemConfigs(page, pageSize, search)
      if (res.code === 200 && res.data) {
        setRows(res.data.list || [])
        setTotal(res.data.total || 0)
      } else {
        showAlert(res.msg || t('common.loadFailed'), 'error')
      }
    } catch (e: unknown) {
      showAlert(extractApiErrorMessage(e, t('common.loadFailed')), 'error')
    } finally {
      setLoading(false)
    }
  }, [page, search, t])

  useEffect(() => {
    void load()
  }, [load])

  const loadRoutePolicy = useCallback(async () => {
    setRouteLoading(true)
    try {
      const [catalogRes, policyRes] = await Promise.all([getAkskRouteCatalog(), getAkskRoutePolicy()])
      if (catalogRes.code === 200 && catalogRes.data) {
        setRouteGroups(catalogRes.data.groups || [])
      }
      if (policyRes.code === 200 && policyRes.data) {
        setRouteEnabled(!!policyRes.data.enabled)
        setRouteSelectedIds(policyRes.data.routeIds || [])
      }
    } catch (e: unknown) {
      showAlert(extractApiErrorMessage(e, t('common.loadFailed')), 'error')
    } finally {
      setRouteLoading(false)
    }
  }, [t])

  useEffect(() => {
    if (routeOpen) void loadRoutePolicy()
  }, [routeOpen, loadRoutePolicy])

  const saveRoutePolicy = async () => {
    if (routeEnabled && routeSelectedIds.length === 0) {
      showAlert(t('akskRoutePolicy.needSelection'), 'error')
      return
    }
    setRouteSaving(true)
    try {
      const res = await updateAkskRoutePolicy({ enabled: routeEnabled, routeIds: routeSelectedIds })
      if (res.code !== 200) {
        showAlert(res.msg || t('common.saveFailed'), 'error')
        return
      }
      showAlert(t('common.saveSuccess'), 'success')
      setRouteOpen(false)
    } catch (e: unknown) {
      showAlert(extractApiErrorMessage(e, t('common.saveFailed')), 'error')
    } finally {
      setRouteSaving(false)
    }
  }

  const submitCreate = async () => {
    setCreating(true)
    try {
      const v = await createForm.validate()
      const res = await createSystemConfig({
        key: String(v.key || '').trim(),
        desc: String(v.desc || '').trim() || undefined,
        value: String(v.value ?? ''),
        format: (v.format as string) || 'text',
        autoload: Boolean(v.autoload),
        public: Boolean(v.public),
      })
      if (res.code === 200) {
        showAlert(t('systemConfigs.createSuccess'), 'success')
        setCreateOpen(false)
        createForm.resetFields()
        void load()
      } else {
        showAlert(res.msg || t('common.failed'), 'error')
      }
    } catch {
      /* validation */
    } finally {
      setCreating(false)
    }
  }

  const openEdit = (r: SystemConfigRow) => {
    setEditRow(r)
    editForm.setFieldsValue({
      desc: r.desc || '',
      value: r.value ?? '',
      format: r.format || 'text',
      autoload: Boolean(r.autoload),
      public: Boolean(r.public),
    })
    setEditOpen(true)
  }

  const submitEdit = async () => {
    if (!editRow?.id) return
    setEditing(true)
    try {
      const v = await editForm.validate()
      const res = await updateSystemConfig(editRow.id, {
        desc: String(v.desc || '').trim(),
        value: String(v.value ?? ''),
        format: (v.format as string) || 'text',
        autoload: Boolean(v.autoload),
        public: Boolean(v.public),
      })
      if (res.code === 200) {
        showAlert(t('common.saveSuccess'), 'success')
        setEditOpen(false)
        setEditRow(null)
        void load()
      } else {
        showAlert(res.msg || t('common.saveFailed'), 'error')
      }
    } catch {
      /* validation */
    } finally {
      setEditing(false)
    }
  }

  const confirmDelete = async () => {
    if (!delTarget?.id) return
    setDelLoading(true)
    try {
      const res = await deleteSystemConfig(delTarget.id)
      if (res.code === 200) {
        showAlert(t('common.deleteSuccess'), 'success')
        setDelTarget(null)
        void load()
      } else {
        showAlert(res.msg || t('common.deleteFailed'), 'error')
      }
    } catch (e: unknown) {
      showAlert(extractApiErrorMessage(e, t('common.deleteFailed')), 'error')
    } finally {
      setDelLoading(false)
    }
  }

  const configColumns: DataListColumn<SystemConfigRow>[] = [
    {
      key: 'key',
      title: 'Key',
      render: (_, r) => (
        <div className="min-w-0">
          <div className="truncate font-mono text-sm font-medium text-neutral-900">{r.key || '—'}</div>
          {r.desc && (
            <div className="mt-0.5 truncate text-xs text-neutral-500">{r.desc}</div>
          )}
        </div>
      ),
    },
    {
      key: 'value',
      title: t('systemConfigs.value'),
      render: (_, r) => (
        <div className="min-w-0 max-w-[280px]">
          <div className="truncate text-sm text-neutral-700" title={String(r.value ?? '')}>
            {String(r.value ?? '') || '—'}
          </div>
        </div>
      ),
    },
    {
      key: 'meta',
      width: 180,
      render: (_, r) => (
        <div className="flex items-center gap-2 text-xs text-neutral-500">
          <Tag className="!rounded-full !text-xs">{r.format || 'text'}</Tag>
          {boolTag(r.autoload, '自动加载', '—')}
          {boolTag(r.public, '公开', '—')}
        </div>
      ),
    },
    {
      key: 'actions',
      width: 120,
      align: 'right',
      render: (_, r) => (
        <Space size="mini">
          <Button size="mini" icon={<Pencil size={12} />} onClick={() => openEdit(r)}>
            {t('common.edit')}
          </Button>
          <Button size="mini" status="danger" icon={<Trash2 size={12} />} onClick={() => setDelTarget(r)}>
            {t('common.delete')}
          </Button>
        </Space>
      ),
    },
  ]

  return (
    <BaseLayout
      title={t('systemConfigs.title')}
      description={t('systemConfigs.description')}
    >
      <div className="mx-auto max-w-6xl space-y-3 p-4">
        <div className="rounded-xl border border-border bg-card px-4 py-3">
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-3">
              <div className="flex h-8 w-8 items-center justify-center rounded-lg bg-amber-50 text-amber-600">
                <Shield size={16} />
              </div>
              <div className="text-sm font-medium text-neutral-900">{t('akskRoutePolicy.title')}</div>
            </div>
            <Button type="primary" icon={<Shield size={14} />} onClick={() => setRouteOpen(true)}>
              {t('akskRoutePolicy.enabled')}
            </Button>
          </div>
        </div>

        <DataList
          data={rows as unknown as (SystemConfigRow & Record<string, unknown>)[]}
          columns={configColumns}
          loading={loading}
          rowKey="id"
          emptyText={t('common.noData')}
          pagination={total > 0 ? { current: page, pageSize, total, onChange: setPage } : null}
          header={
            <div className="flex items-center justify-between gap-3">
              <span className="text-sm font-medium text-neutral-900">{t('systemConfigs.title')}</span>
              <div className="flex items-center gap-2">
                <Input.Search
                  allowClear
                  placeholder={t('systemConfigs.searchPlaceholder')}
                  style={{ width: 240 }}
                  onSearch={(v) => {
                    setPage(1)
                    setSearch(v)
                  }}
                />
                <Button type="primary" icon={<Plus size={14} />} onClick={() => setCreateOpen(true)}>
                  {t('systemConfigs.create')}
                </Button>
              </div>
            </div>
          }
        />
      </div>

      <Drawer
        title={t('akskRoutePolicy.title')}
        width={640}
        visible={routeOpen}
        onCancel={() => setRouteOpen(false)}
        footer={
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-3">
              <span className="text-sm text-neutral-500">{t('akskRoutePolicy.enabled')}</span>
              <Switch checked={routeEnabled} onChange={setRouteEnabled} />
              <Tag color={routeEnabled ? 'green' : 'orangered'} className="!rounded-full">
                {routeEnabled ? t('akskRoutePolicy.statusOpen') : t('akskRoutePolicy.statusClosed')}
              </Tag>
              <Tag className="!rounded-full">{routeSelectedIds.length} selected</Tag>
            </div>
            <div className="flex gap-2">
              <Button type="outline" onClick={() => setRouteOpen(false)}>
                {t('common.cancel')}
              </Button>
              <Button type="primary" loading={routeSaving} disabled={routeLoading} onClick={() => void saveRoutePolicy()}>
                {t('common.save')}
              </Button>
            </div>
          </div>
        }
      >
        <AkskRoutePicker
          groups={routeGroups}
          selectedIds={routeSelectedIds}
          onChange={setRouteSelectedIds}
          disabled={routeLoading || !routeEnabled}
          emptyHint={t('akskRoutePolicy.catalogEmpty')}
        />
      </Drawer>

      <Drawer
        title={t('systemConfigs.create')}
        width={480}
        visible={createOpen}
        onCancel={() => setCreateOpen(false)}
        footer={
          <div className="flex justify-end gap-2">
            <Button type="outline" onClick={() => setCreateOpen(false)}>
              {t('common.cancel')}
            </Button>
            <Button type="primary" loading={creating} onClick={() => void submitCreate()}>
              {t('common.save')}
            </Button>
          </div>
        }
      >
        <Form form={createForm} initialValues={{ format: 'text', autoload: false, public: false }}>
          <ConfigKvForm mode="create" />
        </Form>
      </Drawer>

      <Drawer
        title={t('systemConfigs.editTitle', { key: editRow?.key || '' })}
        width={480}
        visible={editOpen}
        onCancel={() => {
          setEditOpen(false)
          setEditRow(null)
        }}
        footer={
          <div className="flex justify-end gap-2">
            <Button type="outline" onClick={() => setEditOpen(false)}>
              {t('common.cancel')}
            </Button>
            <Button type="primary" loading={editing} onClick={() => void submitEdit()}>
              {t('common.save')}
            </Button>
          </div>
        }
      >
        <Form form={editForm}>
          <ConfigKvForm mode="edit" fixedKey={editRow?.key} />
        </Form>
      </Drawer>

      <Modal
        title={t('common.confirmDelete')}
        visible={Boolean(delTarget)}
        onCancel={() => setDelTarget(null)}
        onOk={() => void confirmDelete()}
        confirmLoading={delLoading}
        okButtonProps={{ status: 'danger' }}
      >
        {t('systemConfigs.deleteConfirm', { key: delTarget?.key || '' })}
      </Modal>
    </BaseLayout>
  )
}
