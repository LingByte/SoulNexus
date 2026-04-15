import { useEffect, useState } from 'react'
import { Lock, RefreshCw, Unlock, Search } from 'lucide-react'
import AdminLayout from '@/components/Layout/AdminLayout'
import Card from '@/components/UI/Card'
import Button from '@/components/UI/Button'
import Input from '@/components/UI/Input'
import Badge from '@/components/UI/Badge'
import EmptyState from '@/components/UI/EmptyState'
import { showAlert } from '@/utils/notification'
import { getAccountLocks, unlockAccount, type AccountLock } from '@/services/adminApi'

const AccountLocks = () => {
  const [locks, setLocks] = useState<AccountLock[]>([])
  const [loading, setLoading] = useState(false)
  const [page, setPage] = useState(1)
  const [pageSize] = useState(20)
  const [total, setTotal] = useState(0)
  const [email, setEmail] = useState('')
  const [active, setActive] = useState('')

  const fetchLocks = async () => {
    try {
      setLoading(true)
      const result = await getAccountLocks({
        page,
        page_size: pageSize,
        email: email || undefined,
        is_active: active === '' ? undefined : active === 'true',
      })
      setLocks(result.locks || [])
      setTotal(result.total || 0)
    } catch (error: any) {
      showAlert('获取账号锁定记录失败', 'error', error?.msg || error?.message)
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    fetchLocks()
  }, [page])

  const onSearch = () => {
    setPage(1)
    fetchLocks()
  }

  const onUnlock = async (id: number) => {
    try {
      await unlockAccount(id)
      showAlert('账号解锁成功', 'success')
      fetchLocks()
    } catch (error: any) {
      showAlert('账号解锁失败', 'error', error?.msg || error?.message)
    }
  }

  const totalPages = Math.ceil(total / pageSize)

  return (
    <AdminLayout title="账号锁定管理" description="查看并解锁被风控锁定的账号">
      <div className="space-y-6">
        <Card>
          <div className="flex flex-col sm:flex-row gap-3 items-end">
            <Input placeholder="邮箱" value={email} onChange={(e) => setEmail(e.target.value)} />
            <Input placeholder="是否激活( true / false )" value={active} onChange={(e) => setActive(e.target.value)} />
            <Button onClick={onSearch} leftIcon={<Search className="w-4 h-4" />}>搜索</Button>
            <Button variant="outline" onClick={fetchLocks} leftIcon={<RefreshCw className={`w-4 h-4 ${loading ? 'animate-spin' : ''}`} />}>刷新</Button>
          </div>
        </Card>

        <Card>
          {loading && locks.length === 0 ? (
            <div className="flex justify-center py-12"><RefreshCw className="w-6 h-6 animate-spin text-slate-400" /></div>
          ) : locks.length === 0 ? (
            <EmptyState icon={Lock} title="暂无锁定记录" description="当前没有匹配的账号锁定记录" />
          ) : (
            <div className="overflow-x-auto">
              <table className="w-full">
                <thead>
                  <tr className="border-b border-slate-200 dark:border-slate-700">
                    <th className="text-left py-3 px-4">邮箱</th>
                    <th className="text-left py-3 px-4">失败次数</th>
                    <th className="text-left py-3 px-4">IP</th>
                    <th className="text-left py-3 px-4">解锁时间</th>
                    <th className="text-left py-3 px-4">状态</th>
                    <th className="text-right py-3 px-4">操作</th>
                  </tr>
                </thead>
                <tbody>
                  {locks.map((lock) => (
                    <tr key={lock.id} className="border-b border-slate-100 dark:border-slate-800">
                      <td className="py-3 px-4">{lock.email}</td>
                      <td className="py-3 px-4">{lock.failedAttempts}</td>
                      <td className="py-3 px-4">{lock.ipAddress}</td>
                      <td className="py-3 px-4">{new Date(lock.unlockAt).toLocaleString('zh-CN')}</td>
                      <td className="py-3 px-4">
                        <Badge variant={lock.isActive ? 'error' : 'success'}>{lock.isActive ? '锁定中' : '已解锁'}</Badge>
                      </td>
                      <td className="py-3 px-4 text-right">
                        <Button
                          size="sm"
                          variant="ghost"
                          onClick={() => onUnlock(lock.id)}
                          disabled={!lock.isActive}
                          leftIcon={<Unlock className="w-4 h-4" />}
                        >
                          解锁
                        </Button>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}

          {totalPages > 1 && (
            <div className="flex items-center justify-between mt-4 pt-4 border-t border-slate-200 dark:border-slate-700">
              <span className="text-sm text-slate-500">共 {total} 条</span>
              <div className="flex items-center gap-2">
                <Button variant="outline" size="sm" onClick={() => setPage((p) => p - 1)} disabled={page === 1}>上一页</Button>
                <span className="text-sm text-slate-500">第 {page} / {totalPages} 页</span>
                <Button variant="outline" size="sm" onClick={() => setPage((p) => p + 1)} disabled={page >= totalPages}>下一页</Button>
              </div>
            </div>
          )}
        </Card>
      </div>
    </AdminLayout>
  )
}

export default AccountLocks
