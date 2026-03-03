import { motion } from 'framer-motion'
import { 
  Edit, 
  Trash2, 
  Power, 
  PowerOff,
  Mic,
  Phone,
  MessageSquare,
  Clock
} from 'lucide-react'
import Button from '@/components/UI/Button'
import type { Scheme } from '@/types/scheme'
import { clsx } from 'clsx'

interface SchemeCardProps {
  scheme: Scheme
  onEdit: () => void
  onDelete: () => void
  onToggleActive: () => void
}

const SchemeCard = ({ scheme, onEdit, onDelete, onToggleActive }: SchemeCardProps) => {
  return (
    <motion.div
      whileHover={{ y: -4 }}
      className={clsx(
        'bg-card border rounded-lg p-6 transition-all duration-200',
        scheme.isActive 
          ? 'border-primary shadow-lg shadow-primary/20' 
          : 'border-border hover:border-primary/50 hover:shadow-md'
      )}
    >
      {/* Header */}
      <div className="flex items-start justify-between mb-4">
        <div className="flex-1">
          <div className="flex items-center gap-2 mb-1">
            <h3 className="text-lg font-semibold text-foreground">
              {scheme.schemeName}
            </h3>
            {scheme.isActive && (
              <span className="px-2 py-0.5 bg-green-100 text-green-700 dark:bg-green-900 dark:text-green-300 rounded-full text-xs font-medium">
                激活中
              </span>
            )}
          </div>
          {scheme.description && (
            <p className="text-sm text-muted-foreground line-clamp-2">
              {scheme.description}
            </p>
          )}
        </div>
      </div>

      {/* Info Grid */}
      <div className="space-y-3 mb-4">
        {/* AI Assistant */}
        <div className="flex items-start gap-2">
          <MessageSquare className="w-4 h-4 text-muted-foreground mt-0.5 flex-shrink-0" />
          <div className="flex-1 min-w-0">
            <div className="text-xs text-muted-foreground mb-0.5">AI助手</div>
            <div className="text-sm text-foreground line-clamp-2">
              {scheme.assistant?.name || (scheme.assistantId ? `助手 #${scheme.assistantId}` : '未绑定')}
            </div>
          </div>
        </div>

        {/* Auto Answer */}
        <div className="flex items-center gap-2">
          <Phone className="w-4 h-4 text-muted-foreground flex-shrink-0" />
          <div className="flex-1 min-w-0">
            <div className="text-xs text-muted-foreground mb-0.5">自动接听</div>
            <div className="text-sm text-foreground">
              {scheme.autoAnswer 
                ? (scheme.autoAnswerDelay > 0 ? `延迟 ${scheme.autoAnswerDelay} 秒` : '立即接听')
                : '已关闭'
              }
            </div>
          </div>
        </div>

        {/* Recording */}
        <div className="flex items-center gap-2">
          <Mic className="w-4 h-4 text-muted-foreground flex-shrink-0" />
          <div className="flex-1 min-w-0">
            <div className="text-xs text-muted-foreground mb-0.5">录音设置</div>
            <div className="text-sm text-foreground">
              {scheme.recordingEnabled 
                ? (scheme.recordingMode === 'full' ? '全程录音' : '仅留言录音')
                : '不录音'
              }
            </div>
          </div>
        </div>

        {/* Stats */}
        <div className="flex items-center gap-2">
          <Clock className="w-4 h-4 text-muted-foreground flex-shrink-0" />
          <div className="flex-1 min-w-0">
            <div className="text-xs text-muted-foreground mb-0.5">使用统计</div>
            <div className="text-sm text-foreground">
              {scheme.callCount || 0} 次通话 · {scheme.messageCount || 0} 条留言
            </div>
          </div>
        </div>
      </div>

      {/* Actions */}
      <div className="flex items-center gap-2 pt-4 border-t border-border">
        <Button
          size="sm"
          variant={scheme.isActive ? 'destructive' : 'primary'}
          onClick={onToggleActive}
          leftIcon={scheme.isActive ? <PowerOff className="w-3.5 h-3.5" /> : <Power className="w-3.5 h-3.5" />}
          className="flex-1"
        >
          {scheme.isActive ? '停用' : '激活'}
        </Button>
        <Button
          size="sm"
          variant="ghost"
          onClick={onEdit}
        >
          <Edit className="w-3.5 h-3.5" />
        </Button>
        <Button
          size="sm"
          variant="ghost"
          onClick={onDelete}
          disabled={scheme.isActive}
        >
          <Trash2 className="w-3.5 h-3.5" />
        </Button>
      </div>
    </motion.div>
  )
}

export default SchemeCard
