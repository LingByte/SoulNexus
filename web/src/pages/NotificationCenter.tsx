import { useState, useEffect } from 'react'
import { motion } from 'framer-motion'
import {
    Bell,
    Check,
    Trash2,
    MoreVertical,
    AlertCircle,
    Info,
    CheckCircle,
    XCircle,
    Clock,
    Eye,
    EyeOff
} from 'lucide-react'
import { useNotificationStore } from '@/stores/notificationStore'
import { useAuthStore } from '@/stores/authStore'
import { useI18nStore } from '@/stores/i18nStore'
import Button from '@/components/UI/Button'
import Card, { CardContent } from '@/components/UI/Card'
import Badge from '@/components/UI/Badge'
import { showAlert } from '@/utils/notification'
import { formatDistanceToNow } from 'date-fns'
import { zhCN } from 'date-fns/locale'
import { highlightContent } from '@/utils/highlight'
import { useSearchHighlight } from '@/hooks/useSearchHighlight'

const NotificationCenter = () => {
  const { t } = useI18nStore()
  const [filter, setFilter] = useState<'all' | 'unread' | 'read'>('all')
  const [showActions, setShowActions] = useState<string | null>(null)
  const [showDrawer, setShowDrawer] = useState(false)
  const [selectedNotification, setSelectedNotification] = useState<any | null>(null)
  const [drawerEntering, setDrawerEntering] = useState(false)
  const [searchQuery, setSearchQuery] = useState('')
  const [currentPage, setCurrentPage] = useState(1)
  const [pageSize] = useState(10)
  const [sortBy] = useState<'newest' | 'oldest'>('newest')
  const [selectedIds, setSelectedIds] = useState<number[]>([])
  const [isSelectMode, setIsSelectMode] = useState(false)
  const [startTime] = useState('')
  const [endTime] = useState('')

  const { isAuthenticated } = useAuthStore()
  const { searchKeyword, highlightFragments, highlightResultId } = useSearchHighlight()
  
  // 从 URL 参数中获取搜索关键词，并设置到搜索框
  useEffect(() => {
    if (searchKeyword) {
      setSearchQuery(searchKeyword)
    }
  }, [searchKeyword])
  const { 
    notifications, 
    isLoading, 
    total, 
    totalUnread, 
    totalRead,
    currentPage: _storeCurrentPage,
    totalPages,
    fetchNotifications,
    markAllAsRead,
    markAsRead,
    deleteNotification,
    batchDeleteNotifications,
    getAllNotificationIds
  } = useNotificationStore()

  // 加载通知数据
  const loadNotifications = () => {
    const params = {
      page: currentPage,
      size: pageSize,
      filter: filter === 'all' ? undefined : filter,
      title: searchQuery || undefined,
      sort: sortBy,
      start_time: startTime || undefined,
      end_time: endTime || undefined,
    }
    fetchNotifications(params)
  }

  // 修正副作用触发逻辑
  useEffect(() => {
    if (isAuthenticated) {
      setCurrentPage(1)
    }
  }, [filter, searchQuery, sortBy, startTime, endTime, isAuthenticated])

  useEffect(() => {
    if (isAuthenticated) {
      loadNotifications()
    }
    // eslint-disable-next-line
  }, [currentPage, isAuthenticated, filter, searchQuery, sortBy, startTime, endTime])

  // 刷新数据
  const refreshNotifications = () => {
    loadNotifications()
    setSelectedIds([])
    setIsSelectMode(false)
  }

  const handleMarkAllAsRead = async () => {
    await markAllAsRead()
    showAlert(t('notification.messages.markAllReadSuccess'), 'success')
    refreshNotifications()
  }

  const handleMarkAsRead = async (id: string) => {
    await markAsRead(id)
    setShowActions(null)
    refreshNotifications()
  }

  const handleDelete = async (id: string) => {
    await deleteNotification(id)
    setShowActions(null)
    showAlert(t('notification.messages.deleteSuccess'), 'success')
    refreshNotifications()
  }

  // 多选相关函数
  const handleSelectAll = async () => {
    if (selectedIds.length > 0) {
      // 如果已有选中项，则清空选择
      setSelectedIds([])
    } else {
      // 获取所有通知ID（根据当前筛选条件）
      const params = {
        filter: filter === 'all' ? undefined : filter,
        title: searchQuery || undefined,
        start_time: startTime || undefined,
        end_time: endTime || undefined,
      }
      const allIds = await getAllNotificationIds(params)
      setSelectedIds(allIds)
    }
  }

  const handleSelectNotification = (id: number) => {
    setSelectedIds(prev => 
      prev.includes(id) 
        ? prev.filter(selectedId => selectedId !== id)
        : [...prev, id]
    )
  }

  const handleBatchDelete = async () => {
    if (selectedIds.length === 0) return
    
    await batchDeleteNotifications(selectedIds)
    showAlert(t('notification.messages.batchDeleteSuccess').replace('{count}', String(selectedIds.length)), 'success')
    setSelectedIds([])
    setIsSelectMode(false)
    refreshNotifications()
  }

  const handleBatchMarkAsRead = async () => {
    if (selectedIds.length === 0) return
    
    for (const id of selectedIds) {
      await markAsRead(id.toString())
    }
    showAlert(t('notification.messages.batchMarkReadSuccess').replace('{count}', String(selectedIds.length)), 'success')
    setSelectedIds([])
    setIsSelectMode(false)
    refreshNotifications()
  }

  const openDrawer = async (notification: any) => {
    setSelectedNotification(notification)
    setShowDrawer(true)
    // 下一帧触发进入动画
    setTimeout(() => setDrawerEntering(true), 0)
    if (!notification.read) {
      await handleMarkAsRead(notification.id.toString())
    }
  }

  const closeDrawer = () => {
    setDrawerEntering(false)
    setTimeout(() => {
      setShowDrawer(false)
      setSelectedNotification(null)
    }, 200)
  }

  const getNotificationIcon = (type?: string, isRead?: boolean) => {
    const iconClass = `w-4 h-4 ${isRead ? 'text-muted-foreground' : 'text-foreground'}`

    switch (type) {
      case 'success':
        return <CheckCircle className={iconClass} />
      case 'warning':
        return <AlertCircle className={iconClass} />
      case 'error':
        return <XCircle className={iconClass} />
      default:
        return <Info className={iconClass} />
    }
  }

  if (!isAuthenticated) {
    return (
      <div className="min-h-screen flex items-center justify-center bg-background">
        <Card className="w-full max-w-md">
          <CardContent className="text-center py-12 px-8">
            <div className="w-16 h-16 bg-muted rounded-lg flex items-center justify-center mx-auto mb-6">
              <Bell className="w-8 h-8 text-muted-foreground" />
            </div>
            <h2 className="text-xl font-semibold mb-3">
              {t('notification.pleaseLogin')}
            </h2>
            <p className="text-muted-foreground">
              {t('notification.loginDesc')}
            </p>
          </CardContent>
        </Card>
      </div>
    )
  }

  return (
    <div className="h-[calc(100vh-4rem)] bg-background flex flex-col">
      {/* 主要内容区域 - 横向布局 */}
      <div className="flex-1 flex flex-col lg:flex-row overflow-hidden">
        {/* 左侧：统计和筛选 */}
        <div className="w-full lg:w-80 flex-shrink-0 border-r bg-card/30 flex flex-col">
          {/* 头部 */}
          <div className="p-3 border-b">
            <div className="flex items-center justify-between mb-2">
              <h1 className="text-base font-bold text-foreground relative pl-4">
                <motion.div
                  layoutId="pageTitleIndicator"
                  className="absolute left-0 top-1/2 -translate-y-1/2 w-1 h-5 bg-primary rounded-r-full"
                  transition={{ type: 'spring', bounce: 0.2, duration: 0.3 }}
                />
                {t('notification.title')}
              </h1>
            </div>
            {/* 操作按钮 - 横向排列 */}
            <div className="flex items-center space-x-1.5">
              <Button
                  variant="outline"
                  size="sm"
                  onClick={refreshNotifications}
                  disabled={isLoading}
                  className="h-7 px-2 text-xs"
              >
                {t('notification.refresh')}
              </Button>

              {!isSelectMode ? (
                  <>
                    <Button
                        variant="outline"
                        size="sm"
                        onClick={() => setIsSelectMode(true)}
                        className="h-7 px-2 text-xs"
                    >
                      {t('notification.select')}
                    </Button>
                    {totalUnread > 0 && (
                        <Button
                            variant="default"
                            size="sm"
                            onClick={handleMarkAllAsRead}
                            className="h-7 px-2 text-xs"
                        >
                          {t('notification.markAllRead')}
                        </Button>
                    )}
                  </>
              ) : (
                  <>
                    <Button
                        variant="outline"
                        size="sm"
                        onClick={() => {
                          setIsSelectMode(false)
                          setSelectedIds([])
                        }}
                        className="h-7 px-2 text-xs"
                    >
                      {t('notification.cancel')}
                    </Button>
                    {selectedIds.length > 0 && (
                        <>
                          <Button
                              variant="outline"
                              size="sm"
                              onClick={handleBatchMarkAsRead}
                              className="h-7 px-2 text-xs"
                          >
                            {t('notification.markRead')} ({selectedIds.length})
                          </Button>
                          <Button
                              variant="destructive"
                              size="sm"
                              onClick={handleBatchDelete}
                              className="h-7 px-2 text-xs"
                          >
                            {t('notification.delete')} ({selectedIds.length})
                          </Button>
                        </>
                    )}
                  </>
              )}
            </div>
          </div>
          
          {/* 统计卡片 */}
          <div className="p-3 border-b">
            <h3 className="text-xs font-semibold text-foreground mb-2">{t('notification.stats')}</h3>
            <div className="space-y-1.5">
              <div className="flex items-center justify-between p-2 bg-blue-50 dark:bg-blue-900/20 rounded-md">
                <div className="flex items-center space-x-1.5">
                  <Bell className="w-3.5 h-3.5 text-blue-600 dark:text-blue-400" />
                  <span className="text-xs text-blue-700 dark:text-blue-300">{t('notification.total')}</span>
                </div>
                <span className="text-sm font-bold text-blue-600 dark:text-blue-400">{total}</span>
              </div>
              
              <div className="flex items-center justify-between p-2 bg-orange-50 dark:bg-orange-900/20 rounded-md">
                <div className="flex items-center space-x-1.5">
                  <EyeOff className="w-3.5 h-3.5 text-orange-600 dark:text-orange-400" />
                  <span className="text-xs text-orange-700 dark:text-orange-300">{t('notification.unread')}</span>
                </div>
                <span className="text-sm font-bold text-orange-600 dark:text-orange-400">{totalUnread}</span>
              </div>
              
              <div className="flex items-center justify-between p-2 bg-green-50 dark:bg-green-900/20 rounded-md">
                <div className="flex items-center space-x-1.5">
                  <Eye className="w-3.5 h-3.5 text-green-600 dark:text-green-400" />
                  <span className="text-xs text-green-700 dark:text-green-300">{t('notification.read')}</span>
                </div>
                <span className="text-sm font-bold text-green-600 dark:text-green-400">{totalRead}</span>
              </div>
            </div>
          </div>

          {/* 搜索和筛选 */}
          <div className="p-3 border-b">
            <h3 className="text-xs font-semibold text-foreground mb-2">{t('notification.filter')}</h3>

            {/* 状态筛选 */}
            <div className="mb-3">
              <label className="text-xs text-muted-foreground mb-1 block">{t('notification.status')}</label>
              <div className="space-y-0.5">
                {[
                  { key: 'all', label: t('notification.all'), count: total },
                  { key: 'unread', label: t('notification.unread'), count: totalUnread },
                  { key: 'read', label: t('notification.read'), count: totalRead },
                ].map((item) => (
                  <button
                    key={item.key}
                    onClick={() => setFilter(item.key as any)}
                    className={`w-full flex items-center justify-between px-2 py-1.5 rounded-md text-xs transition-colors ${
                      filter === item.key
                        ? 'bg-primary text-primary-foreground'
                        : 'bg-secondary text-secondary-foreground hover:bg-secondary/80'
                    }`}
                  >
                    <span>{item.label}</span>
                    <span className="text-xs opacity-75">({item.count})</span>
                  </button>
                ))}
              </div>
            </div>
          </div>

          {/* 多选控制 */}
          {isSelectMode && notifications.length > 0 && (
            <div className="p-3 border-b">
              <h3 className="text-xs font-semibold text-foreground mb-2">{t('notification.batchActions')}</h3>
              <div className="space-y-1">
                <div className="flex items-center space-x-2">
                  <input
                    type="checkbox"
                    checked={selectedIds.length > 0}
                    onChange={handleSelectAll}
                    className="w-3.5 h-3.5 text-primary bg-background border-input rounded focus:ring-2 focus:ring-ring"
                  />
                  <span className="text-xs text-muted-foreground">
                    {selectedIds.length === 0 
                      ? t('notification.selectAll')
                      : `已选择 ${selectedIds.length} 项`
                    }
                  </span>
                </div>
              </div>
            </div>
          )}
        </div>

        {/* 右侧：通知列表 */}
        <div className="flex-1 flex flex-col min-w-0">
          {/* 通知列表标题 */}
          <div className="flex-shrink-0 px-4 py-3 border-b bg-card/30">
            <div className="flex items-center justify-between">
              <h3 className="text-sm font-semibold text-foreground">
                {filter === 'all' ? t('notification.allNotifications') :
                 filter === 'unread' ? t('notification.unreadNotifications') : t('notification.readNotifications')}
              </h3>
              <span className="text-xs text-muted-foreground">
                {t('notification.totalCount').replace('{count}', String(notifications?.length || 0))}
              </span>
            </div>
          </div>

          {/* 通知列表 - 可滚动区域 */}
          <div className="flex-1 overflow-y-auto">
            {notifications.length === 0 ? (
              <div className="flex items-center justify-center h-full">
                <div className="text-center">
                  <div className="w-16 h-16 bg-muted/50 rounded-full flex items-center justify-center mx-auto mb-4">
                    <Bell className="w-8 h-8 text-muted-foreground" />
                  </div>
                  <h3 className="text-lg font-medium mb-2">
                    {filter === 'all' ? t('notification.empty.all') :
                     filter === 'unread' ? t('notification.empty.unread') : t('notification.empty.read')}
                  </h3>
                  <p className="text-sm text-muted-foreground">
                    {filter === 'all' ? t('notification.emptyDesc.all') :
                     filter === 'unread' ? t('notification.emptyDesc.unread') : t('notification.emptyDesc.read')}
                  </p>
                </div>
              </div>
            ) : (
              <div className="p-1">
                {notifications.map((notification) => (
                  <div
                    key={notification.id}
                    className={`group mb-1 p-3 rounded-lg border transition-all duration-200 hover:shadow-sm ${
                      !notification.read 
                        ? 'bg-primary/5 border-primary/20' 
                        : 'bg-card/30 hover:bg-card/50 border-border'
                    } ${selectedIds.includes(notification.id) ? 'ring-2 ring-primary' : ''}`}
                  >
                    <div className="flex items-start space-x-3">
                      {isSelectMode && (
                        <input
                          type="checkbox"
                          checked={selectedIds.includes(notification.id)}
                          onChange={() => handleSelectNotification(notification.id)}
                          onClick={(e) => e.stopPropagation()}
                          className="w-4 h-4 text-primary bg-background border-input rounded focus:ring-2 focus:ring-ring mt-0.5"
                        />
                      )}
                      
                      {/* 通知图标 */}
                      <div className={`flex-shrink-0 w-6 h-6 rounded-md flex items-center justify-center ${
                        !notification.read 
                          ? 'bg-primary/15 text-primary' 
                          : 'bg-muted/60 text-muted-foreground'
                      }`}>
                        {getNotificationIcon(notification.type, notification.read)}
                      </div>
                      
                      {/* 通知内容 */}
                      <div 
                        className={`flex-1 min-w-0 ${!isSelectMode ? 'cursor-pointer' : ''}`}
                        onClick={() => !isSelectMode && openDrawer(notification)}
                      >
                        <div className="flex items-start justify-between">
                          <div className="flex-1 min-w-0">
                            <div className="flex items-center space-x-2 mb-1">
                              <h4 
                                className={`font-medium text-sm truncate ${
                                  !notification.read ? 'text-foreground' : 'text-muted-foreground'
                                } ${highlightResultId === `notification_${notification.id}` ? 'ring-2 ring-yellow-400 rounded px-1' : ''}`}
                                dangerouslySetInnerHTML={{
                    __html: highlightContent(
                      notification.title,
                      searchKeyword,
                      highlightFragments || undefined
                    )
                                }}
                              />
                              
                              {/* 状态标签 */}
                              <div className="flex items-center space-x-1 flex-shrink-0">
                                {!notification.read && (
                                  <Badge className="h-3 px-1 text-[9px] bg-primary text-primary-foreground">
                                    未读
                                  </Badge>
                                )}
                                {notification.type && (
                                  <Badge variant="outline" className="h-3 px-1 text-[9px] border-muted-foreground/40 text-muted-foreground">
                                    {notification.type}
                                  </Badge>
                                )}
                              </div>
                            </div>
                            
                            {/* 时间和操作按钮 */}
                            <div className="flex items-center justify-between">
                              <div className="flex items-center space-x-1 text-xs text-muted-foreground">
                                <Clock className="w-3 h-3" />
                                <span>
                                  {notification.created_at ? formatDistanceToNow(new Date(notification.created_at), {
                                    addSuffix: true,
                                    locale: zhCN
                                  }) : t('notification.unknownTime')}
                                </span>
                              </div>
                              
                              {/* 操作按钮 */}
                              <div className="relative" onClick={(e) => e.stopPropagation()}>
                                <button
                                  onClick={() => setShowActions(
                                    showActions === notification.id.toString() ? null : notification.id.toString()
                                  )}
                                  className="p-1 text-muted-foreground hover:text-foreground rounded hover:bg-accent/50 transition-all duration-200 opacity-0 group-hover:opacity-100"
                                >
                                  <MoreVertical className="w-3.5 h-3.5" />
                                </button>

                                {showActions === notification.id.toString() && (
                                  <div className="absolute right-0 top-full mt-1 w-28 bg-popover rounded-lg shadow-lg border z-10 overflow-hidden">
                                    <div className="py-1">
                                      {!notification.read && (
                                        <button
                                          onClick={() => handleMarkAsRead(notification.id.toString())}
                                          className="flex items-center w-full px-2 py-1.5 text-xs text-foreground hover:bg-accent transition-colors"
                                        >
                                          <Check className="w-3 h-3 mr-1.5" />
                                          {t('notification.markAsRead')}
                                        </button>
                                      )}
                                      <button
                                        onClick={() => handleDelete(notification.id.toString())}
                                        className="flex items-center w-full px-2 py-1.5 text-xs text-destructive hover:bg-destructive/10 transition-colors"
                                      >
                                        <Trash2 className="w-3 h-3 mr-1.5" />
                                        {t('notification.delete')}
                                      </button>
                                    </div>
                                  </div>
                                )}
                              </div>
                            </div>
                          </div>
                        </div>
                      </div>
                    </div>
                  </div>
                ))}
              </div>
            )}
          </div>

          {/* 分页 - 固定在底部 */}
          {totalPages > 1 && (
            <div className="flex-shrink-0 border-t bg-card/30 px-4 py-3">
              <div className="flex flex-col sm:flex-row items-center justify-between gap-3">
                <div className="text-sm text-muted-foreground">
                  {t('notification.totalCount').replace('{count}', String(total))}
                </div>

                <div className="flex items-center space-x-2">
                  <Button
                    variant="outline"
                    size="sm"
                    onClick={() => setCurrentPage(Math.max(1, currentPage - 1))}
                    disabled={currentPage === 1 || isLoading}
                    className="h-8 px-3"
                  >
                    {t('notification.previousPage')}
                  </Button>

                  <div className="flex items-center space-x-1">
                    {Array.from({ length: Math.min(5, totalPages) }, (_, i) => {
                      let pageNum;
                      if (totalPages <= 5) {
                        pageNum = i + 1;
                      } else if (currentPage <= 3) {
                        pageNum = i + 1;
                      } else if (currentPage >= totalPages - 2) {
                        pageNum = totalPages - 4 + i;
                      } else {
                        pageNum = currentPage - 2 + i;
                      }

                      return (
                        <button
                          key={pageNum}
                          onClick={() => setCurrentPage(pageNum)}
                          disabled={isLoading}
                          className={`w-7 h-7 rounded-md text-xs font-medium transition-colors ${
                            currentPage === pageNum
                              ? 'bg-primary text-primary-foreground'
                              : 'bg-secondary text-secondary-foreground hover:bg-secondary/80'
                          } ${isLoading ? 'opacity-50 cursor-not-allowed' : ''}`}
                        >
                          {pageNum}
                        </button>
                      );
                    })}
                  </div>

                  <Button
                    variant="outline"
                    size="sm"
                    onClick={() => setCurrentPage(Math.min(totalPages, currentPage + 1))}
                    disabled={currentPage === totalPages || isLoading}
                    className="h-8 px-3"
                  >
                    {t('notification.nextPage')}
                  </Button>
                </div>
              </div>
            </div>
          )}
        </div>
      </div>

      {/* 详情抽屉 */}
      {showDrawer && selectedNotification && (
        <div className="fixed inset-0 z-40">
          <div className={`absolute inset-0 bg-black/30 transition-opacity duration-200 ${drawerEntering ? 'opacity-100' : 'opacity-0'}`} onClick={closeDrawer} />
          <div className={`absolute right-0 top-0 h-full w-full sm:w-[380px] max-w-full bg-background border-l shadow-xl flex flex-col transform transition-transform duration-200 ease-out ${drawerEntering ? 'translate-x-0' : 'translate-x-full'}`}>
            <div className="px-4 py-3 border-b flex items-center justify-between">
              <div className="flex items-center space-x-2 min-w-0">
                <div className={`w-6 h-6 rounded-full flex items-center justify-center ${
                  !selectedNotification.read ? 'bg-primary/10 text-primary' : 'bg-muted text-muted-foreground'
                }`}>
                  {getNotificationIcon(selectedNotification.type, selectedNotification.read)}
                </div>
                <h2 
                  className="text-sm font-semibold truncate" 
                  title={selectedNotification.title}
                  dangerouslySetInnerHTML={{
                    __html: highlightContent(
                      selectedNotification.title,
                      searchKeyword,
                      highlightFragments || undefined
                    )
                  }}
                />
              </div>
              <button className="text-sm text-muted-foreground hover:text-foreground" onClick={closeDrawer}>{t('notification.close')}</button>
            </div>
            <div className="p-4 flex-1 overflow-auto">
              <div className="border rounded-md p-3">
                {/* 标题 */}
                <div className="mb-2">
                  <div className="text-[11px] text-muted-foreground mb-0.5">{t('notification.title')}</div>
                  <div 
                    className="text-sm font-semibold break-words leading-snug"
                    dangerouslySetInnerHTML={{
                      __html: highlightContent(
                        selectedNotification.title,
                        searchKeyword,
                        highlightFragments || undefined
                      )
                    }}
                  />
                </div>

                {/* 时间 */}
                <div className="mb-2">
                  <div className="text-[11px] text-muted-foreground mb-0.5">{t('notification.time')}</div>
                  <div className="text-[11px] text-muted-foreground flex items-center space-x-1 leading-none">
                    <Clock className="w-3 h-3" />
                    <span>
                      {selectedNotification.created_at ? formatDistanceToNow(new Date(selectedNotification.created_at), {
                        addSuffix: true,
                        locale: zhCN
                      }) : t('notification.unknownTime')}
                    </span>
                  </div>
                </div>

                {/* 内容 */}
                <div>
                  <div className="text-[11px] text-muted-foreground mb-0.5">{t('notification.content')}</div>
                  <div 
                    className="text-sm whitespace-pre-wrap break-words leading-relaxed"
                    dangerouslySetInnerHTML={{
                      __html: highlightContent(
                        selectedNotification.content || t('notification.noContent'),
                        searchKeyword,
                        highlightFragments || undefined
                      )
                    }}
                  />
                </div>
              </div>
            </div>
            <div className="p-3 border-t flex items-center justify-end space-x-2">
              {!selectedNotification.read && (
                <Button size="sm" onClick={async () => { await handleMarkAsRead(selectedNotification.id.toString()); closeDrawer() }}>{t('notification.markAsRead')}</Button>
              )}
              <Button variant="outline" size="sm" onClick={closeDrawer}>{t('notification.done')}</Button>
            </div>
          </div>
        </div>
      )}
    </div>
  )
}

export default NotificationCenter