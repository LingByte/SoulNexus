import { useCallback, useEffect, useMemo, useState } from 'react'
import { Form, Modal, Switch, Tag } from '@arco-design/web-react'
import {
  Building2, Pencil, Trash2, Power, PowerOff, Plus, UserCircle, Users, ChevronRight, ChevronDown,
} from 'lucide-react'
import { Button, Input, Select } from '@/components/ui'
import BaseLayout from '@/components/Layout/BaseLayout'
import { useTranslation } from '@/i18n'
import {
  createTenantUser,
  deleteTenantUser,
  listTenantUsers,
  updateTenantUser,
  updateTenantUserStatus,
  type TenantUserRow,
  type SnowflakeId,
} from '@/api/tenantUsers'
import {
  createOrgGroup, deleteOrgGroup, listOrgGroups, putOrgTenantUserGroups, updateOrgGroup,
  type OrgGroup,
} from '@/api/tenantOrg'
import { useAuthStore } from '@/stores/authStore'
import { showAlert } from '@/utils/notification'
import { extractApiErrorMessage } from '@/utils/apiError'

const statusColor = (s?: string) => {
  const v = String(s || '').toLowerCase()
  if (v === 'active') return 'green'
  if (v === 'disabled') return 'gray'
  return 'orange'
}

const statusLabel = (s?: string, t?: (k: string) => string) => {
  const v = String(s || '').toLowerCase()
  if (v === 'active') return t?.('tenantMembers.statusActive') ?? '正常'
  if (v === 'disabled') return t?.('tenantMembers.statusDisabled') ?? '已禁用'
  return t?.('tenantMembers.statusPending') ?? '待激活'
}

function userGroupIds(u: TenantUserRow): string[] {
  const groups = u.tenantGroups?.length ? u.tenantGroups : u.tenantGroup ? [u.tenantGroup] : []
  return groups.map((g) => String(g.id))
}

