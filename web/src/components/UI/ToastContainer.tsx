import React, { createContext, useContext, useState, useCallback } from 'react'
import { createPortal } from 'react-dom'
import Toast, { ToastType, ToastProps } from './Toast'

interface ToastContextType {
  showToast: (type: ToastType, title: string, message?: string, duration?: number) => void
  success: (title: string, message?: string, duration?: number) => void
  error: (title: string, message?: string, duration?: number) => void
  warning: (title: string, message?: string, duration?: number) => void
  info: (title: string, message?: string, duration?: number) => void
}

const ToastContext = createContext<ToastContextType | undefined>(undefined)

export const useToast = () => {
  const context = useContext(ToastContext)
  if (!context) {
    throw new Error('useToast must be used within a ToastProvider')
  }
  return context
}

interface ToastProviderProps {
  children: React.ReactNode
}

export const ToastProvider: React.FC<ToastProviderProps> = ({ children }) => {
  const [toasts, setToasts] = useState<Omit<ToastProps, 'onClose'>[]>([])

  const removeToast = useCallback((id: string) => {
    setToasts(prev => prev.filter(toast => toast.id !== id))
  }, [])

  const showToast = useCallback((
    type: ToastType,
    title: string,
    message?: string,
    duration?: number
  ) => {
    const id = Math.random().toString(36).substr(2, 9)
    const newToast = {
      id,
      type,
      title,
      message,
      duration
    }
    setToasts(prev => [...prev, newToast])
  }, [])

  const success = useCallback((title: string, message?: string, duration?: number) => {
    showToast('success', title, message, duration)
  }, [showToast])

  const error = useCallback((title: string, message?: string, duration?: number) => {
    showToast('error', title, message, duration)
  }, [showToast])

  const warning = useCallback((title: string, message?: string, duration?: number) => {
    showToast('warning', title, message, duration)
  }, [showToast])

  const info = useCallback((title: string, message?: string, duration?: number) => {
    showToast('info', title, message, duration)
  }, [showToast])

  const contextValue: ToastContextType = {
    showToast,
    success,
    error,
    warning,
    info
  }

  return (
    <ToastContext.Provider value={contextValue}>
      {children}
      {typeof window !== 'undefined' && createPortal(
        <div className="fixed top-4 right-4 z-50 pointer-events-none">
          <div className="flex flex-col space-y-2">
            {toasts.map(toast => (
              <Toast
                key={toast.id}
                {...toast}
                onClose={removeToast}
              />
            ))}
          </div>
        </div>,
        document.body
      )}
    </ToastContext.Provider>
  )
}