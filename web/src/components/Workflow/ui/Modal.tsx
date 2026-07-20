import type { ReactNode } from 'react'
import { Modal as ArcoModal } from '@arco-design/web-react'
import { cn } from '@/utils/utils'

interface ModalProps {
  isOpen: boolean
  onClose: () => void
  children: ReactNode
  title?: ReactNode
  size?: 'sm' | 'md' | 'lg' | 'xl' | 'full'
  closeOnOverlayClick?: boolean
  closeOnEscape?: boolean
  showCloseButton?: boolean
  className?: string
  footer?: ReactNode
}

const widthMap: Record<NonNullable<ModalProps['size']>, number | string> = {
  sm: 480,
  md: 560,
  lg: 720,
  xl: 960,
  full: 'calc(100vw - 32px)',
}

export default function Modal({
  isOpen,
  onClose,
  children,
  title,
  size = 'md',
  closeOnOverlayClick = true,
  className,
  footer,
}: ModalProps) {
  return (
    <ArcoModal
      visible={isOpen}
      onCancel={onClose}
      title={title}
      footer={footer ?? null}
      maskClosable={closeOnOverlayClick}
      escToExit
      unmountOnExit
      style={{ width: widthMap[size], maxWidth: '100%' }}
      className={cn('workflow-modal', className)}
      maskStyle={{ zIndex: 1100 }}
      wrapStyle={{ zIndex: 1100 }}
    >
      {children}
    </ArcoModal>
  )
}

export function ModalContent({ children, className }: { children: ReactNode; className?: string }) {
  return <div className={cn('py-2', className)}>{children}</div>
}

export function ModalFooter({ children, className }: { children: ReactNode; className?: string }) {
  return <div className={cn('flex justify-end gap-2 pt-4', className)}>{children}</div>
}
