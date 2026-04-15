import { useEffect, useState } from 'react'
import AdminLayout from '@/components/Layout/AdminLayout'
import Card from '@/components/UI/Card'
import Button from '@/components/UI/Button'
import Input from '@/components/UI/Input'
import { listAdminGroups, updateAdminGroup, deleteAdminGroup, type AdminGroup } from '@/services/adminApi'
import { showAlert } from '@/utils/notification'

const Groups = () => {
  const [groups, setGroups] = useState<AdminGroup[]>([])
  const [loading, setLoading] = useState(false)
  const [search, setSearch] = useState('')
  const [page, setPage] = useState(1)
  const [total, setTotal] = useState(0)
  const pageSize = 20

  const fetchData = async () => {
    try {
      setLoading(true)
      const res = await listAdminGroups({ page, pageSize, search })
      setGroups(res.groups || [])
      setTotal(res.total || 0)
    } catch (e: any) {
      showAlert('加载企业列表失败', 'error', e?.msg || e?.message)
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => { fetchData() }, [page, search])

  const handleArchiveToggle = async (group: AdminGroup) => {
    await updateAdminGroup(group.id, { isArchived: !group.isArchived })
    fetchData()
  }

  return (
    <AdminLayout title="企业管理" description="管理组织与企业信息">
      <Card className="space-y-4">
        <div className="flex gap-3">
          <Input value={search} onChange={(e) => setSearch(e.target.value)} placeholder="搜索名称/类型" className="max-w-sm" />
          <Button variant="outline" onClick={fetchData}>刷新</Button>
        </div>
        <div className="overflow-x-auto">
          <table className="w-full text-sm">
            <thead><tr className="border-b"><th className="py-2 text-left">ID</th><th className="text-left">名称</th><th className="text-left">类型</th><th className="text-left">成员数</th><th className="text-left">状态</th><th className="text-left">操作</th></tr></thead>
            <tbody>
              {groups.map(g => (
                <tr key={g.id} className="border-b">
                  <td className="py-2">{g.id}</td><td>{g.name}</td><td>{g.type || '-'}</td><td>{g.memberCount || 0}</td><td>{g.isArchived ? '已归档' : '正常'}</td>
                  <td className="flex gap-2 py-2">
                    <Button size="sm" variant="outline" onClick={() => handleArchiveToggle(g)}>{g.isArchived ? '取消归档' : '归档'}</Button>
                    <Button size="sm" variant="ghost" className="text-red-600" onClick={async () => { await deleteAdminGroup(g.id); fetchData() }}>删除</Button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
        <div className="flex justify-between text-sm"><span>共 {total} 条</span><div className="flex gap-2"><Button size="sm" variant="outline" disabled={page <= 1} onClick={() => setPage(p => p - 1)}>上一页</Button><Button size="sm" variant="outline" disabled={page * pageSize >= total} onClick={() => setPage(p => p + 1)}>下一页</Button></div></div>
        {loading && <div className="text-sm text-slate-500">加载中...</div>}
      </Card>
    </AdminLayout>
  )
}

export default Groups
