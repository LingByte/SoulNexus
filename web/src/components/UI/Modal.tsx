import { ReactNode, useEffect } from 'react'
import { motion, AnimatePresence } from 'framer-motion'
import { X } from 'lucide-react'
import { clsx } from 'clsx'

interface ModalProps {
  isOpen: boolean
  onClose: () => void
  children: ReactNode
  title?: string
  size?: 'sm' | 'md' | 'lg' | 'xl' | 'full'
  closeOnOverlayClick?: boolean
  closeOnEscape?: boolean
  showCloseButton?: boolean
  className?: string
}

const Modal = ({ 
  isOpen, 
  onClose, 
  children, 
  title,
  size = 'md',
  closeOnOverlayClick = true,
  closeOnEscape = true,
  showCloseButton = true,
  className
}: ModalProps) => {
  const sizeClasses = {
    sm: 'max-w-md',
    md: 'max-w-lg',
    lg: 'max-w-2xl',
    xl: 'max-w-4xl',
    full: 'max-w-full mx-4',
  }

  useEffect(() => {
    if (!isOpen) return

    const handleEscape = (e: KeyboardEvent) => {
      if (e.key === 'Escape' && closeOnEscape) {
        onClose()
      }
    }

    document.addEventListener('keydown', handleEscape)
    document.body.style.overflow = 'hidden'

    return () => {
      document.removeEventListener('keydown', handleEscape)
      document.body.style.overflow = 'unset'
    }
  }, [isOpen, onClose, closeOnEscape])

  return (
    <AnimatePresence>
      {isOpen && (
        <motion.div
          initial={{ opacity: 0 }}
          animate={{ opacity: 1 }}
          exit={{ opacity: 0 }}
          transition={{ duration: 0.2 }}
          className="fixed inset-0 z-[9999] flex items-center justify-center p-4 min-h-screen"
        >
          {/* Overlay */}
          <motion.div
            initial={{ opacity: 0 }}
            animate={{ opacity: 1 }}
            exit={{ opacity: 0 }}
            onClick={closeOnOverlayClick ? onClose : undefined}
            className="absolute inset-0 bg-background/80 backdrop-blur-sm"
          />
          
          {/* Modal Content */}
          <motion.div
            initial={{ opacity: 0, scale: 0.95, y: 20 }}
            animate={{ opacity: 1, scale: 1, y: 0 }}
            exit={{ opacity: 0, scale: 0.95, y: 20 }}
            transition={{ duration: 0.2, ease: [0.16, 1, 0.3, 1] }}
            className={clsx(
              'relative bg-background rounded-lg shadow-lg w-full border',
              sizeClasses[size],
              className
            )}
            onClick={(e) => e.stopPropagation()}
          >
            {/* Header */}
            {title && (
              <div className="flex items-center justify-between px-6 py-4 border-b">
                <h2 className="text-lg font-semibold text-foreground">
                  {title}
                </h2>
                {showCloseButton && (
                  <motion.button
                    whileHover={{ scale: 1.05 }}
                    whileTap={{ scale: 0.95 }}
                    onClick={onClose}
                    className="p-2 text-muted-foreground hover:text-foreground hover:bg-accent rounded-md transition-colors"
                  >
                    <X className="w-4 h-4" />
                  </motion.button>
                )}
              </div>
            )}
            
            {/* Content */}
            <div className="px-6 py-4">
              {children}
            </div>
          </motion.div>
        </motion.div>
      )}
    </AnimatePresence>
  )
}

interface ModalHeaderProps {
  children: ReactNode
  onClose?: () => void
  showCloseButton?: boolean
  className?: string
}

const ModalHeader = ({ 
  children, 
  onClose, 
  showCloseButton = true,
  className 
}: ModalHeaderProps) => (
  <div className={clsx('flex items-center justify-between px-6 py-4 border-b border-neutral-200 dark:border-neutral-700', className)}>
    <div className="flex-1">{children}</div>
    {showCloseButton && onClose && (
      <motion.button
        whileHover={{ scale: 1.05 }}
        whileTap={{ scale: 0.95 }}
        onClick={onClose}
        className="p-2 text-neutral-400 hover:text-neutral-600 dark:hover:text-neutral-300 hover:bg-neutral-100 dark:hover:bg-neutral-700 rounded-lg transition-all duration-200"
      >
        <X className="w-5 h-5" />
      </motion.button>
    )}
  </div>
)

interface ModalTitleProps {
  children: ReactNode
  className?: string
}

const ModalTitle = ({ children, className }: ModalTitleProps) => (
  <h2 className={clsx('text-xl font-semibold text-neutral-900 dark:text-white', className)}>
    {children}
  </h2>
)

interface ModalContentProps {
  children: ReactNode
  className?: string
}

const ModalContent = ({ children, className }: ModalContentProps) => (
  <div className={clsx('px-6 py-4', className)}>
    {children}
  </div>
)

interface ModalFooterProps {
  children: ReactNode
  className?: string
}

const ModalFooter = ({ children, className }: ModalFooterProps) => (
  <div className={clsx('flex items-center justify-end space-x-3 px-6 py-4 border-t border-neutral-200 dark:border-neutral-700', className)}>
    {children}
  </div>
)

export default Modal
export { Modal, ModalHeader, ModalTitle, ModalContent, ModalFooter }
