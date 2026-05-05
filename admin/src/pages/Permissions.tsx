import { useCallback, useEffect, useState } from 'react'
import { KeyRound } from 'lucide-react'
import AdminLayout from '@/components/Layout/AdminLayout'
import Card from '@/components/UI/Card'
import Button from '@/components/UI/Button'
import Input from '@/components/UI/Input'
import Modal from '@/components/UI/Modal'
import { showAlert } from '@/utils/notification'
import {
  listPermissions,
  createPermission,
  updatePermission,
  deletePermission,
  type PermissionListItem,
} from '@/services/permissionApi'

const Permissions = () => {
  const [rows, setRows] = useState<PermissionListItem[]>([])
  const [total, setTotal] = useState(0)
  const [page, setPage] = useState(1)
  const [search, setSearch] = useState('')
  const [withRoles, setWithRoles] = useState(false)
  const [loading, setLoading] = useState(false)

  const [modalOpen, setModalOpen] = useState(false)
  const [editing, setEditing] = useState<PermissionListItem | null>(null)
  const [form, setForm] = useState({ key: '', name: '', description: '', resource: '' })

  const [viewRoles, setViewRoles] = useState<PermissionListItem | null>(null)

  const load = useCallback(
    async (pageOverride?: number) => {
      const p = pageOverride ?? page
      setLoading(true)
      try {
        const data = await listPermissions({
          page: p,
          pageSize: 20,
          search: search.trim() || undefined,
          withRoles,
        })
        setRows(data.items || [])
        setTotal(data.total || 0)
      } catch (e: any) {
        showAlert(e?.message || '加载失败', 'error')
      } finally {
        setLoading(false)
      }
    },
    [page, search, withRoles]
  )

  useEffect(() => {
    void load()
  }, [load])

  const openCreate = () => {
    setEditing(null)
    setForm({ key: '', name: '', description: '', resource: '' })
    setModalOpen(true)
  }

  const openEdit = (row: PermissionListItem) => {
    setEditing(row)
    setForm({
      key: row.key,
      name: row.name,
      description: row.description || '',
      resource: row.resource || '',
    })
    setModalOpen(true)
  }

  const save = async () => {
    try {
      if (editing) {
        await updatePermission(editing.id, form)
        showAlert('已更新', 'success')
      } else {
        await createPermission(form)
        showAlert('已创建', 'success')
      }
      setModalOpen(false)
      void load(page)
    } catch (e: any) {
      showAlert(e?.message || '保存失败', 'error')
    }
  }

  const remove = async (row: PermissionListItem) => {
    if (!confirm(`删除权限「${row.key}」？`)) return
    try {
      await deletePermission(row.id)
      showAlert('已删除', 'success')
      void load(page)
    } catch (e: any) {
      showAlert(e?.message || '删除失败', 'error')
    }
  }

  return (
    <AdminLayout>
      <div className="max-w-6xl mx-auto space-y-6">
        <div className="flex items-center gap-3">
          <KeyRound className="w-8 h-8 text-indigo-600" />
          <div>
            <h1 className="text-2xl font-semibold text-slate-900 dark:text-white">权限</h1>
            <p className="text-sm text-slate-500 dark:text-slate-400">
              维护权限点；列表展示关联角色数与直连用户数，可选加载关联角色。
            </p>
          </div>
        </div>

        <Card className="p-4 space-y-4">
          <div className="flex flex-wrap gap-3 items-center justify-between">
            <div className="flex flex-wrap gap-2 items-center">
              <Input
                placeholder="搜索 key / 名称"
                value={search}
                onChange={(e) => setSearch(e.target.value)}
                className="w-56"
              />
              <Button
                variant="secondary"
                onClick={() => {
                  setPage(1)
                  void load(1)
                }}
              >
                搜索
              </Button>
              <label className="flex items-center gap-2 text-sm text-slate-600 dark:text-slate-300 cursor-pointer">
                <input type="checkbox" checked={withRoles} onChange={(e) => setWithRoles(e.target.checked)} />
                加载关联角色
              </label>
              <Button onClick={openCreate}>新建权限</Button>
            </div>
            <Button variant="secondary" onClick={() => void load()} disabled={loading}>
              刷新
            </Button>
          </div>

          <div className="overflow-x-auto">
            <table className="min-w-full text-sm">
              <thead>
                <tr className="border-b border-slate-200 dark:border-slate-700 text-left">
                  <th className="py-2 pr-4">Key</th>
                  <th className="py-2 pr-4">名称</th>
                  <th className="py-2 pr-4">资源</th>
                  <th className="py-2 pr-4">关联角色数</th>
                  <th className="py-2 pr-4">直连用户授权</th>
                  <th className="py-2">操作</th>
                </tr>
              </thead>
              <tbody>
                {rows.map((row) => (
                  <tr key={row.id} className="border-b border-slate-100 dark:border-slate-800">
                    <td className="py-2 pr-4 font-mono text-xs">{row.key}</td>
                    <td className="py-2 pr-4">{row.name}</td>
                    <td className="py-2 pr-4">{row.resource || '—'}</td>
                    <td className="py-2 pr-4">{row.roleCount ?? 0}</td>
                    <td className="py-2 pr-4">{row.directUserGrantCount ?? 0}</td>
                    <td className="py-2 flex flex-wrap gap-2">
                      {withRoles && row.roles && row.roles.length > 0 && (
                        <Button size="sm" variant="secondary" onClick={() => setViewRoles(row)}>
                          查看角色
                        </Button>
                      )}
                      <Button size="sm" variant="secondary" onClick={() => openEdit(row)}>
                        编辑
                      </Button>
                      <Button size="sm" variant="secondary" onClick={() => void remove(row)}>
                        删除
                      </Button>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>

          <div className="flex justify-between text-sm text-slate-500">
            <span>
              共 {total} 条 · 第 {page} 页
            </span>
            <div className="flex gap-2">
              <Button
                size="sm"
                variant="secondary"
                disabled={page <= 1}
                onClick={() => {
                  const p = Math.max(1, page - 1)
                  setPage(p)
                  void load(p)
                }}
              >
                上一页
              </Button>
              <Button
                size="sm"
                variant="secondary"
                onClick={() => {
                  const p = page + 1
                  setPage(p)
                  void load(p)
                }}
              >
                下一页
              </Button>
            </div>
          </div>
        </Card>

        <Modal isOpen={modalOpen} onClose={() => setModalOpen(false)} title={editing ? '编辑权限' : '新建权限'}>
          <div className="space-y-3 p-1">
            <div>
              <label className="text-xs text-slate-500">Key</label>
              <Input value={form.key} onChange={(e) => setForm((f) => ({ ...f, key: e.target.value }))} />
            </div>
            <div>
              <label className="text-xs text-slate-500">名称</label>
              <Input value={form.name} onChange={(e) => setForm((f) => ({ ...f, name: e.target.value }))} />
            </div>
            <div>
              <label className="text-xs text-slate-500">描述</label>
              <Input value={form.description} onChange={(e) => setForm((f) => ({ ...f, description: e.target.value }))} />
            </div>
            <div>
              <label className="text-xs text-slate-500">资源分组</label>
              <Input value={form.resource} onChange={(e) => setForm((f) => ({ ...f, resource: e.target.value }))} />
            </div>
            <div className="flex justify-end gap-2 pt-2">
              <Button variant="secondary" onClick={() => setModalOpen(false)}>
                取消
              </Button>
              <Button onClick={() => void save()}>保存</Button>
            </div>
          </div>
        </Modal>

        <Modal
          isOpen={!!viewRoles}
          onClose={() => setViewRoles(null)}
          title={viewRoles ? `关联角色 — ${viewRoles.key}` : ''}
        >
          <ul className="text-sm space-y-2 max-h-72 overflow-y-auto">
            {(viewRoles?.roles || []).map((r) => (
              <li key={r.id}>
                <span className="font-mono text-xs text-indigo-600">{r.slug}</span>
                <span className="ml-2">{r.name}</span>
              </li>
            ))}
            {!viewRoles?.roles?.length && <li className="text-slate-500">暂无</li>}
          </ul>
        </Modal>
      </div>
    </AdminLayout>
  )
}

export default Permissions
