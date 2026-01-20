import React, { useState, useEffect } from 'react'
import { RefreshCw, Loader2 } from 'lucide-react'
import { getCaptcha } from '@/api/auth'

interface CaptchaInputProps {
  value: string
  captchaId: string
  onChange: (code: string, id: string) => void
  onError?: (error: string) => void
  className?: string
}

const CaptchaInput: React.FC<CaptchaInputProps> = ({
  value,
  captchaId,
  onChange,
  onError,
  className = '',
}) => {
  const [captchaImage, setCaptchaImage] = useState<string>('')
  const [loading, setLoading] = useState(false)
  const [currentId, setCurrentId] = useState(captchaId)

  const fetchCaptcha = async () => {
    setLoading(true)
    try {
      const response = await getCaptcha()
      if (response.code === 200 && response.data) {
        setCaptchaImage(response.data.image)
        setCurrentId(response.data.id)
        onChange('', response.data.id) // 清空输入，更新ID
      } else {
        onError?.('获取验证码失败')
      }
    } catch (error: any) {
      onError?.(error.message || '获取验证码失败')
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    if (!captchaImage && !loading) {
      fetchCaptcha()
    }
  }, [])

  const handleRefresh = () => {
    fetchCaptcha()
  }

  return (
    <div className={`space-y-2 ${className}`}>
      <label className="block text-sm font-medium text-gray-700 dark:text-gray-300">
        图形验证码 <span className="text-red-500">*</span>
      </label>
      <div className="flex items-center gap-2">
        <input
          type="text"
          value={value}
          onChange={(e) => onChange(e.target.value, currentId)}
          placeholder="请输入验证码"
          className="flex-1 px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-800 text-gray-900 dark:text-white focus:ring-2 focus:ring-primary focus:border-transparent"
          maxLength={6}
        />
        <div className="relative w-32 h-12 border border-gray-300 dark:border-gray-600 rounded-lg overflow-hidden bg-gray-100 dark:bg-gray-800 flex items-center justify-center">
          {loading ? (
            <Loader2 className="w-6 h-6 animate-spin text-gray-400" />
          ) : captchaImage ? (
            <img
              src={captchaImage}
              alt="验证码"
              className="w-full h-full object-contain cursor-pointer"
              onClick={handleRefresh}
              title="点击刷新验证码"
            />
          ) : (
            <div className="text-xs text-gray-400">加载中...</div>
          )}
          <button
            type="button"
            onClick={handleRefresh}
            disabled={loading}
            className="absolute top-0 right-0 p-1 bg-gray-200 dark:bg-gray-700 hover:bg-gray-300 dark:hover:bg-gray-600 rounded-bl-lg transition-colors"
            title="刷新验证码"
          >
            <RefreshCw className={`w-3 h-3 ${loading ? 'animate-spin' : ''}`} />
          </button>
        </div>
      </div>
    </div>
  )
}

export default CaptchaInput

