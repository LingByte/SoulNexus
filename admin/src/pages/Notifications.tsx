import { useState, useEffect, useMemo } from 'react'
import { CheckCheck, Trash2, Search, CheckCircle2, AlertCircle, Info, XCircle, Eye } from 'lucide-react'
import AdminLayout from '@/components/Layout/AdminLayout'
import Card from '@/components/UI/Card'
import Button from '@/components/UI/Button'
import Input from '@/components/UI/Input'
import ConfirmDialog from '@/components/UI/ConfirmDialog'
import Badge from '@/components/UI/Badge'
import Modal from '@/components/UI/Modal'
import DataTable from '@/components/Data/DataTable'
import { cn } from '@/utils/cn'
import { getNotifications, markAllNotificationsRead, deleteNotification, Notification } from '@/services/adminApi'
import { showAlert } from '@/utils/notification'

const Notifications = () => {
  const [searchQuery, setSearchQuery] = useState('')
  const [filter, setFilter] = useState<'all' | 'unread' | 'read'>('all')
  const [notifications, setNotifications] = useState<Notification[]>([])
  const [loading, setLoading] = useState(true)
  const [currentPage, setCurrentPage] = useState(1)
  const [pageSize] = useState(20)
  const [total, setTotal] = useState(0)
  const [selectedNotification, setSelectedNotification] = useState<Notification | null>(null)
  const [showDetailModal, setShowDetailModal] = useState(false)
  const [deleteConfirm, setDeleteConfirm] = useState<{ open: boolean; id: number | null }>({ open: false, id: null })

  useEffect(() => {
    fetchNotifications()
  }, [currentPage, filter, searchQuery])

  const fetchNotifications = async () => {
    try {
      setLoading(true)
      const response = await getNotifications({
        page: currentPage,
        pageSize,
        search: searchQuery || undefined,
        filter: filter !== 'all' ? filter : undefined,
      })
      setNotifications(response.list || [])
      setTotal(response.total || 0)
    } catch (error) {
      console.error('获取通知列表失败:', error)
      setNotifications([])
      setTotal(0)
    } finally {
      setLoading(false)
    }
  }

  const handleMarkAllRead = async () => {
    try {
      await markAllNotificationsRead()
      showAlert('已全部标记为已读', 'success')
      fetchNotifications()
    } catch (error) {
      console.error('标记全部已读失败:', error)
      showAlert('操作失败', 'error')
    }
  }

  const handleDelete = async (id: number) => {
    setDeleteConfirm({ open: true, id })
  }

  const confirmDelete = async () => {
    if (!deleteConfirm.id) return
    try {
      await deleteNotification(deleteConfirm.id)
      showAlert('删除成功', 'success')
      fetchNotifications()
    } catch (error) {
      console.error('删除通知失败:', error)
      showAlert('删除失败', 'error')
    } finally {
      setDeleteConfirm({ open: false, id: null })
    }
  }

  const handleRowClick = (notification: Notification) => {
    setSelectedNotification(notification)
    setShowDetailModal(true)
  }

  const getTypeIcon = (type?: string) => {
    switch (type) {
      case 'success':
        return <CheckCircle2 className="w-4 h-4" />
      case 'error':
        return <XCircle className="w-4 h-4" />
      case 'warning':
        return <AlertCircle className="w-4 h-4" />
      default:
        return <Info className="w-4 h-4" />
    }
  }

  const getTypeVariant = (type?: string): 'success' | 'error' | 'warning' | 'primary' => {
    switch (type) {
      case 'success':
        return 'success'
      case 'error':
        return 'error'
      case 'warning':
        return 'warning'
      default:
        return 'primary'
    }
  }

  const columns = useMemo(() => [
    {
      key: 'type' as keyof Notification,
      title: '类型',
      width: '16%',
      render: (_value: any, record: Notification) => {
        const typeLabel = record.type === 'success' ? '成功' : 
                         record.type === 'error' ? '错误' : 
                         record.type === 'warning' ? '警告' : '信息'
        return (
          <Badge
            variant={getTypeVariant(record.type)}
            size="xs"
            shape="pill"
            icon={getTypeIcon(record.type)}
          >
            {typeLabel}
          </Badge>
        )
      },
    },
    {
      key: 'title' as keyof Notification,
      title: '标题',
      width: '28%',
      render: (value: any, record: Notification) => (
        <div className="flex items-center gap-2 text-xs" >
          <span className={cn(
            "font-medium",
            !record.read 
              ? "text-slate-900 dark:text-white" 
              : "text-slate-600 dark:text-slate-400"
          )}>
            {value}
          </span>
          {!record.read && (
            <Badge variant="primary" size="xs" shape="pill">
              未读
            </Badge>
          )}
        </div>
      ),
    },
    {
      key: 'content' as keyof Notification,
      title: '内容',
      width: '35%',
      render: (value: any) => (
        <p className="text-sm text-slate-600 dark:text-slate-400 truncate max-w-md">
          {value}
        </p>
      ),
    },
    {
      key: 'created_at' as keyof Notification,
      title: '时间',
      width: '10%',
      render: (value: any) => {
        if (!value) return <span className="text-sm text-slate-500 dark:text-slate-500">-</span>
        try {
          const date = new Date(value)
          if (isNaN(date.getTime())) {
            return <span className="text-sm text-slate-500 dark:text-slate-500">-</span>
          }
          return (
            <span className="text-xs text-slate-500 dark:text-slate-500">
              {date.toLocaleString('zh-CN', {
                year: 'numeric',
                month: '2-digit',
                day: '2-digit',
                hour: '2-digit',
                minute: '2-digit'
              })}
            </span>
          )
        } catch (e) {
          return <span className="text-sm text-slate-500 dark:text-slate-500">-</span>
        }
      },
    },
    {
      key: 'actions' as keyof Notification,
      title: '操作',
      width: '5%',
      render: (_: any, record: Notification) => (
        <div className="flex items-center gap-2">
          <Button
            variant="ghost"
            size="sm"
            leftIcon={<Eye className="w-4 h-4" />}
            onClick={(e) => {
              e.stopPropagation()
              handleRowClick(record)
            }}
          >
            详情
          </Button>
          <Button
            variant="ghost"
            size="sm"
            leftIcon={<Trash2 className="w-4 h-4" />}
            className="text-red-600 hover:text-red-700 dark:text-red-400 hover:bg-red-50 dark:hover:bg-red-950/20"
            onClick={(e) => {
              e.stopPropagation()
              handleDelete(record.id)
            }}
          >
            删除
          </Button>
        </div>
      ),
    },
  ], [])

  const unreadCount = notifications?.filter(n => !n.read).length || 0

  return (
    <AdminLayout
      title="消息中心"
      description={`您有 ${unreadCount} 条未读消息`}
      actions={
        <div className="flex gap-2">
          <Button 
            variant="outline" 
            leftIcon={<CheckCheck className="w-4 h-4" />}
            onClick={handleMarkAllRead}
          >
            <span>全部已读</span>
          </Button>
        </div>
      }
    >
      <div className="space-y-6">
        {/* 搜索和筛选 */}
        <Card className="p-4">
          <div className="flex flex-col sm:flex-row gap-4">
            <div className="flex-1">
              <Input
                placeholder="搜索消息..."
                value={searchQuery}
                onChange={(e) => {
                  setSearchQuery(e.target.value)
                  setCurrentPage(1)
                }}
                leftIcon={<Search className="w-4 h-4" />}
                className="w-full"
              />
            </div>
            <div className="flex gap-2">
              <Button
                variant={filter === 'all' ? 'primary' : 'outline'}
                size="sm"
                onClick={() => {
                  setFilter('all')
                  setCurrentPage(1)
                }}
              >
                <span>全部</span>
              </Button>
              <Button
                variant={filter === 'unread' ? 'primary' : 'outline'}
                size="sm"
                onClick={() => {
                  setFilter('unread')
                  setCurrentPage(1)
                }}
              >
                <span>未读</span>
              </Button>
              <Button
                variant={filter === 'read' ? 'primary' : 'outline'}
                size="sm"
                onClick={() => {
                  setFilter('read')
                  setCurrentPage(1)
                }}
              >
                <span>已读</span>
              </Button>
            </div>
          </div>
        </Card>

        {/* 通知列表表格 */}
        <Card className="p-6">
          <DataTable
            data={notifications}
            columns={columns}
            loading={loading}
            searchable={false}
            onRowClick={handleRowClick}
            emptyText="暂无通知"
            pageSize={pageSize}
            showPagination={true}
          />
          {/* 自定义分页 */}
          {total > pageSize && (
            <div className="flex items-center justify-between mt-4 pt-4 border-t border-slate-200 dark:border-slate-700">
              <div className="text-sm text-slate-500 dark:text-slate-400">
                显示 {Math.min((currentPage - 1) * pageSize + 1, total)} 到{' '}
                {Math.min(currentPage * pageSize, total)} 条，共 {total} 条
              </div>
              <div className="flex items-center gap-2">
                <Button
                  variant="outline"
                  size="sm"
                  onClick={() => setCurrentPage(prev => Math.max(prev - 1, 1))}
                  disabled={currentPage === 1}
                >
                  <span>上一页</span>
                </Button>
                <div className="flex items-center gap-1">
                  {Array.from({ length: Math.min(5, Math.ceil(total / pageSize)) }, (_, i) => {
                    const page = i + 1
                    const totalPages = Math.ceil(total / pageSize)
                    if (totalPages <= 5) {
                      return (
                        <Button
                          key={page}
                          variant={currentPage === page ? 'primary' : 'outline'}
                          size="sm"
                          onClick={() => setCurrentPage(page)}
                          className="w-8 h-8 p-0"
                        >
                          <span>{page}</span>
                        </Button>
                      )
                    }
                    // 显示当前页附近的页码
                    const startPage = Math.max(1, currentPage - 2)
                    const endPage = Math.min(totalPages, currentPage + 2)
                    if (page >= startPage && page <= endPage) {
                      return (
                        <Button
                          key={page}
                          variant={currentPage === page ? 'primary' : 'outline'}
                          size="sm"
                          onClick={() => setCurrentPage(page)}
                          className="w-8 h-8 p-0"
                        >
                          <span>{page}</span>
                        </Button>
                      )
                    }
                    return null
                  })}
                </div>
                <Button
                  variant="outline"
                  size="sm"
                  onClick={() => setCurrentPage(prev => Math.min(prev + 1, Math.ceil(total / pageSize)))}
                  disabled={currentPage >= Math.ceil(total / pageSize)}
                >
                  <span>下一页</span>
                </Button>
              </div>
            </div>
          )}
        </Card>
      </div>

      {/* 详情模态框 */}
      <Modal
        isOpen={showDetailModal}
        onClose={() => {
          setShowDetailModal(false)
          setSelectedNotification(null)
        }}
        title="通知详情"
        size="lg"
      >
        {selectedNotification && (
          <div className="space-y-4">
            <div className="flex items-start gap-4">
              <div className="flex-shrink-0">
                <Badge
                  variant={getTypeVariant(selectedNotification.type)}
                  size="md"
                  shape="pill"
                  icon={getTypeIcon(selectedNotification.type)}
                >
                  {selectedNotification.type === 'success' ? '成功' : 
                   selectedNotification.type === 'error' ? '错误' : 
                   selectedNotification.type === 'warning' ? '警告' : '信息'}
                </Badge>
              </div>
              <div className="flex-1">
                <div className="flex items-center gap-2 mb-2">
                  <h3 className="text-lg font-semibold text-slate-900 dark:text-white">
                    {selectedNotification.title}
                  </h3>
                  {!selectedNotification.read && (
                    <Badge variant="primary" size="xs" shape="pill">
                      未读
                    </Badge>
                  )}
                </div>
                <div className="text-sm text-slate-600 dark:text-slate-400 mb-4">
                  {(() => {
                    if (!selectedNotification.created_at) return '-'
                    try {
                      const date = new Date(selectedNotification.created_at)
                      if (isNaN(date.getTime())) return '-'
                      return date.toLocaleString('zh-CN', {
                        year: 'numeric',
                        month: 'long',
                        day: 'numeric',
                        hour: '2-digit',
                        minute: '2-digit'
                      })
                    } catch (e) {
                      return '-'
                    }
                  })()}
                </div>
                <div className="prose dark:prose-invert max-w-none">
                  <p className="text-slate-700 dark:text-slate-300 leading-relaxed whitespace-pre-wrap">
                    {selectedNotification.content}
                  </p>
                </div>
              </div>
            </div>
          </div>
        )}
      </Modal>

      {/* 删除确认对话框 */}
      <ConfirmDialog
        isOpen={deleteConfirm.open}
        onClose={() => setDeleteConfirm({ open: false, id: null })}
        onConfirm={confirmDelete}
        title="确认删除"
        message="确定要删除这条通知吗？此操作不可恢复。"
        variant="danger"
        confirmText="删除"
      />
    </AdminLayout>
  )
}

export default Notifications
