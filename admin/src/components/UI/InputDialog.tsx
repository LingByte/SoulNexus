import { useState, useEffect } from 'react'
import Button from './Button'
import Input from './Input'
import Modal from './Modal'

interface InputDialogProps {
  isOpen: boolean
  onClose: () => void
  onConfirm: (value: string) => void
  title?: string
  label?: string
  placeholder?: string
  defaultValue?: string
  confirmText?: string
  cancelText?: string
  loading?: boolean
  validation?: (value: string) => string | null
}

const InputDialog = ({
  isOpen,
  onClose,
  onConfirm,
  title = '输入',
  label = '请输入',
  placeholder = '',
  defaultValue = '',
  confirmText = '确认',
  cancelText = '取消',
  loading = false,
  validation
}: InputDialogProps) => {
  const [value, setValue] = useState(defaultValue)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    if (isOpen) {
      setValue(defaultValue)
      setError(null)
    }
  }, [isOpen, defaultValue])

  const handleConfirm = () => {
    if (validation) {
      const errorMsg = validation(value)
      if (errorMsg) {
        setError(errorMsg)
        return
      }
    }
    onConfirm(value)
  }

  const handleKeyPress = (e: React.KeyboardEvent) => {
    if (e.key === 'Enter' && !loading) {
      handleConfirm()
    }
  }

  return (
    <Modal
      isOpen={isOpen}
      onClose={onClose}
      size="sm"
      closeOnOverlayClick={!loading}
      closeOnEscape={!loading}
    >
      <div className="p-6">
        <h3 className="text-lg font-semibold text-slate-900 dark:text-white mb-4">
          {title}
        </h3>
        <div className="mb-6">
          <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">
            {label}
          </label>
          <Input
            value={value}
            onChange={(e) => {
              setValue(e.target.value)
              setError(null)
            }}
            onKeyPress={handleKeyPress}
            placeholder={placeholder}
            disabled={loading}
            className={error ? 'border-red-500' : ''}
            autoFocus
          />
          {error && (
            <p className="mt-1 text-sm text-red-600 dark:text-red-400">{error}</p>
          )}
        </div>
        <div className="flex items-center justify-end gap-3">
          <Button
            variant="outline"
            onClick={onClose}
            disabled={loading}
          >
            {cancelText}
          </Button>
          <Button
            onClick={handleConfirm}
            loading={loading}
          >
            {confirmText}
          </Button>
        </div>
      </div>
    </Modal>
  )
}

export default InputDialog
