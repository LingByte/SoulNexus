import { useState, useEffect } from 'react'
import { motion } from 'framer-motion'
import { History, CheckCircle2, XCircle, AlertTriangle, Search, RefreshCw } from 'lucide-react'
import AdminLayout from '@/components/Layout/AdminLayout'
import Card from '@/components/UI/Card'
import Button from '@/components/UI/Button'
import Input from '@/components/UI/Input'
import Badge from '@/components/UI/Badge'
import EmptyState from '@/components/UI/EmptyState'
import { Select, SelectTrigger, SelectContent, SelectItem, SelectValue } from '@/components/UI/Select'
import { getLoginHistory, type LoginHistory } from '@/services/adminApi'
import { showAlert } from '@/utils/notification'
import { cn } from '@/utils/cn'

const LoginHistoryPage = () => {
  const [histories, setHistories] = useState<LoginHistory[]>([])
  const [loading, setLoading] = useState(false)
  const [page, setPage] = useState(1)
  const [pageSize] = useState(20)
  const [total, setTotal] = useState(0)
  const [search, setSearch] = useState('')
  const [successFilter, setSuccessFilter] = useState<string>('')
  const [suspiciousFilter, setSuspiciousFilter] = useState<string>('')

  // Reset to first page when filters change
  useEffect(() => {
    if (page === 1) {
      fetchHistory()
    } else {
      setPage(1)
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [search, successFilter, suspiciousFilter])

  // Fetch when page changes
  useEffect(() => {
    fetchHistory()
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [page])

  const fetchHistory = async () => {
    try {
      setLoading(true)
      const params: any = { page, page_size: pageSize }
      if (search) params.search = search
      if (successFilter) params.success = successFilter === 'true'
      if (suspiciousFilter) params.is_suspicious = suspiciousFilter === 'true'
      
      const data = await getLoginHistory(params)
      setHistories(data.histories)
      setTotal(data.total)
    } catch (error: any) {
      showAlert('获取登录历史失败', 'error', error?.msg || error?.message)
    } finally {
      setLoading(false)
    }
  }

  const formatDate = (dateStr: string) => {
    return new Date(dateStr).toLocaleString('zh-CN')
  }

  const totalPages = Math.ceil(total / pageSize)

  return (
    <AdminLayout title="登录历史" description="查看登录历史记录">
      <div className="space-y-6">
        {/* 搜索和过滤 */}
        <Card>
          <div className="flex flex-col sm:flex-row gap-4">
            <div className="flex-1 relative">
              <Search className="absolute left-3 top-1/2 transform -translate-y-1/2 w-4 h-4 text-muted-foreground" />
              <Input
                placeholder="搜索邮箱、IP地址或位置..."
                value={search}
                onChange={(e) => setSearch(e.target.value)}
                className="pl-10"
              />
            </div>
            <Select value={successFilter} onValueChange={setSuccessFilter}>
              <SelectTrigger className="w-full sm:w-[180px]">
                <SelectValue placeholder="登录状态">
                  {successFilter === '' ? '登录状态: 全部' : successFilter === 'true' ? '登录状态: 成功' : '登录状态: 失败'}
                </SelectValue>
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="">全部</SelectItem>
                <SelectItem value="true">成功</SelectItem>
                <SelectItem value="false">失败</SelectItem>
              </SelectContent>
            </Select>
            <Select value={suspiciousFilter} onValueChange={setSuspiciousFilter}>
              <SelectTrigger className="w-full sm:w-[180px]">
                <SelectValue placeholder="可疑登录">
                  {suspiciousFilter === '' ? '可疑登录: 全部' : suspiciousFilter === 'true' ? '可疑登录: 是' : '可疑登录: 否'}
                </SelectValue>
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="">全部</SelectItem>
                <SelectItem value="true">是</SelectItem>
                <SelectItem value="false">否</SelectItem>
              </SelectContent>
            </Select>
            <Button
              variant="outline"
              onClick={fetchHistory}
              disabled={loading}
              leftIcon={<RefreshCw className={cn('w-4 h-4', loading && 'animate-spin')} />}
            >
              刷新
            </Button>
          </div>
        </Card>

        {/* 登录历史列表 */}
        <Card>
          {loading && histories.length === 0 ? (
            <div className="flex items-center justify-center py-12">
              <RefreshCw className="w-6 h-6 animate-spin text-muted-foreground" />
            </div>
          ) : histories.length === 0 ? (
            <EmptyState
              icon={History}
              title="暂无登录历史"
              description="还没有登录记录"
            />
          ) : (
            <div className="overflow-x-auto">
              <table className="w-full">
                <thead>
                  <tr className="border-b border-border">
                    <th className="text-left p-4 font-medium text-sm w-[10%]">状态</th>
                    <th className="text-left p-4 font-medium text-sm w-[20%]">邮箱</th>
                    <th className="text-left p-4 font-medium text-sm w-[12%]">IP地址</th>
                    <th className="text-left p-4 font-medium text-sm w-[15%]">位置</th>
                    <th className="text-left p-4 font-medium text-sm w-[25%]">设备信息</th>
                    <th className="text-left p-4 font-medium text-sm w-[15%]">时间</th>
                    <th className="text-center p-4 font-medium text-sm w-[8%]">可疑</th>
                  </tr>
                </thead>
                <tbody>
                  {histories.map((history) => (
                    <motion.tr
                      key={history.id}
                      initial={{ opacity: 0, y: 10 }}
                      animate={{ opacity: 1, y: 0 }}
                      className="border-b border-border hover:bg-accent/50 transition-colors"
                    >
                      <td className="p-4">
                        <div className="flex items-center gap-2">
                          {history.success ? (
                            <CheckCircle2 className="w-5 h-5 text-green-600 dark:text-green-400" />
                          ) : (
                            <XCircle className="w-5 h-5 text-red-600 dark:text-red-400" />
                          )}
                          <Badge variant={history.success ? 'default' : 'destructive'}>
                            {history.success ? '成功' : '失败'}
                          </Badge>
                        </div>
                      </td>
                      <td className="p-4">
                        <code className="text-sm font-mono bg-muted px-2 py-1 rounded">
                          {history.email}
                        </code>
                      </td>
                      <td className="p-4">
                        <code className="text-sm font-mono bg-muted px-2 py-1 rounded">
                          {history.ipAddress}
                        </code>
                      </td>
                      <td className="p-4 text-sm">
                        <div>
                          <div>{history.location}</div>
                          <div className="text-muted-foreground text-xs">
                            {history.country}, {history.city}
                          </div>
                        </div>
                      </td>
                      <td className="p-4">
                        <div className="text-sm text-muted-foreground max-w-xs truncate" title={history.userAgent}>
                          {history.userAgent}
                        </div>
                      </td>
                      <td className="p-4 text-sm">
                        {formatDate(history.createdAt)}
                      </td>
                      <td className="p-4 text-center">
                        {history.isSuspicious ? (
                          <div className="flex items-center justify-center">
                            <AlertTriangle className="w-5 h-5 text-orange-600 dark:text-orange-400" />
                          </div>
                        ) : (
                          <div className="flex items-center justify-center">
                            <CheckCircle2 className="w-5 h-5 text-green-600 dark:text-green-400" />
                          </div>
                        )}
                      </td>
                    </motion.tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}

          {/* 分页 */}
          {totalPages > 1 && (
            <div className="flex items-center justify-between mt-4 pt-4 border-t border-border">
              <div className="text-sm text-muted-foreground">
                共 {total} 条，第 {page} / {totalPages} 页
              </div>
              <div className="flex gap-2">
                <Button
                  variant="outline"
                  size="sm"
                  onClick={() => setPage((p) => Math.max(1, p - 1))}
                  disabled={page === 1}
                >
                  上一页
                </Button>
                <Button
                  variant="outline"
                  size="sm"
                  onClick={() => setPage((p) => Math.min(totalPages, p + 1))}
                  disabled={page === totalPages}
                >
                  下一页
                </Button>
              </div>
            </div>
          )}
        </Card>
      </div>
    </AdminLayout>
  )
}

export default LoginHistoryPage
