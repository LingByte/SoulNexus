import { useState, useEffect } from 'react'
import { motion } from 'framer-motion'
import { 
  Mail, 
  MailOpen, 
  Star, 
  Trash2, 
  Search,
  Filter,
  CheckSquare,
  Square
} from 'lucide-react'
import { useNavigate } from 'react-router-dom'
import Button from '@/components/UI/Button'
import Input from '@/components/UI/Input'
import { showAlert } from '@/utils/notification'
import { 
  getVoicemails, 
  getVoicemailStats,
  batchDeleteVoicemails,
  batchMarkAsRead
} from '@/api/voicemail'
import type { Voicemail, VoicemailStats } from '@/types/voicemail'
import VoicemailList from '@/components/Voicemail/VoicemailList'

const VoicemailBox = () => {
  const navigate = useNavigate()
  const [voicemails, setVoicemails] = useState<Voicemail[]>([])
  const [stats, setStats] = useState<VoicemailStats | null>(null)
  const [loading, setLoading] = useState(false)
  const [searchTerm, setSearchTerm] = useState('')
  const [filterStatus, setFilterStatus] = useState<'all' | 'unread' | 'important'>('all')
  const [selectedIds, setSelectedIds] = useState<number[]>([])

  useEffect(() => {
    loadVoicemails()
    loadStats()
  }, [filterStatus])

  const loadVoicemails = async () => {
    try {
      setLoading(true)
      const params: any = {}
      
      if (filterStatus === 'unread') {
        params.isRead = false
      } else if (filterStatus === 'important') {
        params.isImportant = true
      }

      const res = await getVoicemails(params)
      if (res.code === 200 && res.data) {
        // 后端返回的是 { list, total, page, size } 结构
        const data = res.data as any
        if (Array.isArray(data)) {
          setVoicemails(data)
        } else if (data.list && Array.isArray(data.list)) {
          setVoicemails(data.list)
        } else {
          setVoicemails([])
        }
      }
    } catch (error: any) {
      showAlert(error.msg || '加载留言失败', 'error')
      setVoicemails([]) // 确保出错时也设置为空数组
    } finally {
      setLoading(false)
    }
  }

  const loadStats = async () => {
    try {
      const res = await getVoicemailStats()
      if (res.code === 200 && res.data) {
        setStats(res.data)
      }
    } catch (error: any) {
      console.error('加载统计失败:', error)
    }
  }

  const handleBatchDelete = async () => {
    if (selectedIds.length === 0) {
      showAlert('请选择要删除的留言', 'warning')
      return
    }

    if (!confirm(`确定要删除 ${selectedIds.length} 条留言吗？`)) {
      return
    }

    try {
      const res = await batchDeleteVoicemails(selectedIds)
      if (res.code === 200) {
        showAlert('删除成功', 'success')
        setSelectedIds([])
        loadVoicemails()
        loadStats()
      } else {
        throw new Error(res.msg)
      }
    } catch (error: any) {
      showAlert(error.msg || '删除失败', 'error')
    }
  }

  const handleBatchMarkRead = async () => {
    if (selectedIds.length === 0) {
      showAlert('请选择要标记的留言', 'warning')
      return
    }

    try {
      const res = await batchMarkAsRead(selectedIds)
      if (res.code === 200) {
        showAlert('已标记为已读', 'success')
        setSelectedIds([])
        loadVoicemails()
        loadStats()
      } else {
        throw new Error(res.msg)
      }
    } catch (error: any) {
      showAlert(error.msg || '操作失败', 'error')
    }
  }

  const toggleSelectAll = () => {
    if (selectedIds.length === filteredVoicemails.length) {
      setSelectedIds([])
    } else {
      setSelectedIds(filteredVoicemails.map(v => v.id))
    }
  }

  const toggleSelect = (id: number) => {
    setSelectedIds(prev => 
      prev.includes(id) 
        ? prev.filter(i => i !== id)
        : [...prev, id]
    )
  }

  const filteredVoicemails = voicemails.filter(v => {
    if (searchTerm) {
      const search = searchTerm.toLowerCase()
      return (
        v.callerNumber.includes(search) ||
        v.callerName?.toLowerCase().includes(search) ||
        v.transcribedText?.toLowerCase().includes(search) ||
        v.summary?.toLowerCase().includes(search)
      )
    }
    return true
  })

  return (
    <div className="p-6 max-w-7xl mx-auto">
      {/* Header */}
      <div className="mb-6">
        <h1 className="text-3xl font-bold text-foreground mb-2">
          留言箱
        </h1>
        <p className="text-muted-foreground">
          查看和管理您的通话留言
        </p>
      </div>

      {/* Stats Cards */}
      {stats && (
        <div className="grid grid-cols-1 md:grid-cols-4 gap-4 mb-6">
          <motion.div
            initial={{ opacity: 0, y: 20 }}
            animate={{ opacity: 1, y: 0 }}
            className="p-4 bg-card border border-border rounded-lg"
          >
            <div className="flex items-center justify-between">
              <div>
                <div className="text-2xl font-bold text-foreground">{stats.total}</div>
                <div className="text-sm text-muted-foreground">总留言</div>
              </div>
              <Mail className="w-8 h-8 text-muted-foreground" />
            </div>
          </motion.div>

          <motion.div
            initial={{ opacity: 0, y: 20 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ delay: 0.1 }}
            className="p-4 bg-card border border-border rounded-lg"
          >
            <div className="flex items-center justify-between">
              <div>
                <div className="text-2xl font-bold text-foreground">{stats.unread}</div>
                <div className="text-sm text-muted-foreground">未读</div>
              </div>
              <MailOpen className="w-8 h-8 text-blue-500" />
            </div>
          </motion.div>

          <motion.div
            initial={{ opacity: 0, y: 20 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ delay: 0.2 }}
            className="p-4 bg-card border border-border rounded-lg"
          >
            <div className="flex items-center justify-between">
              <div>
                <div className="text-2xl font-bold text-foreground">{stats.important}</div>
                <div className="text-sm text-muted-foreground">重要</div>
              </div>
              <Star className="w-8 h-8 text-yellow-500" />
            </div>
          </motion.div>

          <motion.div
            initial={{ opacity: 0, y: 20 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ delay: 0.3 }}
            className="p-4 bg-card border border-border rounded-lg"
          >
            <div className="flex items-center justify-between">
              <div>
                <div className="text-2xl font-bold text-foreground">{stats.today}</div>
                <div className="text-sm text-muted-foreground">今日</div>
              </div>
              <Mail className="w-8 h-8 text-green-500" />
            </div>
          </motion.div>
        </div>
      )}

      {/* Toolbar */}
      <div className="mb-6 space-y-4">
        {/* Search and Filter */}
        <div className="flex flex-col sm:flex-row gap-4">
          <div className="flex-1">
            <Input
              placeholder="搜索号码、姓名或内容..."
              value={searchTerm}
              onChange={(e) => setSearchTerm(e.target.value)}
              leftIcon={<Search className="w-4 h-4" />}
              clearable
              onClear={() => setSearchTerm('')}
            />
          </div>
          <div className="flex gap-2">
            <Button
              variant={filterStatus === 'all' ? 'primary' : 'outline'}
              onClick={() => setFilterStatus('all')}
              size="md"
            >
              全部
            </Button>
            <Button
              variant={filterStatus === 'unread' ? 'primary' : 'outline'}
              onClick={() => setFilterStatus('unread')}
              size="md"
            >
              未读
            </Button>
            <Button
              variant={filterStatus === 'important' ? 'primary' : 'outline'}
              onClick={() => setFilterStatus('important')}
              size="md"
            >
              重要
            </Button>
          </div>
        </div>

        {/* Batch Actions */}
        {selectedIds.length > 0 && (
          <motion.div
            initial={{ opacity: 0, y: -10 }}
            animate={{ opacity: 1, y: 0 }}
            className="flex items-center gap-3 p-3 bg-primary/10 border border-primary/20 rounded-lg"
          >
            <span className="text-sm text-foreground">
              已选择 {selectedIds.length} 项
            </span>
            <div className="flex-1" />
            <Button
              size="sm"
              variant="outline"
              onClick={handleBatchMarkRead}
              leftIcon={<MailOpen className="w-3.5 h-3.5" />}
            >
              标记已读
            </Button>
            <Button
              size="sm"
              variant="destructive"
              onClick={handleBatchDelete}
              leftIcon={<Trash2 className="w-3.5 h-3.5" />}
            >
              删除
            </Button>
          </motion.div>
        )}
      </div>

      {/* Voicemail List */}
      {loading ? (
        <div className="text-center py-12">
          <div className="inline-block w-8 h-8 border-4 border-primary border-t-transparent rounded-full animate-spin" />
          <p className="mt-4 text-muted-foreground">加载中...</p>
        </div>
      ) : filteredVoicemails.length === 0 ? (
        <motion.div
          initial={{ opacity: 0, scale: 0.95 }}
          animate={{ opacity: 1, scale: 1 }}
          className="text-center py-12 bg-card border border-border rounded-lg"
        >
          <Mail className="w-16 h-16 mx-auto text-muted-foreground mb-4" />
          <h3 className="text-lg font-medium text-foreground mb-2">
            {searchTerm ? '没有找到匹配的留言' : '还没有留言'}
          </h3>
          <p className="text-muted-foreground">
            {searchTerm ? '尝试使用其他关键词搜索' : '当有人给您留言时，会显示在这里'}
          </p>
        </motion.div>
      ) : (
        <VoicemailList
          voicemails={filteredVoicemails}
          selectedIds={selectedIds}
          onToggleSelect={toggleSelect}
          onToggleSelectAll={toggleSelectAll}
          onRefresh={() => {
            loadVoicemails()
            loadStats()
          }}
          onViewDetail={(id) => navigate(`/voicemail/${id}`)}
        />
      )}
    </div>
  )
}

export default VoicemailBox
