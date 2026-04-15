import { useState, useEffect, useRef } from 'react'
import { RefreshCw, MousePointer2 } from 'lucide-react'
import { get, post } from '@/utils/request'
import { getApiBaseURL } from '@/config/apiConfig'

export interface CaptchaData {
  id: string
  type: 'image' | 'click'
  data: any
  expires: string
}

interface CaptchaProps {
  onVerify: (captchaId: string, captchaType: string, captchaData: any) => void
  onError?: (error: string) => void
}

const Captcha = ({ onVerify, onError }: CaptchaProps) => {
  const [captcha, setCaptcha] = useState<CaptchaData | null>(null)
  const [loading, setLoading] = useState(false)
  const [verifying, setVerifying] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [verified, setVerified] = useState(false)
  
  // Image captcha
  const [imageCode, setImageCode] = useState('')
  
  // Click captcha
  const [clickedPositions, setClickedPositions] = useState<Array<{ x: number; y: number }>>([])
  const clickImageRef = useRef<HTMLImageElement>(null)

  // Load captcha
  const loadCaptcha = async () => {
    setLoading(true)
    setError(null)
    try {
      const response = await get<CaptchaData>(
        `${getApiBaseURL()}/auth/captcha`
      )
      
      if (response.code === 200 && response.data) {
        // 后端返回扁平结构 { id, image }，适配成组件期望的 { id, type, data: { image } }
        const raw = response.data as any
        let adapted: CaptchaData
        if (raw.data?.image !== undefined) {
          // 已经是嵌套结构
          adapted = raw as CaptchaData
        } else {
          adapted = {
            id: raw.id,
            type: raw.type || 'image',
            data: { image: raw.image },
            expires: raw.expires || '',
          }
        }

        // 清理重复的 base64 前缀
        if (adapted.data?.image && typeof adapted.data.image === 'string') {
          let imgData = adapted.data.image
          while (imgData.includes('data:image/png;base64,data:image/png;base64,')) {
            imgData = imgData.replace('data:image/png;base64,data:image/png;base64,', 'data:image/png;base64,')
          }
          adapted.data.image = imgData
        }

        setCaptcha(adapted)
        // Reset states based on type
        setImageCode('')
        setClickedPositions([])
        setVerified(false)
      } else {
        throw new Error(response.msg || 'Failed to load captcha')
      }
    } catch (err: any) {
      const errorMsg = err?.msg || err?.message || 'Failed to load captcha'
      setError(errorMsg)
      onError?.(errorMsg)
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    loadCaptcha()
  }, [])

  // Verify captcha - 直接回调，不预先消费验证码，让登录接口统一验证
  const verifyCaptcha = async (data: any) => {
    if (!captcha) return
    // 直接触发回调，把 id 和 code 传给登录接口去验证
    onVerify(captcha.id, captcha.type, data)
    setVerified(true)
  }

  // Handle image captcha submit
  const handleImageSubmit = () => {
    if (!imageCode.trim()) {
      setError('请输入验证码')
      return
    }
    if (!captcha?.id) {
      setError('验证码已过期，请刷新')
      loadCaptcha()
      return
    }
    verifyCaptcha(imageCode.trim())
  }

  // Handle click captcha
  const handleClickImage = (e: React.MouseEvent<HTMLImageElement>) => {
    if (!clickImageRef.current) return
    const rect = clickImageRef.current.getBoundingClientRect()
    const img = clickImageRef.current
    
    // 计算点击位置相对于图片的坐标
    // 需要考虑图片的实际显示尺寸和原始尺寸的缩放比例
    const scaleX = img.naturalWidth / rect.width
    const scaleY = img.naturalHeight / rect.height
    
    const x = Math.round((e.clientX - rect.left) * scaleX)
    const y = Math.round((e.clientY - rect.top) * scaleY)
    
    const newPositions = [...clickedPositions, { x, y }]
    setClickedPositions(newPositions)
    
    // If we've clicked enough positions, verify
    if (captcha?.data?.count && newPositions.length >= captcha.data.count) {
      verifyCaptcha(newPositions)
    }
  }

  if (loading) {
    return (
      <div className="flex items-center justify-center p-8">
        <RefreshCw className="w-5 h-5 animate-spin text-slate-400" />
        <span className="ml-2 text-sm text-slate-500">Loading captcha...</span>
      </div>
    )
  }

  if (!captcha) {
    return (
      <div className="text-center p-4">
        <p className="text-sm text-red-500 mb-2">{error || 'Failed to load captcha'}</p>
        <button
          onClick={loadCaptcha}
          className="text-sm text-blue-600 hover:text-blue-700 underline"
        >
          Retry
        </button>
      </div>
    )
  }

  return (
    <div className="space-y-4">
      {/* Image Captcha */}
      {captcha.type === 'image' && (
        <div className="space-y-4">
          {verified ? (
            <div className="flex items-center gap-2 p-3 bg-green-50 dark:bg-green-900/20 border border-green-200 dark:border-green-800 rounded-md">
              <svg className="w-5 h-5 text-green-600 dark:text-green-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" />
              </svg>
              <span className="text-sm text-green-600 dark:text-green-400">验证成功</span>
            </div>
          ) : (
            <>
              {/* 第一行：验证码图片 + 换一张 */}
              <div className="flex items-center gap-3">
                <img
                  src={captcha.data?.image?.startsWith('data:') ? captcha.data.image : `data:image/png;base64,${captcha.data?.image}`}
                  alt="验证码"
                  className="h-12 flex-1 object-contain border border-slate-200 dark:border-slate-700 rounded-lg cursor-pointer hover:opacity-80 transition-opacity"
                  onClick={loadCaptcha}
                  title="点击刷新"
                />
                <button
                  type="button"
                  onClick={loadCaptcha}
                  className="flex items-center gap-1 text-xs text-slate-500 dark:text-slate-400 hover:text-blue-600 dark:hover:text-blue-400 transition-colors whitespace-nowrap"
                >
                  <RefreshCw className="w-3 h-3" />
                  换一张
                </button>
              </div>

              {/* 第二行：输入框 */}
              <input
                type="text"
                value={imageCode}
                onChange={(e) => setImageCode(e.target.value)}
                placeholder="请输入验证码"
                autoFocus
                className="w-full px-4 py-3 border border-slate-300 dark:border-slate-600 rounded-lg text-sm text-center tracking-widest focus:outline-none focus:ring-2 focus:ring-blue-500 dark:bg-slate-700 dark:text-white placeholder:tracking-normal"
                onKeyDown={(e) => {
                  if (e.key === 'Enter' && !e.nativeEvent.isComposing && imageCode.trim()) {
                    handleImageSubmit()
                  }
                }}
              />

              {/* 第三行：确认按钮 */}
              <button
                type="button"
                onClick={handleImageSubmit}
                disabled={!imageCode.trim()}
                className="w-full py-2.5 bg-blue-600 text-white rounded-lg text-sm font-medium hover:bg-blue-700 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
              >
                确认验证
              </button>
            </>
          )}
        </div>
      )}

      {/* Click Captcha */}
      {captcha.type === 'click' && (
        <div className="space-y-3">
          <div className="flex items-center justify-between">
            <label className="text-sm font-medium text-slate-700 dark:text-slate-300 flex items-center gap-2">
              <MousePointer2 className="w-4 h-4" />
              点击指定位置 ({clickedPositions.length}/{captcha.data?.count || 0})
            </label>
            <button
              type="button"
              onClick={loadCaptcha}
              className="text-xs text-blue-600 hover:text-blue-700 flex items-center gap-1"
            >
              <RefreshCw className="w-3 h-3" />
              换一张
            </button>
          </div>
          {verified ? (
            <div className="flex items-center gap-2 p-3 bg-green-50 dark:bg-green-900/20 border border-green-200 dark:border-green-800 rounded-md">
              <svg className="w-5 h-5 text-green-600 dark:text-green-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" />
              </svg>
              <span className="text-sm text-green-600 dark:text-green-400">验证成功</span>
            </div>
          ) : (
            <div className="relative">
            <img
              ref={clickImageRef}
              src={captcha.data?.image?.startsWith('data:') ? captcha.data.image : `data:image/png;base64,${captcha.data?.image}`}
              alt="Click captcha"
              className="w-full border border-slate-200 dark:border-slate-700 rounded-md cursor-crosshair"
              onClick={handleClickImage}
            />
              {clickedPositions.map((pos, idx) => {
                // 计算标记的显示位置（考虑图片缩放）
                const img = clickImageRef.current
                if (!img) return null
                const rect = img.getBoundingClientRect()
                const scaleX = rect.width / (img.naturalWidth || rect.width)
                const scaleY = rect.height / (img.naturalHeight || rect.height)
                const displayX = pos.x * scaleX
                const displayY = pos.y * scaleY
                return (
                  <div
                    key={idx}
                    className="absolute w-4 h-4 bg-blue-500 rounded-full border-2 border-white transform -translate-x-1/2 -translate-y-1/2 pointer-events-none"
                    style={{ left: `${displayX}px`, top: `${displayY}px` }}
                  />
                )
              })}
              {verifying && (
                <div className="absolute inset-0 bg-black/20 rounded-md flex items-center justify-center">
                  <RefreshCw className="w-6 h-6 animate-spin text-white" />
                </div>
              )}
            </div>
          )}
        </div>
      )}

      {error && (
        <p className="text-xs text-red-500 mt-2">{error}</p>
      )}
    </div>
  )
}

export default Captcha
