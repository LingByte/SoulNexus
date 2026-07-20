import { useCallback, useEffect, useMemo, useState } from 'react'
import { Form, Input as ArcoInput, Modal, Tree, Tag } from '@arco-design/web-react'
import { Shield, Trash2, Plus, Key, Users } from 'lucide-react'
import { Button, DataList, Select } from '@/components/ui'
import type { DataListColumn } from '@/components/ui'
import type { TreeDataType } from '@arco-design/web-react/es/Tree/interface'
import BaseLayout from '@/components/Layout/BaseLayout'
import { useTranslation } from '@/i18n'
import {
  createOrgRole, deleteOrgRole, getOrgRole, listOrgPermissions, listOrgRoles, putOrgRolePermissions, putOrgTenantUserRoles,
  type OrgPermission, type OrgRole, type SnowflakeId,
} from '@/api/tenantOrg'
import { listTenantUsers, type TenantUserRow } from '@/api/tenantUsers'
import { showAlert } from '@/utils/notification'
import { extractApiErrorMessage } from '@/utils/apiError'

const FormItem = Form.Item

const KIND_RANK: Record<string, number> = { module: 0, menu: 1, button: 2, api: 3, data: 4 }
function kindRank(k?: string): number { return KIND_RANK[k || ''] ?? 99 }

function permNodeTitle(p: OrgPermission, t: (key: string) => string): string {
  const k = p.kind || ''
  if (k === 'module') return t('tenantRoles.kindModule').replace('{name}', p.name)
  if (k === 'menu') return t('tenantRoles.kindMenu').replace('{name}', p.name)
  if (k === 'button') return t('tenantRoles.kindButton').replace('{name}', p.name)
  if (k === 'api') return t('tenantRoles.kindApi').replace('{name}', p.name)
  if (k === 'data') return t('tenantRoles.kindData').replace('{name}', p.name)
  return p.name
}

function buildPermissionTree(perms: OrgPermission[], t: (key: string) => string): TreeDataType[] {
  const codeSet = new Set(perms.map((p) => p.code))
  const resolvedParent = (p: OrgPermission): string => { const pc = (p.parentCode || '').trim(); return pc && codeSet.has(pc) ? pc : '' }
  const byParent = new Map<string, OrgPermission[]>()
  for (const p of perms) { const par = resolvedParent(p); if (!byParent.has(par)) byParent.set(par, []); byParent.get(par)!.push(p) }
  for (const arr of byParent.values()) arr.sort((a, b) => { const d = kindRank(a.kind) - kindRank(b.kind); return d !== 0 ? d : a.code.localeCompare(b.code) })
  const walk = (parentCode: string): TreeDataType[] => {
    const kids = byParent.get(parentCode) || []
    return kids.map((p) => { const children = walk(p.code); return { key: String(p.id), title: permNodeTitle(p, t), children: children.length ? children : undefined } })
  }
  return walk('')
}

function collectKeysWithChildren(nodes: TreeDataType[]): string[] {
  const out: string[] = []; const walk = (ns: TreeDataType[]) => { for (const n of ns) { if (n.children && n.children.length > 0) { out.push(String(n.key)); walk(n.children) } } }; walk(nodes); return out
}

