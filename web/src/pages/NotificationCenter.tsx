import { useState, useEffect } from 'react'
import { useNotificationStore } from '@/stores/notificationStore'
import { useAuthStore } from '@/stores/authStore'
import { useI18nStore } from '@/stores/i18nStore'
import { Input as ArcoInput, Pagination, Checkbox, Tag } from '@arco-design/web-react'
import Button from '@/components/UI/Button'
import Modal from '@/components/UI/Modal'
import { showAlert } from '@/utils/notification'
import { formatDistanceToNow } from 'date-fns'
import { zhCN } from 'date-fns/locale'
import { highlightContent } from '@/utils/highlight'
import { useSearchHighlight } from '@/hooks/useSearchHighlight'
import { Bell, Check, Trash2, AlertCircle, Info, CheckCircle, XCircle, Clock } from 'lucide-react'

const NotificationCenter = () => {
  const { t } = useI18nStore()
  const [filter, setFilter] = useState<'all' | 'unread' | 'read'>('all')
  const [searchQuery, setSearchQuery] = useState('')
  const [currentPage, setCurrentPage] = useState(1)
  const [pageSize] = useState(10)
  const [selectedIds, setSelectedIds] = useState<number[]>([])
  const [isSelectMode, setIsSelectMode] = useState(false)
  const [selectedNotification, setSelectedNotification] = useState<any | null>(null)

  const { isAuthenticated } = useAuthStore()
  const { searchKeyword, highlightFragments, highlightResultId: _highlightResultId } = useSearchHighlight()

  const {
    notifications, isLoading, total, totalUnread, totalRead, totalPages,
    fetchNotifications, markAllAsRead, markAsRead, deleteNotification,
    batchDeleteNotifications, getAllNotificationIds
  } = useNotificationStore()

  const loadNotifications = () => {
    fetchNotifications({
      page: currentPage, size: pageSize,
      filter: filter === 'all' ? undefined : filter,
      title: searchQuery || undefined,
    })
  }

  useEffect(() => { if (isAuthenticated) setCurrentPage(1) }, [filter, searchQuery, isAuthenticated])
  useEffect(() => { if (isAuthenticated) loadNotifications() }, [currentPage, isAuthenticated, filter, searchQuery])

  const refresh = () => { loadNotifications(); setSelectedIds([]); setIsSelectMode(false) }

  const handleMarkAllAsRead = async () => { await markAllAsRead(); showAlert(t('notification.messages.markAllReadSuccess'), 'success'); refresh() }
  const handleMarkAsRead = async (id: string) => { await markAsRead(id); refresh() }
  const handleDelete = async (id: string) => { await deleteNotification(id); showAlert(t('notification.messages.deleteSuccess'), 'success'); refresh() }

  const handleSelectAll = async () => {
    if (selectedIds.length > 0) { setSelectedIds([]) }
    else {
      const allIds = await getAllNotificationIds({ filter: filter === 'all' ? undefined : filter, title: searchQuery || undefined })
      setSelectedIds(allIds)
    }
  }

  const toggleSelect = (id: number) => setSelectedIds(prev => prev.includes(id) ? prev.filter(x => x !== id) : [...prev, id])

  const handleBatchDelete = async () => {
    if (!selectedIds.length) return
    await batchDeleteNotifications(selectedIds)
    showAlert(t('notification.messages.batchDeleteSuccess').replace('{count}', String(selectedIds.length)), 'success')
    setSelectedIds([]); setIsSelectMode(false); refresh()
  }

  const handleBatchMarkAsRead = async () => {
    if (!selectedIds.length) return
    for (const id of selectedIds) await markAsRead(id.toString())
    showAlert(t('notification.messages.batchMarkReadSuccess').replace('{count}', String(selectedIds.length)), 'success')
    setSelectedIds([]); setIsSelectMode(false); refresh()
  }

  const getIcon = (type?: string) => {
    const cls = 'w-4 h-4'
    switch (type) {
      case 'success': return <CheckCircle className={`${cls} text-green-500`} />
      case 'warning': return <AlertCircle className={`${cls} text-yellow-500`} />
      case 'error': return <XCircle className={`${cls} text-red-500`} />
      default: return <Info className={`${cls} text-blue-500`} />
    }
  }

  if (!isAuthenticated) return (
    <div className="flex items-center justify-center h-[60vh]">
      <div className="text-center">
        <Bell className="w-12 h-12 text-gray-300 mx-auto mb-3" />
        <p className="text-gray-500">{t('notification.pleaseLogin')}</p>
      </div>
    </div>
  )

  return (
    <div className="w-full space-y-3">
      {/* Stats + Filter */}
      <div className="flex flex-wrap items-center gap-3 justify-between">
        <div className="flex items-center gap-2 flex-wrap">
          <Tag color="blue" className="text-xs">{t('notification.total')} {total}</Tag>
          <Tag color="orange" className="text-xs">{t('notification.unread')} {totalUnread}</Tag>
          <Tag color="green" className="text-xs">{t('notification.read')} {totalRead}</Tag>
          <div className="flex gap-0.5 p-0.5 bg-gray-100 dark:bg-gray-800 rounded-lg ml-2">
            {(['all', 'unread', 'read'] as const).map(f => (
              <button key={f} onClick={() => setFilter(f)}
                className={`px-2.5 py-1 rounded-md text-xs font-medium transition-all ${
                  filter === f ? 'bg-white dark:bg-gray-700 shadow-sm' : 'text-gray-500 hover:text-gray-700'
                }`}>
                {t(`notification.${f === 'all' ? 'all' : f}`)}
              </button>
            ))}
          </div>
          {isSelectMode && (
            <Checkbox checked={selectedIds.length > 0} onChange={handleSelectAll}>
              <span className="text-xs">{selectedIds.length === 0 ? t('notification.selectAll') : `${t('notification.selected')} ${selectedIds.length}`}</span>
            </Checkbox>
          )}
        </div>
        <ArcoInput.Search placeholder={t('notification.searchPlaceholder')} value={searchQuery} onChange={setSearchQuery} className="w-48" />
      </div>

      {/* Actions */}
      <div className="flex items-center gap-1.5">
        <Button variant="outline" size="sm" onClick={refresh} loading={isLoading}>{t('notification.refresh')}</Button>
        {!isSelectMode ? (
          <>
            <Button variant="outline" size="sm" onClick={() => setIsSelectMode(true)}>{t('notification.select')}</Button>
            {totalUnread > 0 && <Button variant="primary" size="sm" onClick={handleMarkAllAsRead}>{t('notification.markAllRead')}</Button>}
          </>
        ) : (
          <>
            <Button variant="outline" size="sm" onClick={() => { setIsSelectMode(false); setSelectedIds([]) }}>{t('notification.cancel')}</Button>
            {selectedIds.length > 0 && (
              <>
                <Button variant="outline" size="sm" onClick={handleBatchMarkAsRead}>{t('notification.markRead')} ({selectedIds.length})</Button>
                <Button variant="destructive" size="sm" onClick={handleBatchDelete}>{t('notification.delete')} ({selectedIds.length})</Button>
              </>
            )}
          </>
        )}
      </div>

      {/* Notification List */}
      <div className="border border-gray-200 dark:border-gray-700 rounded-lg overflow-hidden">
        {notifications.length === 0 ? (
          <div className="text-center py-16 text-gray-400">
            <Bell className="w-10 h-10 mx-auto mb-2 opacity-40" />
            <p className="text-sm">{t(`notification.empty.${filter}`)}</p>
          </div>
        ) : (
          <div className="divide-y divide-gray-100 dark:divide-gray-800">
            {notifications.map(n => (
              <div key={n.id} onClick={() => !isSelectMode && setSelectedNotification(n)}
                className={`flex items-center gap-3 px-4 py-2.5 cursor-pointer transition-colors ${
                  !n.read ? 'bg-blue-50/50 dark:bg-blue-900/10 hover:bg-blue-50' : 'hover:bg-gray-50 dark:hover:bg-gray-800/50'
                } ${selectedIds.includes(n.id) ? 'bg-blue-100 dark:bg-blue-900/20' : ''}`}>
                {isSelectMode && (
                  <Checkbox checked={selectedIds.includes(n.id)} onClick={e => e.stopPropagation()}
                    onChange={() => toggleSelect(n.id)} />
                )}
                <div className="shrink-0">{getIcon(n.type)}</div>
                <div className="flex-1 min-w-0">
                  <div className="flex items-center gap-2">
                    <span className={`text-sm truncate ${!n.read ? 'font-medium' : 'text-gray-500'}`}
                      dangerouslySetInnerHTML={{ __html: highlightContent(n.title, searchKeyword, highlightFragments || undefined) }} />
                    {!n.read && <span className="shrink-0 w-1.5 h-1.5 rounded-full bg-blue-500" />}
                  </div>
                </div>
                <div className="flex items-center gap-2 shrink-0">
                  <span className="text-xs text-gray-400 whitespace-nowrap flex items-center gap-1">
                    <Clock className="w-3 h-3" />
                    {n.created_at ? formatDistanceToNow(new Date(n.created_at), { addSuffix: true, locale: zhCN }) : ''}
                  </span>
                  {!isSelectMode && (
                    <div className="flex gap-0.5 opacity-0 group-hover:opacity-100 transition-opacity">
                      {!n.read && (
                        <button onClick={e => { e.stopPropagation(); handleMarkAsRead(n.id.toString()) }}
                          className="p-1 hover:bg-gray-100 rounded text-gray-400 hover:text-green-500">
                          <Check className="w-3.5 h-3.5" />
                        </button>
                      )}
                      <button onClick={e => { e.stopPropagation(); handleDelete(n.id.toString()) }}
                        className="p-1 hover:bg-gray-100 rounded text-gray-400 hover:text-red-500">
                        <Trash2 className="w-3.5 h-3.5" />
                      </button>
                    </div>
                  )}
                </div>
              </div>
            ))}
          </div>
        )}
      </div>

      {/* Pagination */}
      {totalPages > 1 && (
        <div className="flex justify-center">
          <Pagination current={currentPage} total={total} pageSize={pageSize} onChange={setCurrentPage} size="small" />
        </div>
      )}

      <Modal
        isOpen={!!selectedNotification}
        onClose={() => setSelectedNotification(null)}
        title={selectedNotification?.title || t('notification.title')}
        size="sm"
      >
        {selectedNotification && (
          <div className="space-y-4">
            <div className="flex items-center gap-2 text-sm text-gray-500">
              <Clock className="w-4 h-4" />
              {selectedNotification.created_at ? formatDistanceToNow(new Date(selectedNotification.created_at), { addSuffix: true, locale: zhCN }) : ''}
            </div>
            <div className="p-4 bg-gray-50 dark:bg-gray-800/50 rounded-lg">
              <div className="text-xs text-gray-400 mb-1">{t('notification.content')}</div>
              <div className="text-sm whitespace-pre-wrap break-words leading-relaxed"
                dangerouslySetInnerHTML={{ __html: highlightContent(selectedNotification.content || t('notification.noContent'), searchKeyword, highlightFragments || undefined) }} />
            </div>
            <div className="flex justify-end gap-2 pt-1">
              {!selectedNotification.read && (
                <Button
                  size="sm"
                  variant="primary"
                  onClick={async () => {
                    await handleMarkAsRead(selectedNotification.id.toString())
                    setSelectedNotification(null)
                  }}
                >
                  {t('notification.markAsRead')}
                </Button>
              )}
              <Button variant="outline" size="sm" onClick={() => setSelectedNotification(null)}>
                {t('notification.close')}
              </Button>
            </div>
          </div>
        )}
      </Modal>
    </div>
  )
}

export default NotificationCenter
