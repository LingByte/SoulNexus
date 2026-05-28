import React from 'react'
import { motion } from 'framer-motion'
import { Loader2, Mic, PhoneOff } from 'lucide-react'
import { cn } from '@/utils/cn'

interface VoiceBallProps {
  isCalling: boolean
  isConnecting?: boolean
  onToggleCall: () => void
  className?: string
}

/** 侧边栏语音通话主按钮：纯视觉层，不单独申请麦克风（由通话流程统一管理）。 */
const VoiceBall: React.FC<VoiceBallProps> = ({
  isCalling,
  isConnecting = false,
  onToggleCall,
  className = '',
}) => {
  const active = isCalling || isConnecting

  return (
    <div className={cn('flex items-center justify-center', className)}>
      <motion.button
        type="button"
        aria-label={isCalling ? '结束对话' : isConnecting ? '正在连接' : '开始语音对话'}
        disabled={isConnecting}
        onClick={onToggleCall}
        className={cn(
          'relative flex h-[6.25rem] w-[6.25rem] items-center justify-center rounded-full',
          'transition-all duration-300 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-offset-2',
          'focus-visible:ring-violet-400 focus-visible:ring-offset-white dark:focus-visible:ring-offset-neutral-800',
          isCalling
            ? 'bg-gradient-to-br from-rose-400 via-rose-500 to-red-600 shadow-xl shadow-rose-500/45'
            : 'bg-gradient-to-br from-violet-500 via-fuchsia-500 to-indigo-600 shadow-lg shadow-violet-500/40',
          !active && 'hover:scale-[1.03] hover:shadow-xl hover:shadow-violet-500/50',
          isConnecting && 'cursor-wait opacity-95',
        )}
        whileTap={isConnecting ? undefined : { scale: 0.97 }}
      >
        {/* 外圈彩色光晕 */}
        <span
          className={cn(
            'pointer-events-none absolute inset-[-6px] rounded-full opacity-70 blur-md',
            isCalling
              ? 'bg-gradient-to-br from-rose-400/60 to-orange-400/40'
              : 'bg-gradient-to-br from-violet-400/70 via-fuchsia-400/50 to-cyan-400/40',
          )}
        />

        {/* 待机：双层呼吸环 */}
        {!active && (
          <>
            <motion.span
              className="pointer-events-none absolute inset-[-2px] rounded-full border-2 border-violet-300/50 dark:border-violet-400/40"
              animate={{ scale: [1, 1.14], opacity: [0.65, 0.2] }}
              transition={{ duration: 2.4, repeat: Infinity, ease: 'easeOut' }}
            />
            <motion.span
              className="pointer-events-none absolute inset-[-8px] rounded-full border border-fuchsia-300/35 dark:border-fuchsia-400/25"
              animate={{ scale: [1, 1.1], opacity: [0.45, 0.12] }}
              transition={{ duration: 2.4, repeat: Infinity, ease: 'easeOut', delay: 0.35 }}
            />
            <motion.span
              className="pointer-events-none absolute inset-0 rounded-full bg-white/15"
              animate={{ opacity: [0.25, 0.45, 0.25] }}
              transition={{ duration: 2.8, repeat: Infinity, ease: 'easeInOut' }}
            />
          </>
        )}

        {/* 通话中：玫红扩散波纹 */}
        {isCalling && (
          <>
            <motion.span
              className="pointer-events-none absolute inset-0 rounded-full border-2 border-rose-200/70"
              animate={{ scale: [1, 1.5], opacity: [0.6, 0] }}
              transition={{ duration: 1.6, repeat: Infinity, ease: 'easeOut' }}
            />
            <motion.span
              className="pointer-events-none absolute inset-0 rounded-full border border-orange-200/50"
              animate={{ scale: [1, 1.32], opacity: [0.45, 0] }}
              transition={{ duration: 1.6, repeat: Infinity, ease: 'easeOut', delay: 0.5 }}
            />
          </>
        )}

        {/* 连接中：旋转描边 */}
        {isConnecting && (
          <span className="pointer-events-none absolute inset-[-4px] rounded-full border-[3px] border-transparent border-t-white border-r-fuchsia-200/80 animate-spin" />
        )}

        <span className="relative z-10 flex items-center justify-center drop-shadow-md">
          {isConnecting ? (
            <Loader2 className="h-7 w-7 animate-spin" strokeWidth={2.25} />
          ) : isCalling ? (
            <PhoneOff className="h-7 w-7" strokeWidth={2.25} />
          ) : (
            <Mic className="h-7 w-7" strokeWidth={2.25} />
          )}
        </span>
      </motion.button>
    </div>
  )
}

export default VoiceBall
