import { useState, useEffect, useRef, useCallback } from 'react'
import { useNavigate } from 'react-router-dom'
import { useAuthStore } from '@/stores/authStore.ts'
import { useAuthModalStore } from '@/stores/authModalStore.ts'
import { showAlert } from '@/utils/notification'
import {
  sendEmailCode,
  registerUserByEmail,
  registerUser,
  loginWithPassword,
  loginWithEmailCode,
  forgotPassword,
} from '@/api/auth.ts'
import { encryptPasswordToString } from '@/utils/passwordEncrypt.ts'
import { getSystemInit } from '@/api/system.ts'
import { BehaviorTracker } from '@/utils/behaviorTracker.ts'

export type AuthMode = 'login' | 'register' | 'forgot-password'
export type LoginType = 'email' | 'password'

export interface AuthFormData {
  email: string
  password: string
  confirmPassword: string
  userName: string
  displayName: string
  verificationCode: string
}

const emptyFormData = (): AuthFormData => ({
  email: '',
  password: '',
  confirmPassword: '',
  userName: '',
  displayName: '',
  verificationCode: '',
})

export function useAuthModalLogic() {
  const { isOpen, mode: storeMode, nextPath, required, close } = useAuthModalStore()
  const { login, updateProfile: updateAuthStore } = useAuthStore()
  const navigate = useNavigate()

  const [mode, setMode] = useState<AuthMode>('login')
  const [loginType, setLoginType] = useState<LoginType>('email')
  const [isLoading, setIsLoading] = useState(false)
  const [isSendingCode, setIsSendingCode] = useState(false)
  const [countdown, setCountdown] = useState(0)
  const [isRegisterSuccess, setIsRegisterSuccess] = useState(false)
  const [registerSuccessData, setRegisterSuccessData] = useState<any>(null)
  const [isLoginSuccess, setIsLoginSuccess] = useState(false)
  const [loginSuccessData, setLoginSuccessData] = useState<any>(null)
  const [showTwoFactorInput, setShowTwoFactorInput] = useState(false)
  const [twoFactorCode, setTwoFactorCode] = useState('')
  const [showCaptchaModal, setShowCaptchaModal] = useState(false)
  const [pendingAction, setPendingAction] = useState<AuthMode | null>(null)
  const [isForgotPasswordSuccess, setIsForgotPasswordSuccess] = useState(false)
  const [forgotPasswordEmail, setForgotPasswordEmail] = useState('')
  const [showDeviceVerification, setShowDeviceVerification] = useState(false)
  const [deviceVerificationData, setDeviceVerificationData] = useState({
    email: '',
    deviceId: '',
    message: '',
  })
  const [showMemoryDBWarning, setShowMemoryDBWarning] = useState(false)
  const [emailEnabled, setEmailEnabled] = useState(true)
  const [formData, setFormData] = useState<AuthFormData>(emptyFormData())

  const behaviorTrackerRef = useRef<BehaviorTracker | null>(null)
  const formRef = useRef<HTMLDivElement>(null)

  const resetForm = useCallback(() => {
    setFormData(emptyFormData())
    setCountdown(0)
    setIsRegisterSuccess(false)
    setRegisterSuccessData(null)
    setIsLoginSuccess(false)
    setLoginSuccessData(null)
    setIsForgotPasswordSuccess(false)
    setForgotPasswordEmail('')
    setShowTwoFactorInput(false)
    setTwoFactorCode('')
  }, [])

  const handleCloseModal = useCallback(() => {
    if (required) return
    close()
    resetForm()
  }, [required, close, resetForm])

  const finishLoginSuccess = useCallback(() => {
    setTimeout(() => {
      if (nextPath) navigate(nextPath)
      close()
      resetForm()
    }, 1000)
  }, [nextPath, navigate, close, resetForm])

  useEffect(() => {
    if (isOpen) {
      setMode(storeMode === 'register' ? 'register' : 'login')
      setShowTwoFactorInput(false)
      setTwoFactorCode('')
    }
  }, [isOpen, storeMode])

  useEffect(() => {
    if (!isOpen) return
    getSystemInit()
      .then((res) => {
        if (res.code === 200 && res.data) {
          setEmailEnabled(res.data.email.configured)
          if (!res.data.email.configured) setLoginType('password')
          if (res.data.database.isMemoryDB && !localStorage.getItem('memoryDBWarningDismissed')) {
            setShowMemoryDBWarning(true)
          }
        }
      })
      .catch(() => setEmailEnabled(true))
  }, [isOpen])

  useEffect(() => {
    if (!isOpen) {
      behaviorTrackerRef.current?.stopTracking()
      behaviorTrackerRef.current = null
      return
    }
    behaviorTrackerRef.current = new BehaviorTracker()
    const timer = window.setTimeout(() => {
      if (behaviorTrackerRef.current && formRef.current) {
        behaviorTrackerRef.current.startTracking(formRef.current)
      }
    }, 500)
    return () => {
      clearTimeout(timer)
      behaviorTrackerRef.current?.stopTracking()
      behaviorTrackerRef.current = null
    }
  }, [isOpen])

  useEffect(() => {
    if (countdown <= 0) return
    const timer = window.setTimeout(() => setCountdown(countdown - 1), 1000)
    return () => clearTimeout(timer)
  }, [countdown])

  const handleDismissMemoryDBWarning = () => {
    setShowMemoryDBWarning(false)
    localStorage.setItem('memoryDBWarningDismissed', 'true')
  }

  const patchForm = (field: keyof AuthFormData, value: string) => {
    setFormData((prev) => ({ ...prev, [field]: value }))
  }

  const extractToken = (data: any) =>
    data?.token || data?.user?.token || data?.user?.authToken || data?.user?.AuthToken

  const completeLogin = async (responseData: any) => {
    const token = extractToken(responseData)
    if (!token) throw new Error('登录成功但未获取到认证令牌，请重试')
    const ok = await login(token)
    if (!ok) throw new Error('登录处理失败：无法获取用户信息')
    if (responseData.user) updateAuthStore(responseData.user)
    setLoginSuccessData(responseData)
    setIsLoginSuccess(true)
    const displayName =
      responseData.user?.displayName ||
      responseData.user?.DisplayName ||
      responseData.displayName ||
      formData.email
    showAlert(`欢迎回来，${displayName}！`, 'success', '登录成功')
    finishLoginSuccess()
  }

  const handleTwoFactorSubmit = async () => {
    if (!twoFactorCode.trim()) {
      showAlert('请输入两步验证码', 'error')
      return
    }
    setIsLoading(true)
    try {
      const encryptedPassword = await encryptPasswordToString(formData.password)
      const response = await loginWithPassword({
        email: formData.email,
        password: encryptedPassword,
        timezone: Intl.DateTimeFormat().resolvedOptions().timeZone,
        remember: true,
        authToken: true,
        twoFactorCode,
      })
      if (response.code === 200) {
        await completeLogin(response.data)
      } else {
        throw new Error(response.data?.message || response.msg || '登录失败')
      }
    } catch (error: any) {
      showAlert(error?.msg || error?.message || '登录失败', 'error', '登录失败')
    } finally {
      setIsLoading(false)
    }
  }

  const sendVerificationCode = async () => {
    if (!formData.email) {
      showAlert('请先输入邮箱', 'warning')
      return
    }
    if (!/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(formData.email)) {
      showAlert('请输入有效的邮箱地址', 'warning')
      return
    }
    setIsSendingCode(true)
    try {
      const response = await sendEmailCode({
        email: formData.email,
        clientIp: '',
        userAgent: navigator.userAgent,
      })
      if (response.code === 200) {
        showAlert('验证码已发送到您的邮箱，请在5分钟内验证', 'success', '发送成功')
        setCountdown(60)
      } else {
        throw new Error(response.msg || '验证码发送失败')
      }
    } catch (error: any) {
      let msg = error?.msg || error?.message || '验证码发送失败，请重试'
      if (error?.code === -1 && error?.msg?.includes('无法连接到服务器')) {
        msg = '无法连接到服务器，请检查后端服务是否已启动'
      }
      showAlert(msg, 'error', '发送失败')
    } finally {
      setIsSendingCode(false)
    }
  }

  const performForgotPassword = async () => {
    if (!formData.email || !/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(formData.email)) {
      showAlert('请输入有效的邮箱地址', 'warning')
      return
    }
    setIsLoading(true)
    try {
      const response = await forgotPassword(formData.email)
      if (response.code === 200) {
        setForgotPasswordEmail(formData.email)
        setIsForgotPasswordSuccess(true)
        showAlert('重置密码邮件已发送，请查收邮箱', 'success', '发送成功')
      } else {
        throw new Error(response.msg || '发送失败')
      }
    } catch (error: any) {
      showAlert(error?.msg || error?.message || '发送重置密码邮件失败，请重试', 'error', '发送失败')
    } finally {
      setIsLoading(false)
    }
  }

  const captchaFields = (
    captchaId: string,
    captchaType: 'image' | 'click',
    captchaCode: string,
    captchaData?: string,
  ) => ({
    captchaId,
    captchaType,
    captchaCode,
    ...(captchaData ? { captchaData } : {}),
  })

  const performLogin = async (
    captchaId: string,
    captchaType: 'image' | 'click',
    captchaCode: string,
    captchaData?: string,
  ) => {
    setIsLoading(true)
    try {
      if (loginType === 'email') {
        if (!formData.verificationCode) {
          showAlert('请输入验证码', 'warning')
          return
        }
        const response = await loginWithEmailCode({
          email: formData.email,
          code: formData.verificationCode,
          timezone: Intl.DateTimeFormat().resolvedOptions().timeZone,
          remember: true,
          authToken: true,
          ...captchaFields(captchaId, captchaType, captchaCode, captchaData),
        })
        if (response.code === 200) {
          await completeLogin(response.data)
        } else {
          throw new Error(response.data?.message || response.msg || '登录失败')
        }
        return
      }

      if (!formData.password) {
        showAlert('请输入密码', 'warning')
        return
      }
      const encryptedPassword = await encryptPasswordToString(formData.password)
      const response = await loginWithPassword({
        email: formData.email,
        password: encryptedPassword,
        timezone: Intl.DateTimeFormat().resolvedOptions().timeZone,
        remember: true,
        authToken: true,
        ...captchaFields(captchaId, captchaType, captchaCode, captchaData),
      })
      if (response.code !== 200) {
        throw new Error(response.data?.message || response.msg || '登录失败')
      }
      if (response.data.requiresEmailVerification) {
        showAlert('密码登录次数过多，请使用邮箱验证码登录', 'warning', '需要邮箱验证')
        setLoginType('email')
        return
      }
      if (response.data.requiresDeviceVerification) {
        setShowDeviceVerification(true)
        setDeviceVerificationData({
          email: formData.email,
          deviceId: response.data.deviceId || '',
          message: response.data.message || '此设备不受信任，需要验证',
        })
        showAlert('检测到新设备登录，请验证设备', 'warning', '需要设备验证')
        return
      }
      if (response.data.requiresTwoFactor) {
        setShowTwoFactorInput(true)
        showAlert('请输入两步验证码', 'info', '需要两步验证')
        return
      }
      await completeLogin(response.data)
    } catch (error: any) {
      showAlert(error?.msg || error?.message || '登录失败', 'error', '登录失败')
    } finally {
      setIsLoading(false)
    }
  }

  const performRegister = async (
    captchaId: string,
    captchaType: 'image' | 'click',
    captchaCode: string,
    captchaData?: string,
  ) => {
    setIsLoading(true)
    try {
      if (formData.password !== formData.confirmPassword) {
        showAlert('密码不匹配', 'warning')
        return
      }
      if (!formData.displayName) {
        showAlert('请输入显示名', 'warning')
        return
      }

      let behaviorData = null
      if (behaviorTrackerRef.current) {
        behaviorData = behaviorTrackerRef.current.getBehaviorData({
          email: formData.email,
          displayName: formData.displayName,
          userName: formData.userName,
        })
      }

      let response
      if (emailEnabled) {
        if (!formData.verificationCode) {
          showAlert('请输入验证码', 'warning')
          return
        }
        if (!formData.userName) {
          showAlert('请输入用户名', 'warning')
          return
        }
        const encryptedPassword = await encryptPasswordToString(formData.password)
          response = await registerUserByEmail({
            email: formData.email,
            password: encryptedPassword,
            userName: formData.userName,
            displayName: formData.displayName,
            code: formData.verificationCode,
            firstName: formData.userName.split(' ')[0] || formData.userName,
            lastName: formData.userName.split(' ')[1] || '',
            locale: 'zh-CN',
            timezone: Intl.DateTimeFormat().resolvedOptions().timeZone,
            source: 'WEB',
            ...captchaFields(captchaId, captchaType, captchaCode, captchaData),
            mouseTrack: behaviorData ? JSON.stringify(behaviorData.mouseTrack) : '',
            formFillTime: behaviorData ? behaviorData.formFillTime : 0,
            keystrokePattern: behaviorData ? behaviorData.keystrokePattern : '',
          })
      } else {
        const encryptedPassword = await encryptPasswordToString(formData.password)
        response = await registerUser({
            email: formData.email,
            password: encryptedPassword,
            displayName: formData.displayName,
            ...captchaFields(captchaId, captchaType, captchaCode, captchaData),
            firstName: formData.userName?.split(' ')[0] || formData.displayName,
            lastName: formData.userName?.split(' ')[1] || '',
            locale: 'zh-CN',
            timezone: Intl.DateTimeFormat().resolvedOptions().timeZone,
            source: 'WEB',
            mouseTrack: behaviorData ? JSON.stringify(behaviorData.mouseTrack) : '',
            formFillTime: behaviorData ? behaviorData.formFillTime : 0,
            keystrokePattern: behaviorData ? behaviorData.keystrokePattern : '',
          })
      }

      const responseData = (response.data || response) as any
      if (response.code === 200 || (responseData && (responseData.email || responseData.activation !== undefined))) {
        if (response.code === 200 && response.data && (response.data as any).displayName) {
          setRegisterSuccessData(response.data)
          showAlert(
            `注册成功！欢迎 ${(response.data as any).displayName || (response.data as any).email}，您的账号已创建完成。`,
            'success',
            '注册完成',
          )
        } else {
          const registerData = responseData
          setRegisterSuccessData({
            email: registerData.email,
            displayName: registerData.email?.split('@')[0] || '用户',
            activation: registerData.activation || false,
          })
          const activationMsg = registerData.activation
            ? '您的账号已激活，可以立即使用。'
            : `激活邮件已发送至 ${registerData.email}，请查收并激活账号。`
          showAlert(`注册成功！${activationMsg}`, 'success', '注册完成')
        }
        setIsRegisterSuccess(true)
        setTimeout(() => {
          close()
          resetForm()
        }, 3000)
      } else {
        throw new Error(response.msg || '注册失败')
      }
    } catch (error: any) {
      showAlert(error?.msg || error?.message || '注册失败', 'error', '注册失败')
    } finally {
      setIsLoading(false)
    }
  }

  const handleCaptchaVerify = (
    captchaId: string,
    captchaType: 'image' | 'click',
    payload: string | Array<{ x: number; y: number }>,
  ) => {
    setShowCaptchaModal(false)
    const captchaCode = captchaType === 'image' ? String(payload) : ''
    const captchaData =
      captchaType === 'click' ? JSON.stringify(payload as Array<{ x: number; y: number }>) : undefined
    if (pendingAction === 'login') performLogin(captchaId, captchaType, captchaCode, captchaData)
    else if (pendingAction === 'register') performRegister(captchaId, captchaType, captchaCode, captchaData)
    else if (pendingAction === 'forgot-password') performForgotPassword()
    setPendingAction(null)
  }

  const handleSubmit = (e?: React.FormEvent) => {
    e?.preventDefault()
    setPendingAction(mode)
    setShowCaptchaModal(true)
  }

  const modalTitle =
    mode === 'login' ? '登录' : mode === 'register' ? '注册' : '忘记密码'

  const goToLoginMode = useCallback(() => {
    setIsRegisterSuccess(false)
    setRegisterSuccessData(null)
    setIsForgotPasswordSuccess(false)
    setForgotPasswordEmail('')
    setIsLoginSuccess(false)
    setLoginSuccessData(null)
    setShowTwoFactorInput(false)
    setTwoFactorCode('')
    setMode('login')
  }, [])

  const dismissRegisterSuccess = useCallback(() => {
    setIsRegisterSuccess(false)
    setRegisterSuccessData(null)
  }, [])

  return {
    isOpen,
    required,
    mode,
    setMode,
    loginType,
    setLoginType,
    isLoading,
    isSendingCode,
    countdown,
    isRegisterSuccess,
    registerSuccessData,
    isLoginSuccess,
    loginSuccessData,
    showTwoFactorInput,
    twoFactorCode,
    setTwoFactorCode,
    setShowTwoFactorInput,
    showCaptchaModal,
    setShowCaptchaModal,
    setPendingAction,
    isForgotPasswordSuccess,
    forgotPasswordEmail,
    showDeviceVerification,
    setShowDeviceVerification,
    deviceVerificationData,
    setDeviceVerificationData,
    showMemoryDBWarning,
    emailEnabled,
    formData,
    patchForm,
    formRef,
    modalTitle,
    handleCloseModal,
    handleDismissMemoryDBWarning,
    handleTwoFactorSubmit,
    sendVerificationCode,
    handleCaptchaVerify,
    handleSubmit,
    resetForm,
    close,
    nextPath,
    navigate,
    goToLoginMode,
    dismissRegisterSuccess,
    setPendingAction,
  }
}
