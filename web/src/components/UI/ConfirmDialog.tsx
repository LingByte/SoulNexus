import React, { useState } from 'react'
import { motion } from 'framer-motion'
import { AlertTriangle, Info, CheckCircle, XCircle } from 'lucide-react'
import Modal, { ModalContent, ModalFooter } from './Modal'
import Button from './Button'

interface ConfirmDialogProps {
  isOpen: boolean
  onClose: () => void
  onConfirm: () => void | Promise<void>
  title: string
  message: string
  confirmText?: string
  cancelText?: string
  type?: 'warning' | 'danger' | 'info' | 'success'
  loading?: boolean
}

const ConfirmDialog: React.FC<ConfirmDialogProps> = ({
  isOpen,
  onClose,
  onConfirm,
  title,
  message,
  confirmText = '确认',
  cancelText = '取消',
  type = 'warning',
  loading = false
}) => {
  const [confirmBusy, setConfirmBusy] = useState(false)
  const busy = loading || confirmBusy

  const getIcon = () => {
    switch (type) {
      case 'danger':
        return <XCircle className="w-6 h-6 text-red-500" />
      case 'warning':
        return <AlertTriangle className="w-6 h-6 text-yellow-500" />
      case 'info':
        return <Info className="w-6 h-6 text-blue-500" />
      case 'success':
        return <CheckCircle className="w-6 h-6 text-green-500" />
      default:
        return <AlertTriangle className="w-6 h-6 text-yellow-500" />
    }
  }

  const getConfirmButtonVariant = () => {
    switch (type) {
      case 'danger':
        return 'destructive'
      case 'warning':
        return 'destructive'
      case 'info':
        return 'primary'
      case 'success':
        return 'primary'
      default:
        return 'destructive'
    }
  }

  const handleConfirm = async () => {
    if (busy) return
    setConfirmBusy(true)
    try {
      await Promise.resolve(onConfirm())
      onClose()
    } catch {
      /* keep dialog open; parent shows error toast */
    } finally {
      setConfirmBusy(false)
    }
  }

  return (
    <Modal
      isOpen={isOpen}
      onClose={onClose}
      size="sm"
      closeOnOverlayClick={!busy}
      closeOnEscape={!busy}
      showCloseButton={false}
    >
      <ModalContent>
        <div className="flex items-start space-x-4">
          <motion.div
            initial={{ scale: 0 }}
            animate={{ scale: 1 }}
            transition={{ delay: 0.1, type: "spring", stiffness: 200 }}
            className="flex-shrink-0"
          >
            {getIcon()}
          </motion.div>
          <div className="flex-1">
            <h3 className="text-lg font-semibold text-gray-900 dark:text-white mb-2">
              {title}
            </h3>
            <p className="text-sm text-gray-600 dark:text-gray-400">
              {message}
            </p>
          </div>
        </div>
      </ModalContent>
      <ModalFooter>
        <Button
          variant="outline"
          onClick={onClose}
          disabled={busy}
        >
          {cancelText}
        </Button>
        <Button
          variant={getConfirmButtonVariant()}
          onClick={() => void handleConfirm()}
          loading={busy}
        >
          {confirmText}
        </Button>
      </ModalFooter>
    </Modal>
  )
}

export default ConfirmDialog