export default function TenantRolePermissions() {
  const { t } = useTranslation()
  const [roles, setRoles] = useState<OrgRole[]>([])
  const [perms, setPerms] = useState<OrgPermission[]>([])
  const [users, setUsers] = useState<TenantUserRow[]>([])
  const [loading, setLoading] = useState(false)

  const [roleModalOpen, setRoleModalOpen] = useState(false)
  const [roleForm] = Form.useForm()

  const [permModalOpen, setPermModalOpen] = useState(false)
  const [permRoleId, setPermRoleId] = useState<SnowflakeId | null>(null)
  const [permRoleName, setPermRoleName] = useState('')
  const [permSelected, setPermSelected] = useState<string[]>([])

  const [assignOpen, setAssignOpen] = useState(false)
  const [assignUserId, setAssignUserId] = useState<SnowflakeId | undefined>(undefined)
  const [assignRoleIds, setAssignRoleIds] = useState<SnowflakeId[]>([])

  const loadRoles = useCallback(async () => { const res = await listOrgRoles(); if (res.code === 200 && res.data?.list) setRoles(res.data.list) }, [])
  const loadPerms = useCallback(async () => { const res = await listOrgPermissions(); if (res.code === 200 && res.data?.list) setPerms(res.data.list) }, [])
  const loadUsers = useCallback(async () => { const res = await listTenantUsers(1, 200); if (res.code === 200 && res.data?.list) setUsers(res.data.list) }, [])

  const refreshAll = useCallback(async () => { setLoading(true); try { await Promise.all([loadRoles(), loadPerms(), loadUsers()]) } finally { setLoading(false) } }, [loadPerms, loadRoles, loadUsers])
  useEffect(() => { void refreshAll() }, [refreshAll])

  const treeData = useMemo(() => buildPermissionTree(perms, t), [perms, t])
  const defaultExpandedKeys = useMemo(() => collectKeysWithChildren(treeData), [treeData])

  const openEditPerms = async (row: OrgRole) => {
    try {
      const res = await getOrgRole(String(row.id))
      if (res.code !== 200 || !res.data) { showAlert(res.msg || t('tenantRoles.loadFailed'), 'error'); return }
      setPermRoleId(String(row.id)); setPermRoleName(row.name)
      const ids = res.data.permissionIds || []; setPermSelected(ids.map(String)); setPermModalOpen(true)
    } catch (e: unknown) { showAlert(extractApiErrorMessage(e, t('tenantRoles.loadFailed')), 'error') }
  }

  const columns: DataListColumn<Record<string, unknown>>[] = [
    {
      key: 'icon', width: 40,
      render: () => (
        <div className="flex h-9 w-9 shrink-0 items-center justify-center rounded-full bg-amber-50 text-amber-600">
          <Shield size={18} strokeWidth={1.75} />
        </div>
      ),
    },
    {
      key: 'info', title: t('tenantRoles.colName'),
      render: (_, r) => (
        <div className="min-w-0 flex-1">
          <div className="truncate text-sm font-medium text-neutral-900">{String(r.name ?? '—')}</div>
          <div className="mt-0.5 flex items-center gap-2 text-xs text-neutral-500">
            {r.description ? <span className="truncate">{String(r.description)}</span> : null}
            {r.isSystem ? <Tag color="blue" className="!rounded-full !text-xs">系统</Tag> : null}
          </div>
        </div>
      ),
    },
    {
      key: 'actions', width: 180, align: 'right',
      render: (_, r) => (
        <div className="flex items-center justify-end gap-1">
          <Button size="mini" icon={<Key size={12} />} disabled={!!r.isSystem} onClick={() => { if (!r.isSystem) void openEditPerms(r as unknown as OrgRole) }}>{t('tenantRoles.btnPermissions')}</Button>
          <Button size="mini" status="danger" icon={<Trash2 size={12} />} disabled={!!r.isSystem} onClick={() => { if (r.isSystem) return; Modal.confirm({ title: t('tenantRoles.deleteTitle'), content: t('tenantRoles.deleteConfirm').replace('{name}', String(r.name || '')), onOk: async () => { try { const res = await deleteOrgRole(String(r.id)); if (res.code !== 200) { showAlert(res.msg || t('tenantRoles.deleteFailed'), 'error'); return } showAlert(t('tenantRoles.deleted'), 'success'); await loadRoles() } catch (e: unknown) { showAlert(extractApiErrorMessage(e, t('tenantRoles.deleteFailed')), 'error') } } }) }}>{t('common.delete')}</Button>
        </div>
      ),
    },
  ]

  return (
    <BaseLayout title={t('pages.rolePermissions.title')} description={t('pages.rolePermissions.description')}>
      <DataList
        data={roles as unknown as (OrgRole & Record<string, unknown>)[]}
        columns={columns}
        loading={loading}
        rowKey="id"
        emptyText={t('common.noData')}
        header={
          <div className="flex items-center justify-between gap-3">
            <span className="text-sm font-medium text-neutral-900">{t('tenantRoles.title')}</span>
            <div className="flex items-center gap-2">
              <Button type="primary" icon={<Plus size={14} />} onClick={() => { roleForm.resetFields(); setRoleModalOpen(true) }}>{t('tenantRoles.newRole')}</Button>
              <Button type="outline" icon={<Users size={14} />} onClick={() => setAssignOpen(true)}>{t('tenantRoles.assignRoleToUser')}</Button>
            </div>
          </div>
        }
      />

      <Modal title={t('tenantRoles.modalNewRole')} visible={roleModalOpen} onCancel={() => setRoleModalOpen(false)}
        onOk={async () => {
          try {
            const v = await roleForm.validate(); const name = String(v.name || '').trim()
            if (!name) { showAlert(t('tenantRoles.nameRequired'), 'warning'); return }
            const r = await createOrgRole({ name, description: String(v.description || '').trim() })
            if (r.code !== 200) { showAlert(r.msg || t('tenantRoles.createFailed'), 'error'); return }
            showAlert(t('tenantRoles.created'), 'success'); setRoleModalOpen(false); await loadRoles()
          } catch (e: unknown) { if (e && typeof e === 'object' && 'errorFields' in e) return; showAlert(extractApiErrorMessage(e, t('tenantRoles.createFailed')), 'error') }
        }}
      >
        <Form form={roleForm} layout="vertical">
          <FormItem label={t('tenantRoles.formName')} field="name" rules={[{ required: true }]}><ArcoInput placeholder={t('tenantRoles.namePlaceholder')} /></FormItem>
          <FormItem label={t('tenantRoles.formDescription')} field="description"><ArcoInput placeholder={t('tenantRoles.optional')} /></FormItem>
        </Form>
      </Modal>

      <Modal title={permRoleName ? `${t('tenantRoles.modalRolePermissions')} — ${permRoleName}` : t('tenantRoles.modalRolePermissions')} style={{ width: 720 }} visible={permModalOpen} onCancel={() => setPermModalOpen(false)}
        onOk={async () => {
          if (permRoleId == null) return
          try {
            const permissionIds = permSelected.map(String).filter(Boolean)
            const r = await putOrgRolePermissions(permRoleId, { permissionIds })
            if (r.code !== 200) { showAlert(r.msg || t('tenantRoles.saveFailed'), 'error'); return }
            showAlert(t('tenantRoles.permissionsUpdated'), 'success'); setPermModalOpen(false)
          } catch (e: unknown) { showAlert(extractApiErrorMessage(e, t('tenantRoles.saveFailed')), 'error') }
        }}
      >
        <p className="mb-3 text-sm text-neutral-500">{t('tenantRoles.permHint')}</p>
        <div className="max-h-[420px] overflow-auto rounded-lg border border-neutral-200 bg-neutral-50 p-3">
          <Tree checkable blockNode treeData={treeData} checkedKeys={permSelected} defaultExpandedKeys={defaultExpandedKeys} onCheck={(keys) => setPermSelected(keys as string[])} />
        </div>
      </Modal>

      <Modal title={t('tenantRoles.modalAssignRole')} style={{ width: 560 }} visible={assignOpen} onCancel={() => setAssignOpen(false)}
        onOk={async () => {
          if (!assignUserId) { showAlert(t('tenantRoles.selectUserRequired'), 'warning'); return }
          try {
            const r = await putOrgTenantUserRoles(assignUserId, { roleIds: assignRoleIds })
            if (r.code !== 200) { showAlert(r.msg || t('tenantRoles.saveFailed'), 'error'); return }
            showAlert(t('tenantRoles.userRolesUpdated'), 'success'); setAssignOpen(false)
          } catch (e: unknown) { showAlert(extractApiErrorMessage(e, t('tenantRoles.saveFailed')), 'error') }
        }}
      >
        <div className="space-y-4">
          <div>
            <label className="mb-1 block text-sm text-neutral-500">{t('tenantRoles.sectionUser')}</label>
            <Select placeholder={t('tenantRoles.selectMember')} style={{ width: '100%' }} value={assignUserId} onChange={(v) => setAssignUserId(v != null ? String(v) : undefined)} options={users.map((u) => ({ value: String(u.id), label: `${u.displayName || u.username || u.email || u.id}` }))} showSearch />
          </div>
          <div>
            <label className="mb-1 block text-sm text-neutral-500">{t('tenantRoles.sectionRoles')}</label>
            <Select mode="multiple" placeholder={t('tenantRoles.selectRoles')} style={{ width: '100%' }} value={assignRoleIds} onChange={(v) => setAssignRoleIds((v as string[]).map(String))} options={roles.map((r) => ({ value: String(r.id), label: r.name + (r.isSystem ? t('tenantRoles.systemSuffix') : '') }))} />
          </div>
        </div>
      </Modal>
    </BaseLayout>
  )
}
