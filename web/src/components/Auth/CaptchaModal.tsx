import { useState, useEffect, useRef } from 'react'
import { Modal, Input, Button, Spin, Typography, Tag, Space, ConfigProvider } from '@arco-design/web-react'
import { IconRefresh } from '@arco-design/web-react/icon'
import { getCaptcha } from '@/api/auth'

export const AUTH_PRIMARY_COLOR = '#6d28d9'

export interface CaptchaPayload {
  id: string
  type: 'image' | 'click'
  image: string
  count?: number
  tolerance?: number
  words?: string[]
}

interface CaptchaModalProps {
  isOpen: boolean
  onClose: () => void
  onVerify: (
    captchaId: string,
    captchaType: 'image' | 'click',
    payload: string | Array<{ x: number; y: number }>,
  ) => void
  title?: string
}

function normalizeImageSrc(image?: string): string {
  if (!image) return ''
  let img = image
  while (img.includes('data:image/png;base64,data:image/png;base64,')) {
    img = img.replace('data:image/png;base64,data:image/png;base64,', 'data:image/png;base64,')
  }
  if (img.startsWith('data:')) return img
  return `data:image/png;base64,${img}`
}

function adaptCaptchaResponse(raw: any): CaptchaPayload | null {
  if (!raw?.id) return null
  return {
    id: raw.id,
    type: (raw.type ?? 'image') as 'image' | 'click',
    image: normalizeImageSrc(raw.image),
    count: raw.count,
    tolerance: raw.tolerance,
    words: raw.words,
  }
}

