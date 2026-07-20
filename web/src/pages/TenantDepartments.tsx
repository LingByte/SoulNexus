import { useCallback, useEffect, useState } from 'react'
import { Form, Modal, Switch } from '@arco-design/web-react'
import { Building2, Pencil, Trash2, Plus, Users } from 'lucide-react'
import { Button, DataList, Input, Select } from '@/components/ui'
import type { DataListColumn } from '@/components/ui'
import { showAlert } from '@/utils/notification'
import BaseLayout from '@/components/Layout/BaseLayout'
import { useTranslation } from '@/i18n'
import {
  createOrgGroup, deleteOrgGroup, listOrgGroups, putOrgTenantUserGroups, updateOrgGroup,
  type OrgGroup, type SnowflakeId,
} from '@/api/tenantOrg'
import { listTenantUsers, type TenantUserRow } from '@/api/tenantUsers'
import { extractApiErrorMessage } from '@/utils/apiError'

const FormItem = Form.Item

export default function TenantDepartments() {
  const { t } = useTranslation()
  const [groups, setGroups] = useState<OrgGroup[]>([])
  const [users, setUsers] = useState<TenantUserRow[]>([])
  const [loading, setLoading] = useState(false)

  const [groupModalOpen, setGroupModalOpen] = useState(false)
  const [groupEditing, setGroupEditing] = useState<OrgGroup | null>(null)
  const [groupForm] = Form.useForm()

  const [assignGroupOpen, setAssignGroupOpen] = useState(false)
  const [assignGroupUserId, setAssignGroupUserId] = useState<SnowflakeId | undefined>(undefined)
  const [assignGroupIds, setAssignGroupIds] = useState<SnowflakeId[]>([])

  const loadGroups = useCallback(async () => {
    const res = await listOrgGroups()
    if (res.code === 200 && res.data?.list) setGroups(res.data.list)
  }, [])

  const loadUsers = useCallback(async () => {
    const res = await listTenantUsers(1, 200)
    if (res.code === 200 && res.data?.list) setUsers(res.data.list)
  }, [])

  const refreshAll = useCallback(async () => {
    setLoading(true)
    try { await Promise.all([loadGroups(), loadUsers()]) } finally { setLoading(false) }
  }, [loadGroups, loadUsers])

  useEffect(() => { void refreshAll() }, [refreshAll])

  const columns: DataListColumn<Record<string, unknown>>[] = [
    {
      key: 'icon', width: 40,
      render: () => (
        <div className="flex h-9 w-9 shrink-0 items-center justify-center rounded-full bg-violet-50 text-violet-600">
          <Building2 size={18} strokeWidth={1.75} />
        </div>
      ),
    },
    {
      key: 'info', title: t('tenantDepartments.colName'),
      render: (_, r) => (
        <div className="min-w-0 flex-1">
          <div className="truncate text-sm font-medium text-neutral-900">{String(r.name ?? '—')}</div>
          {r.isDefault ? <span className="mt-0.5 inline-block rounded-full bg-blue-50 px-2 py-0.5 text-xs text-blue-600">默认</span> : null}
        </div>
      ),
    },
    {
      key: 'actions', width: 140, align: 'right',
      render: (_, r) => (
        <div className="flex items-center justify-end gap-1">
          <Button size="mini" icon={<Pencil size={12} />} onClick={() => { setGroupEditing(r as unknown as OrgGroup); groupForm.setFieldsValue({ name: (r as Record<string, unknown>).name, isDefault: !!(r as Record<string, unknown>).isDefault }); setGroupModalOpen(true) }}>{t('common.edit')}</Button>
          <Button size="mini" status="danger" icon={<Trash2 size={12} />} onClick={() => { Modal.confirm({ title: t('tenantDepartments.deleteTitle'), content: t('tenantDepartments.deleteConfirm', { name: String((r as Record<string, unknown>).name || '') }), onOk: async () => { try { const res = await deleteOrgGroup(String((r as Record<string, unknown>).id)); if (res.code !== 200) { showAlert(res.msg || t('common.deleteFailed'), 'error'); return } showAlert(t('common.deleteSuccess'), 'success'); await loadGroups() } catch (e: unknown) { showAlert(extractApiErrorMessage(e, t('common.deleteFailed')), 'error') } } }) }}>{t('common.delete')}</Button>
        </div>
      ),
    },
  ]

  return (
    <BaseLayout title={t('pages.departments.title')} description={t('pages.departments.description')}>
      <DataList
        data={groups as unknown as (OrgGroup & Record<string, unknown>)[]}
        columns={columns}
        loading={loading}
        rowKey="id"
        emptyText={t('common.noData')}
        header={
          <div className="flex items-center justify-between gap-3">
            <span className="text-sm font-medium text-neutral-900">{t('tenantDepartments.departmentList')}</span>
            <div className="flex items-center gap-2">
              <Button type="primary" icon={<Plus size={14} />} onClick={() => { setGroupEditing(null); groupForm.resetFields(); groupForm.setFieldsValue({ name: '', isDefault: false }); setGroupModalOpen(true) }}>{t('tenantDepartments.createDepartment')}</Button>
              <Button type="outline" icon={<Users size={14} />} onClick={() => { setAssignGroupUserId(undefined); setAssignGroupIds([]); setAssignGroupOpen(true) }}>{t('tenantDepartments.assignDepartment')}</Button>
            </div>
          </div>
        }
      />

      <Modal title={groupEditing ? t('tenantDepartments.editDepartment') : t('tenantDepartments.createDepartment')} visible={groupModalOpen} onCancel={() => setGroupModalOpen(false)}
        onOk={async () => {
          try {
            const v = await groupForm.validate()
            const name = String(v.name || '').trim()
            if (!name) { showAlert(t('tenantDepartments.nameRequired'), 'warning'); return }
            if (groupEditing) {
              const r = await updateOrgGroup(String(groupEditing.id), { name, isDefault: !!v.isDefault })
              if (r.code !== 200) { showAlert(r.msg || t('common.saveFailed'), 'error'); return }
            } else {
              const r = await createOrgGroup({ name, isDefault: !!v.isDefault })
              if (r.code !== 200) { showAlert(r.msg || t('tenantDepartments.createFailed'), 'error'); return }
            }
            showAlert(t('common.saveSuccess'), 'success')
            setGroupModalOpen(false)
            await loadGroups()
          } catch (e: unknown) {
            if (e && typeof e === 'object' && 'errorFields' in e) return
            showAlert(extractApiErrorMessage(e, t('common.saveFailed')), 'error')
          }
        }}
      >
        <Form form={groupForm} layout="vertical">
          <FormItem label={t('tenantDepartments.formName')} field="name" rules={[{ required: true }]}><Input placeholder={t('tenantDepartments.namePlaceholder')} /></FormItem>
          <FormItem label={t('tenantDepartments.formDefault')} field="isDefault" triggerPropName="checked"><Switch /></FormItem>
        </Form>
      </Modal>

      <Modal title={t('tenantDepartments.assignDepartment')} style={{ width: 560 }} visible={assignGroupOpen} onCancel={() => setAssignGroupOpen(false)}
        onOk={async () => {
          if (!assignGroupUserId) { showAlert(t('tenantDepartments.selectUserRequired'), 'warning'); return }
          try {
            const r = await putOrgTenantUserGroups(assignGroupUserId, { groupIds: assignGroupIds })
            if (r.code !== 200) { showAlert(r.msg || t('common.saveFailed'), 'error'); return }
            showAlert(t('tenantDepartments.updateSuccess'), 'success')
            setAssignGroupOpen(false)
          } catch (e: unknown) { showAlert(extractApiErrorMessage(e, t('common.saveFailed')), 'error') }
        }}
      >
        <div className="space-y-4">
          <div>
            <label className="mb-1 block text-sm text-neutral-500">{t('tenantDepartments.sectionUser')}</label>
            <Select placeholder={t('tenantDepartments.selectMember')} style={{ width: '100%' }} value={assignGroupUserId} onChange={(v) => setAssignGroupUserId(v != null ? String(v) : undefined)} options={users.map((u) => ({ value: String(u.id), label: `${u.displayName || u.username || u.email || u.id}` }))} showSearch />
          </div>
          <div>
            <label className="mb-1 block text-sm text-neutral-500">{t('tenantDepartments.sectionDepartments')}</label>
            <Select mode="multiple" placeholder={t('tenantDepartments.selectDepartment')} style={{ width: '100%' }} value={assignGroupIds} onChange={(v) => setAssignGroupIds((v as string[]).map(String))} options={groups.map((g) => ({ value: String(g.id), label: g.name }))} />
          </div>
        </div>
      </Modal>
    </BaseLayout>
  )
}
