import { useState, useEffect } from 'react'
import { motion, AnimatePresence } from 'framer-motion'
import { useNavigate } from 'react-router-dom'
import { User, Mail, Lock, Eye, EyeOff, Shield, Clock, Globe, AlertTriangle, X } from 'lucide-react'
import Modal from '../UI/Modal'
import Button from '../UI/Button'
import Input from '../UI/Input'
import { useAuthStore } from '@/stores/authStore.ts'
import { showAlert } from '@/utils/notification.ts'
// TODO: 实现认证相关的API调用
// import { sendEmailCode, registerUserByEmail, registerUser, loginWithPassword, loginWithEmailCode } from '@/api/auth.ts'
// import { getSystemInit } from '@/api/system.ts'

interface AuthModalProps {
  isOpen: boolean
  onClose: () => void
  initialMode?: 'login' | 'register'
}

const AuthModal = ({ isOpen, onClose, initialMode = 'login' }: AuthModalProps) => {
  const [mode, setMode] = useState<'login' | 'register'>(initialMode)
  const [loginType, setLoginType] = useState<'email' | 'password'>('email') // 登录方式
  const [showPassword, setShowPassword] = useState(false)
  const [showConfirmPassword, setShowConfirmPassword] = useState(false)
  const [isLoading, setIsLoading] = useState(false)
  const [isSendingCode, setIsSendingCode] = useState(false)
  const [countdown, setCountdown] = useState(0)
  const [isRegisterSuccess, setIsRegisterSuccess] = useState(false)
  const [registerSuccessData, setRegisterSuccessData] = useState<any>(null)
  const [isLoginSuccess, setIsLoginSuccess] = useState(false)
  const [loginSuccessData, setLoginSuccessData] = useState<any>(null)
  const [showTwoFactorInput, setShowTwoFactorInput] = useState(false)
  const [twoFactorCode, setTwoFactorCode] = useState('')
  
  // 系统初始化信息
  const [showMemoryDBWarning, setShowMemoryDBWarning] = useState(false)
  const [emailEnabled, setEmailEnabled] = useState(true) // 默认启用邮箱登录

  const { login } = useAuthStore()
  const navigate = useNavigate()

  // 表单数据
  const [formData, setFormData] = useState({
    email: '',
    password: '',
    confirmPassword: '',
    userName: '',
    displayName: '',
    verificationCode: ''
  })

  // 获取系统初始化信息
  useEffect(() => {
    if (isOpen) {
      // TODO: 实现获取系统初始化信息的API调用
      // getSystemInit().then(res => {
      //   if (res.code === 200 && res.data) {
      //     setEmailEnabled(res.data.email.configured)
      //     
      //     // 如果没有配置邮箱，默认使用密码登录
      //     if (!res.data.email.configured) {
      //       setLoginType('password')
      //     }
      //     
      //     // 检查是否需要显示内存数据库警告
      //     if (res.data.database.isMemoryDB) {
      //       const warningDismissed = localStorage.getItem('memoryDBWarningDismissed')
      //       if (!warningDismissed) {
      //         setShowMemoryDBWarning(true)
      //       }
      //     }
      //   }
      // }).catch(err => {
      //   console.error('Failed to get system init info:', err)
      //   // 如果获取失败，默认启用邮箱登录
      //   setEmailEnabled(true)
      // })
      // 暂时默认启用邮箱登录
      setEmailEnabled(true)
    }
  }, [isOpen])

  // 倒计时效果
  useEffect(() => {
    let timer: number
    if (countdown > 0) {
      timer = window.setTimeout(() => setCountdown(countdown - 1), 1000)
    }
    return () => clearTimeout(timer)
  }, [countdown])
  
  // 关闭内存数据库警告
  const handleDismissMemoryDBWarning = () => {
    setShowMemoryDBWarning(false)
    localStorage.setItem('memoryDBWarningDismissed', 'true')
  }

  const handleInputChange = (field: string, value: string) => {
    setFormData(prev => ({
      ...prev,
      [field]: value
    }))
  }

  // 处理两步验证码提交
  const handleTwoFactorSubmit = async () => {
    if (!twoFactorCode.trim()) {
      showAlert('请输入两步验证码', 'error', '验证失败')
      return
    }

    setIsLoading(true)
    try {
      const response = await loginWithPassword({
        email: formData.email,
        password: formData.password,
        timezone: Intl.DateTimeFormat().resolvedOptions().timeZone,
        remember: true,
        authToken: true,
        twoFactorCode: twoFactorCode
      })
      
      if (response.code === 200) {
        // 使用authStore的login方法处理登录成功
        const loginSuccess = await login(response.data.token)
        if (loginSuccess) {
          setLoginSuccessData(response.data)
          setIsLoginSuccess(true)
          showAlert(`欢迎回来，${response.data.displayName}！`, 'success', '登录成功')
          
          // 检查是否有重定向路径
          const redirectPath = localStorage.getItem('redirectAfterLogin')
          if (redirectPath) {
            localStorage.removeItem('redirectAfterLogin')
            setTimeout(() => {
              navigate(redirectPath)
              onClose()
            }, 1000)
          } else {
            setTimeout(() => {
              onClose()
            }, 1000)
          }
        } else {
          throw new Error('登录处理失败')
        }
      } else {
        throw new Error(response.msg || '登录失败')
      }
    } catch (error: any) {
      showAlert(error?.msg || error?.message || '登录失败', 'error', '登录失败')
    } finally {
      setIsLoading(false)
    }
  }

  // 发送验证码
  const sendVerificationCode = async () => {
    if (!formData.email) {
      showAlert('请先输入邮箱', 'error', '验证失败')
      return
    }

    const emailRegex = /^[^\s@]+@[^\s@]+\.[^\s@]+$/
    if (!emailRegex.test(formData.email)) {
      showAlert('请输入有效的邮箱地址', 'error', '验证失败')
      return
    }

    setIsSendingCode(true)
    try {
      // 调用发送验证码的API
      const response = await sendEmailCode({
        email: formData.email,
        clientIp: '', // 由后端自动获取
        userAgent: navigator.userAgent
      })
      
      if (response.code === 200) {
        showAlert('验证码已发送到您的邮箱，请在5分钟内验证', 'success', '发送成功')
        setCountdown(60) // 60秒倒计时
      } else {
        throw new Error(response.msg || '验证码发送失败')
      }
    } catch (error: any) {
      console.error('Send code error:', error)
      let errorMessage = error?.msg || error?.message || '验证码发送失败，请重试'
      
      // 特殊处理网络连接错误
      if (error?.code === -1 && error?.msg?.includes('无法连接到服务器')) {
        errorMessage = '无法连接到服务器，请检查后端服务是否已启动'
      }
      
      showAlert(errorMessage, 'error', '发送失败')
    } finally {
      setIsSendingCode(false)
    }
  }

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setIsLoading(true)

    try {
      if (mode === 'login') {
        if (loginType === 'email') {
          // 邮箱验证码登录
          if (!formData.verificationCode) {
            showAlert('请输入验证码', 'error', '验证失败')
            return
          }
          
          const response = await loginWithEmailCode({
            email: formData.email,
            code: formData.verificationCode,
            timezone: Intl.DateTimeFormat().resolvedOptions().timeZone,
            remember: true,
            authToken: true
          })
          
          if (response.code === 200) {
            // 使用authStore的login方法处理登录成功
            const loginSuccess = await login(response.data.token)
            if (loginSuccess) {
              setLoginSuccessData(response.data)
              setIsLoginSuccess(true)
              showAlert(`欢迎回来，${response.data.displayName}！`, 'success', '登录成功')
              
              // 检查是否有重定向路径
              const redirectPath = localStorage.getItem('redirectAfterLogin')
              if (redirectPath) {
                localStorage.removeItem('redirectAfterLogin')
                setTimeout(() => {
                  navigate(redirectPath)
                  onClose()
                }, 1000)
              } else {
                setTimeout(() => {
                  onClose()
                }, 1000)
              }
            } else {
              throw new Error('登录处理失败')
            }
          } else {
            throw new Error(response.msg || '登录失败')
          }
        } else {
          // 密码登录
          if (!formData.password) {
            showAlert('请输入密码', 'error', '验证失败')
            return
          }
          
          const response = await loginWithPassword({
            email: formData.email,
            password: formData.password,
            timezone: Intl.DateTimeFormat().resolvedOptions().timeZone,
            remember: true,
            authToken: true
          })
          
          if (response.code === 200) {
            // 检查是否需要两步验证
            if (response.data.requiresTwoFactor) {
              setShowTwoFactorInput(true)
              showAlert('请输入两步验证码', 'info', '需要两步验证')
              return
            }
            
            // 调试：打印响应数据
            console.log('Login response:', response.data)
            console.log('Token from response:', response.data.token)
            
            // 使用authStore的login方法处理登录成功
            const loginSuccess = await login(response.data.token)
            if (loginSuccess) {
              setLoginSuccessData(response.data)
              setIsLoginSuccess(true)
              showAlert(`欢迎回来，${response.data.displayName}！`, 'success', '登录成功')
              
              // 检查是否有重定向路径
              const redirectPath = localStorage.getItem('redirectAfterLogin')
              if (redirectPath) {
                localStorage.removeItem('redirectAfterLogin')
                setTimeout(() => {
                  navigate(redirectPath)
                  onClose()
                }, 1000)
              } else {
                setTimeout(() => {
                  onClose()
                }, 1000)
              }
            } else {
              throw new Error('登录处理失败')
            }
          } else {
            throw new Error(response.msg || '登录失败')
          }
        }
      } else {
        // 注册
        if (formData.password !== formData.confirmPassword) {
          showAlert('密码不匹配', 'error', '验证失败')
          return
        }
        if (!formData.displayName) {
          showAlert('请输入显示名', 'error', '验证失败')
          return
        }
        
        // 根据邮件配置状态选择注册方式
        let response
        if (emailEnabled) {
          // 如果配置了邮箱，使用邮箱验证码注册
          if (!formData.verificationCode) {
            showAlert('请输入验证码', 'error', '验证失败')
            return
          }
          if (!formData.userName) {
            showAlert('请输入用户名', 'error', '验证失败')
            return
          }
          
          response = await registerUserByEmail({
            email: formData.email,
            password: formData.password,
            userName: formData.userName,
            displayName: formData.displayName,
            code: formData.verificationCode,
            firstName: formData.userName.split(' ')[0] || formData.userName,
            lastName: formData.userName.split(' ')[1] || '',
            locale: 'zh-CN',
            timezone: Intl.DateTimeFormat().resolvedOptions().timeZone,
            source: 'WEB'
          })
        } else {
          // 如果没有配置邮箱，使用普通注册（不需要验证码）
          response = await registerUser({
            email: formData.email,
            password: formData.password,
            displayName: formData.displayName,
            firstName: formData.userName?.split(' ')[0] || formData.displayName,
            lastName: formData.userName?.split(' ')[1] || '',
            locale: 'zh-CN',
            timezone: Intl.DateTimeFormat().resolvedOptions().timeZone,
            source: 'WEB'
          })
        }
        
        // 注册成功处理
        // 后端可能返回两种格式：
        // 1. 标准格式: {code: 200, msg: "...", data: {...}}
        // 2. 直接格式: {email: "...", activation: false} (HTTP 200 但响应体直接是对象)
        const responseData = (response.data || response) as any
        
        // 检查是否是成功响应
        if (response.code === 200 || (responseData && (responseData.email || responseData.activation !== undefined))) {
          // 处理标准格式
          if (response.code === 200 && response.data && (response.data as any).displayName) {
            setRegisterSuccessData(response.data as any)
            setIsRegisterSuccess(true)
            
            showAlert(
              `注册成功！欢迎 ${(response.data as any).displayName || (response.data as any).email}，您的账号已创建完成。`,
              'success', 
              '注册完成'
            )
          } else {
            // 处理直接格式 {email, activation}
            const registerData = responseData
            setRegisterSuccessData({
              email: registerData.email,
              displayName: registerData.email?.split('@')[0] || '用户',
              activation: registerData.activation || false
            })
            setIsRegisterSuccess(true)
            
            const activationMsg = registerData.activation 
              ? '您的账号已激活，可以立即使用。'
              : '您的账号已创建，请等待管理员激活。'
            
            showAlert(
              `注册成功！${activationMsg}`,
              'success', 
              '注册完成'
            )
          }
        } else {
          throw new Error(response.msg || (response as any).error || '注册失败')
        }
      }
    } catch (error: any) {
      console.error('Auth error:', error)
      const errorMessage = error?.msg || error?.message || (mode === 'login' ? '登录失败' : '注册失败')
      showAlert(errorMessage, 'error', '操作失败')
    } finally {
      setIsLoading(false)
    }
  }

  const resetForm = () => {
    setFormData({
      email: '',
      password: '',
      confirmPassword: '',
      userName: '',
      displayName: '',
      verificationCode: ''
    })
    setCountdown(0)
    setIsRegisterSuccess(false)
    setRegisterSuccessData(null)
    setIsLoginSuccess(false)
    setLoginSuccessData(null)
  }

  const switchMode = () => {
    setMode(mode === 'login' ? 'register' : 'login')
    resetForm()
  }

  return (
    <Modal
      className="z-20"
      isOpen={isOpen}
      onClose={() => {
        onClose()
        resetForm()
      }}
      size="md"
      title={mode === 'login' ? '登录' : '注册'}
    >
      {/* 注册成功状态显示 */}
      {isRegisterSuccess && registerSuccessData && (
        <motion.div
          initial={{ opacity: 0, scale: 0.9 }}
          animate={{ opacity: 1, scale: 1 }}
          className="text-center py-8"
        >
          <motion.div
            initial={{ scale: 0 }}
            animate={{ scale: 1 }}
            transition={{ delay: 0.2, type: "spring", stiffness: 200 }}
            className="w-16 h-16 bg-green-100 dark:bg-green-900/20 rounded-full flex items-center justify-center mx-auto mb-4"
          >
            <motion.div
              initial={{ pathLength: 0 }}
              animate={{ pathLength: 1 }}
              transition={{ delay: 0.4, duration: 0.6 }}
              className="w-8 h-8 text-green-600 dark:text-green-400"
            >
              <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
                <path d="M9 12l2 2 4-4" />
                <circle cx="12" cy="12" r="10" />
              </svg>
            </motion.div>
          </motion.div>
          
          <motion.h3
            initial={{ opacity: 0, y: 20 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ delay: 0.3 }}
            className="text-xl font-semibold text-gray-900 dark:text-white mb-2"
          >
            注册成功！
          </motion.h3>
          
          <motion.p
            initial={{ opacity: 0, y: 20 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ delay: 0.4 }}
            className="text-gray-600 dark:text-gray-400 mb-6"
          >
            欢迎 <span className="font-medium text-primary">{registerSuccessData.displayName}</span>，
            您的账号已创建完成！
          </motion.p>
          
          <motion.div
            initial={{ opacity: 0, y: 20 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ delay: 0.5 }}
            className="space-y-3"
          >
            <div className="text-sm text-gray-500 dark:text-gray-400 space-y-1">
              <p className="flex items-center">
                <Mail className="w-4 h-4 mr-2" />
                {registerSuccessData.email}
              </p>
              <p className="flex items-center">
                <User className="w-4 h-4 mr-2" />
                {registerSuccessData.displayName}
              </p>
              <p className="flex items-center">
                <Globe className="w-4 h-4 mr-2" />
                {registerSuccessData.timezone}
              </p>
            </div>
            
            <div className="flex space-x-3 pt-4">
              <Button
                variant="outline"
                onClick={() => {
                  setIsRegisterSuccess(false)
                  setRegisterSuccessData(null)
                }}
                className="flex-1"
              >
                继续注册
              </Button>
              <Button
                variant="primary"
                onClick={() => {
                  onClose()
                  resetForm()
                  setMode('login')
                  showAlert('请使用您的邮箱和密码登录', 'info', '提示')
                }}
                className="flex-1"
              >
                立即登录
              </Button>
            </div>
          </motion.div>
        </motion.div>
      )}
      
      {/* 登录成功状态显示 */}
      {isLoginSuccess && loginSuccessData && (
        <motion.div
          initial={{ opacity: 0, scale: 0.9 }}
          animate={{ opacity: 1, scale: 1 }}
          className="text-center py-8"
        >
          <motion.div
            initial={{ scale: 0 }}
            animate={{ scale: 1 }}
            transition={{ delay: 0.2, type: "spring", stiffness: 200 }}
            className="w-16 h-16 bg-green-100 dark:bg-green-900/20 rounded-full flex items-center justify-center mx-auto mb-4"
          >
            <motion.div
              initial={{ pathLength: 0 }}
              animate={{ pathLength: 1 }}
              transition={{ delay: 0.4, duration: 0.6 }}
              className="w-8 h-8 text-green-600 dark:text-green-400"
            >
              <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
                <path d="M9 12l2 2 4-4" />
                <circle cx="12" cy="12" r="10" />
              </svg>
            </motion.div>
          </motion.div>
          
          <motion.h3
            initial={{ opacity: 0, y: 20 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ delay: 0.3 }}
            className="text-xl font-semibold text-gray-900 dark:text-white mb-2"
          >
            登录成功！
          </motion.h3>
          
          <motion.p
            initial={{ opacity: 0, y: 20 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ delay: 0.4 }}
            className="text-gray-600 dark:text-gray-400 mb-6"
          >
            欢迎回来，<span className="font-medium text-primary">{loginSuccessData.displayName}</span>！
          </motion.p>
          
          <motion.div
            initial={{ opacity: 0, y: 20 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ delay: 0.5 }}
            className="space-y-3"
          >
            <div className="text-sm text-gray-500 dark:text-gray-400 space-y-2 text-center">
              <div className="flex items-center justify-center gap-2">
                <Mail className="w-4 h-4" />
                <span>邮箱：{loginSuccessData.email}</span>
              </div>
              <div className="flex items-center justify-center gap-2">
                <User className="w-4 h-4" />
                <span>显示名：{loginSuccessData.displayName}</span>
              </div>
              <div className="flex items-center justify-center gap-2">
                <Clock className="w-4 h-4" />
                <span>最后登录：{new Date(loginSuccessData.lastLogin).toLocaleString()}</span>
              </div>
            </div>
            
            <div className="pt-4">
              <Button
                variant="primary"
                onClick={() => {
                  onClose()
                  resetForm()
                }}
                className="w-full"
              >
                进入应用
              </Button>
            </div>
          </motion.div>
        </motion.div>
      )}
      
      {/* 内存数据库警告提示 */}
      <AnimatePresence>
        {showMemoryDBWarning && (
          <motion.div
            initial={{ opacity: 0, y: -10 }}
            animate={{ opacity: 1, y: 0 }}
            exit={{ opacity: 0, y: -10 }}
            className="mb-4 p-3 bg-yellow-50 dark:bg-yellow-900/20 border border-yellow-200 dark:border-yellow-800 rounded-lg flex items-start gap-3"
          >
            <AlertTriangle className="w-5 h-5 text-yellow-600 dark:text-yellow-400 flex-shrink-0 mt-0.5" />
            <div className="flex-1 text-sm text-yellow-800 dark:text-yellow-200">
              <p className="font-medium mb-1">检测到您目前使用的是内存数据库</p>
              <p className="text-xs">数据可能会丢失，如果您有需要请配置持久化数据库（如 MySQL 或 PostgreSQL）。</p>
            </div>
            <button
              type="button"
              onClick={handleDismissMemoryDBWarning}
              className="flex-shrink-0 text-yellow-600 dark:text-yellow-400 hover:text-yellow-800 dark:hover:text-yellow-200 transition-colors"
            >
              <X className="w-4 h-4" />
            </button>
          </motion.div>
        )}
      </AnimatePresence>

      {/* 正常表单显示 */}
      {!isRegisterSuccess && !isLoginSuccess && (
        <form onSubmit={handleSubmit} className="space-y-6">
        {/* 登录表单 */}
        {mode === 'login' && (
          <motion.div
            key="login"
            initial={{ opacity: 0, x: -20 }}
            animate={{ opacity: 1, x: 0 }}
            exit={{ opacity: 0, x: 20 }}
            transition={{ duration: 0.3 }}
            className="space-y-4"
          >
            {/* 登录方式切换 - 仅在邮箱配置启用时显示 */}
            {emailEnabled && (
              <div className="flex bg-gray-100 dark:bg-gray-800 rounded-lg p-1">
                <button
                  type="button"
                  onClick={() => setLoginType('email')}
                  className={`flex-1 flex items-center justify-center space-x-2 py-2 px-4 rounded-md text-sm font-medium transition-colors ${
                    loginType === 'email'
                      ? 'bg-white dark:bg-gray-700 text-gray-900 dark:text-white shadow-sm'
                      : 'text-gray-600 dark:text-gray-400 hover:text-gray-900 dark:hover:text-white'
                  }`}
                >
                  <Mail className="w-4 h-4" />
                  <span>邮箱验证码</span>
                </button>
                <button
                  type="button"
                  onClick={() => setLoginType('password')}
                  className={`flex-1 flex items-center justify-center space-x-2 py-2 px-4 rounded-md text-sm font-medium transition-colors ${
                    loginType === 'password'
                      ? 'bg-white dark:bg-gray-700 text-gray-900 dark:text-white shadow-sm'
                      : 'text-gray-600 dark:text-gray-400 hover:text-gray-900 dark:hover:text-white'
                  }`}
                >
                  <Lock className="w-4 h-4" />
                  <span>密码登录</span>
                </button>
              </div>
            )}

            <Input
              label="邮箱"
              type="email"
              placeholder="请输入邮箱"
              value={formData.email}
              onChange={(e) => handleInputChange('email', e.target.value)}
              leftIcon={<Mail className="w-5 h-5" />}
              required
            />

            {loginType === 'email' ? (
              <div className="space-y-3">
                <Input
                  label="验证码"
                  placeholder="请输入验证码"
                  value={formData.verificationCode}
                  onChange={(e) => handleInputChange('verificationCode', e.target.value)}
                  leftIcon={<Shield className="w-5 h-5" />}
                  rightIcon={
                    <motion.button
                      type="button"
                      onClick={sendVerificationCode}
                      disabled={isSendingCode || countdown > 0}
                      className="text-primary hover:text-primary/80 disabled:text-gray-400 disabled:cursor-not-allowed text-sm font-medium transition-all duration-300 relative group"
                      whileHover={!isSendingCode && countdown === 0 ? { scale: 1.05 } : {}}
                      whileTap={!isSendingCode && countdown === 0 ? { scale: 0.95 } : {}}
                    >
                      {/* 背景光效 */}
                      <motion.div
                        className="absolute inset-0 bg-primary/10 rounded-md opacity-0 group-hover:opacity-100"
                        transition={{ duration: 0.2 }}
                      />
                      
                      {/* 按钮内容 */}
                      <div className="relative z-10 flex items-center space-x-1">
                        {isSendingCode ? (
                          <motion.div
                            className="flex items-center space-x-1"
                            initial={{ opacity: 0 }}
                            animate={{ opacity: 1 }}
                            transition={{ duration: 0.2 }}
                          >
                            <motion.div
                              animate={{ rotate: 360 }}
                              transition={{ duration: 1, repeat: Infinity, ease: 'linear' }}
                              className="w-4 h-4 border-2 border-primary border-t-transparent rounded-full"
                            />
                            <span>发送中</span>
                          </motion.div>
                        ) : countdown > 0 ? (
                          <motion.div
                            className="flex items-center space-x-1"
                            initial={{ opacity: 0, scale: 0.8 }}
                            animate={{ opacity: 1, scale: 1 }}
                            transition={{ duration: 0.2 }}
                          >
                            <motion.div
                              animate={{ rotate: [0, 360] }}
                              transition={{ duration: 2, repeat: Infinity, ease: 'linear' }}
                            >
                              <Clock className="w-4 h-4" />
                            </motion.div>
                            <motion.span
                              key={countdown}
                              initial={{ scale: 1.2 }}
                              animate={{ scale: 1 }}
                              transition={{ duration: 0.2 }}
                            >
                              {countdown}s
                            </motion.span>
                          </motion.div>
                        ) : (
                          <motion.div
                            className="flex items-center space-x-1"
                            initial={{ opacity: 0, x: 10 }}
                            animate={{ opacity: 1, x: 0 }}
                            transition={{ duration: 0.3 }}
                          >
                            <motion.span
                              animate={{
                                scale: [1, 1.1, 1],
                              }}
                              transition={{
                                duration: 1.5,
                                repeat: Infinity,
                                ease: "easeInOut"
                              }}
                            >
                              发送验证码
                            </motion.span>
                          </motion.div>
                        )}
                      </div>
                    </motion.button>
                  }
                  required
                />
              </div>
            ) : (
              <Input
                label="密码"
                type={showPassword ? 'text' : 'password'}
                placeholder="请输入密码"
                value={formData.password}
                onChange={(e) => handleInputChange('password', e.target.value)}
                leftIcon={<Lock className="w-5 h-5" />}
                rightIcon={
                  <button
                    type="button"
                    onClick={() => setShowPassword(!showPassword)}
                    className="text-neutral-400 hover:text-neutral-600 dark:hover:text-neutral-300"
                  >
                    {showPassword ? <EyeOff className="w-5 h-5" /> : <Eye className="w-5 h-5" />}
                  </button>
                }
                required
              />
            )}

            {/* 两步验证码输入框 */}
            {showTwoFactorInput && (
              <motion.div
                initial={{ opacity: 0, y: 20 }}
                animate={{ opacity: 1, y: 0 }}
                exit={{ opacity: 0, y: -20 }}
                className="space-y-3"
              >
                <Input
                  label="两步验证码"
                  placeholder="请输入两步验证码"
                  value={twoFactorCode}
                  onChange={(e) => setTwoFactorCode(e.target.value)}
                  leftIcon={<Shield className="w-5 h-5" />}
                  required
                />
                <Button
                  type="button"
                  variant="primary"
                  onClick={handleTwoFactorSubmit}
                  disabled={isLoading}
                  className="w-full"
                >
                  {isLoading ? '验证中...' : '验证登录'}
                </Button>
                <Button
                  type="button"
                  variant="secondary"
                  onClick={() => setShowTwoFactorInput(false)}
                  className="w-full"
                >
                  返回
                </Button>
              </motion.div>
            )}
          </motion.div>
        )}

        {/* 注册表单 */}
        {mode === 'register' && (
          <motion.div
            key="register"
            initial={{ opacity: 0, x: -20 }}
            animate={{ opacity: 1, x: 0 }}
            exit={{ opacity: 0, x: 20 }}
            transition={{ duration: 0.3 }}
            className="space-y-4"
          >
                  <div className={`grid gap-4 ${emailEnabled ? 'grid-cols-2' : 'grid-cols-1'}`}>
                    {emailEnabled && (
                      <Input
                        label="用户名"
                        placeholder="请输入用户名"
                        value={formData.userName}
                        onChange={(e) => handleInputChange('userName', e.target.value)}
                        leftIcon={<User className="w-5 h-5" />}
                        required
                      />
                    )}
                    <Input
                      label="显示名"
                      placeholder="请输入显示名"
                      value={formData.displayName}
                      onChange={(e) => handleInputChange('displayName', e.target.value)}
                      leftIcon={<User className="w-5 h-5" />}
                      required
                    />
                  </div>

            <Input
              label="邮箱"
              type="email"
              placeholder="请输入邮箱"
              value={formData.email}
              onChange={(e) => handleInputChange('email', e.target.value)}
              leftIcon={<Mail className="w-5 h-5" />}
              required
            />

            {/* 验证码输入 - 仅在配置了邮箱时显示 */}
            {emailEnabled && (
              <Input
                label="验证码"
                placeholder="请输入验证码"
                value={formData.verificationCode}
                onChange={(e) => handleInputChange('verificationCode', e.target.value)}
                leftIcon={<Shield className="w-5 h-5" />}
                rightIcon={
                  <motion.button
                    type="button"
                    onClick={sendVerificationCode}
                    disabled={isSendingCode || countdown > 0}
                    className="text-primary hover:text-primary/80 disabled:text-gray-400 disabled:cursor-not-allowed text-sm font-medium transition-all duration-300 relative group"
                    whileHover={!isSendingCode && countdown === 0 ? { scale: 1.05 } : {}}
                    whileTap={!isSendingCode && countdown === 0 ? { scale: 0.95 } : {}}
                  >
                    {/* 背景光效 */}
                    <motion.div
                      className="absolute inset-0 bg-primary/10 rounded-md opacity-0 group-hover:opacity-100"
                      transition={{ duration: 0.2 }}
                    />
                    
                    {/* 按钮内容 */}
                    <div className="relative z-10 flex items-center space-x-1">
                      {isSendingCode ? (
                        <motion.div
                          className="flex items-center space-x-1"
                          initial={{ opacity: 0 }}
                          animate={{ opacity: 1 }}
                          transition={{ duration: 0.2 }}
                        >
                          <motion.div
                            animate={{ rotate: 360 }}
                            transition={{ duration: 1, repeat: Infinity, ease: 'linear' }}
                            className="w-4 h-4 border-2 border-primary border-t-transparent rounded-full"
                          />
                          <span>发送中</span>
                        </motion.div>
                      ) : countdown > 0 ? (
                        <motion.div
                          className="flex items-center space-x-1"
                          initial={{ opacity: 0, scale: 0.8 }}
                          animate={{ opacity: 1, scale: 1 }}
                          transition={{ duration: 0.2 }}
                        >
                          <motion.div
                            animate={{ rotate: [0, 360] }}
                            transition={{ duration: 2, repeat: Infinity, ease: 'linear' }}
                          >
                            <Clock className="w-4 h-4" />
                          </motion.div>
                          <motion.span
                            key={countdown}
                            initial={{ scale: 1.2 }}
                            animate={{ scale: 1 }}
                            transition={{ duration: 0.2 }}
                          >
                            {countdown}s
                          </motion.span>
                        </motion.div>
                      ) : (
                        <motion.div
                          className="flex items-center space-x-1"
                          initial={{ opacity: 0, x: 10 }}
                          animate={{ opacity: 1, x: 0 }}
                          transition={{ duration: 0.3 }}
                        >
                          <motion.span
                            animate={{
                              scale: [1, 1.1, 1],
                            }}
                            transition={{
                              duration: 1.5,
                              repeat: Infinity,
                              ease: "easeInOut"
                            }}
                          >
                            发送验证码
                          </motion.span>
                        </motion.div>
                      )}
                    </div>
                  </motion.button>
                }
                required
              />
            )}

            <Input
              label="密码"
              type={showPassword ? 'text' : 'password'}
              placeholder="请输入密码"
              value={formData.password}
              onChange={(e) => handleInputChange('password', e.target.value)}
              leftIcon={<Lock className="w-5 h-5" />}
              rightIcon={
                <button
                  type="button"
                  onClick={() => setShowPassword(!showPassword)}
                  className="text-neutral-400 hover:text-neutral-600 dark:hover:text-neutral-300"
                >
                  {showPassword ? <EyeOff className="w-5 h-5" /> : <Eye className="w-5 h-5" />}
                </button>
              }
              required
            />

            <Input
              label="确认密码"
              type={showConfirmPassword ? 'text' : 'password'}
              placeholder="请再次输入密码"
              value={formData.confirmPassword}
              onChange={(e) => handleInputChange('confirmPassword', e.target.value)}
              leftIcon={<Lock className="w-5 h-5" />}
              rightIcon={
                <button
                  type="button"
                  onClick={() => setShowConfirmPassword(!showConfirmPassword)}
                  className="text-neutral-400 hover:text-neutral-600 dark:hover:text-neutral-300"
                >
                  {showConfirmPassword ? <EyeOff className="w-5 h-5" /> : <Eye className="w-5 h-5" />}
                </button>
              }
              required
            />
          </motion.div>
        )}

        {/* 提交按钮 */}
        <div className="space-y-4">
          <motion.div
            initial={{ opacity: 0, y: 20 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ duration: 0.5, delay: 0.2 }}
          >
            <Button
              type="submit"
              variant="primary"
              animation="pulse"
              className="w-full relative overflow-hidden group"
              disabled={isLoading}
            >
              {/* 背景渐变效果 */}
              <motion.div
                className="absolute inset-0 bg-gradient-to-r from-blue-600 via-purple-600 to-blue-600 opacity-0 group-hover:opacity-100"
                initial={{ opacity: 0 }}
                whileHover={{ opacity: 1 }}
                transition={{ duration: 0.3 }}
              />
              
              {/* 脉冲光效 */}
              <motion.div
                className="absolute inset-0 bg-white/20 rounded-lg"
                animate={{
                  scale: [1, 1.1, 1],
                  opacity: [0.3, 0.6, 0.3],
                }}
                transition={{
                  duration: 2,
                  repeat: Infinity,
                  ease: "easeInOut"
                }}
              />
              
              {/* 按钮内容 */}
              <div className="relative z-10 flex items-center justify-center space-x-2">
                {isLoading ? (
                  <>
                    <motion.div
                      animate={{ rotate: 360 }}
                      transition={{ duration: 1, repeat: Infinity, ease: 'linear' }}
                      className="w-4 h-4 border-2 border-white border-t-transparent rounded-full"
                    />
                    <span>处理中...</span>
                  </>
                ) : (
                  <>
                    <motion.span
                      initial={{ opacity: 0, x: -10 }}
                      animate={{ opacity: 1, x: 0 }}
                      transition={{ duration: 0.3 }}
                    >
                      {mode === 'login' ? '登录' : '注册'}
                    </motion.span>
                    <motion.div
                      initial={{ opacity: 0, x: 10 }}
                      animate={{ opacity: 1, x: 0 }}
                      transition={{ duration: 0.3, delay: 0.1 }}
                      className="w-2 h-2 bg-white/80 rounded-full"
                    />
                  </>
                )}
              </div>
            </Button>
          </motion.div>

          {/* 切换模式 */}
          <motion.div
            initial={{ opacity: 0, y: 10 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ duration: 0.4, delay: 0.3 }}
            className="text-center"
          >
            <span className="text-sm text-neutral-600 dark:text-neutral-400">
              {mode === 'login' ? '还没有账号？' : '已有账号？'}
            </span>
            <motion.button
              type="button"
              onClick={switchMode}
              className="ml-2 text-sm text-primary hover:text-primary/80 font-medium transition-all duration-300 relative group"
              whileHover={{ scale: 1.05 }}
              whileTap={{ scale: 0.95 }}
            >
              {/* 下划线动画 */}
              <motion.div
                className="absolute bottom-0 left-0 w-0 h-0.5 bg-primary group-hover:w-full"
                transition={{ duration: 0.3, ease: "easeOut" }}
              />
              
              {/* 文字内容 */}
              <motion.span
                key={mode}
                initial={{ opacity: 0, y: 5 }}
                animate={{ opacity: 1, y: 0 }}
                exit={{ opacity: 0, y: -5 }}
                transition={{ duration: 0.2 }}
              >
                {mode === 'login' ? '立即注册' : '立即登录'}
              </motion.span>
            </motion.button>
          </motion.div>
        </div>
      </form>
      )}
    </Modal>
  )
}

export default AuthModal
