import { useState, useRef } from 'react'
import { Upload, X, File, Image, FileText } from 'lucide-react'
import { cn } from '@/utils/cn.ts'
import { showAlert } from '@/utils/notification.ts'

interface FileUploadProps {
  onFileSelect: (files: File[]) => void
  accept?: string
  multiple?: boolean
  maxSize?: number // in MB
  maxFiles?: number
  className?: string
  label?: string
  error?: string
  disabled?: boolean
}

interface FileWithPreview extends File {
  id: string
  preview?: string
}

const FileUpload = ({
  onFileSelect,
  accept = '*/*',
  multiple = false,
  maxSize = 10,
  maxFiles = 5,
  className,
  label,
  error,
  disabled = false
}: FileUploadProps) => {
  const [files, setFiles] = useState<FileWithPreview[]>([])
  const [dragActive, setDragActive] = useState(false)
  const fileInputRef = useRef<HTMLInputElement>(null)

  const getFileIcon = (file: File) => {
    if (file.type.startsWith('image/')) return Image
    if (file.type.startsWith('text/')) return FileText
    return File
  }

  const formatFileSize = (bytes: number) => {
    if (bytes === 0) return '0 Bytes'
    const k = 1024
    const sizes = ['Bytes', 'KB', 'MB', 'GB']
    const i = Math.floor(Math.log(bytes) / Math.log(k))
    return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i]
  }

  const validateFile = (file: File) => {
    if (file.size > maxSize * 1024 * 1024) {
      throw new Error(`文件大小不能超过 ${maxSize}MB`)
    }
    return true
  }

  const handleFiles = (newFiles: FileList | null) => {
    if (!newFiles) return

    const fileArray = Array.from(newFiles)
    const validFiles: FileWithPreview[] = []

    try {
      fileArray.forEach(file => {
        validateFile(file)
        
        if (files.length + validFiles.length >= maxFiles) {
          throw new Error(`最多只能上传 ${maxFiles} 个文件`)
        }

        const fileWithPreview: FileWithPreview = Object.assign(file, {
          id: Math.random().toString(36).substr(2, 9)
        })

        if (file.type.startsWith('image/')) {
          fileWithPreview.preview = URL.createObjectURL(file)
        }

        validFiles.push(fileWithPreview)
      })

      const updatedFiles = multiple ? [...files, ...validFiles] : validFiles
      setFiles(updatedFiles)
      onFileSelect(updatedFiles)
    } catch (error) {
      showAlert(error instanceof Error ? error.message : '文件验证失败', 'error', '验证失败')
    }
  }

  const handleDrag = (e: React.DragEvent) => {
    e.preventDefault()
    e.stopPropagation()
    if (e.type === 'dragenter' || e.type === 'dragover') {
      setDragActive(true)
    } else if (e.type === 'dragleave') {
      setDragActive(false)
    }
  }

  const handleDrop = (e: React.DragEvent) => {
    e.preventDefault()
    e.stopPropagation()
    setDragActive(false)
    
    if (disabled) return
    
    handleFiles(e.dataTransfer.files)
  }

  const handleFileInput = (e: React.ChangeEvent<HTMLInputElement>) => {
    handleFiles(e.target.files)
  }

  const removeFile = (fileId: string) => {
    const updatedFiles = files.filter(file => file.id !== fileId)
    setFiles(updatedFiles)
    onFileSelect(updatedFiles)
  }

  const openFileDialog = () => {
    if (!disabled) {
      fileInputRef.current?.click()
    }
  }

  return (
    <div className={cn('w-full', className)}>
      {label && (
        <label className="block text-sm font-medium text-neutral-700 dark:text-neutral-300 mb-2">
          {label}
        </label>
      )}

      <div
        className={cn(
          'relative border-2 border-dashed rounded-lg p-6 transition-colors',
          dragActive && !disabled
            ? 'border-primary bg-primary/5'
            : 'border-neutral-300 dark:border-neutral-600',
          disabled && 'opacity-50 cursor-not-allowed',
          error && 'border-red-500'
        )}
        onDragEnter={handleDrag}
        onDragLeave={handleDrag}
        onDragOver={handleDrag}
        onDrop={handleDrop}
      >
        <input
          ref={fileInputRef}
          type="file"
          accept={accept}
          multiple={multiple}
          onChange={handleFileInput}
          className="hidden"
          disabled={disabled}
        />

        <div className="text-center">
          <Upload className={cn(
            'mx-auto h-12 w-12 text-neutral-400',
            !disabled && 'group-hover:text-primary'
          )} />
          <div className="mt-4">
            <button
              type="button"
              onClick={openFileDialog}
              disabled={disabled}
              className={cn(
                'text-sm font-medium text-primary hover:text-primary/80 transition-colors',
                disabled && 'cursor-not-allowed'
              )}
            >
              点击上传
            </button>
            <p className="text-xs text-neutral-500 dark:text-neutral-400 mt-1">
              或拖拽文件到此处
            </p>
            <p className="text-xs text-neutral-400 dark:text-neutral-500 mt-1">
              最大 {maxSize}MB，最多 {maxFiles} 个文件
            </p>
          </div>
        </div>
      </div>

      {/* 文件列表 */}
      {files.length > 0 && (
        <div className="mt-4 space-y-2">
          {files.map((file) => {
            const Icon = getFileIcon(file)
            return (
              <div
                key={file.id}
                className="flex items-center justify-between p-3 bg-neutral-50 dark:bg-neutral-800 rounded-lg"
              >
                <div className="flex items-center gap-3">
                  {file.preview ? (
                    <img
                      src={file.preview}
                      alt={file.name}
                      className="w-8 h-8 object-cover rounded"
                    />
                  ) : (
                    <Icon className="w-8 h-8 text-neutral-400" />
                  )}
                  <div>
                    <p className="text-sm font-medium text-neutral-900 dark:text-white">
                      {file.name}
                    </p>
                    <p className="text-xs text-neutral-500 dark:text-neutral-400">
                      {formatFileSize(file.size)}
                    </p>
                  </div>
                </div>
                <button
                  type="button"
                  onClick={() => removeFile(file.id)}
                  className="p-1 text-neutral-400 hover:text-red-500 transition-colors"
                >
                  <X className="w-4 h-4" />
                </button>
              </div>
            )
          })}
        </div>
      )}

      {error && (
        <p className="mt-1 text-sm text-red-600 dark:text-red-400">{error}</p>
      )}
    </div>
  )
}

export default FileUpload
