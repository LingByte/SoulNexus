import { motion } from 'framer-motion'
import { 
  Mail, 
  MailOpen, 
  Star, 
  Phone,
  Clock,
  MapPin,
  CheckSquare,
  Square,
  Trash2
} from 'lucide-react'
import { clsx } from 'clsx'
import Button from '@/components/UI/Button'
import { showAlert } from '@/utils/notification'
import { 
  markVoicemailAsRead, 
  markVoicemailAsImportant,
  deleteVoicemail
} from '@/api/voicemail'
import type { Voicemail } from '@/types/voicemail'

interface VoicemailListProps {
  voicemails: Voicemail[]
  selectedIds: number[]
  onToggleSelect: (id: number) => void
  onToggleSelectAll: () => void
  onRefresh: () => void
  onViewDetail: (id: number) => void
}

const VoicemailList = ({ 
  voicemails, 
  selectedIds, 
  onToggleSelect, 
  onToggleSelectAll,
  onRefresh,
  onViewDetail
}: VoicemailListProps) => {
  
  const handleMarkRead = async (e: React.MouseEvent, id: number) => {
    e.stopPropagation()
    try {
      const res = await markVoicemailAsRead(id)
      if (res.code === 200) {
        showAlert('已标记为已读', 'success')
        onRefresh()
      }
    } catch (error: any) {
      showAlert(error.msg || '操作失败', 'error')
    }
  }

  const handleToggleImportant = async (e: React.MouseEvent, voicemail: Voicemail) => {
    e.stopPropagation()
    try {
      const res = await markVoicemailAsImportant(voicemail.id, !voicemail.isImportant)
      if (res.code === 200) {
        showAlert(voicemail.isImportant ? '已取消重要标记' : '已标记为重要', 'success')
        onRefresh()
      }
    } catch (error: any) {
      showAlert(error.msg || '操作失败', 'error')
    }
  }

  const handleDelete = async (e: React.MouseEvent, id: number) => {
    e.stopPropagation()
    if (!confirm('确定要删除这条留言吗？')) {
      return
    }
    try {
      const res = await deleteVoicemail(id)
      if (res.code === 200) {
        showAlert('删除成功', 'success')
        onRefresh()
      }
    } catch (error: any) {
      showAlert(error.msg || '删除失败', 'error')
    }
  }

  const formatDuration = (seconds: number) => {
    const mins = Math.floor(seconds / 60)
    const secs = seconds % 60
    return `${mins}:${secs.toString().padStart(2, '0')}`
  }

  const formatDate = (dateStr: string) => {
    const date = new Date(dateStr)
    const now = new Date()
    const diff = now.getTime() - date.getTime()
    const days = Math.floor(diff / (1000 * 60 * 60 * 24))
    
    if (days === 0) {
      return date.toLocaleTimeString('zh-CN', { hour: '2-digit', minute: '2-digit' })
    } else if (days === 1) {
      return '昨天'
    } else if (days < 7) {
      return `${days}天前`
    } else {
      return date.toLocaleDateString('zh-CN', { month: '2-digit', day: '2-digit' })
    }
  }

  const allSelected = voicemails.length > 0 && selectedIds.length === voicemails.length

  return (
    <div className="bg-card border border-border rounded-lg overflow-hidden">
      {/* Header */}
      <div className="flex items-center gap-3 px-4 py-3 border-b border-border bg-muted/30">
        <button
          onClick={onToggleSelectAll}
          className="p-1 hover:bg-muted rounded transition-colors"
        >
          {allSelected ? (
            <CheckSquare className="w-5 h-5 text-primary" />
          ) : (
            <Square className="w-5 h-5 text-muted-foreground" />
          )}
        </button>
        <span className="text-sm text-muted-foreground">
          {voicemails.length} 条留言
        </span>
      </div>

      {/* List */}
      <div className="divide-y divide-border">
        {voicemails.map((voicemail, index) => (
          <motion.div
            key={voicemail.id}
            initial={{ opacity: 0, x: -20 }}
            animate={{ opacity: 1, x: 0 }}
            transition={{ delay: index * 0.05 }}
            onClick={() => onViewDetail(voicemail.id)}
            className={clsx(
              'flex items-center gap-3 px-4 py-4 hover:bg-muted/50 cursor-pointer transition-colors',
              !voicemail.isRead && 'bg-primary/5'
            )}
          >
            {/* Checkbox */}
            <button
              onClick={(e) => {
                e.stopPropagation()
                onToggleSelect(voicemail.id)
              }}
              className="p-1 hover:bg-muted rounded transition-colors"
            >
              {selectedIds.includes(voicemail.id) ? (
                <CheckSquare className="w-5 h-5 text-primary" />
              ) : (
                <Square className="w-5 h-5 text-muted-foreground" />
              )}
            </button>

            {/* Important Star */}
            <button
              onClick={(e) => handleToggleImportant(e, voicemail)}
              className="p-1 hover:bg-muted rounded transition-colors"
            >
              <Star 
                className={clsx(
                  'w-5 h-5',
                  voicemail.isImportant 
                    ? 'text-yellow-500 fill-yellow-500' 
                    : 'text-muted-foreground'
                )}
              />
            </button>

            {/* Read Status */}
            <div className="flex-shrink-0">
              {voicemail.isRead ? (
                <MailOpen className="w-5 h-5 text-muted-foreground" />
              ) : (
                <Mail className="w-5 h-5 text-primary" />
              )}
            </div>

            {/* Content */}
            <div className="flex-1 min-w-0">
              <div className="flex items-center gap-2 mb-1">
                <span className={clsx(
                  'font-medium truncate',
                  voicemail.isRead ? 'text-muted-foreground' : 'text-foreground'
                )}>
                  {voicemail.callerName || voicemail.callerNumber}
                </span>
                {voicemail.callerLocation && (
                  <span className="flex items-center gap-1 text-xs text-muted-foreground">
                    <MapPin className="w-3 h-3" />
                    {voicemail.callerLocation}
                  </span>
                )}
              </div>
              
              {voicemail.summary ? (
                <p className="text-sm text-muted-foreground line-clamp-1">
                  {voicemail.summary}
                </p>
              ) : voicemail.transcribedText ? (
                <p className="text-sm text-muted-foreground line-clamp-1">
                  {voicemail.transcribedText}
                </p>
              ) : (
                <p className="text-sm text-muted-foreground italic">
                  {voicemail.transcribeStatus === 'pending' ? '等待转录...' : '暂无内容'}
                </p>
              )}

              {voicemail.keywords && (
                <div className="flex gap-1 mt-1">
                  {voicemail.keywords.split(',').slice(0, 3).map((keyword, i) => (
                    <span 
                      key={i}
                      className="px-2 py-0.5 bg-primary/10 text-primary text-xs rounded-full"
                    >
                      {keyword.trim()}
                    </span>
                  ))}
                </div>
              )}
            </div>

            {/* Meta */}
            <div className="flex flex-col items-end gap-1 flex-shrink-0">
              <span className="text-xs text-muted-foreground">
                {formatDate(voicemail.createdAt)}
              </span>
              <div className="flex items-center gap-1 text-xs text-muted-foreground">
                <Clock className="w-3 h-3" />
                {formatDuration(voicemail.duration)}
              </div>
            </div>

            {/* Actions */}
            <div className="flex items-center gap-1 flex-shrink-0">
              {!voicemail.isRead && (
                <Button
                  size="xs"
                  variant="ghost"
                  onClick={(e) => handleMarkRead(e, voicemail.id)}
                  title="标记已读"
                >
                  <MailOpen className="w-3.5 h-3.5" />
                </Button>
              )}
              <Button
                size="xs"
                variant="ghost"
                onClick={(e) => handleDelete(e, voicemail.id)}
                title="删除"
              >
                <Trash2 className="w-3.5 h-3.5" />
              </Button>
            </div>
          </motion.div>
        ))}
      </div>
    </div>
  )
}

export default VoicemailList
