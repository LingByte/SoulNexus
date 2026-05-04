import { useCallback, useEffect, useState } from 'react'
import { Users } from 'lucide-react'
import AdminLayout from '@/components/Layout/AdminLayout'
import Card from '@/components/UI/Card'
import Button from '@/components/UI/Button'
import Input from '@/components/UI/Input'
import Modal from '@/components/UI/Modal'
import Badge from '@/components/UI/Badge'
import { showAlert } from '@/utils/notification'
import {
  listRoles,
  createRole,
  updateRole,
  deleteRole,
  setRolePermissions,
  type Role,
} from '@/services/roleApi'
import { listPermissions, type PermissionListItem } from '@/services/permissionApi'

const Roles = () => {
  const [rows, setRows] = useState<Role[]>([])
  const [total, setTotal] = useState(0)
  const [page, setPage] = useState(1)
  const [search, setSearch] = useState('')
  const [loading, setLoading] = useState(false)

  const [modalOpen, setModalOpen] = useState(false)
  const [editing, setEditing] = useState<Role | null>(null)
  const [form, setForm] = useState({ name: '', slug: '', description: '' })

  const [permModalRole, setPermModalRole] = useState<Role | null>(null)
  const [allPerms, setAllPerms] = useState<PermissionListItem[]>([])
  const [permPick, setPermPick] = useState<Set<number>>(new Set())

  const load = useCallback(
    async (pageOverride?: number) => {
      const p = pageOverride ?? page
      setLoading(true)
      try {
        const data = await listRoles({ page: p, pageSize: 20, search: search.trim() || undefined })
        setRows(data.items || [])
        setTotal(data.total || 0)
      } catch (e: any) {
        showAlert(e?.message || '加载失败', 'error')
      } finally {
        setLoading(false)
      }
    },
    [page, search]
  )

  useEffect(() => {
    void load()
  }, [load])

  const loadAllPerms = async () => {
    const data = await listPermissions({ page: 1, pageSize: 500 })
    setAllPerms(data.items || [])
  }

  const openPermEditor = async (r: Role) => {
    setPermModalRole(r)
    const ids = new Set((r.permissions || []).map((x) => x.id))
    setPermPick(ids)
    if (allPerms.length === 0) await loadAllPerms()
  }

  const savePermPick = async () => {
    if (!permModalRole) return
    try {
      const updated = await setRolePermissions(permModalRole.id, [...permPick])
      showAlert('角色权限已保存', 'success')
      setPermModalRole(null)
      setRows((prev) => prev.map((x) => (x.id === updated.id ? updated : x)))
    } catch (e: any) {
      showAlert(e?.message || '保存失败', 'error')
    }
  }

  const openCreate = () => {
    setEditing(null)
    setForm({ name: '', slug: '', description: '' })
    setModalOpen(true)
  }

  const openEdit = (r: Role) => {
    setEditing(r)
    setForm({ name: r.name, slug: r.slug, description: r.description || '' })
    setModalOpen(true)
  }

  const save = async () => {
    try {
      if (editing) {
        await updateRole(editing.id, form)
        showAlert('已更新', 'success')
      } else {
        await createRole(form)
        showAlert('已创建', 'success')
      }
      setModalOpen(false)
      void load(page)
    } catch (e: any) {
      showAlert(e?.message || '保存失败', 'error')
    }
  }

  const remove = async (r: Role) => {
    if (r.isSystem) {
      showAlert('系统角色不可删除', 'error')
      return
    }
    if (!confirm(`删除角色「${r.slug}」？`)) return
    try {
      await deleteRole(r.id)
      showAlert('已删除', 'success')
      void load(page)
    } catch (e: any) {
      showAlert(e?.message || '删除失败', 'error')
    }
  }

  const togglePerm = (id: number) => {
    const n = new Set(permPick)
    if (n.has(id)) n.delete(id)
    else n.add(id)
    setPermPick(n)
  }

  return (
    <AdminLayout>
      <div className="max-w-6xl mx-auto space-y-6">
        <div className="flex items-center gap-3">
          <Users className="w-8 h-8 text-indigo-600" />
          <div>
            <h1 className="text-2xl font-semibold text-slate-900 dark:text-white">角色</h1>
            <p className="text-sm text-slate-500 dark:text-slate-400">列表已包含每个角色下的权限（关联查询）。</p>
          </div>
        </div>

        <Card className="p-4 space-y-4">
          <div className="flex flex-wrap gap-2 justify-between">
            <div className="flex flex-wrap gap-2">
              <Input
                placeholder="搜索 slug / 名称"
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
              <Button onClick={openCreate}>新建角色</Button>
            </div>
            <Button variant="secondary" onClick={() => void load()} disabled={loading}>
              刷新
            </Button>
          </div>

          <div className="space-y-4">
            {rows.map((r) => (
              <div
                key={r.id}
                className="border border-slate-200 dark:border-slate-700 rounded-lg p-4 space-y-2"
              >
                <div className="flex flex-wrap justify-between gap-2">
                  <div>
                    <span className="font-mono text-sm text-indigo-600">{r.slug}</span>
                    <span className="ml-2 font-medium">{r.name}</span>
                    {r.isSystem && (
                      <Badge className="ml-2" variant="secondary">
                        系统
                      </Badge>
                    )}
                  </div>
                  <div className="flex gap-2">
                    <Button size="sm" variant="secondary" onClick={() => void openPermEditor(r)}>
                      分配权限
                    </Button>
                    <Button size="sm" variant="secondary" onClick={() => openEdit(r)}>
                      编辑
                    </Button>
                    <Button size="sm" variant="secondary" onClick={() => void remove(r)}>
                      删除
                    </Button>
                  </div>
                </div>
                <div className="flex flex-wrap gap-1">
                  {(r.permissions || []).length === 0 && (
                    <span className="text-xs text-slate-500">暂无权限</span>
                  )}
                  {(r.permissions || []).map((p) => (
                    <Badge key={p.id} variant="outline" className="text-xs font-normal">
                      {p.key}
                    </Badge>
                  ))}
                </div>
              </div>
            ))}
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

        <Modal isOpen={modalOpen} onClose={() => setModalOpen(false)} title={editing ? '编辑角色' : '新建角色'}>
          <div className="space-y-3 p-1">
            <div>
              <label className="text-xs text-slate-500">Slug</label>
              <Input
                value={form.slug}
                disabled={!!editing?.isSystem}
                onChange={(e) => setForm((f) => ({ ...f, slug: e.target.value }))}
              />
            </div>
            <div>
              <label className="text-xs text-slate-500">名称</label>
              <Input value={form.name} onChange={(e) => setForm((f) => ({ ...f, name: e.target.value }))} />
            </div>
            <div>
              <label className="text-xs text-slate-500">描述</label>
              <Input
                value={form.description}
                onChange={(e) => setForm((f) => ({ ...f, description: e.target.value }))}
              />
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
          isOpen={!!permModalRole}
          onClose={() => setPermModalRole(null)}
          title={permModalRole ? `分配权限 — ${permModalRole.slug}` : ''}
          size="lg"
        >
          <div className="max-h-[60vh] overflow-y-auto grid sm:grid-cols-2 gap-2 p-1">
            {allPerms.map((p) => (
              <label key={p.id} className="flex items-start gap-2 text-xs cursor-pointer">
                <input
                  type="checkbox"
                  checked={permPick.has(p.id)}
                  onChange={() => togglePerm(p.id)}
                  className="mt-0.5"
                />
                <span>
                  <span className="font-mono text-indigo-600">{p.key}</span>
                  <span className="text-slate-600 dark:text-slate-300 ml-1">{p.name}</span>
                </span>
              </label>
            ))}
          </div>
          <div className="flex justify-end gap-2 mt-4">
            <Button variant="secondary" onClick={() => setPermModalRole(null)}>
              取消
            </Button>
            <Button onClick={() => void savePermPick()}>保存</Button>
          </div>
        </Modal>
      </div>
    </AdminLayout>
  )
}

export default Roles
