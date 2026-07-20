import { useState, useCallback, useRef, createContext, useContext, type ReactNode } from 'react'
import { createPortal } from 'react-dom'
import { cn } from '@/utils/utils.ts'
import { X, CheckCircle2, XCircle, AlertTriangle, Info } from 'lucide-react'

type ToastType = 'success' | 'error' | 'warning' | 'info'

const MAX_VISIBLE = 5
const FADE_MS = 200

interface ToastItem {
  id: number
  type: ToastType
  message: string
  removing: boolean
}

interface ToastContextValue {
  toast: (message: string, type?: ToastType, duration?: number) => void
}

const ToastContext = createContext<ToastContextValue>({ toast: () => {} })

export function useToast() {
  return useContext(ToastContext)
}

let _id = 0

export function ToastProvider({ children }: { children: ReactNode }) {
  const [toasts, setToasts] = useState<ToastItem[]>([])
  const timersRef = useRef<Map<string, ReturnType<typeof setTimeout>>>(new Map())

  const actualRemove = useCallback((id: number) => {
    timersRef.current.delete(String(id))
    setToasts((prev) => prev.filter((t) => t.id !== id))
  }, [])

  const startFadeAndRemove = useCallback(
    (id: number) => {
      const fadeKey = id + ':fade'
      if (timersRef.current.has(fadeKey)) return
      setToasts((prev) => prev.map((t) => (t.id === id ? { ...t, removing: true } : t)))
      const fadeTimer = setTimeout(() => actualRemove(id), FADE_MS)
      timersRef.current.set(fadeKey, fadeTimer)
    },
    [actualRemove],
  )

  const dismiss = useCallback(
    (id: number) => {
      const t = timersRef.current.get(String(id))
      if (t) clearTimeout(t)
      timersRef.current.delete(String(id))
      startFadeAndRemove(id)
    },
    [startFadeAndRemove],
  )

  const toast = useCallback(
    (message: string, type: ToastType = 'info', duration = 3000) => {
      const id = ++_id
      setToasts((prev) => {
        let next = [...prev]
        if (next.length >= MAX_VISIBLE) {
          const victim = next.find((t) => !t.removing) || next[0]
          dismiss(victim.id)
          next = next.filter((t) => t.id !== victim.id)
        }
        return [...next, { id, type, message, removing: false }]
      })
      if (duration > 0) {
        const timer = setTimeout(() => dismiss(id), duration)
        timersRef.current.set(String(id), timer)
      }
    },
    [dismiss],
  )

  return (
    <ToastContext.Provider value={{ toast }}>
      {children}
      {createPortal(
        <div className="fixed top-4 right-4 z-[9999] flex flex-col gap-2 pointer-events-none">
          {toasts.map((t) => (
            <ToastItemComp key={t.id} item={t} onDismiss={() => dismiss(t.id)} />
          ))}
        </div>,
        document.body,
      )}
    </ToastContext.Provider>
  )
}

function ToastItemComp({ item, onDismiss }: { item: ToastItem; onDismiss: () => void }) {
  const iconMap: Record<ToastType, ReactNode> = {
    success: <CheckCircle2 className="h-4 w-4 text-emerald-400 shrink-0" />,
    error: <XCircle className="h-4 w-4 text-red-400 shrink-0" />,
    warning: <AlertTriangle className="h-4 w-4 text-amber-400 shrink-0" />,
    info: <Info className="h-4 w-4 text-blue-400 shrink-0" />,
  }

  const borderMap: Record<ToastType, string> = {
    success: 'border-emerald-500/30',
    error: 'border-red-500/30',
    warning: 'border-amber-500/30',
    info: 'border-blue-500/30',
  }

  return (
    <div
      className={cn(
        'pointer-events-auto flex items-center gap-2.5 pl-3 pr-2 py-2.5 min-w-[280px] max-w-[420px]',
        'rounded-lg border backdrop-blur-xl',
        'bg-white/70 dark:bg-neutral-900/70',
        'shadow-[0_8px_32px_rgba(0,0,0,0.12)]',
        'text-sm text-neutral-800 dark:text-neutral-100',
        borderMap[item.type],
        item.removing ? 'animate-toast-out' : 'animate-toast-in',
      )}
    >
      {iconMap[item.type]}
      <span className="flex-1 min-w-0 break-words">{item.message}</span>
      <button
        onClick={onDismiss}
        className="shrink-0 p-0.5 rounded hover:bg-black/5 dark:hover:bg-white/10 transition-colors"
      >
        <X className="h-3.5 w-3.5 text-neutral-400 dark:text-neutral-500" />
      </button>
    </div>
  )
}
