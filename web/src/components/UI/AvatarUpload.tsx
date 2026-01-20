import React, { useState, useRef } from 'react'
import { motion } from 'framer-motion'
import { Upload, X, User, Check, AlertCircle } from 'lucide-react'
import { useAuthStore } from '@/stores/authStore'
import { useUIStore } from '@/stores/uiStore'

interface AvatarUploadProps {
  currentAvatar?: string
  onAvatarChange?: (avatarUrl: string) => void
  size?: 'sm' | 'md' | 'lg' | 'xl'
  className?: string
}

const AvatarUpload: React.FC<AvatarUploadProps> = ({
  currentAvatar,
  onAvatarChange,
  size = 'md',
  className = ''
}) => {
  const [isUploading, setIsUploading] = useState(false)
  const [uploadProgress, setUploadProgress] = useState(0)
  const [error, setError] = useState<string | null>(null)
  const [preview, setPreview] = useState<string | null>(null)
  const fileInputRef = useRef<HTMLInputElement>(null)
  
  const { user, updateUser } = useAuthStore()
  const { addNotification } = useUIStore()

  const sizeClasses = {
    sm: 'w-16 h-16',
    md: 'w-24 h-24',
    lg: 'w-32 h-32',
    xl: 'w-40 h-40'
  }

  const handleFileSelect = (event: React.ChangeEvent<HTMLInputElement>) => {
    const file = event.target.files?.[0]
    if (!file) return

    // 验证文件类型
    const allowedTypes = ['image/jpeg', 'image/jpg', 'image/png', 'image/gif', 'image/webp']
    if (!allowedTypes.includes(file.type)) {
      setError('只支持 JPEG、PNG、GIF、WebP 格式的图片')
      return
    }

    // 验证文件大小 (5MB)
    if (file.size > 5 * 1024 * 1024) {
      setError('图片大小不能超过 5MB')
      return
    }

    setError(null)
    
    // 创建预览
    const reader = new FileReader()
    reader.onload = (e) => {
      setPreview(e.target?.result as string)
    }
    reader.readAsDataURL(file)

    // 上传文件
    uploadAvatar(file)
  }

  const uploadAvatar = async (file: File) => {
    setIsUploading(true)
    setUploadProgress(0)
    setError(null)

    try {
      const formData = new FormData()
      formData.append('avatar', file)

      // 模拟上传进度
      const progressInterval = setInterval(() => {
        setUploadProgress(prev => {
          if (prev >= 90) {
            clearInterval(progressInterval)
            return 90
          }
          return prev + 10
        })
      }, 100)

      const response = await fetch('/api/auth/avatar/upload', {
        method: 'POST',
        headers: {
          'Authorization': `Bearer ${user?.authToken || ''}`
        },
        body: formData
      })

      clearInterval(progressInterval)
      setUploadProgress(100)

      if (!response.ok) {
        const errorData = await response.json()
        throw new Error(errorData.message || '上传失败')
      }

      const data = await response.json()
      
      // 更新用户头像
      if (updateUser) {
        updateUser({ ...user, avatar: data.data.avatar })
      }

      if (onAvatarChange) {
        onAvatarChange(data.data.avatar)
      }

      addNotification({
        type: 'success',
        title: '头像上传成功',
        message: '您的头像已更新',
        duration: 3000
      })

      setPreview(null)
    } catch (err) {
      setError(err instanceof Error ? err.message : '上传失败')
      addNotification({
        type: 'error',
        title: '头像上传失败',
        message: err instanceof Error ? err.message : '上传失败',
        duration: 5000
      })
    } finally {
      setIsUploading(false)
      setUploadProgress(0)
    }
  }

  const deleteAvatar = async () => {
    if (!user?.avatar) return

    try {
      const response = await fetch('/api/auth/avatar', {
        method: 'DELETE',
        headers: {
          'Authorization': `Bearer ${user?.authToken || ''}`
        }
      })

      if (!response.ok) {
        const errorData = await response.json()
        throw new Error(errorData.message || '删除失败')
      }

      // 更新用户头像
      if (updateUser) {
        updateUser({ ...user, avatar: '' })
      }

      if (onAvatarChange) {
        onAvatarChange('')
      }

      addNotification({
        type: 'success',
        title: '头像删除成功',
        message: '已恢复默认头像',
        duration: 3000
      })
    } catch (err) {
      addNotification({
        type: 'error',
        title: '头像删除失败',
        message: err instanceof Error ? err.message : '删除失败',
        duration: 5000
      })
    }
  }

  const handleClick = () => {
    if (!isUploading) {
      fileInputRef.current?.click()
    }
  }

  const displayAvatar = preview || currentAvatar || user?.avatar

  return (
    <div className={`relative ${className}`}>
      {/* 头像显示区域 */}
      <div
        className={`
          ${sizeClasses[size]}
          relative rounded-full overflow-hidden cursor-pointer
          bg-gray-200 dark:bg-gray-700 flex items-center justify-center
          group hover:bg-gray-300 dark:hover:bg-gray-600 transition-colors
          ${isUploading ? 'opacity-50 cursor-not-allowed' : ''}
        `}
        onClick={handleClick}
      >
        {displayAvatar ? (
          <img
            src={displayAvatar}
            alt="用户头像"
            className="w-full h-full object-cover"
          />
        ) : (
          <User className="w-1/2 h-1/2 text-gray-400" />
        )}

        {/* 上传遮罩 */}
        {!isUploading && (
          <div className="absolute inset-0 bg-black bg-opacity-0 group-hover:bg-opacity-50 transition-all duration-200 flex items-center justify-center">
            <Upload className="w-6 h-6 text-white opacity-0 group-hover:opacity-100 transition-opacity" />
          </div>
        )}

        {/* 上传进度 */}
        {isUploading && (
          <div className="absolute inset-0 bg-black bg-opacity-50 flex items-center justify-center">
            <div className="text-white text-sm font-medium">
              {uploadProgress}%
            </div>
          </div>
        )}

        {/* 删除按钮 */}
        {displayAvatar && !isUploading && (
          <button
            onClick={(e) => {
              e.stopPropagation()
              deleteAvatar()
            }}
            className="absolute -top-2 -right-2 w-6 h-6 bg-red-500 text-white rounded-full flex items-center justify-center hover:bg-red-600 transition-colors"
          >
            <X className="w-4 h-4" />
          </button>
        )}
      </div>

      {/* 错误提示 */}
      {error && (
        <motion.div
          initial={{ opacity: 0, y: -10 }}
          animate={{ opacity: 1, y: 0 }}
          className="absolute top-full left-0 right-0 mt-2 p-2 bg-red-100 dark:bg-red-900 text-red-700 dark:text-red-300 text-sm rounded-md flex items-center gap-2"
        >
          <AlertCircle className="w-4 h-4 flex-shrink-0" />
          <span>{error}</span>
        </motion.div>
      )}

      {/* 成功提示 */}
      {uploadProgress === 100 && !error && (
        <motion.div
          initial={{ opacity: 0, y: -10 }}
          animate={{ opacity: 1, y: 0 }}
          className="absolute top-full left-0 right-0 mt-2 p-2 bg-green-100 dark:bg-green-900 text-green-700 dark:text-green-300 text-sm rounded-md flex items-center gap-2"
        >
          <Check className="w-4 h-4 flex-shrink-0" />
          <span>上传成功</span>
        </motion.div>
      )}

      {/* 隐藏的文件输入 */}
      <input
        ref={fileInputRef}
        type="file"
        accept="image/jpeg,image/jpg,image/png,image/gif,image/webp"
        onChange={handleFileSelect}
        className="hidden"
      />
    </div>
  )
}

export default AvatarUpload
