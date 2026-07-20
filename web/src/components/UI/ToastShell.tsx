import { useEffect, type ReactNode } from 'react'
import { ToastProvider, useToast } from '@/components/ui/toast'
import { setToastFn } from '@/utils/notification'

function ToastBridge() {
  const { toast } = useToast()
  useEffect(() => {
    setToastFn(toast)
    return () => setToastFn(null)
  }, [toast])
  return null
}

export default function ToastShell({ children }: { children: ReactNode }) {
  return (
    <ToastProvider>
      <ToastBridge />
      {children}
    </ToastProvider>
  )
}
