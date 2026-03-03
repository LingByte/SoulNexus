import { motion } from 'framer-motion'
import { 
  Edit, 
  Trash2, 
  Star,
  Phone,
  Link as LinkIcon,
  Unlink,
  BarChart3
} from 'lucide-react'
import Button from '@/components/UI/Button'
import type { PhoneNumber } from '@/types/phoneNumber'
import { clsx } from 'clsx'

interface PhoneNumberCardProps {
  number: PhoneNumber
  onEdit: () => void
  onDelete: () => void
  onSetPrimary: () => void
  onBind: () => void
  onUnbind: () => void
}

const PhoneNumberCard = ({ 
  number, 
  onEdit, 
  onDelete, 
  onSetPrimary,
  onBind,
  onUnbind
}: PhoneNumberCardProps) => {
  return (
    <motion.div
      whileHover={{ y: -4 }}
      className={clsx(
        'bg-card border rounded-lg p-6 transition-all duration-200',
        number.isPrimary 
          ? 'border-primary shadow-lg shadow-primary/20' 
          : 'border-border hover:border-primary/50 hover:shadow-md'
      )}
    >
      {/* Header */}
      <div className="flex items-start justify-between mb-4">
        <div className="flex-1">
          <div className="flex items-center gap-2 mb-1">
            <Phone className="w-5 h-5 text-primary" />
            <h3 className="text-lg font-semibold text-foreground">
              {number.phoneNumber}
            </h3>
            {number.isPrimary && (
              <Star className="w-4 h-4 text-yellow-500 fill-yellow-500" />
            )}
          </div>
          {number.displayName && (
            <p className="text-sm text-muted-foreground">
              {number.displayName}
            </p>
          )}
        </div>
      </div>

      {/* Info */}
      <div className="space-y-3 mb-4">
        {/* Carrier */}
        {number.carrier && (
          <div className="flex items-center justify-between text-sm">
            <span className="text-muted-foreground">运营商</span>
            <span className="text-foreground font-medium">{number.carrier}</span>
          </div>
        )}

        {/* Scheme */}
        <div className="flex items-center justify-between text-sm">
          <span className="text-muted-foreground">绑定方案</span>
          {number.schemeName ? (
            <span className="text-foreground font-medium">{number.schemeName}</span>
          ) : (
            <span className="text-muted-foreground">未绑定</span>
          )}
        </div>

        {/* Stats */}
        <div className="flex items-center justify-between text-sm">
          <span className="text-muted-foreground">通话次数</span>
          <span className="text-foreground font-medium">{number.callCount || 0}</span>
        </div>

        <div className="flex items-center justify-between text-sm">
          <span className="text-muted-foreground">留言数量</span>
          <span className="text-foreground font-medium">{number.voicemailCount || 0}</span>
        </div>
      </div>

      {/* Status */}
      <div className="mb-4">
        <span className={clsx(
          'inline-flex items-center px-2 py-1 rounded-full text-xs font-medium',
          number.enabled
            ? 'bg-green-100 text-green-700 dark:bg-green-900 dark:text-green-300'
            : 'bg-gray-100 text-gray-700 dark:bg-gray-900 dark:text-gray-300'
        )}>
          {number.enabled ? '已启用' : '已禁用'}
        </span>
      </div>

      {/* Actions */}
      <div className="space-y-2 pt-4 border-t border-border">
        <div className="flex items-center gap-2">
          {!number.isPrimary && (
            <Button
              size="sm"
              variant="outline"
              onClick={onSetPrimary}
              leftIcon={<Star className="w-3.5 h-3.5" />}
              className="flex-1"
            >
              设为主号
            </Button>
          )}
          {number.schemeId ? (
            <Button
              size="sm"
              variant="outline"
              onClick={onUnbind}
              leftIcon={<Unlink className="w-3.5 h-3.5" />}
              className="flex-1"
            >
              解绑方案
            </Button>
          ) : (
            <Button
              size="sm"
              variant="outline"
              onClick={onBind}
              leftIcon={<LinkIcon className="w-3.5 h-3.5" />}
              className="flex-1"
            >
              绑定方案
            </Button>
          )}
        </div>
        <div className="flex items-center gap-2">
          <Button
            size="sm"
            variant="ghost"
            onClick={onEdit}
            className="flex-1"
          >
            <Edit className="w-3.5 h-3.5 mr-1" />
            编辑
          </Button>
          <Button
            size="sm"
            variant="ghost"
            onClick={onDelete}
            disabled={number.isPrimary}
            className="flex-1"
          >
            <Trash2 className="w-3.5 h-3.5 mr-1" />
            删除
          </Button>
        </div>
      </div>
    </motion.div>
  )
}

export default PhoneNumberCard
