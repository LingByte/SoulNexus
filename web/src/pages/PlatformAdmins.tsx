import { useCallback, useEffect, useState } from 'react'
import {
  Form,
  Modal,
  Space,
  Tag,
  Typography,
} from '@arco-design/web-react'
import { UserCircle } from 'lucide-react'
import { Button, Input, Select, DataList } from '@/components/ui'
import type { DataListColumn } from '@/components/ui'
import BaseLayout from '@/components/Layout/BaseLayout'
import { useTranslation } from '@/i18n'
import {
  createPlatformAdmin,
  deletePlatformAdmin,
  listPlatformAdmins,
  resetPlatformAdminPassword,
  updatePlatformAdmin,
  updatePlatformAdminStatus,
  type PlatformAdminRow,
} from '@/api/platformAdmins'
import { useAuthStore } from '@/stores/authStore'
import { showAlert } from '@/utils/notification'
import { extractApiErrorMessage } from '@/utils/apiError'

export default function PlatformAdmins() {
  const { t } = useTranslation()
  const meId = Number(useAuthStore((s) => s.user?.id) ?? 0)

  const statusOpts = [
    { label: t('platformAdmins.statusActive'), value: 'active' },
    { label: t('platformAdmins.statusDisabled'), value: 'disabled' },
  ]

  const fmtStatus = (s?: string) => {
    const v = String(s || '').toLowerCase()
    if (v === 'active') return t('platformAdmins.statusActive')
    if (v === 'disabled') return t('platformAdmins.statusDisabled')
    return s || '—'
  }
  const [rows, setRows] = useState<PlatformAdminRow[]>([])
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
  const [editRow, setEditRow] = useState<PlatformAdminRow | null>(null)

  const [pwdOpen, setPwdOpen] = useState(false)
  const [pwdForm] = Form.useForm()
  const [pwdLoading, setPwdLoading] = useState(false)
  const [pwdTarget, setPwdTarget] = useState<PlatformAdminRow | null>(null)

  const [delTarget, setDelTarget] = useState<PlatformAdminRow | null>(null)
  const [delLoading, setDelLoading] = useState(false)

  const pageSize = 20

  const load = useCallback(async () => {
    setLoading(true)
    try {
      const res = await listPlatformAdmins(page, pageSize, search)
      if (res.code === 200 && res.data) {
        setRows(res.data.list || [])
        setTotal(res.data.total || 0)
      } else {
        showAlert(res.msg || t('platformAdmins.loadFailed'), 'error')
      }
    } catch (e: unknown) {
      showAlert(extractApiErrorMessage(e, t('platformAdmins.loadFailed')), 'error')
    } finally {
      setLoading(false)
    }
  }, [page, search])

  useEffect(() => {
    void load()
  }, [load])

  const submitCreate = async () => {
    setCreating(true)
    try {
      const v = await createForm.validate()
      const res = await createPlatformAdmin({
        email: String(v.email || '').trim(),
        password: String(v.password || ''),
        displayName: String(v.displayName || '').trim() || undefined,
        status: (v.status as string) || 'active',
      })
      if (res.code === 200) {
        showAlert(t('platformAdmins.createSuccess'), 'success')
        setCreateOpen(false)
        createForm.resetFields()
        void load()
      } else {
        showAlert(res.msg || t('platformAdmins.createFailed'), 'error')
      }
    } catch {
      /* validation */
    } finally {
      setCreating(false)
    }
  }

  const openEdit = (r: PlatformAdminRow) => {
    setEditRow(r)
    editForm.setFieldsValue({
      email: r.email || '',
      displayName: r.displayName || '',
    })
    setEditOpen(true)
  }

  const submitEdit = async () => {
    if (!editRow) return
    setEditing(true)
    try {
      const v = await editForm.validate()
      const res = await updatePlatformAdmin(editRow.id, {
        email: String(v.email || '').trim() || undefined,
        displayName: String(v.displayName || '').trim() || undefined,
      })
      if (res.code === 200) {
        showAlert(t('platformAdmins.saveSuccess'), 'success')
        setEditOpen(false)
        void load()
      } else {
        showAlert(res.msg || t('platformAdmins.saveFailed'), 'error')
      }
    } catch {
      /* validation */
    } finally {
      setEditing(false)
    }
  }

  const quickStatus = async (r: PlatformAdminRow, status: 'active' | 'disabled') => {
    try {
      const res = await updatePlatformAdminStatus(r.id, status)
      if (res.code === 200) {
        showAlert(t('platformAdmins.statusUpdated'), 'success')
        void load()
      } else {
        showAlert(res.msg || t('platformAdmins.operationFailed'), 'error')
      }
    } catch (e: unknown) {
      showAlert(extractApiErrorMessage(e, t('platformAdmins.operationFailed')), 'error')
    }
  }

  const submitPassword = async () => {
    if (!pwdTarget) return
    setPwdLoading(true)
    try {
      const v = await pwdForm.validate()
      const res = await resetPlatformAdminPassword(pwdTarget.id, String(v.password || ''))
      if (res.code === 200) {
        showAlert(t('platformAdmins.passwordReset'), 'success')
        setPwdOpen(false)
        pwdForm.resetFields()
      } else {
        showAlert(res.msg || t('platformAdmins.resetFailed'), 'error')
      }
    } catch {
      /* validation */
    } finally {
      setPwdLoading(false)
    }
  }

  const confirmDelete = async () => {
    if (!delTarget) return
    setDelLoading(true)
    try {
      const res = await deletePlatformAdmin(delTarget.id)
      if (res.code === 200) {
        showAlert(t('platformAdmins.deleteSuccess'), 'success')
        setDelTarget(null)
        void load()
      } else {
        showAlert(res.msg || t('platformAdmins.deleteFailed'), 'error')
      }
    } catch (e: unknown) {
      showAlert(extractApiErrorMessage(e, t('platformAdmins.deleteFailed')), 'error')
    } finally {
      setDelLoading(false)
    }
  }

  const columns: DataListColumn<PlatformAdminRow>[] = [
    {
      key: 'avatar',
      width: 40,
      render: () => (
        <div className="flex h-9 w-9 shrink-0 items-center justify-center rounded-full bg-neutral-100">
          <UserCircle size={20} className="text-neutral-400" />
        </div>
      ),
    },
    {
      key: 'info',
      title: t('platformAdmins.colEmail'),
      render: (_, r) => (
        <div className="min-w-0">
          <div className="truncate text-sm font-medium text-neutral-900">{r.email || '—'}</div>
          {r.displayName && (
            <div className="mt-0.5 truncate text-xs text-neutral-500">{r.displayName}</div>
          )}
        </div>
      ),
    },
    {
      key: 'status',
      title: t('platformAdmins.colStatus'),
      width: 100,
      render: (_, r) => (
        <Tag color={r.status === 'active' ? 'green' : 'gray'} className="!rounded-full">
          {fmtStatus(r.status)}
        </Tag>
      ),
    },
    {
      key: 'actions',
      width: 280,
      align: 'right',
      render: (_, r) => {
        const isSelf = meId > 0 && Number(r.id) === meId
        return (
          <Space wrap size="mini">
            <Button size="mini" onClick={() => openEdit(r)}>
              {t('platformAdmins.btnEdit')}
            </Button>
            <Button size="mini" onClick={() => { setPwdTarget(r); setPwdOpen(true) }}>
              {t('platformAdmins.btnResetPassword')}
            </Button>
            {r.status === 'active' ? (
              <Button
                size="mini"
                status="warning"
                disabled={isSelf}
                onClick={() => void quickStatus(r, 'disabled')}
              >
                {t('platformAdmins.btnDisable')}
              </Button>
            ) : (
              <Button size="mini" type="outline" onClick={() => void quickStatus(r, 'active')}>
                {t('platformAdmins.btnEnable')}
              </Button>
            )}
            <Button size="mini" status="danger" disabled={isSelf} onClick={() => setDelTarget(r)}>
              {t('platformAdmins.btnDelete')}
            </Button>
          </Space>
        )
      },
    },
  ]

  return (
    <BaseLayout title={t('pages.platformAdmins.title')} description={t('pages.platformAdmins.description')}>
      <DataList
        data={rows as unknown as (PlatformAdminRow & Record<string, unknown>)[]}
        columns={columns}
        loading={loading}
        rowKey="id"
        emptyText={t('common.noData')}
        pagination={total > 0 ? { current: page, pageSize, total, onChange: setPage } : null}
        header={
          <div className="flex items-center justify-between gap-3">
            <Typography.Title heading={6} style={{ margin: 0 }}>
              {t('platformAdmins.adminList')}
            </Typography.Title>
            <div className="flex items-center gap-2">
              <Input.Search
                allowClear
                placeholder={t('platformAdmins.searchPlaceholder')}
                style={{ width: 240 }}
                onSearch={(v) => {
                  setSearch(v)
                  setPage(1)
                }}
              />
              <Button type="primary" onClick={() => setCreateOpen(true)}>
                {t('platformAdmins.createAdmin')}
              </Button>
            </div>
          </div>
        }
      />

      <Modal title={t('platformAdmins.modalCreate')} visible={createOpen} onOk={() => void submitCreate()} onCancel={() => setCreateOpen(false)} confirmLoading={creating}>
        <Form form={createForm} layout="vertical" initialValues={{ status: 'active' }}>
          <Form.Item label={t('platformAdmins.formEmail')} field="email" rules={[{ required: true, type: 'email' }]}>
            <Input />
          </Form.Item>
          <Form.Item label={t('platformAdmins.formPassword')} field="password" rules={[{ required: true, minLength: 8 }]}>
            <Input.Password />
          </Form.Item>
          <Form.Item label={t('platformAdmins.formDisplayName')} field="displayName">
            <Input />
          </Form.Item>
          <Form.Item label={t('platformAdmins.formStatus')} field="status">
            <Select options={statusOpts} />
          </Form.Item>
        </Form>
      </Modal>

      <Modal title={t('platformAdmins.modalEdit')} visible={editOpen} onOk={() => void submitEdit()} onCancel={() => setEditOpen(false)} confirmLoading={editing}>
        <Form form={editForm} layout="vertical">
          <Form.Item label={t('platformAdmins.formEmail')} field="email" rules={[{ type: 'email' }]}>
            <Input />
          </Form.Item>
          <Form.Item label={t('platformAdmins.formDisplayName')} field="displayName">
            <Input />
          </Form.Item>
        </Form>
      </Modal>

      <Modal title={`${t('platformAdmins.modalResetPassword')} ${pwdTarget?.email || ''}`} visible={pwdOpen} onOk={() => void submitPassword()} onCancel={() => setPwdOpen(false)} confirmLoading={pwdLoading}>
        <Form form={pwdForm} layout="vertical">
          <Form.Item label={t('platformAdmins.formNewPassword')} field="password" rules={[{ required: true, minLength: 8 }]}>
            <Input.Password />
          </Form.Item>
        </Form>
      </Modal>

      <Modal
        title={t('platformAdmins.modalDelete')}
        visible={!!delTarget}
        onOk={() => void confirmDelete()}
        onCancel={() => setDelTarget(null)}
        confirmLoading={delLoading}
      >
        {t('platformAdmins.deleteConfirm', { email: delTarget?.email || '' })}
      </Modal>
    </BaseLayout>
  )
}
