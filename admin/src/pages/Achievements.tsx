import { useCallback, useEffect, useState } from 'react'
import AdminLayout from '@/components/Layout/AdminLayout'
import { get, post, put, del } from '@/utils/request'
import { getApiBaseURL } from '@/config/apiConfig'
import { showAlert } from '@/utils/notification'
import { Plus, Pencil, Trash2, Search, ChevronLeft, ChevronRight, X } from 'lucide-react'

interface Achievement {
  id: number
  key: string
  title: string
  description: string
  icon: string
  category: string
  threshold: number
  rewardPoints: number
  sortOrder: number
  isActive: boolean
  createdAt: string
}

const emptyForm: Partial<Achievement> = {
  key: '',
  title: '',
  description: '',
  icon: '',
  category: 'streak',
  threshold: 1,
  rewardPoints: 0,
  sortOrder: 0,
  isActive: true,
}

export default function Achievements() {
  const [list, setList] = useState<Achievement[]>([])
  const [total, setTotal] = useState(0)
  const [page, setPage] = useState(1)
  const [keyword, setKeyword] = useState('')
  const pageSize = 20
  const [loading, setLoading] = useState(false)

  const [modalOpen, setModalOpen] = useState(false)
  const [editing, setEditing] = useState<Achievement | null>(null)
  const [form, setForm] = useState<Partial<Achievement>>({ ...emptyForm })
  const [saving, setSaving] = useState(false)

  const fetchList = useCallback(async () => {
    setLoading(true)
    try {
      const params = new URLSearchParams({ page: String(page), pageSize: String(pageSize) })
      if (keyword.trim()) params.append('keyword', keyword.trim())
      const res = await get<any>(`${getApiBaseURL()}/admin/achievements?${params}`)
      if (res.code === 200) {
        setList(res.data?.list || [])
        setTotal(res.data?.total || 0)
      }
    } finally {
      setLoading(false)
    }
  }, [page, pageSize, keyword])

  useEffect(() => { fetchList() }, [fetchList])

  const openCreate = () => {
    setEditing(null)
    setForm({ ...emptyForm })
    setModalOpen(true)
  }

  const openEdit = (a: Achievement) => {
    setEditing(a)
    setForm({ ...a })
    setModalOpen(true)
  }

  const handleSave = async () => {
    if (!form.key?.trim() || !form.title?.trim() || !form.category?.trim()) {
      showAlert('key、title、category 为必填', 'error')
      return
    }
    setSaving(true)
    try {
      if (editing) {
        await put(`${getApiBaseURL()}/admin/achievements/${editing.id}`, form)
        showAlert('更新成功', 'success')
      } else {
        await post(`${getApiBaseURL()}/admin/achievements`, form)
        showAlert('创建成功', 'success')
      }
      setModalOpen(false)
      fetchList()
    } catch (e: any) {
      showAlert(e?.msg || e?.message || '操作失败', 'error')
    } finally {
      setSaving(false)
    }
  }

  const handleDelete = async (a: Achievement) => {
    if (!confirm(`确认删除成就「${a.title}」？`)) return
    try {
      await del(`${getApiBaseURL()}/admin/achievements/${a.id}`)
      showAlert('删除成功', 'success')
      fetchList()
    } catch (e: any) {
      showAlert(e?.msg || e?.message || '删除失败', 'error')
    }
  }

  const totalPages = Math.max(1, Math.ceil(total / pageSize))

  return (
    <AdminLayout>
      <div className="p-6 space-y-6">
        <div className="flex items-center justify-between">
          <div>
            <h1 className="text-2xl font-bold text-slate-900 dark:text-slate-100">成就管理</h1>
            <p className="text-slate-500 dark:text-slate-400 mt-1">管理成就定义（阈值、奖励、排序、启用状态）</p>
          </div>
          <button
            onClick={openCreate}
            className="inline-flex items-center gap-2 px-4 py-2 bg-blue-600 hover:bg-blue-700 text-white rounded-lg"
          >
            <Plus className="w-4 h-4" />
            新建成就
          </button>
        </div>

        <div className="bg-white dark:bg-slate-900 rounded-xl border border-slate-200 dark:border-slate-800 p-4">
          <div className="flex items-center gap-3">
            <div className="relative flex-1">
              <Search className="w-4 h-4 text-slate-400 absolute left-3 top-1/2 -translate-y-1/2" />
              <input
                value={keyword}
                onChange={(e) => { setKeyword(e.target.value); setPage(1) }}
                placeholder="搜索 title / key"
                className="w-full pl-9 pr-3 py-2 border border-slate-200 dark:border-slate-700 rounded-lg bg-transparent"
              />
            </div>
            <button
              onClick={fetchList}
              className="px-4 py-2 rounded-lg border border-slate-200 dark:border-slate-700 hover:bg-slate-50 dark:hover:bg-slate-800"
            >
              刷新
            </button>
          </div>
        </div>

        <div className="bg-white dark:bg-slate-900 rounded-xl border border-slate-200 dark:border-slate-800 overflow-hidden">
          <div className="overflow-x-auto">
            <table className="min-w-full">
              <thead className="bg-slate-50 dark:bg-slate-800">
                <tr className="text-left text-sm text-slate-600 dark:text-slate-300">
                  <th className="px-4 py-3">ID</th>
                  <th className="px-4 py-3">Key</th>
                  <th className="px-4 py-3">标题</th>
                  <th className="px-4 py-3">分类</th>
                  <th className="px-4 py-3">阈值</th>
                  <th className="px-4 py-3">积分</th>
                  <th className="px-4 py-3">排序</th>
                  <th className="px-4 py-3">启用</th>
                  <th className="px-4 py-3">操作</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-slate-100 dark:divide-slate-800">
                {loading ? (
                  <tr><td colSpan={9} className="px-4 py-8 text-center text-slate-500">加载中...</td></tr>
                ) : list.length === 0 ? (
                  <tr><td colSpan={9} className="px-4 py-8 text-center text-slate-500">暂无数据</td></tr>
                ) : (
                  list.map((a) => (
                    <tr key={a.id} className="text-sm">
                      <td className="px-4 py-3 text-slate-500">{a.id}</td>
                      <td className="px-4 py-3 font-mono">{a.key}</td>
                      <td className="px-4 py-3">{a.title}</td>
                      <td className="px-4 py-3">{a.category}</td>
                      <td className="px-4 py-3">{a.threshold}</td>
                      <td className="px-4 py-3">{a.rewardPoints}</td>
                      <td className="px-4 py-3">{a.sortOrder}</td>
                      <td className="px-4 py-3">{a.isActive ? '是' : '否'}</td>
                      <td className="px-4 py-3">
                        <div className="flex items-center gap-2">
                          <button
                            onClick={() => openEdit(a)}
                            className="inline-flex items-center gap-1 px-3 py-1.5 rounded-md border border-slate-200 dark:border-slate-700 hover:bg-slate-50 dark:hover:bg-slate-800"
                          >
                            <Pencil className="w-4 h-4" /> 编辑
                          </button>
                          <button
                            onClick={() => handleDelete(a)}
                            className="inline-flex items-center gap-1 px-3 py-1.5 rounded-md border border-red-200 dark:border-red-900 text-red-600 hover:bg-red-50 dark:hover:bg-red-950/20"
                          >
                            <Trash2 className="w-4 h-4" /> 删除
                          </button>
                        </div>
                      </td>
                    </tr>
                  ))
                )}
              </tbody>
            </table>
          </div>

          <div className="flex items-center justify-between p-4 border-t border-slate-200 dark:border-slate-800">
            <div className="text-sm text-slate-500">共 {total} 条</div>
            <div className="flex items-center gap-2">
              <button
                onClick={() => setPage((p) => Math.max(1, p - 1))}
                disabled={page <= 1}
                className="inline-flex items-center gap-1 px-3 py-2 rounded-lg border border-slate-200 dark:border-slate-700 disabled:opacity-50"
              >
                <ChevronLeft className="w-4 h-4" /> 上一页
              </button>
              <div className="text-sm text-slate-600 dark:text-slate-300">{page} / {totalPages}</div>
              <button
                onClick={() => setPage((p) => Math.min(totalPages, p + 1))}
                disabled={page >= totalPages}
                className="inline-flex items-center gap-1 px-3 py-2 rounded-lg border border-slate-200 dark:border-slate-700 disabled:opacity-50"
              >
                下一页 <ChevronRight className="w-4 h-4" />
              </button>
            </div>
          </div>
        </div>

        {modalOpen && (
          <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50 p-4">
            <div className="w-full max-w-2xl bg-white dark:bg-slate-900 rounded-xl border border-slate-200 dark:border-slate-800 shadow-xl">
              <div className="flex items-center justify-between px-5 py-4 border-b border-slate-200 dark:border-slate-800">
                <div className="font-semibold text-slate-900 dark:text-slate-100">{editing ? '编辑成就' : '新建成就'}</div>
                <button
                  onClick={() => setModalOpen(false)}
                  className="p-2 rounded-md hover:bg-slate-100 dark:hover:bg-slate-800"
                >
                  <X className="w-4 h-4" />
                </button>
              </div>

              <div className="p-5 grid grid-cols-1 md:grid-cols-2 gap-4">
                <div>
                  <div className="text-sm text-slate-500 mb-1">Key</div>
                  <input
                    value={form.key || ''}
                    onChange={(e) => setForm((f) => ({ ...f, key: e.target.value }))}
                    className="w-full px-3 py-2 rounded-lg border border-slate-200 dark:border-slate-700 bg-transparent font-mono"
                    placeholder="streak_7"
                  />
                </div>
                <div>
                  <div className="text-sm text-slate-500 mb-1">标题</div>
                  <input
                    value={form.title || ''}
                    onChange={(e) => setForm((f) => ({ ...f, title: e.target.value }))}
                    className="w-full px-3 py-2 rounded-lg border border-slate-200 dark:border-slate-700 bg-transparent"
                    placeholder="一周坚持"
                  />
                </div>
                <div className="md:col-span-2">
                  <div className="text-sm text-slate-500 mb-1">描述</div>
                  <textarea
                    value={form.description || ''}
                    onChange={(e) => setForm((f) => ({ ...f, description: e.target.value }))}
                    className="w-full px-3 py-2 rounded-lg border border-slate-200 dark:border-slate-700 bg-transparent"
                    rows={3}
                    placeholder="连续学习7天"
                  />
                </div>
                <div>
                  <div className="text-sm text-slate-500 mb-1">分类</div>
                  <input
                    value={form.category || ''}
                    onChange={(e) => setForm((f) => ({ ...f, category: e.target.value }))}
                    className="w-full px-3 py-2 rounded-lg border border-slate-200 dark:border-slate-700 bg-transparent"
                    placeholder="streak / words / mastered / review / milestone"
                  />
                </div>
                <div>
                  <div className="text-sm text-slate-500 mb-1">Icon</div>
                  <input
                    value={form.icon || ''}
                    onChange={(e) => setForm((f) => ({ ...f, icon: e.target.value }))}
                    className="w-full px-3 py-2 rounded-lg border border-slate-200 dark:border-slate-700 bg-transparent"
                    placeholder="trophy"
                  />
                </div>
                <div>
                  <div className="text-sm text-slate-500 mb-1">阈值</div>
                  <input
                    type="number"
                    value={form.threshold ?? 1}
                    onChange={(e) => setForm((f) => ({ ...f, threshold: Number(e.target.value) }))}
                    className="w-full px-3 py-2 rounded-lg border border-slate-200 dark:border-slate-700 bg-transparent"
                  />
                </div>
                <div>
                  <div className="text-sm text-slate-500 mb-1">积分</div>
                  <input
                    type="number"
                    value={form.rewardPoints ?? 0}
                    onChange={(e) => setForm((f) => ({ ...f, rewardPoints: Number(e.target.value) }))}
                    className="w-full px-3 py-2 rounded-lg border border-slate-200 dark:border-slate-700 bg-transparent"
                  />
                </div>
                <div>
                  <div className="text-sm text-slate-500 mb-1">排序</div>
                  <input
                    type="number"
                    value={form.sortOrder ?? 0}
                    onChange={(e) => setForm((f) => ({ ...f, sortOrder: Number(e.target.value) }))}
                    className="w-full px-3 py-2 rounded-lg border border-slate-200 dark:border-slate-700 bg-transparent"
                  />
                </div>
                <div>
                  <div className="text-sm text-slate-500 mb-1">启用</div>
                  <select
                    value={String(form.isActive ?? true)}
                    onChange={(e) => setForm((f) => ({ ...f, isActive: e.target.value === 'true' }))}
                    className="w-full px-3 py-2 rounded-lg border border-slate-200 dark:border-slate-700 bg-transparent"
                  >
                    <option value="true">是</option>
                    <option value="false">否</option>
                  </select>
                </div>
              </div>

              <div className="flex justify-end gap-2 px-5 py-4 border-t border-slate-200 dark:border-slate-800">
                <button
                  onClick={() => setModalOpen(false)}
                  className="px-4 py-2 rounded-lg border border-slate-200 dark:border-slate-700"
                >
                  取消
                </button>
                <button
                  onClick={handleSave}
                  disabled={saving}
                  className="px-4 py-2 rounded-lg bg-blue-600 hover:bg-blue-700 text-white disabled:opacity-50"
                >
                  {saving ? '保存中...' : '保存'}
                </button>
              </div>
            </div>
          </div>
        )}
      </div>
    </AdminLayout>
  )
}
