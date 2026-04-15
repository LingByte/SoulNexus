import React, { useState, useEffect } from 'react'
import { motion, AnimatePresence } from 'framer-motion'
import { X, Loader2 } from 'lucide-react'
import { getCaptcha } from '@/api/auth'

interface CaptchaModalProps {
  isOpen: boolean
  onClose: () => void
  onVerify: (captchaId: string, captchaCode: string, captchaType: 'image' | 'click') => void
  title?: string
}

const CaptchaModal: React.FC<CaptchaModalProps> = ({
  isOpen,
  onClose,
  onVerify,
  title = '图形验证码'
}) => {
  const [captchaImage, setCaptchaImage] = useState<string>('')
  const [captchaId, setCaptchaId] = useState<string>('')
  const [captchaCode, setCaptchaCode] = useState<string>('')
  const [captchaType, setCaptchaType] = useState<'image' | 'click'>('image')
  const [captchaCount, setCaptchaCount] = useState<number>(0)
  const [captchaWords, setCaptchaWords] = useState<string[]>([])
  const [captchaPoints, setCaptchaPoints] = useState<Array<{ x: number; y: number }>>([])
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string>('')

  const fetchCaptcha = async () => {
    setLoading(true)
    setError('')
    try {
      const response = await getCaptcha()
      if (response.code === 200 && response.data) {
        setCaptchaImage(response.data.image)
        setCaptchaId(response.data.id)
        setCaptchaType(response.data.type || 'image')
        setCaptchaCount(response.data.count || 0)
        setCaptchaWords(response.data.words || [])
        setCaptchaPoints([])
        setCaptchaCode('') // 清空输入
      } else {
        setError('获取验证码失败')
      }
    } catch (err: any) {
      setError(err.message || '获取验证码失败')
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    if (isOpen) {
      fetchCaptcha()
    } else {
      // 关闭时重置
      setCaptchaCode('')
      setCaptchaId('')
      setCaptchaImage('')
      setCaptchaType('image')
      setCaptchaCount(0)
      setCaptchaWords([])
      setCaptchaPoints([])
      setError('')
    }
  }, [isOpen])

  const handleRefresh = () => {
    fetchCaptcha()
  }

  const handleVerify = () => {
    if (!captchaId) {
      setError('验证码未加载，请刷新重试')
      return
    }
    if (captchaType === 'click') {
      if (!captchaPoints.length || (captchaCount > 0 && captchaPoints.length !== captchaCount)) {
        setError('请按提示完成点击验证码')
        return
      }
      onVerify(captchaId, JSON.stringify(captchaPoints), captchaType)
      return
    }
    if (!captchaCode.trim()) {
      setError('请输入验证码')
      return
    }
    onVerify(captchaId, captchaCode, captchaType)
  }

  const handleKeyPress = (e: React.KeyboardEvent) => {
    if (e.key === 'Enter') {
      handleVerify()
    }
  }

  const handleCaptchaImageClick = (e: React.MouseEvent<HTMLImageElement>) => {
    if (captchaType !== 'click') {
      fetchCaptcha()
      return
    }
    const rect = e.currentTarget.getBoundingClientRect()
    const x = Math.round(e.clientX - rect.left)
    const y = Math.round(e.clientY - rect.top)
    setCaptchaPoints((prev) => {
      if (captchaCount > 0 && prev.length >= captchaCount) return prev
      return [...prev, { x, y }]
    })
    setError('')
  }

  return (
    <AnimatePresence>
      {isOpen && (
        <>
          {/* 背景遮罩 */}
          <motion.div
            initial={{ opacity: 0 }}
            animate={{ opacity: 1 }}
            exit={{ opacity: 0 }}
            onClick={onClose}
            className="fixed inset-0 bg-black/50 backdrop-blur-sm z-[10000]"
          />
          
          {/* 弹窗内容 */}
          <div className="fixed inset-0 z-[10000] flex items-center justify-center p-4">
            <motion.div
              initial={{ opacity: 0, scale: 0.95, y: 20 }}
              animate={{ opacity: 1, scale: 1, y: 0 }}
              exit={{ opacity: 0, scale: 0.95, y: 20 }}
              onClick={(e) => e.stopPropagation()}
              className="bg-white dark:bg-gray-800 rounded-2xl shadow-2xl w-full max-w-md p-6 relative z-[10001]"
            >
              {/* 关闭按钮 */}
              <button
                onClick={onClose}
                className="absolute top-4 right-4 text-gray-400 hover:text-gray-600 dark:hover:text-gray-300 transition-colors"
              >
                <X className="w-5 h-5" />
              </button>

              {/* 标题 */}
              <h3 className="text-xl font-semibold text-gray-900 dark:text-white mb-6 pr-8">
                {captchaType === 'click' ? `请依次点击图中字符：${captchaWords.join('、') || '图中字符'}` : title}
              </h3>

              {/* 验证码图片 */}
              <div className="mb-4">
                <div className="relative w-full h-32 border-2 border-gray-300 dark:border-gray-600 rounded-lg overflow-hidden bg-gray-100 dark:bg-gray-700 flex items-center justify-center">
                  {loading ? (
                    <Loader2 className="w-8 h-8 animate-spin text-gray-400" />
                  ) : captchaImage ? (
                    <img
                      src={captchaImage}
                      alt="验证码"
                      className="w-full h-full object-contain cursor-pointer"
                      onClick={handleCaptchaImageClick}
                      title="点击刷新验证码"
                    />
                  ) : (
                    <div className="text-sm text-gray-400">加载中...</div>
                  )}
                </div>
              </div>

              {/* 输入框 */}
              <div className={`mb-4 ${captchaType === 'click' ? 'hidden' : ''}`}>
                <input
                  type="text"
                  value={captchaCode}
                  onChange={(e) => {
                    setCaptchaCode(e.target.value)
                    setError('')
                  }}
                  onKeyDown={handleKeyPress}
                  placeholder="请输入验证码"
                  className="w-full px-4 py-3 border-2 border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-800 text-gray-900 dark:text-white focus:ring-2 focus:ring-primary focus:border-transparent text-center text-lg tracking-widest uppercase"
                  maxLength={6}
                  autoFocus
                />
              </div>

              {/* 错误提示 */}
              {error && (
                <div className="mb-4 text-sm text-red-500 dark:text-red-400">
                  {error}
                </div>
              )}

              {/* 按钮 */}
              <div className="flex gap-3">
                <button
                  onClick={onClose}
                  className="flex-1 px-4 py-2.5 border border-gray-300 dark:border-gray-600 rounded-lg text-gray-700 dark:text-gray-300 hover:bg-gray-50 dark:hover:bg-gray-700 transition-colors font-medium"
                >
                  取消
                </button>
                <button
                  onClick={handleVerify}
                  disabled={loading || !captchaCode.trim()}
                  className="flex-1 px-4 py-2.5 bg-primary text-white rounded-lg hover:bg-primary/90 transition-colors font-medium disabled:opacity-50 disabled:cursor-not-allowed"
                >
                  验证
                </button>
              </div>
            </motion.div>
          </div>
        </>
      )}
    </AnimatePresence>
  )
}

export default CaptchaModal