export default function TenantMembers() {
  const { t } = useTranslation()
  const statusOpts = [
    { label: t('tenantMembers.statusActive'), value: 'active' },
    { label: t('tenantMembers.statusDisabled'), value: 'disabled' },
    { label: t('tenantMembers.statusPending'), value: 'pending' },
  ]
  const meId = String(useAuthStore((s) => s.user?.id) ?? '')
  const [users, setUsers] = useState<TenantUserRow[]>([])
  const [groups, setGroups] = useState<OrgGroup[]>([])
  const [loading, setLoading] = useState(false)
  const [expanded, setExpanded] = useState<Record<string, boolean>>({})

  const [createOpen, setCreateOpen] = useState(false)
  const [createForm] = Form.useForm()
  const [creating, setCreating] = useState(false)

  const [editOpen, setEditOpen] = useState(false)
  const [editForm] = Form.useForm()
  const [editing, setEditing] = useState(false)
  const [editId, setEditId] = useState<SnowflakeId | null>(null)

  const [delTarget, setDelTarget] = useState<TenantUserRow | null>(null)
  const [delLoading, setDelLoading] = useState(false)

  const [groupModalOpen, setGroupModalOpen] = useState(false)
  const [groupEditing, setGroupEditing] = useState<OrgGroup | null>(null)
  const [groupForm] = Form.useForm()

  const [assignOpen, setAssignOpen] = useState(false)
  const [assignUser, setAssignUser] = useState<TenantUserRow | null>(null)
  const [assignGroupIds, setAssignGroupIds] = useState<SnowflakeId[]>([])

  const refreshAll = useCallback(async () => {
    setLoading(true)
    try {
      const [uRes, gRes] = await Promise.all([
        listTenantUsers(1, 500),
        listOrgGroups(),
      ])
      if (uRes.code === 200 && uRes.data?.list) setUsers(uRes.data.list)
      else if (uRes.code !== 200) showAlert(uRes.msg || t('tenantMembers.loadFailed'), 'error')
      if (gRes.code === 200 && gRes.data?.list) {
        setGroups(gRes.data.list)
        setExpanded((prev) => {
          const next: Record<string, boolean> = {}
          for (const g of gRes.data!.list) {
            const key = `dept-${g.id}`
            next[key] = prev[key] === true
          }
          if (prev['root-unassigned']) next['root-unassigned'] = true
          return next
        })
      }
    } catch (e: unknown) {
      showAlert(extractApiErrorMessage(e, t('tenantMembers.loadFailed')), 'error')
    } finally {
      setLoading(false)
    }
  }, [t])

  useEffect(() => { void refreshAll() }, [refreshAll])

  const toggle = (key: string) => {
    setExpanded((prev) => ({ ...prev, [key]: !prev[key] }))
  }

  const submitCreate = async () => {
    setCreating(true)
    try {
      const v = await createForm.validate()
      const res = await createTenantUser({
        email: String(v.email || '').trim(),
        password: String(v.password || ''),
        displayName: String(v.displayName || '').trim() || undefined,
        phone: String(v.phone || '').trim() || undefined,
        username: String(v.username || '').trim() || undefined,
        status: (v.status as string) || 'active',
      })
      if (res.code === 200) {
        showAlert(t('tenantMembers.createSuccess'), 'success')
        setCreateOpen(false)
        createForm.resetFields()
        void refreshAll()
      } else {
        showAlert(res.msg || t('tenantMembers.createFailed'), 'error')
      }
    } catch (e: unknown) {
      if (e && typeof e === 'object' && 'errorFields' in e) return
      showAlert(extractApiErrorMessage(e, t('tenantMembers.createFailed')), 'error')
    } finally {
      setCreating(false)
    }
  }

  const openEdit = (r: TenantUserRow) => {
    setEditId(String(r.id))
    editForm.setFieldsValue({
      email: r.email || '', phone: r.phone || '', username: r.username || '',
      displayName: r.displayName || '', status: r.status || 'active',
    })
    setEditOpen(true)
  }

  const submitEdit = async () => {
    if (editId == null) return
    setEditing(true)
    try {
      const v = await editForm.validate()
      const res = await updateTenantUser(editId, {
        email: String(v.email || '').trim() || undefined,
        phone: String(v.phone || '').trim() || undefined,
        username: String(v.username || '').trim() || undefined,
        displayName: String(v.displayName || '').trim() || undefined,
        status: String(v.status || '').trim() || undefined,
      })
      if (res.code === 200) {
        showAlert(t('tenantMembers.saveSuccess'), 'success')
        setEditOpen(false)
        void refreshAll()
      } else {
        showAlert(res.msg || t('tenantMembers.saveFailed'), 'error')
      }
    } catch (e: unknown) {
      if (e && typeof e === 'object' && 'errorFields' in e) return
      showAlert(extractApiErrorMessage(e, t('tenantMembers.saveFailed')), 'error')
    } finally {
      setEditing(false)
    }
  }

  const quickStatus = async (r: TenantUserRow, status: string) => {
    try {
      const res = await updateTenantUserStatus(String(r.id), status)
      if (res.code === 200) { showAlert(t('tenantMembers.statusUpdated'), 'success'); void refreshAll() }
      else showAlert(res.msg || t('tenantMembers.operationFailed'), 'error')
    } catch (e: unknown) { showAlert(extractApiErrorMessage(e, t('tenantMembers.operationFailed')), 'error') }
  }

  const confirmDelete = async () => {
    if (!delTarget) return
    setDelLoading(true)
    try {
      const res = await deleteTenantUser(String(delTarget.id))
      if (res.code === 200) { showAlert(t('tenantMembers.deleteSuccess'), 'success'); setDelTarget(null); void refreshAll() }
      else showAlert(res.msg || t('tenantMembers.deleteFailed'), 'error')
    } catch (e: unknown) { showAlert(extractApiErrorMessage(e, t('tenantMembers.deleteFailed')), 'error') }
    finally { setDelLoading(false) }
  }

  const openAssign = (u: TenantUserRow) => {
    setAssignUser(u)
    setAssignGroupIds(userGroupIds(u))
    setAssignOpen(true)
  }

  const saveAssign = async () => {
    if (!assignUser) return
    try {
      const res = await putOrgTenantUserGroups(String(assignUser.id), assignGroupIds.map(String))
      if (res.code !== 200) {
        showAlert(res.msg || t('tenantDepartments.assignFailed'), 'error')
        return
      }
      showAlert(t('tenantDepartments.assignSuccess'), 'success')
      setAssignOpen(false)
      void refreshAll()
    } catch (e: unknown) {
      showAlert(extractApiErrorMessage(e, t('tenantDepartments.assignFailed')), 'error')
    }
  }

  const memberRow = (u: TenantUserRow) => (
    <div
      key={String(u.id)}
      className="flex min-w-0 items-center justify-between gap-3 border-t border-border/60 py-2.5 pl-10 pr-2"
    >
      <div className="flex min-w-0 items-center gap-2">
        <div className="flex h-8 w-8 shrink-0 items-center justify-center rounded-full bg-blue-50 text-blue-600">
          <UserCircle size={16} strokeWidth={1.75} />
        </div>
        <div className="min-w-0">
          <div className="truncate text-sm font-medium text-neutral-900">{u.email || '—'}</div>
          <div className="mt-0.5 flex flex-wrap items-center gap-2 text-xs text-neutral-500">
            {u.displayName ? <span>{u.displayName}</span> : null}
            {u.phone ? <><span className="text-neutral-300">·</span><span>{u.phone}</span></> : null}
            <Tag color={statusColor(u.status)} size="small" className="!rounded-full">{statusLabel(u.status, t)}</Tag>
          </div>
        </div>
      </div>
      <div className="flex shrink-0 items-center gap-1">
        <Button size="mini" icon={<Users size={12} />} onClick={() => openAssign(u)}>
          {t('tenantDepartments.assignDepartment')}
        </Button>
        <Button size="mini" icon={<Pencil size={12} />} onClick={() => openEdit(u)}>{t('tenantMembers.btnEdit')}</Button>
        {String(u.status) === 'active' ? (
          <Button size="mini" icon={<PowerOff size={12} />} onClick={() => void quickStatus(u, 'disabled')}>{t('tenantMembers.btnDisable')}</Button>
        ) : (
          <Button size="mini" icon={<Power size={12} />} onClick={() => void quickStatus(u, 'active')}>{t('tenantMembers.btnEnable')}</Button>
        )}
        <Button size="mini" status="danger" icon={<Trash2 size={12} />} disabled={String(u.id) === meId} onClick={() => setDelTarget(u)}>{t('tenantMembers.btnDelete')}</Button>
      </div>
    </div>
  )

  const { unassigned, byDept } = useMemo(() => {
    const assigned = new Set<string>()
    const map = new Map<string, TenantUserRow[]>()
    for (const g of groups) map.set(String(g.id), [])
    for (const u of users) {
      const ids = userGroupIds(u)
      if (ids.length === 0) continue
      for (const gid of ids) {
        assigned.add(String(u.id))
        const list = map.get(gid) || []
        list.push(u)
        map.set(gid, list)
      }
    }
    return {
      unassigned: users.filter((u) => !assigned.has(String(u.id))),
      byDept: map,
    }
  }, [users, groups])

  return (
    <BaseLayout title={t('pages.tenantMembers.title')} description={t('tenantOrg.pageDesc')}>
      <div className="rounded-xl border border-border bg-card">
        <div className="flex items-center justify-between gap-3 border-b border-border px-4 py-3">
          <span className="text-sm font-medium text-neutral-900">{t('tenantOrg.treeTitle')}</span>
          <div className="flex items-center gap-2">
            <Button
              type="outline"
              icon={<Building2 size={14} />}
              onClick={() => {
                setGroupEditing(null)
                groupForm.resetFields()
                groupForm.setFieldsValue({ name: '', isDefault: false })
                setGroupModalOpen(true)
              }}
            >
              {t('tenantDepartments.createDepartment')}
            </Button>
            <Button
              type="primary"
              icon={<Plus size={14} />}
              onClick={() => {
                createForm.resetFields()
                createForm.setFieldsValue({ status: 'active' })
                setCreateOpen(true)
              }}
            >
              {t('tenantMembers.createMember')}
            </Button>
          </div>
        </div>
        <div className="min-h-[320px] p-2">
          {loading ? (
            <div className="py-16 text-center text-sm text-neutral-400">{t('common.loading')}</div>
          ) : (
            <div className="divide-y divide-border/70">
              {unassigned.length > 0 ? (
                <div>
                  <button
                    type="button"
                    className="flex w-full items-center gap-2 px-3 py-2.5 text-left hover:bg-muted/40"
                    onClick={() => toggle('root-unassigned')}
                  >
                    {expanded['root-unassigned']
                      ? <ChevronDown size={16} className="shrink-0 text-neutral-400" />
                      : <ChevronRight size={16} className="shrink-0 text-neutral-400" />}
                    <div className="flex h-8 w-8 shrink-0 items-center justify-center rounded-full bg-neutral-100 text-neutral-600">
                      <Users size={16} strokeWidth={1.75} />
                    </div>
                    <div className="min-w-0">
                      <div className="text-sm font-medium text-neutral-900">{t('tenantOrg.unassigned')}</div>
                      <div className="text-xs text-neutral-500">{t('tenantOrg.memberCount', { count: unassigned.length })}</div>
                    </div>
                  </button>
                  {expanded['root-unassigned'] ? unassigned.map(memberRow) : null}
                </div>
              ) : null}

              {groups.map((g) => {
                const key = `dept-${g.id}`
                const members = byDept.get(String(g.id)) || []
                const open = !!expanded[key]
                return (
                  <div key={key}>
                    <div className="flex items-center gap-1 px-1 py-1">
                      <button
                        type="button"
                        className="flex min-w-0 flex-1 items-center gap-2 rounded-lg px-2 py-2 text-left hover:bg-muted/40"
                        onClick={() => toggle(key)}
                      >
                        {open
                          ? <ChevronDown size={16} className="shrink-0 text-neutral-400" />
                          : <ChevronRight size={16} className="shrink-0 text-neutral-400" />}
                        <div className="flex h-8 w-8 shrink-0 items-center justify-center rounded-full bg-violet-50 text-violet-600">
                          <Building2 size={16} strokeWidth={1.75} />
                        </div>
                        <div className="min-w-0">
                          <div className="truncate text-sm font-medium text-neutral-900">
                            {g.name}
                            {g.isDefault ? (
                              <span className="ml-2 inline-block rounded-full bg-blue-50 px-2 py-0.5 text-xs font-normal text-blue-600">
                                {t('tenantDepartments.defaultBadge')}
                              </span>
                            ) : null}
                          </div>
                          <div className="text-xs text-neutral-500">
                            {t('tenantOrg.memberCount', { count: members.length })}
                          </div>
                        </div>
                      </button>
                      <div className="flex shrink-0 items-center gap-1 pr-2">
                        <Button
                          size="mini"
                          icon={<Pencil size={12} />}
                          onClick={() => {
                            setGroupEditing(g)
                            groupForm.setFieldsValue({ name: g.name, isDefault: !!g.isDefault })
                            setGroupModalOpen(true)
                          }}
                        >
                          {t('common.edit')}
                        </Button>
                        <Button
                          size="mini"
                          status="danger"
                          icon={<Trash2 size={12} />}
                          onClick={() => {
                            Modal.confirm({
                              title: t('tenantDepartments.deleteTitle'),
                              content: t('tenantDepartments.deleteConfirm', { name: g.name }),
                              onOk: async () => {
                                try {
                                  const res = await deleteOrgGroup(String(g.id))
                                  if (res.code !== 200) {
                                    showAlert(res.msg || t('common.deleteFailed'), 'error')
                                    return
                                  }
                                  showAlert(t('common.deleteSuccess'), 'success')
                                  await refreshAll()
                                } catch (e: unknown) {
                                  showAlert(extractApiErrorMessage(e, t('common.deleteFailed')), 'error')
                                }
                              },
                            })
                          }}
                        >
                          {t('common.delete')}
                        </Button>
                      </div>
                    </div>
                    {open ? (
                      members.length
                        ? members.map(memberRow)
                        : (
                          <div className="border-t border-border/60 py-3 pl-10 text-xs text-neutral-400">
                            {t('tenantOrg.emptyDept')}
                          </div>
                          )
                    ) : null}
                  </div>
                )
              })}

              {!loading && groups.length === 0 && unassigned.length === 0 ? (
                <div className="py-16 text-center text-sm text-neutral-400">{t('tenantMembers.loadFailed')}</div>
              ) : null}
            </div>
          )}
        </div>
      </div>

      <Modal title={t('tenantMembers.modalCreate')} visible={createOpen} onCancel={() => !creating && setCreateOpen(false)} onOk={() => void submitCreate()} confirmLoading={creating}>
        <Form form={createForm} layout="vertical">
          <Form.Item label={t('tenantMembers.formEmail')} field="email" rules={[{ required: true, message: t('tenantMembers.required') }]}><Input placeholder="login@example.com" /></Form.Item>
          <Form.Item label={t('tenantMembers.formPassword')} field="password" rules={[{ required: true, message: t('tenantMembers.required') }, { validator: (v, cb) => { if (v && String(v).length >= 8) return cb(); cb(t('tenantMembers.passwordMin8')) } }]}><Input.Password placeholder={t('tenantMembers.passwordMin8')} /></Form.Item>
          <Form.Item label={t('tenantMembers.formDisplayName')} field="displayName"><Input /></Form.Item>
          <Form.Item label={t('tenantMembers.formPhone')} field="phone"><Input /></Form.Item>
          <Form.Item label={t('tenantMembers.formUsername')} field="username"><Input /></Form.Item>
          <Form.Item label={t('tenantMembers.formStatus')} field="status" initialValue="active"><Select options={statusOpts} /></Form.Item>
        </Form>
      </Modal>

      <Modal title={t('tenantMembers.modalEdit')} visible={editOpen} onCancel={() => !editing && setEditOpen(false)} onOk={() => void submitEdit()} confirmLoading={editing}>
        <Form form={editForm} layout="vertical">
          <Form.Item label={t('tenantMembers.formEmail')} field="email" rules={[{ required: true }]}><Input /></Form.Item>
          <Form.Item label={t('tenantMembers.formDisplayName')} field="displayName"><Input /></Form.Item>
          <Form.Item label={t('tenantMembers.formPhone')} field="phone"><Input /></Form.Item>
          <Form.Item label={t('tenantMembers.formUsername')} field="username"><Input /></Form.Item>
          <Form.Item label={t('tenantMembers.formStatus')} field="status"><Select options={statusOpts} /></Form.Item>
        </Form>
      </Modal>

      <Modal title={t('tenantMembers.modalDelete')} visible={!!delTarget} onCancel={() => !delLoading && setDelTarget(null)} onOk={() => void confirmDelete()} confirmLoading={delLoading}>
        <p className="text-sm text-neutral-600">{t('tenantMembers.deleteConfirm', { email: delTarget?.email || '' })}</p>
      </Modal>

      <Modal
        title={groupEditing ? t('tenantDepartments.editDepartment') : t('tenantDepartments.createDepartment')}
        visible={groupModalOpen}
        onCancel={() => setGroupModalOpen(false)}
        onOk={async () => {
          try {
            const v = await groupForm.validate()
            const body = { name: String(v.name || '').trim(), isDefault: Boolean(v.isDefault) }
            const res = groupEditing
              ? await updateOrgGroup(String(groupEditing.id), body)
              : await createOrgGroup(body)
            if (res.code !== 200) {
              showAlert(res.msg || t('common.saveFailed'), 'error')
              return
            }
            showAlert(t('common.saveSuccess'), 'success')
            setGroupModalOpen(false)
            void refreshAll()
          } catch (e: unknown) {
            if (e && typeof e === 'object' && 'errorFields' in e) return
            showAlert(extractApiErrorMessage(e, t('common.saveFailed')), 'error')
          }
        }}
      >
        <Form form={groupForm} layout="vertical">
          <Form.Item label={t('tenantDepartments.colName')} field="name" rules={[{ required: true }]}>
            <Input />
          </Form.Item>
          <Form.Item label={t('tenantDepartments.defaultBadge')} field="isDefault" triggerPropName="checked">
            <Switch />
          </Form.Item>
        </Form>
      </Modal>

      <Modal
        title={t('tenantDepartments.assignDepartment')}
        visible={assignOpen}
        onCancel={() => setAssignOpen(false)}
        onOk={() => void saveAssign()}
      >
        <p className="mb-3 text-sm text-neutral-600">{assignUser?.email}</p>
        <Select
          mode="multiple"
          value={assignGroupIds.map(String)}
          onChange={(v) => setAssignGroupIds((v as string[]) || [])}
          options={groups.map((g) => ({ label: g.name, value: String(g.id) }))}
          placeholder={t('tenantDepartments.selectDepartments')}
          style={{ width: '100%' }}
        />
      </Modal>
    </BaseLayout>
  )
}
