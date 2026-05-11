import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { motion } from 'framer-motion'
import { User, Eye, EyeOff, LogIn, Lock as LockIcon, Shield } from 'lucide-react'
import Button from '@/components/UI/Button'
import Input from '@/components/UI/Input'
import Modal from '@/components/UI/Modal'
import Captcha from '@/components/Auth/Captcha'
import { useAuthStore } from '@/stores/authStore'
import { showAlert } from '@/utils/notification'
import { loginByPassword, sendDeviceVerificationCode, verifyDeviceForLogin } from '@/api/authApi'
import faviconUrl from '/favicon.png'

const Login = () => {
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [showPassword, setShowPassword] = useState(false)
  const [loading, setLoading] = useState(false)

  // 验证码状态
  const [showCaptchaModal, setShowCaptchaModal] = useState(false)
  const [captchaId, setCaptchaId] = useState('')
  const [captchaType, setCaptchaType] = useState('')
  const [captchaCode, setCaptchaCode] = useState<any>(null)

  // 两步验证
  const [requiresTwoFactor, setRequiresTwoFactor] = useState(false)
  const [twoFactorCode, setTwoFactorCode] = useState('')

  // 新设备验证
  const [requiresDeviceVerification, setRequiresDeviceVerification] = useState(false)
  const [showDeviceVerifyModal, setShowDeviceVerifyModal] = useState(false)
  const [deviceId, setDeviceId] = useState('')
  const [deviceVerifyCode, setDeviceVerifyCode] = useState('')
  const [deviceVerifyLoading, setDeviceVerifyLoading] = useState(false)
  const [deviceSendLoading, setDeviceSendLoading] = useState(false)

  const { login } = useAuthStore()
  const navigate = useNavigate()

  // 验证码验证成功后执行登录
  const performLogin = async (cId: string, cType: string, cData: any) => {
    setLoading(true)
    try {
      const loginData: any = {
        email,
        password,
        captchaId: cId,
        captchaType: cType,
        timezone: Intl.DateTimeFormat().resolvedOptions().timeZone,
        remember: true,
      }

      if (cType === 'image') {
        loginData.captchaCode = cData || ''
      } else {
        loginData.captchaData = typeof cData === 'string' ? cData : JSON.stringify(cData || [])
      }

      if (requiresTwoFactor && twoFactorCode) {
        loginData.twoFactorCode = twoFactorCode
      }

      const response = await loginByPassword(loginData)

      if (response.code !== 200) {
        throw new Error(response.msg || '登录失败')
      }

      if (response.data?.requiresTwoFactor) {
        setRequiresTwoFactor(true)
        setLoading(false)
        return
      }

      if (response.data?.requiresDeviceVerification) {
        const incomingDeviceId = response.data?.deviceId || ''
        setDeviceId(incomingDeviceId)
        setRequiresDeviceVerification(true)
        setShowDeviceVerifyModal(true)
        setDeviceVerifyCode('')
        if (response.data?.verificationSent) {
          showAlert('新设备验证码已发送到邮箱', 'success')
        }
        setLoading(false)
        return
      }

      const payload = response.data || {}
      const token =
        (payload as any).token ||
        (payload as any).access_token ||
        (payload as any).accessToken ||
        (payload as any).user?.token
      const userData = (payload as any).user
      const refreshTok = (payload as any).refreshToken || (payload as any).refresh_token

      if (!token) throw new Error('登录失败：未获取到 token')
      if (!userData) throw new Error('登录失败：未获取到用户信息')

      // 管理员校验已下沉到后端（非管理员后端直接返回 4xx），前端无需重复判断
      if (refreshTok) {
        try {
          localStorage.setItem('refresh_token', refreshTok)
        } catch {
          /* ignore */
        }
      }

      if ((payload as any).suspiciousLogin) {
        showAlert(
          (payload as any).message || '检测到陌生环境登录，已发送提醒邮件，仍将继续进入后台。',
          'warning',
          '安全提示',
        )
      }

      const prof = (userData as any)?.profile || {}
      await login(token, {
        id: userData.id,
        email: userData.email,
        displayName: (prof.displayName as string) || userData.displayName || email,
        role: userData.role,
        avatar: (prof.avatar as string) || userData.avatar,
        profile: userData.profile,
      })

      showAlert('登录成功', 'success', '欢迎回来')
      await new Promise<void>((r) => queueMicrotask(r))
      navigate('/users', { replace: true })
    } catch (error: any) {
      showAlert(error?.msg || error?.message || '登录失败，请检查用户名和密码', 'error', '登录失败')
    } finally {
      setLoading(false)
    }
  }

  const handleCaptchaVerify = (id: string, type: string, data: any) => {
    setCaptchaId(id)
    setCaptchaType(type)
    setCaptchaCode(data)
    setShowCaptchaModal(false)
    performLogin(id, type, data)
  }

  const handleCaptchaError = (error: string) => {
    showAlert(error, 'error', '验证码错误')
  }

  const handleSendDeviceCode = async () => {
    if (!email || !deviceId) {
      showAlert('设备信息不完整，请重新登录', 'error')
      return
    }
    try {
      setDeviceSendLoading(true)
      const res = await sendDeviceVerificationCode({ email, deviceId })
      if (res.code !== 200) throw new Error(res.msg || '发送验证码失败')
      showAlert('验证码已发送到邮箱', 'success')
    } catch (error: any) {
      showAlert(error?.msg || error?.message || '发送验证码失败', 'error')
    } finally {
      setDeviceSendLoading(false)
    }
  }

  const handleVerifyDevice = async () => {
    if (!email || !deviceId || !deviceVerifyCode) {
      showAlert('请输入完整设备验证码', 'error')
      return
    }
    try {
      setDeviceVerifyLoading(true)
      const res = await verifyDeviceForLogin({
        email,
        deviceId,
        verifyCode: deviceVerifyCode,
      })
      if (res.code !== 200) throw new Error(res.msg || '设备验证失败')
      showAlert('设备验证成功，正在继续登录', 'success')
      setRequiresDeviceVerification(false)
      setShowDeviceVerifyModal(false)
      if (captchaId) {
        await performLogin(captchaId, captchaType, captchaCode)
      } else {
        setShowCaptchaModal(true)
      }
    } catch (error: any) {
      showAlert(error?.msg || error?.message || '设备验证失败', 'error')
    } finally {
      setDeviceVerifyLoading(false)
    }
  }

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!email || !password) {
      showAlert('请输入邮箱和密码', 'error', '登录失败')
      return
    }

    // 2FA 阶段：已有验证码信息，直接登录
    if (requiresTwoFactor && twoFactorCode && captchaId) {
      await performLogin(captchaId, captchaType, captchaCode)
      return
    }

    // 弹出验证码
    setShowCaptchaModal(true)
  }

  return (
    <div className="min-h-screen flex items-center justify-center relative overflow-hidden">
      <div className="absolute inset-0 overflow-hidden">
        <div className="absolute -top-40 -right-40 w-96 h-96 bg-teal-300/80 dark:bg-teal-900 rounded-full mix-blend-multiply filter blur-3xl opacity-30 animate-pulse" />
        <div className="absolute -bottom-40 -left-40 w-96 h-96 bg-emerald-300/80 dark:bg-emerald-900 rounded-full mix-blend-multiply filter blur-3xl opacity-30 animate-pulse" />
      </div>

      <motion.div
        initial={{ opacity: 0, y: 20 }}
        animate={{ opacity: 1, y: 0 }}
        transition={{ duration: 0.5 }}
        className="w-full max-w-md relative z-10 px-4"
      >
        <div className="bg-white/95 dark:bg-slate-800/95 backdrop-blur-xl rounded-3xl shadow-2xl p-8 md:p-10 border border-slate-200/50 dark:border-slate-700/50">
          {/* 标题 */}
          <div className="text-center mb-8">
            <motion.div
              initial={{ scale: 0.8, opacity: 0 }}
              animate={{ scale: 1, opacity: 1 }}
              transition={{ delay: 0.2, type: 'spring', stiffness: 200 }}
              className="flex justify-center mb-4"
            >
              <img src={faviconUrl} alt="SoulNexus" className="w-16 h-16 rounded-2xl shadow-md" />
            </motion.div>
            <motion.h1
              initial={{ opacity: 0, y: 10 }}
              animate={{ opacity: 1, y: 0 }}
              transition={{ delay: 0.2 }}
              className="text-3xl font-bold bg-gradient-to-r from-[#4ECDC4] to-[#45b8b0] bg-clip-text mb-2"
            >
              SoulNexus
            </motion.h1>
            <motion.p
              initial={{ opacity: 0, y: 10 }}
              animate={{ opacity: 1, y: 0 }}
              transition={{ delay: 0.3 }}
              className="text-slate-600 dark:text-slate-400 text-sm"
            >
              管理后台
            </motion.p>
          </div>

          <form onSubmit={handleSubmit} className="space-y-5">
            <Input
              type="text"
              label="邮箱"
              placeholder="请输入邮箱"
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              leftIcon={<User className="w-4 h-4" />}
              size="lg"
              required
              disabled={loading}
            />

            <Input
              type={showPassword ? 'text' : 'password'}
              label="密码"
              placeholder="请输入密码"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              leftIcon={<LockIcon className="w-4 h-4" />}
              rightIcon={
                <button
                  type="button"
                  onClick={() => setShowPassword(!showPassword)}
                  className="text-slate-400 hover:text-slate-600 dark:hover:text-slate-300"
                >
                  {showPassword ? <EyeOff className="w-4 h-4" /> : <Eye className="w-4 h-4" />}
                </button>
              }
              size="lg"
              required
              disabled={loading}
            />

            {requiresTwoFactor && (
              <div className="space-y-3 p-4 bg-blue-50 dark:bg-blue-900/20 rounded-lg border border-blue-200 dark:border-blue-800">
                <div className="flex items-center gap-2 text-blue-600 dark:text-blue-400">
                  <Shield className="w-5 h-5" />
                  <p className="text-sm font-medium">两步验证</p>
                </div>
                <Input
                  type="text"
                  label="验证码"
                  placeholder="请输入6位验证码"
                  value={twoFactorCode}
                  onChange={(e) => setTwoFactorCode(e.target.value.replace(/\D/g, '').slice(0, 6))}
                  leftIcon={<Shield className="w-4 h-4" />}
                  size="lg"
                  required
                  disabled={loading}
                  maxLength={6}
                />
              </div>
            )}

            <Button
              type="submit"
              variant="primary"
              size="lg"
              fullWidth
              loading={loading}
              leftIcon={<LogIn className="w-4 h-4" />}
              className="mt-6"
            >
              {loading ? '登录中...' : requiresTwoFactor ? '验证并登录' : requiresDeviceVerification ? '验证设备后登录' : '登录'}
            </Button>
          </form>
        </div>
      </motion.div>

      {/* 验证码弹窗 */}
      <Modal
        isOpen={showCaptchaModal}
        onClose={() => setShowCaptchaModal(false)}
        title="安全验证"
        size="md"
        closeOnOverlayClick={false}
      >
        <Captcha
          onVerify={handleCaptchaVerify}
          onError={handleCaptchaError}
        />
      </Modal>

      {/* 新设备验证弹窗 */}
      <Modal
        isOpen={showDeviceVerifyModal}
        onClose={() => setShowDeviceVerifyModal(false)}
        title="新设备登录验证"
        size="md"
        closeOnOverlayClick={false}
      >
        <div className="space-y-4">
          <div className="text-sm text-slate-600 dark:text-slate-400">
            检测到新设备登录，请输入邮箱验证码完成验证。
          </div>
          <Input
            label="设备ID"
            value={deviceId}
            disabled
          />
          <Input
            label="验证码"
            placeholder="请输入6位验证码"
            value={deviceVerifyCode}
            onChange={(e) => setDeviceVerifyCode(e.target.value.replace(/\D/g, '').slice(0, 6))}
            maxLength={6}
            disabled={deviceVerifyLoading}
          />
          <div className="flex gap-3 justify-end">
            <Button variant="outline" onClick={handleSendDeviceCode} loading={deviceSendLoading}>
              发送验证码
            </Button>
            <Button variant="primary" onClick={handleVerifyDevice} loading={deviceVerifyLoading}>
              验证并继续登录
            </Button>
          </div>
        </div>
      </Modal>
    </div>
  )
}

export default Login
