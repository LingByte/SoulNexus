import { useCallback, useEffect, useState } from 'react'
import {
  Drawer,
  Form,
  Modal,
  Space,
  Tag,
} from '@arco-design/web-react'
import { Building2, Mail, Users, Settings, Trash2, Bot } from 'lucide-react'
import { Button, Input, Select, DataList } from '@/components/ui'
import type { DataListColumn } from '@/components/ui'
import { showAlert } from '@/utils/notification'
import { useNavigate } from 'react-router-dom'
import BaseLayout from '@/components/Layout/BaseLayout'
import { useTranslation } from '@/i18n'
import {
  createTenantPlatform,
  deleteTenantPlatform,
  listTenants,
  updateTenantPlatform,
  type TenantRow,
} from '@/api/tenants'

const FormItem = Form.Item

export default function TenantManagement() {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const [rows, setRows] = useState<TenantRow[]>([])
  const [total, setTotal] = useState(0)
  const [page, setPage] = useState(1)
  const [size] = useState(20)
  const [search, setSearch] = useState('')
  const [loading, setLoading] = useState(false)

  const [createOpen, setCreateOpen] = useState(false)
  const [createForm] = Form.useForm()

  const [editOpen, setEditOpen] = useState(false)
  const [editForm] = Form.useForm()
  const [editing, setEditing] = useState<TenantRow | null>(null)

  const load = useCallback(async () => {
    setLoading(true)
    try {
      const res = await listTenants(page, size, { search: search.trim() || undefined })
      if (res.code === 200 && res.data) {
        setRows(res.data.list || [])
        setTotal(res.data.total ?? 0)
      } else {
        showAlert(res.msg || t('common.loadFailed'), 'error')
      }
    } finally {
      setLoading(false)
    }
  }, [page, search, size])

  useEffect(() => {
    void load()
  }, [load])

  const columns: DataListColumn<TenantRow>[] = [
    {
      key: 'icon',
      width: 44,
      render: () => (
        <div className="flex h-10 w-10 shrink-0 items-center justify-center rounded-xl bg-blue-50 text-blue-600">
          <Building2 size={20} strokeWidth={1.75} />
        </div>
      ),
    },
    {
      key: 'info',
      title: t('tenantManagement.colCompany'),
      render: (_, r) => (
        <div className="min-w-0 flex-1">
          <div className="truncate text-sm font-medium text-neutral-900">{r.name || '—'}</div>
          <div className="mt-0.5 flex items-center gap-2 text-xs text-neutral-500">
            <span className="font-mono">{r.slug || '—'}</span>
            {r.contactEmail && (
              <>
                <span className="text-neutral-300">·</span>
                <span className="flex items-center gap-1 truncate">
                  <Mail size={10} className="shrink-0 text-neutral-400" />
                  {r.contactEmail}
                </span>
              </>
            )}
          </div>
        </div>
      ),
    },
    {
      key: 'meta',
      width: 160,
      render: (_, r) => (
        <div className="flex items-center gap-4 text-xs text-neutral-500">
          <span className="flex items-center gap-1">
            <Users size={12} className="text-neutral-400" />
            {r.maxUserCount && r.maxUserCount > 0 ? r.maxUserCount : 5}
          </span>
          <Tag
            color={r.status === 'suspended' ? 'gray' : 'green'}
            className="!rounded-full !text-xs"
          >
            {r.status === 'suspended' ? t('tenantManagement.statusSuspended') : t('tenantManagement.statusActive')}
          </Tag>
        </div>
      ),
    },
    {
      key: 'actions',
      width: 220,
      align: 'right',
      render: (_, r) => (
        <Space wrap size="mini">
          <Button
            size="mini"
            icon={<Settings size={13} />}
            onClick={() => {
              setEditing(r)
              editForm.setFieldsValue({
                name: r.name,
                description: r.description || '',
                status: r.status || 'active',
                contactEmail: r.contactEmail || '',
                maxUserCount: r.maxUserCount || 5,
              })
              setEditOpen(true)
            }}
          >
            {t('common.edit')}
          </Button>
          <Button
            size="mini"
            icon={<Bot size={13} />}
            onClick={() => navigate(`/tenant-management/${r.id}/ai`)}
          >
            {t('tenantManagement.aiConfig')}
          </Button>
          <Button
            size="mini"
            status="danger"
            icon={<Trash2 size={13} />}
            onClick={() => {
              Modal.confirm({
                title: t('tenantManagement.deleteTitle'),
                content: t('tenantManagement.deleteContent', { name: r.name }),
                onOk: async () => {
                  const res = await deleteTenantPlatform(r.id)
                  if (res.code !== 200) {
                    showAlert(res.msg || t('common.deleteFailed'), 'error')
                    return
                  }
                  showAlert(t('tenantManagement.deleted'), 'success')
                  await load()
                },
              })
            }}
          >
            {t('common.delete')}
          </Button>
        </Space>
      ),
    },
  ]

  return (
    <BaseLayout
      title={t('pages.tenantManagement.title')}
      description={t('pages.tenantManagement.description')}
    >
      <DataList
        data={rows as unknown as (TenantRow & Record<string, unknown>)[]}
        columns={columns}
        loading={loading}
        rowKey="id"
        emptyText={t('common.noData')}
        pagination={total > 0 ? { current: page, pageSize: size, total, onChange: setPage } : null}
        header={
          <div className="flex items-center justify-between gap-3">
            <div className="flex items-center gap-2">
              <Building2 size={16} className="text-neutral-500" />
              <span className="text-sm font-medium text-neutral-900">{t('tenantManagement.searchPlaceholder')}</span>
            </div>
            <div className="flex items-center gap-2">
              <Input.Search
                placeholder={t('tenantManagement.searchPlaceholder')}
                style={{ width: 240 }}
                allowClear
                onSearch={(v) => {
                  setPage(1)
                  setSearch(v)
                }}
              />
              <Button
                type="primary"
                onClick={() => {
                  createForm.resetFields()
                  setCreateOpen(true)
                }}
              >
                {t('tenantManagement.createTenant')}
              </Button>
            </div>
          </div>
        }
      />

      <Modal
        title={t('tenantManagement.modalCreate')}
        style={{ width: 520 }}
        visible={createOpen}
        onCancel={() => setCreateOpen(false)}
        onOk={async () => {
          try {
            const v = await createForm.validate()
            const r = await createTenantPlatform({
              companyName: String(v.companyName || '').trim(),
              adminEmail: String(v.adminEmail || '').trim(),
              adminPassword: String(v.adminPassword || ''),
              adminDisplayName: String(v.adminDisplayName || '').trim(),
              tenantDescription: String(v.tenantDescription || '').trim(),
              maxUserCount: Number(v.maxUserCount) || 5,
            })
            if (r.code !== 200) {
              showAlert(r.msg || t('tenantManagement.createFailed'), 'error')
              return
            }
            showAlert(t('tenantManagement.createSuccess'), 'success')
            setCreateOpen(false)
            await load()
          } catch {
            /* validate */
          }
        }}
      >
        <Form form={createForm} layout="vertical">
          <FormItem label={t('tenantManagement.formCompany')} field="companyName" rules={[{ required: true }]}>
            <Input placeholder={t('tenantManagement.companyPlaceholder')} />
          </FormItem>
          <FormItem label={t('tenantManagement.formAdminEmail')} field="adminEmail" rules={[{ required: true }]}>
            <Input placeholder={t('tenantManagement.loginEmailPlaceholder')} />
          </FormItem>
          <FormItem label={t('tenantManagement.formAdminPassword')} field="adminPassword" rules={[{ required: true }]}>
            <Input.Password placeholder={t('auth.passwordMin8Short')} />
          </FormItem>
          <FormItem label={t('tenantManagement.formAdminDisplay')} field="adminDisplayName">
            <Input placeholder={t('tenantManagement.optional')} />
          </FormItem>
          <FormItem label={t('tenantManagement.formRemark')} field="tenantDescription">
            <Input.TextArea placeholder={t('tenantManagement.optional')} autoSize={{ minRows: 2 }} />
          </FormItem>
          <FormItem label={t('tenantManagement.formMaxUsers')} field="maxUserCount" initialValue={5}>
            <Input type="number" min={1} />
          </FormItem>
        </Form>
      </Modal>

      <Drawer
        title={t('tenantManagement.modalEdit')}
        width={480}
        visible={editOpen}
        onCancel={() => setEditOpen(false)}
        footer={
          <div className="flex justify-end gap-2">
            <Button type="outline" onClick={() => setEditOpen(false)}>
              {t('common.cancel')}
            </Button>
            <Button
              type="primary"
              onClick={async () => {
                if (!editing) return
                try {
                  const v = await editForm.validate()
                  const r = await updateTenantPlatform(editing.id, {
                    name: String(v.name || '').trim(),
                    description: String(v.description || '').trim(),
                    status: String(v.status || 'active'),
                    contactEmail: String(v.contactEmail || '').trim(),
                    maxUserCount: Number(v.maxUserCount) || 5,
                  })
                  if (r.code !== 200) {
                    showAlert(r.msg || t('common.saveFailed'), 'error')
                    return
                  }
                  showAlert(t('common.saveSuccess'), 'success')
                  setEditOpen(false)
                  await load()
                } catch {
                  /* validate */
                }
              }}
            >
              {t('common.save')}
            </Button>
          </div>
        }
      >
        <Form form={editForm} layout="vertical">
          <FormItem label={t('tenantManagement.formCompany')} field="name" rules={[{ required: true }]}>
            <Input />
          </FormItem>
          <FormItem label={t('tenantManagement.formDescription')} field="description">
            <Input.TextArea autoSize={{ minRows: 2 }} />
          </FormItem>
          <FormItem label={t('tenantManagement.formStatus')} field="status" rules={[{ required: true }]}>
            <Select
              options={[
                { value: 'active', label: t('tenantManagement.statusNormal') },
                { value: 'suspended', label: t('tenantManagement.statusPaused') },
              ]}
            />
          </FormItem>
          <FormItem label={t('tenantManagement.formContactEmail')} field="contactEmail">
            <Input />
          </FormItem>
          <FormItem label={t('tenantManagement.formMaxUsers')} field="maxUserCount">
            <Input type="number" min={1} />
          </FormItem>
        </Form>
      </Drawer>
    </BaseLayout>
  )
}