const CaptchaModal = ({
  isOpen,
  onClose,
  onVerify,
  title = '安全验证',
}: CaptchaModalProps) => {
  const [captcha, setCaptcha] = useState<CaptchaPayload | null>(null)
  const [imageCode, setImageCode] = useState('')
  const [clickedPositions, setClickedPositions] = useState<Array<{ x: number; y: number }>>([])
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')
  const clickImageRef = useRef<HTMLImageElement>(null)
  const imageContainerRef = useRef<HTMLDivElement>(null)

  const fetchCaptcha = async () => {
    setLoading(true)
    setError('')
    try {
      const response = await getCaptcha()
      if (response.code === 200 && response.data) {
        const adapted = adaptCaptchaResponse(response.data)
        if (!adapted?.id) {
          setError('获取验证码失败')
          return
        }
        setCaptcha(adapted)
        setImageCode('')
        setClickedPositions([])
      } else {
        setError(response.msg || '获取验证码失败')
      }
    } catch (err: any) {
      setError(err?.msg || err?.message || '获取验证码失败')
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    if (isOpen) fetchCaptcha()
    else {
      setCaptcha(null)
      setImageCode('')
      setClickedPositions([])
      setError('')
    }
  }, [isOpen])

  const handleImageVerify = () => {
    if (!captcha?.id) {
      setError('验证码已过期，请刷新')
      fetchCaptcha()
      return
    }
    if (!imageCode.trim()) {
      setError('请输入验证码')
      return
    }
    onVerify(captcha.id, 'image', imageCode.trim())
  }

  const handleClickImage = (e: React.MouseEvent<HTMLImageElement>) => {
    if (!clickImageRef.current || !captcha) return
    const rect = clickImageRef.current.getBoundingClientRect()
    const img = clickImageRef.current
    const scaleX = img.naturalWidth / rect.width
    const scaleY = img.naturalHeight / rect.height
    const x = Math.round((e.clientX - rect.left) * scaleX)
    const y = Math.round((e.clientY - rect.top) * scaleY)
    const next = [...clickedPositions, { x, y }]
    setClickedPositions(next)
    setError('')
    const required = Number(captcha.count || 0)
    if (required > 0 && next.length >= required) {
      onVerify(captcha.id, 'click', next)
    }
  }

  const requiredClicks = Number(captcha?.count || 0)
  const imageSrc = captcha?.image || ''

  return (
    <ConfigProvider theme={{ primaryColor: AUTH_PRIMARY_COLOR }}>
      <Modal
        visible={isOpen}
        title={title}
        onCancel={onClose}
        footer={null}
        style={{ width: captcha?.type === 'click' ? 440 : 400 }}
        unmountOnExit
      >
        {loading && !captcha ? (
          <div style={{ display: 'flex', justifyContent: 'center', padding: 32 }}>
            <Spin />
          </div>
        ) : !captcha ? (
          <Space direction="vertical" style={{ width: '100%' }}>
            <Typography.Text type="error">{error || '加载失败'}</Typography.Text>
            <Button type="primary" onClick={fetchCaptcha}>
              重试
            </Button>
          </Space>
        ) : captcha.type === 'click' ? (
          <Space direction="vertical" style={{ width: '100%' }} size={12}>
            <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
              <Typography.Text>
                请依次点击图片中的字符
                {captcha.words?.length ? (
                  <span style={{ marginLeft: 8 }}>
                    {captcha.words.map((w) => (
                      <Tag key={w} color="arcoblue" style={{ marginRight: 4 }}>
                        {w}
                      </Tag>
                    ))}
                  </span>
                ) : null}
              </Typography.Text>
              <Button type="text" size="small" icon={<IconRefresh />} onClick={fetchCaptcha}>
                换一张
              </Button>
            </div>
            <Typography.Text type="secondary" style={{ fontSize: 12 }}>
              已点击 {clickedPositions.length}/{requiredClicks || '?'}
            </Typography.Text>
            <div ref={imageContainerRef} style={{ position: 'relative', lineHeight: 0 }}>
              <img
                ref={clickImageRef}
                src={imageSrc}
                alt="点击验证码"
                style={{
                  width: '100%',
                  borderRadius: 8,
                  border: '1px solid var(--color-border-2)',
                  cursor: 'crosshair',
                }}
                onClick={handleClickImage}
              />
              {clickedPositions.map((pos, idx) => {
                const img = clickImageRef.current
                if (!img) return null
                const rect = img.getBoundingClientRect()
                const scaleX = rect.width / (img.naturalWidth || rect.width)
                const scaleY = rect.height / (img.naturalHeight || rect.height)
                return (
                  <span
                    key={idx}
                    style={{
                      position: 'absolute',
                      left: pos.x * scaleX,
                      top: pos.y * scaleY,
                      width: 20,
                      height: 20,
                      marginLeft: -10,
                      marginTop: -10,
                      borderRadius: '50%',
                      background: AUTH_PRIMARY_COLOR,
                      color: '#fff',
                      fontSize: 11,
                      fontWeight: 600,
                      display: 'flex',
                      alignItems: 'center',
                      justifyContent: 'center',
                      pointerEvents: 'none',
                      boxShadow: '0 0 0 2px #fff',
                    }}
                  >
                    {idx + 1}
                  </span>
                )
              })}
            </div>
          </Space>
        ) : (
          <Space direction="vertical" style={{ width: '100%' }} size={12}>
            <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
              <div
                style={{
                  flex: 1,
                  height: 48,
                  border: '1px solid var(--color-border-2)',
                  borderRadius: 8,
                  overflow: 'hidden',
                  display: 'flex',
                  alignItems: 'center',
                  justifyContent: 'center',
                  background: 'var(--color-fill-2)',
                  cursor: 'pointer',
                }}
                onClick={fetchCaptcha}
                title="点击刷新"
              >
                <img src={imageSrc} alt="图形验证码" style={{ height: '100%', objectFit: 'contain' }} />
              </div>
              <Button type="text" icon={<IconRefresh />} onClick={fetchCaptcha}>
                换一张
              </Button>
            </div>
            <Input
              placeholder="请输入图形验证码"
              value={imageCode}
              onChange={setImageCode}
              onPressEnter={handleImageVerify}
              allowClear
            />
            <Button type="primary" long onClick={handleImageVerify} disabled={!imageCode.trim()}>
              确认
            </Button>
          </Space>
        )}

        {error && (
          <Typography.Text type="error" style={{ display: 'block', marginTop: 12 }}>
            {error}
          </Typography.Text>
        )}
      </Modal>
    </ConfigProvider>
  )
}

export default CaptchaModal
