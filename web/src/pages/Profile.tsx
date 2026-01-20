import { useState, useEffect } from 'react'
import { 
  User, Mail, Shield, Camera, Save, Edit3, X, Lock, Eye, EyeOff, 
  Clock, Phone, Settings, Bell, Key, Heart, 
  CheckCircle, AlertCircle, Zap
} from 'lucide-react'
import { useAuthStore } from '../stores/authStore'
import { useI18nStore } from '../stores/i18nStore'
import Button from '../components/UI/Button'
import Input from '../components/UI/Input'
import Card from '../components/UI/Card'
import Badge from '../components/UI/Badge'
import Switch from '../components/UI/Switch'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '../components/UI/Tabs'
import FadeIn from '../components/Animations/FadeIn'
import LoadingAnimation from '../components/Animations/LoadingAnimation'
import { showAlert } from '../utils/notification'
import { getProfile, updateProfile, updatePreferences, changePassword, changePasswordByEmail, uploadAvatar, setupTwoFactor, enableTwoFactor, disableTwoFactor, getUserActivity, getUserDevices, deleteUserDevice, trustUserDevice, untrustUserDevice, TwoFactorSetupResponse, ActivityLog, UserDevice } from '../api/profile'
import { sendEmailCode, sendEmailVerification } from '../api/auth'
import { motion, AnimatePresence } from 'framer-motion'
import ConfirmDialog from '../components/UI/ConfirmDialog'
import AudioController from '../components/UI/AudioController'
import AuthModal from '../components/Auth/AuthModal'

const Profile = () => {
  const { user, isAuthenticated, updateProfile: updateAuthStore } = useAuthStore()
  const { t } = useI18nStore()
  const [isEditing, setIsEditing] = useState(false)
  const [isChangingPassword, setIsChangingPassword] = useState(false)
  const [isLoading, setIsLoading] = useState(false)
  const [isPageLoading, setIsPageLoading] = useState(true)
  const [showCurrentPassword, setShowCurrentPassword] = useState(false)
  const [showNewPassword, setShowNewPassword] = useState(false)
  const [showConfirmPassword, setShowConfirmPassword] = useState(false)
  const [activeTab, setActiveTab] = useState('profile')
  
  // 两步验证相关状态
  const [twoFactorSetup, setTwoFactorSetup] = useState<TwoFactorSetupResponse | null>(null)
  const [twoFactorCode, setTwoFactorCode] = useState('')
  const [isTwoFactorLoading, setIsTwoFactorLoading] = useState(false)
  const [showTwoFactorSetup, setShowTwoFactorSetup] = useState(false)
  const [showTwoFactorDisable, setShowTwoFactorDisable] = useState(false)

  // 活动记录相关状态
  const [activities, setActivities] = useState<ActivityLog[]>([])
  const [isLoadingActivities, setIsLoadingActivities] = useState(false)
  const [activityPage, setActivityPage] = useState(1)
  const [activityTotalPages, setActivityTotalPages] = useState(1)
  
  // 设备管理相关状态
  const [devices, setDevices] = useState<UserDevice[]>([])
  const [isLoadingDevices, setIsLoadingDevices] = useState(false)
  
  // 确认对话框状态
  const [confirmDialog, setConfirmDialog] = useState<{
    isOpen: boolean
    title: string
    message: string
    onConfirm: () => void
    type?: 'warning' | 'danger' | 'info' | 'success'
  }>({
    isOpen: false,
    title: '',
    message: '',
    onConfirm: () => {},
    type: 'warning'
  })
  
  // 邮箱验证相关状态
  const [isSendingEmailVerification, setIsSendingEmailVerification] = useState(false)
  
  const [formData, setFormData] = useState({
    email: user?.email || '',
    phone: user?.phone || '',
    displayName: user?.displayName || '',
    firstName: user?.firstName || '',
    lastName: user?.lastName || '',
    locale: user?.locale || 'zh-CN',
    timezone: user?.timezone || 'Asia/Shanghai',
    gender: user?.gender || '',
    city: user?.city || '',
    region: user?.region || '',
    extra: user?.extra || '',
    avatar: user?.avatar || '',
  })

  const [passwordData, setPasswordData] = useState({
    currentPassword: '',
    newPassword: '',
    confirmPassword: '',
  })
  
  // 密码更改方式：'password' | 'email'
  const [passwordChangeMethod, setPasswordChangeMethod] = useState<'password' | 'email'>('password')
  const [emailCode, setEmailCode] = useState('')
  const [isSendingCode, setIsSendingCode] = useState(false)
  const [countdown, setCountdown] = useState(0)

  // 页面加载时获取最新用户信息
  useEffect(() => {
    // 只有在用户已登录的情况下才发送请求
    if (!isAuthenticated) {
      setIsPageLoading(false)
      return
    }

    if (user) {
      setIsPageLoading(false); // 如果用户信息已存在，直接结束加载
      return; // 退出 useEffect，避免重复请求
    }

    const fetchUserProfile = async () => {
      try {
        setIsPageLoading(true)
        const response = await getProfile()
        if (response.code === 200 && response.data) {
          // 更新auth store中的用户信息
          updateAuthStore(response.data)
          // 更新表单数据
          setFormData({
            email: response.data.email || '',
            phone: response.data.phone || '',
            displayName: response.data.displayName || '',
            firstName: response.data.firstName || '',
            lastName: response.data.lastName || '',
            locale: response.data.locale || 'zh-CN',
            timezone: response.data.timezone || 'Asia/Shanghai',
            gender: response.data.gender || '',
            city: response.data.city || '',
            region: response.data.region || '',
            extra: response.data.extra || '',
            avatar: response.data.avatar || '',
          })
          showAlert(t('profile.messages.userInfoUpdated'), 'success', t('profile.messages.loadSuccess'))
        } else {
          throw new Error(response.msg || t('profile.messages.getUserInfoFailed'))
        }
      } catch (error: any) {
        showAlert(error?.msg || error?.message || t('profile.messages.getUserInfoFailed'), 'error', t('profile.messages.loadFailed'))
      } finally {
        setIsPageLoading(false)
      }
    }

    fetchUserProfile()
  }, [isAuthenticated, user, updateAuthStore])

  // 当切换到活动记录标签页时加载活动记录
  useEffect(() => {
    if (activeTab === 'activity' && isAuthenticated) {
      loadActivities(1)
    }
  }, [activeTab, isAuthenticated])

  // 当切换到安全设置标签页时加载设备列表
  useEffect(() => {
    if (activeTab === 'security' && isAuthenticated) {
      loadDevices()
    }
  }, [activeTab, isAuthenticated])

  // 设置两步验证
  const handleTwoFactorSetup = async () => {
    setIsTwoFactorLoading(true)
    try {
      const response = await setupTwoFactor()
      if (response.code === 200) {
        setTwoFactorSetup(response.data)
        setShowTwoFactorSetup(true)
        showAlert(t('profile.scanQRCode'), 'info', t('profile.twoFactor'))
      } else {
        throw new Error(response.msg || '设置失败')
      }
    } catch (error: any) {
      showAlert(error?.msg || error?.message || '设置失败', 'error', '操作失败')
    } finally {
      setIsTwoFactorLoading(false)
    }
  }

  // 启用两步验证
  const handleTwoFactorEnable = async () => {
    if (!twoFactorCode.trim()) {
      showAlert(t('profile.enterCode'), 'error', t('profile.messages.verifyFailed'))
      return
    }

    setIsTwoFactorLoading(true)
    try {
      const response = await enableTwoFactor(twoFactorCode)
      if (response.code === 200) {
        setTwoFactorCode('')
        setShowTwoFactorSetup(false)
        setTwoFactorSetup(null)
        // 更新用户状态
        if (user) {
          updateAuthStore({ ...user, twoFactorEnabled: true })
        }
        showAlert(t('profile.messages.enableSuccess'), 'success', t('profile.messages.loadSuccess'))
      } else {
        throw new Error(response.msg || '启用失败')
      }
    } catch (error: any) {
      showAlert(error?.msg || error?.message || '启用失败', 'error', '操作失败')
    } finally {
      setIsTwoFactorLoading(false)
    }
  }

  // 禁用两步验证
  const handleTwoFactorDisable = async () => {
    if (!twoFactorCode.trim()) {
      showAlert(t('profile.enterCode'), 'error', t('profile.messages.verifyFailed'))
      return
    }

    setIsTwoFactorLoading(true)
    try {
      const response = await disableTwoFactor(twoFactorCode)
      if (response.code === 200) {
        setTwoFactorCode('')
        setShowTwoFactorDisable(false)
        // 更新用户状态
        if (user) {
          updateAuthStore({ ...user, twoFactorEnabled: false })
        }
        showAlert(t('profile.messages.disableSuccess'), 'success', t('profile.messages.loadSuccess'))
      } else {
        throw new Error(response.msg || '禁用失败')
      }
    } catch (error: any) {
      showAlert(error?.msg || error?.message || '禁用失败', 'error', '操作失败')
    } finally {
      setIsTwoFactorLoading(false)
    }
  }

  // 加载设备列表
  const loadDevices = async () => {
    setIsLoadingDevices(true)
    try {
      const response = await getUserDevices()
      if (response.code === 200) {
        setDevices(response.data.devices || [])
      } else {
        throw new Error(response.msg || '获取设备列表失败')
      }
    } catch (error: any) {
      showAlert(error?.msg || error?.message || '获取设备列表失败', 'error', '操作失败')
    } finally {
      setIsLoadingDevices(false)
    }
  }

  // 删除设备
  const handleDeleteDevice = async (deviceId: string) => {
    if (!confirm('确定要删除此设备吗？删除后该设备将无法继续登录。')) {
      return
    }

    setIsLoading(true)
    try {
      const response = await deleteUserDevice(deviceId)
      if (response.code === 200) {
        showAlert('设备删除成功', 'success', '操作成功')
        loadDevices() // 重新加载设备列表
      } else {
        throw new Error(response.msg || '删除设备失败')
      }
    } catch (error: any) {
      showAlert(error?.msg || error?.message || '删除设备失败', 'error', '操作失败')
    } finally {
      setIsLoading(false)
    }
  }

  // 信任设备
  const handleTrustDevice = async (deviceId: string) => {
    setIsLoading(true)
    try {
      const response = await trustUserDevice(deviceId)
      if (response.code === 200) {
        showAlert('设备已信任', 'success', '操作成功')
        loadDevices() // 重新加载设备列表
      } else {
        throw new Error(response.msg || '信任设备失败')
      }
    } catch (error: any) {
      showAlert(error?.msg || error?.message || '信任设备失败', 'error', '操作失败')
    } finally {
      setIsLoading(false)
    }
  }

  // 取消信任设备
  const handleUntrustDevice = async (deviceId: string) => {
    setConfirmDialog({
      isOpen: true,
      title: '取消信任设备',
      message: '确定要取消信任此设备吗？取消后该设备登录时将需要重新验证。',
      type: 'warning',
      onConfirm: () => performUntrustDevice(deviceId)
    })
  }

  // 执行取消信任设备操作
  const performUntrustDevice = async (deviceId: string) => {
    setIsLoading(true)
    try {
      const response = await untrustUserDevice(deviceId)
      if (response.code === 200) {
        showAlert('已取消信任设备', 'success', '操作成功')
        loadDevices() // 重新加载设备列表
      } else {
        throw new Error(response.msg || '取消信任设备失败')
      }
    } catch (error: any) {
      showAlert(error?.msg || error?.message || '取消信任设备失败', 'error', '操作失败')
    } finally {
      setIsLoading(false)
    }
  }

  // 加载活动记录
  const loadActivities = async (page: number = 1) => {
    setIsLoadingActivities(true)
    try {
      const response = await getUserActivity({ page, limit: 10 })
      if (response.code === 200) {
        setActivities(response.data.activities)
        setActivityTotalPages(response.data.pagination.totalPages)
        setActivityPage(page)
      }
    } catch (error: any) {
      console.error('Failed to load activities:', error)
    } finally {
      setIsLoadingActivities(false)
    }
  }

  // 获取活动记录图标
  const getActivityIcon = (action: string) => {
    switch (action.toLowerCase()) {
      case 'post':
        return <CheckCircle className="w-5 h-5 text-blue-600 dark:text-blue-400" />
      case 'put':
        return <Edit3 className="w-5 h-5 text-green-600 dark:text-green-400" />
      case 'delete':
        return <AlertCircle className="w-5 h-5 text-red-600 dark:text-red-400" />
      case 'get':
        return <Settings className="w-5 h-5 text-purple-600 dark:text-purple-400" />
      default:
        return <Settings className="w-5 h-5 text-gray-600 dark:text-gray-400" />
    }
  }

  // 获取活动记录背景色
  const getActivityBgColor = (action: string) => {
    switch (action.toLowerCase()) {
      case 'post':
        return 'bg-blue-100 dark:bg-blue-900/30'
      case 'put':
        return 'bg-green-100 dark:bg-green-900/30'
      case 'delete':
        return 'bg-red-100 dark:bg-red-900/30'
      case 'get':
        return 'bg-purple-100 dark:bg-purple-900/30'
      default:
        return 'bg-gray-100 dark:bg-gray-800'
    }
  }

  // 格式化活动记录描述
  const formatActivityDescription = (activity: ActivityLog) => {
    const actionMap: { [key: string]: string } = {
      'POST': t('profile.activity.action.create'),
      'PUT': t('profile.activity.action.update'),
      'DELETE': t('profile.activity.action.delete'),
      'GET': t('profile.activity.action.view'),
      'PATCH': t('profile.activity.action.modify')
    }
    
    const targetMap: { [key: string]: string } = {
      '/api/auth/login': t('profile.activity.target.login'),
      '/api/auth/update': t('profile.activity.target.profile'),
      '/api/auth/change-password': t('profile.activity.target.password'),
      '/api/auth/update/preferences': t('profile.activity.target.preferences'),
      '/api/auth/two-factor': t('profile.activity.target.twoFactor')
    }
    
    const action = actionMap[activity.action] || activity.action
    const target = targetMap[activity.target] || t('profile.activity.target.system')
    
    return `${action}${target}`
  }

  // 格式化时间
  const formatTimeAgo = (dateString: string) => {
    const date = new Date(dateString)
    const now = new Date()
    const diffInSeconds = Math.floor((now.getTime() - date.getTime()) / 1000)
    
    if (diffInSeconds < 60) return t('profile.activity.justNow')
    if (diffInSeconds < 3600) return `${Math.floor(diffInSeconds / 60)}${t('profile.activity.minutesAgo')}`
    if (diffInSeconds < 86400) return `${Math.floor(diffInSeconds / 3600)}${t('profile.activity.hoursAgo')}`
    if (diffInSeconds < 2592000) return `${Math.floor(diffInSeconds / 86400)}${t('profile.activity.daysAgo')}`
    return date.toLocaleDateString('zh-CN')
  }

  const handleSave = async () => {
    setIsLoading(true)
    try {
      const response = await updateProfile(formData)
      if (response.code === 200) {
        // 更新全局用户状态
        if (response.data) {
          updateAuthStore(response.data)
        }
        setIsEditing(false)
        showAlert(t('profile.messages.updateSuccess'), 'success', t('profile.messages.loadSuccess'))
      } else {
        throw new Error(response.msg || '更新失败')
      }
    } catch (error: any) {
      showAlert(error?.msg || error?.message || '更新失败', 'error', '操作失败')
    } finally {
      setIsLoading(false)
    }
  }

  const handleCancel = () => {
    setFormData({
      email: user?.email || '',
      phone: user?.phone || '',
      displayName: user?.displayName || '',
      firstName: user?.firstName || '',
      lastName: user?.lastName || '',
      locale: user?.locale || 'zh-CN',
      timezone: user?.timezone || 'Asia/Shanghai',
      gender: user?.gender || '',
      city: user?.city || '',
      region: user?.region || '',
      extra: user?.extra || '',
      avatar: user?.avatar || '',
    })
    setIsEditing(false)
  }

  // 发送邮箱验证码
  const handleSendEmailCode = async () => {
    if (!user?.email) {
      showAlert('邮箱地址不存在', 'error', '操作失败')
      return
    }

    setIsSendingCode(true)
    try {
      const response = await sendEmailCode({ email: user.email })
      if (response.code === 200) {
        showAlert('验证码已发送到您的邮箱', 'success', '发送成功')
        setCountdown(60)
        const timer = setInterval(() => {
          setCountdown((prev) => {
            if (prev <= 1) {
              clearInterval(timer)
              return 0
            }
            return prev - 1
          })
        }, 1000)
      } else {
        throw new Error(response.msg || '发送验证码失败')
      }
    } catch (error: any) {
      showAlert(error?.msg || error?.message || '发送验证码失败', 'error', '操作失败')
    } finally {
      setIsSendingCode(false)
    }
  }

  // 发送邮箱验证邮件
  const handleSendEmailVerification = async () => {
    if (!user?.email) {
      showAlert('邮箱地址不存在', 'error', '操作失败')
      return
    }

    if (user.emailVerified) {
      showAlert('邮箱已经验证过了', 'info', '提示')
      return
    }

    setIsSendingEmailVerification(true)
    try {
      const response = await sendEmailVerification()
      if (response.code === 200) {
        showAlert('验证邮件已发送到您的邮箱，请查收并点击邮件中的链接完成验证', 'success', '发送成功')
      } else {
        throw new Error(response.msg || '发送验证邮件失败')
      }
    } catch (error: any) {
      showAlert(error?.msg || error?.message || '发送验证邮件失败', 'error', '操作失败')
    } finally {
      setIsSendingEmailVerification(false)
    }
  }

  const handlePasswordChange = async () => {
    if (passwordData.newPassword !== passwordData.confirmPassword) {
      showAlert(t('profile.passwordMismatch'), 'error', t('profile.messages.verifyFailed'))
      return
    }

    setIsLoading(true)
    try {
      let response
      if (passwordChangeMethod === 'password') {
        // 使用原密码方式
        if (!passwordData.currentPassword) {
          showAlert('请输入当前密码', 'error', '验证失败')
          setIsLoading(false)
          return
        }
        response = await changePassword(passwordData)
      } else {
        // 使用邮箱验证码方式
        if (!emailCode) {
          showAlert('请输入邮箱验证码', 'error', '验证失败')
          setIsLoading(false)
          return
        }
        response = await changePasswordByEmail({
          emailCode,
          newPassword: passwordData.newPassword,
          confirmPassword: passwordData.confirmPassword,
        })
      }

      if (response.code === 200) {
        setPasswordData({ currentPassword: '', newPassword: '', confirmPassword: '' })
        setEmailCode('')
        setIsChangingPassword(false)
        setPasswordChangeMethod('password')
        setCountdown(0)
        showAlert(t('profile.messages.passwordChangeSuccess'), 'success', t('profile.messages.loadSuccess'))
        
        // 如果返回了logout标识，说明需要重新登录
        if ((response.data as any)?.logout) {
          setTimeout(() => {
            window.location.href = '/'
          }, 2000)
        }
      } else {
        throw new Error(response.msg || '密码修改失败')
      }
    } catch (error: any) {
      showAlert(error?.msg || error?.message || '密码修改失败', 'error', '操作失败')
    } finally {
      setIsLoading(false)
    }
  }

  const handleAvatarUpload = async (event: React.ChangeEvent<HTMLInputElement>) => {
    const file = event.target.files?.[0]
    if (!file) return

    // 验证文件类型
    const allowedTypes = ['image/jpeg', 'image/jpg', 'image/png', 'image/gif', 'image/webp']
    if (!allowedTypes.includes(file.type)) {
      showAlert(t('profile.messages.invalidFileFormat'), 'error', t('profile.messages.fileFormatError'))
      return
    }

    if (file.size > 5 * 1024 * 1024) {
      showAlert(t('profile.messages.fileTooLarge'), 'error', t('profile.messages.uploadFailed'))
      return
    }

    setIsLoading(true)
    try {
      const response = await uploadAvatar(file)
      if (response.code === 200) {
        // 更新用户头像
        updateAuthStore({ ...user, avatar: response.data.avatar })
        // 更新表单数据
        setFormData(prev => ({ ...prev, avatar: response.data.avatar }))
        showAlert(t('profile.messages.avatarUploadSuccess'), 'success', t('profile.messages.loadSuccess'))
      } else {
        throw new Error(response.msg || '头像上传失败')
      }
    } catch (error: any) {
      showAlert(error?.msg || error?.message || '头像上传失败', 'error', '上传失败')
    } finally {
      setIsLoading(false)
      // 清空文件输入
      event.target.value = ''
    }
  }


  if (!isAuthenticated) {
    return (
      <>
        <AuthModal isOpen={true} onClose={() => { window.location.href = '/' }} />
        {/* 只保留弹窗，移除下方警告区域，避免h1嵌套 */}
      </>
    )
  }

  if (isPageLoading) {
    return (
      <div className="min-h-screen flex items-center justify-center">
        <div className="text-center">
          <LoadingAnimation type="spinner" size="lg" className="mx-auto mb-4" />
          <p className="text-2xl font-bold text-neutral-900 dark:text-neutral-100 mb-4">
            {t('profile.loading')}
          </p>
          <p className="text-neutral-600 dark:text-neutral-400">
            {t('profile.loadingDesc')}
          </p>
        </div>
      </div>
    )
  }

  return (
    <div className="min-h-screen bg-gray-50 dark:bg-gray-900">
      <div className="max-w-6xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
        {/* 头部操作栏 */}
        <FadeIn direction="down">
          <div className="mb-6">
            <div className="flex items-center justify-between">
              <div className="flex items-center space-x-4">
                  <Shield className="w-3 h-3 mr-1" />
                <Badge variant={user?.role === 'admin' ? 'primary' : 'secondary'} className="text-xs">
                  {user?.role === 'admin' ? t('profile.admin') : t('profile.user')}
                </Badge>
                <div className="text-sm text-gray-500 dark:text-gray-400">
                  {t('profile.lastLogin')}：{user?.lastLogin ? new Date(user.lastLogin).toLocaleDateString('zh-CN') : t('profile.unknown')}
                </div>
              </div>
            </div>
          </div>
        </FadeIn>

        {/* 主要内容区域 */}
        <div className="grid grid-cols-1 lg:grid-cols-3 gap-8">
          {/* 左侧：用户信息卡片 */}
          <div className="lg:col-span-1">
            <FadeIn direction="left">
              <Card className="sticky top-8">
                {/* 用户头像和基本信息 */}
                <div className="p-6 border-b border-gray-200 dark:border-gray-700">
                  <div className="flex items-center space-x-4">
                    <div className="relative group">
                      <div className="w-16 h-16 rounded-lg bg-gray-100 dark:bg-gray-800 overflow-hidden">
                        <img
                          src={user?.avatar || `https://ui-avatars.com/api/?name=${user?.displayName || 'User'}&background=6366f1&color=fff&size=64`}
                          alt={user?.displayName || 'User'}
                          className="w-full h-full object-cover"
                        />
                      </div>
                      
                      {/* 上传按钮 */}
                      <label className="absolute -bottom-1 -right-1 p-1.5 bg-white dark:bg-gray-800 rounded-full shadow-md hover:shadow-lg transition-all cursor-pointer border border-gray-200 dark:border-gray-700 group-hover:scale-110">
                        <Camera className="w-3 h-3 text-gray-600 dark:text-gray-300" />
                        <input
                          type="file"
                          accept="image/jpeg,image/jpg,image/png,image/gif,image/webp"
                          onChange={handleAvatarUpload}
                          className="hidden"
                          disabled={isLoading}
                        />
                      </label>


                      {/* 加载状态 */}
                      {isLoading && (
                        <div className="absolute inset-0 bg-black bg-opacity-50 rounded-lg flex items-center justify-center">
                          <LoadingAnimation type="spinner" size="sm" color="#ffffff" />
                        </div>
                      )}
                    </div>
                    <div className="flex-1 min-w-0">
                      <h2 className="text-lg font-semibold text-gray-900 dark:text-white truncate">
                        {(() => {
                          if (user?.displayName) return user.displayName
                          if (user?.firstName && user?.lastName) return `${user.firstName} ${user.lastName}`
                          if (user?.firstName) return user.firstName
                          if (user?.lastName) return user.lastName
                          if (user?.email) return user.email.split('@')[0]
                          return '用户'
                        })()}
                      </h2>
                      <p className="text-sm text-gray-600 dark:text-gray-400 truncate">
                        {user?.email || '未设置邮箱'}
                      </p>
                      <div className="flex items-center mt-1">
                        <div className="w-2 h-2 bg-green-500 rounded-full mr-2"></div>
                        <span className="text-xs text-gray-500 dark:text-gray-400">
                          {user?.lastLogin 
                            ? `${t('profile.lastLogin')}: ${new Date(user.lastLogin).toLocaleString('zh-CN')}` 
                            : t('profile.online')}
                        </span>
                      </div>
                    </div>
                  </div>
                </div>

                {/* 账户信息 */}
                <div className="p-6">
                  <div className="space-y-4">
                    <div className="flex justify-between items-center">
                      <span className="text-sm text-gray-600 dark:text-gray-400">{t('profile.userId')}</span>
                      <span className="text-sm font-mono text-gray-900 dark:text-white">#{user?.id || 'N/A'}</span>
                    </div>
                    <div className="flex justify-between items-center">
                      <span className="text-sm text-gray-600 dark:text-gray-400">{t('profile.registerTime')}</span>
                      <span className="text-sm text-gray-900 dark:text-white">
                        {user?.createdAt ? new Date(user.createdAt).toLocaleDateString('zh-CN') : t('profile.unknown')}
                      </span>
                    </div>
                    <div className="flex justify-between items-center">
                      <span className="text-sm text-gray-600 dark:text-gray-400">{t('profile.accountStatus')}</span>
                      <Badge variant="success" className="text-xs">{t('profile.active')}</Badge>
                    </div>
                    <div className="flex justify-between items-center">
                      <span className="text-sm text-gray-600 dark:text-gray-400">邮箱状态</span>
                      <div className="flex items-center space-x-2">
                        <Badge variant={user?.emailVerified ? "success" : "warning"} className="text-xs">
                          {user?.emailVerified ? '已验证' : '未验证'}
                        </Badge>
                        {!user?.emailVerified && (
                          <button
                            onClick={handleSendEmailVerification}
                            disabled={isSendingEmailVerification}
                            className="text-xs text-blue-600 hover:text-blue-800 dark:text-blue-400 dark:hover:text-blue-300 underline disabled:opacity-50"
                          >
                            {isSendingEmailVerification ? '发送中...' : '验证'}
                          </button>
                        )}
                      </div>
                    </div>
                    {user?.phone && (
                      <div className="flex justify-between items-center">
                        <span className="text-sm text-gray-600 dark:text-gray-400">{t('profile.phone')}</span>
                        <span className="text-sm text-gray-900 dark:text-white">{user.phone}</span>
                      </div>
                    )}
                    {(user?.city || user?.region) && (
                      <div className="flex justify-between items-center">
                        <span className="text-sm text-gray-600 dark:text-gray-400">地区</span>
                        <span className="text-sm text-gray-900 dark:text-white" title={[user?.city, user?.region].filter(Boolean).join(', ')}>
                          {(() => {
                            const location = [user?.city, user?.region].filter(Boolean).join(', ') || '未设置';
                            return location.length > 8 ? location.substring(0, 8) + '...' : location;
                          })()}
                        </span>
                      </div>
                    )}
                    <div className="flex justify-between items-center">
                      <span className="text-sm text-gray-600 dark:text-gray-400">登录次数</span>
                      <span className="text-sm text-gray-900 dark:text-white">{user?.loginCount || 0} 次</span>
                    </div>
                    {user?.lastPasswordChange && (
                      <div className="flex justify-between items-center">
                        <span className="text-sm text-gray-600 dark:text-gray-400">最后修改密码</span>
                        <span className="text-sm text-gray-900 dark:text-white">
                          {new Date(user.lastPasswordChange).toLocaleDateString('zh-CN')}
                        </span>
                      </div>
                    )}
                    <div className="flex justify-between items-center">
                      <span className="text-sm text-gray-600 dark:text-gray-400">资料完整度</span>
                      <div className="flex items-center space-x-2">
                        <div className="w-16 bg-gray-200 dark:bg-gray-700 rounded-full h-2">
                          <div 
                            className="bg-blue-500 h-2 rounded-full transition-all duration-300" 
                            style={{ width: `${user?.profileComplete || 0}%` }}
                          ></div>
                        </div>
                        <span className="text-xs text-gray-600 dark:text-gray-400">{user?.profileComplete || 0}%</span>
                      </div>
                    </div>
                    {(user?.profileComplete || 0) < 100 && (
                      <div className="mt-2 p-2 bg-yellow-50 dark:bg-yellow-900/20 rounded-md border border-yellow-200 dark:border-yellow-800">
                        <p className="text-xs text-yellow-700 dark:text-yellow-300">
                          <span className="font-medium">待完善：</span>
                          {(() => {
                            const missing = [];
                            if (!user?.displayName) missing.push('显示名称');
                            if (!user?.firstName) missing.push('名字');
                            if (!user?.lastName) missing.push('姓氏');
                            if (!user?.avatar) missing.push('头像');
                            if (!user?.phone) missing.push('手机号码');
                            if (!user?.emailVerified) missing.push('邮箱验证');
                            if (!user?.city) missing.push('城市');
                            if (!user?.region) missing.push('地区');
                            if (!user?.gender) missing.push('性别');
                            return missing.slice(0, 3).join('、') + (missing.length > 3 ? '等' : '');
                          })()}
                        </p>
                      </div>
                    )}
                  </div>
                </div>
              </Card>
            </FadeIn>
          </div>

          {/* 右侧：主要内容区域 */}
          <div className="lg:col-span-2">
            <FadeIn direction="right">
              <Tabs value={activeTab} onValueChange={setActiveTab} className="space-y-0">
                <TabsList className="grid w-full grid-cols-4 bg-white dark:bg-gray-800 border border-gray-200 dark:border-gray-700 rounded-lg p-1">
                  <TabsTrigger value="profile" className="flex items-center space-x-2 text-sm py-2">
                    <User className="w-4 h-4" />
                    <span>{t('profile.tabs.profile')}</span>
                  </TabsTrigger>
                  <TabsTrigger value="settings" className="flex items-center space-x-2 text-sm py-2">
                    <Settings className="w-4 h-4" />
                    <span>{t('profile.tabs.settings')}</span>
                  </TabsTrigger>
                  <TabsTrigger value="security" className="flex items-center space-x-2 text-sm py-2">
                    <Shield className="w-4 h-4" />
                    <span>{t('profile.tabs.security')}</span>
                  </TabsTrigger>
                  <TabsTrigger value="activity" className="flex items-center space-x-2 text-sm py-2">
                    <Clock className="w-4 h-4" />
                    <span>{t('profile.tabs.activity')}</span>
                  </TabsTrigger>
                </TabsList>

                {/* 个人资料标签页 */}
                <TabsContent value="profile" className="mt-6">
                  <Card>
                    <div className="p-6">
                      <div className="flex items-center justify-between mb-4">
                        <h3 className="text-lg font-semibold text-gray-900 dark:text-white">{t('profile.basicInfo')}</h3>
                        {!isEditing ? (
                          <Button
                            variant="outline"
                            size="sm"
                            leftIcon={<Edit3 className="w-4 h-4" />}
                            onClick={() => setIsEditing(true)}
                            disabled={isLoading}
                          >
                            {t('profile.edit')}
                          </Button>
                        ) : (
                          <div className="flex space-x-2">
                            <Button
                              variant="outline"
                              size="sm"
                              leftIcon={<X className="w-4 h-4" />}
                              onClick={handleCancel}
                              disabled={isLoading}
                            >
                              {t('profile.cancel')}
                            </Button>
                            <Button
                              variant="primary"
                              size="sm"
                              leftIcon={<Save className="w-4 h-4" />}
                              onClick={handleSave}
                              disabled={isLoading}
                            >
                              {isLoading ? t('profile.saving') : t('profile.save')}
                            </Button>
                          </div>
                        )}
                      </div>
                      <div className="space-y-4">
                        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                          <Input
                            label={t('profile.displayName')}
                            value={formData.displayName}
                            onChange={(e) => setFormData(prev => ({ ...prev, displayName: e.target.value }))}
                            disabled={!isEditing}
                            leftIcon={<User className="w-4 h-4" />}
                            placeholder={t('profile.displayNamePlaceholder')}
                          />
                          
                          <Input
                            label={t('profile.email')}
                            type="email"
                            value={formData.email}
                            onChange={(e) => setFormData(prev => ({ ...prev, email: e.target.value }))}
                            disabled={!isEditing}
                            leftIcon={<Mail className="w-4 h-4" />}
                            placeholder={t('profile.emailPlaceholder')}
                          />
                        </div>

                        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                          <Input
                            label={t('profile.firstName')}
                            value={formData.firstName}
                            onChange={(e) => setFormData(prev => ({ ...prev, firstName: e.target.value }))}
                            disabled={!isEditing}
                            leftIcon={<User className="w-4 h-4" />}
                            placeholder={t('profile.firstNamePlaceholder')}
                          />
                          
                          <Input
                            label={t('profile.lastName')}
                            value={formData.lastName}
                            onChange={(e) => setFormData(prev => ({ ...prev, lastName: e.target.value }))}
                            disabled={!isEditing}
                            leftIcon={<User className="w-4 h-4" />}
                            placeholder={t('profile.lastNamePlaceholder')}
                          />
                        </div>

                        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                          <Input
                            label={t('profile.phone')}
                            value={formData.phone}
                            onChange={(e) => setFormData(prev => ({ ...prev, phone: e.target.value }))}
                            disabled={!isEditing}
                            leftIcon={<Phone className="w-4 h-4" />}
                            placeholder={t('profile.phonePlaceholder')}
                          />
                          
                          <div>
                            <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
                              {t('profile.gender')}
                            </label>
                            <select
                              value={formData.gender}
                              onChange={(e) => setFormData(prev => ({ ...prev, gender: e.target.value }))}
                              disabled={!isEditing}
                              className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-md bg-white dark:bg-gray-800 text-gray-900 dark:text-gray-100 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent disabled:opacity-50"
                            >
                              <option value="">{t('profile.genderSelect')}</option>
                              <option value="male">{t('profile.gender.male')}</option>
                              <option value="female">{t('profile.gender.female')}</option>
                              <option value="other">{t('profile.gender.other')}</option>
                            </select>
                          </div>
                        </div>

                        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                          <Input
                            label="城市"
                            value={formData.city}
                            onChange={(e) => setFormData(prev => ({ ...prev, city: e.target.value }))}
                            disabled={!isEditing}
                            placeholder="请输入所在城市"
                          />
                          
                          <Input
                            label="地区"
                            value={formData.region}
                            onChange={(e) => setFormData(prev => ({ ...prev, region: e.target.value }))}
                            disabled={!isEditing}
                            placeholder="请输入所在地区"
                          />
                        </div>

                        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                          <div>
                            <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
                              {t('profile.timezone')}
                            </label>
                            <select
                              value={formData.timezone}
                              onChange={(e) => setFormData(prev => ({ ...prev, timezone: e.target.value }))}
                              disabled={!isEditing}
                              className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-md bg-white dark:bg-gray-800 text-gray-900 dark:text-gray-100 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent disabled:opacity-50"
                            >
                              <option value="Asia/Shanghai">Asia/Shanghai</option>
                              <option value="Asia/Tokyo">Asia/Tokyo</option>
                              <option value="America/New_York">America/New_York</option>
                              <option value="Europe/London">Europe/London</option>
                              <option value="UTC">UTC</option>
                            </select>
                          </div>
                          
                          <div>
                            <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
                              {t('profile.language')}
                            </label>
                            <select
                              value={formData.locale}
                              onChange={(e) => setFormData(prev => ({ ...prev, locale: e.target.value }))}
                              disabled={!isEditing}
                              className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-md bg-white dark:bg-gray-800 text-gray-900 dark:text-gray-100 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent disabled:opacity-50"
                            >
                              <option value="zh-CN">简体中文</option>
                              <option value="zh-TW">繁體中文</option>
                              <option value="en-US">English</option>
                              <option value="ja-JP">日本語</option>
                            </select>
                          </div>
                        </div>

                        <Input
                          label={t('profile.bio')}
                          value={formData.extra}
                          onChange={(e) => setFormData(prev => ({ ...prev, extra: e.target.value }))}
                          disabled={!isEditing}
                          leftIcon={<Heart className="w-4 h-4" />}
                          placeholder={t('profile.bioPlaceholder')}
                          helperText={t('profile.bioHelper')}
                        />
                      </div>
                    </div>
                  </Card>
                </TabsContent>

                {/* 偏好设置标签页 */}
                <TabsContent value="settings" className="mt-6">
                  <Card>
                    <div className="p-6">
                      <h3 className="text-lg font-semibold text-gray-900 dark:text-white mb-4">{t('profile.notificationPreferences')}</h3>
                      <div className="space-y-4">
                        {/* 这里插入声音控制器，仅在个人页 settings tab 下展示 */}
                        <div className="mb-4">
                          <AudioController />
                        </div>
                        <div className="flex items-center justify-between p-4 bg-gray-50 dark:bg-gray-800 rounded-lg">
                          <div className="flex items-center space-x-3">
                            <div className="p-2 bg-blue-100 dark:bg-blue-900/30 rounded-lg">
                              <Mail className="w-5 h-5 text-blue-600 dark:text-blue-400" />
                            </div>
                            <div>
                              <h4 className="font-medium text-gray-900 dark:text-white">{t('profile.emailNotifications')}</h4>
                              <p className="text-sm text-gray-600 dark:text-gray-400">{t('profile.emailNotificationsDesc')}</p>
                            </div>
                          </div>
                          <Switch
                            checked={user?.emailNotifications || false}
                            onCheckedChange={async (checked) => {
                              updateAuthStore({ emailNotifications: checked })
                              try {
                                const response = await updatePreferences({
                                  emailNotifications: checked
                                })
                                if (response.code === 200) {
                                  showAlert(t('profile.preferencesUpdated'), 'success', t('profile.messages.loadSuccess'))
                                } else {
                                  throw new Error(response.msg || '更新失败')
                                }
                              } catch (error: any) {
                                updateAuthStore({ emailNotifications: !checked })
                                showAlert(error?.msg || error?.message || '更新失败', 'error', '操作失败')
                              }
                            }}
                          />
                        </div>

                        <div className="flex items-center justify-between p-4 bg-gray-50 dark:bg-gray-800 rounded-lg">
                          <div className="flex items-center space-x-3">
                            <div className="p-2 bg-green-100 dark:bg-green-900/30 rounded-lg">
                              <Bell className="w-5 h-5 text-green-600 dark:text-green-400" />
                            </div>
                            <div>
                              <h4 className="font-medium text-gray-900 dark:text-white">{t('profile.pushNotifications')}</h4>
                              <p className="text-sm text-gray-600 dark:text-gray-400">{t('profile.pushNotificationsDesc')}</p>
                            </div>
                          </div>
                          <Switch
                            checked={user?.pushNotifications || false}
                            onCheckedChange={async (checked) => {
                              updateAuthStore({ pushNotifications: checked })
                              try {
                                const response = await updatePreferences({
                                  pushNotifications: checked
                                })
                                if (response.code === 200) {
                                  showAlert(t('profile.preferencesUpdated'), 'success', t('profile.messages.loadSuccess'))
                                } else {
                                  throw new Error(response.msg || '更新失败')
                                }
                              } catch (error: any) {
                                updateAuthStore({ pushNotifications: !checked })
                                showAlert(error?.msg || error?.message || '更新失败', 'error', '操作失败')
                              }
                            }}
                          />
                        </div>

                        <div className="flex items-center justify-between p-4 bg-gray-50 dark:bg-gray-800 rounded-lg">
                          <div className="flex items-center space-x-3">
                            <div className="p-2 bg-purple-100 dark:bg-purple-900/30 rounded-lg">
                              <Zap className="w-5 h-5 text-purple-600 dark:text-purple-400" />
                            </div>
                            <div>
                              <h4 className="font-medium text-gray-900 dark:text-white">{t('profile.systemNotifications')}</h4>
                              <p className="text-sm text-gray-600 dark:text-gray-400">{t('profile.systemNotificationsDesc')}</p>
                            </div>
                          </div>
                          <Switch
                            checked={user?.systemNotifications || false}
                            onCheckedChange={async (checked) => {
                              updateAuthStore({ systemNotifications: checked })
                              try {
                                const response = await updatePreferences({
                                  systemNotifications: checked
                                })
                                if (response.code === 200) {
                                  showAlert(t('profile.preferencesUpdated'), 'success', t('profile.messages.loadSuccess'))
                                } else {
                                  throw new Error(response.msg || '更新失败')
                                }
                              } catch (error: any) {
                                updateAuthStore({ systemNotifications: !checked })
                                showAlert(error?.msg || error?.message || '更新失败', 'error', '操作失败')
                              }
                            }}
                          />
                        </div>

                        <div className="flex items-center justify-between p-4 bg-gray-50 dark:bg-gray-800 rounded-lg">
                          <div className="flex items-center space-x-3">
                            <div className="p-2 bg-orange-100 dark:bg-orange-900/30 rounded-lg">
                              <Mail className="w-5 h-5 text-orange-600 dark:text-orange-400" />
                            </div>
                            <div>
                              <h4 className="font-medium text-gray-900 dark:text-white">自动清理未读邮件</h4>
                              <p className="text-sm text-gray-600 dark:text-gray-400">自动清理超过七天未读的邮件</p>
                            </div>
                          </div>
                          <Switch
                            checked={user?.autoCleanUnreadEmails || false}
                            onCheckedChange={async (checked) => {
                              updateAuthStore({ autoCleanUnreadEmails: checked })
                              try {
                                const response = await updatePreferences({
                                  autoCleanUnreadEmails: checked
                                })
                                if (response.code === 200) {
                                  showAlert(t('profile.preferencesUpdated'), 'success', t('profile.messages.loadSuccess'))
                                } else {
                                  throw new Error(response.msg || '更新失败')
                                }
                              } catch (error: any) {
                                updateAuthStore({ autoCleanUnreadEmails: !checked })
                                showAlert(error?.msg || error?.message || '更新失败', 'error', '操作失败')
                              }
                            }}
                          />
                        </div>
                      </div>
                    </div>
                  </Card>
                </TabsContent>

                {/* 安全设置标签页 */}
                <TabsContent value="security" className="mt-6">
                  <Card>
                    <div className="p-6">
                      <h3 className="text-lg font-semibold text-gray-900 dark:text-white mb-4">密码安全</h3>
                      <div className="space-y-4">
                        <div className="flex items-center justify-between p-4 bg-red-50 dark:bg-red-900/20 rounded-lg border border-red-200 dark:border-red-800">
                          <div className="flex items-center space-x-3">
                            <div className="p-2 bg-red-100 dark:bg-red-900/30 rounded-lg">
                              <Key className="w-5 h-5 text-red-600 dark:text-red-400" />
                            </div>
                            <div>
                              <h4 className="font-medium text-gray-900 dark:text-white">更改密码</h4>
                              <p className="text-sm text-gray-600 dark:text-gray-400">定期更新密码以保护账户安全</p>
                            </div>
                          </div>
                          <Button 
                            variant="outline" 
                            size="sm"
                            onClick={() => setIsChangingPassword(!isChangingPassword)}
                            disabled={isLoading}
                          >
                            {isChangingPassword ? '取消' : '更改密码'}
                          </Button>
                        </div>

                        <AnimatePresence>
                          {isChangingPassword && (
                            <motion.div
                              initial={{ opacity: 0, height: 0 }}
                              animate={{ opacity: 1, height: 'auto' }}
                              exit={{ opacity: 0, height: 0 }}
                              className="space-y-4 p-4 bg-gray-50 dark:bg-gray-800 rounded-lg"
                            >
                              {/* 更改密码方式选择 */}
                              <div className="mb-4">
                                <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
                                  更改密码方式
                                </label>
                                <div className="flex space-x-4">
                                  <button
                                    type="button"
                                    onClick={() => {
                                      setPasswordChangeMethod('password')
                                      setEmailCode('')
                                    }}
                                    className={`flex-1 px-4 py-2 rounded-md text-sm font-medium transition-colors ${
                                      passwordChangeMethod === 'password'
                                        ? 'bg-blue-500 text-white'
                                        : 'bg-gray-200 dark:bg-gray-700 text-gray-700 dark:text-gray-300 hover:bg-gray-300 dark:hover:bg-gray-600'
                                    }`}
                                  >
                                    使用当前密码
                                  </button>
                                  <button
                                    type="button"
                                    onClick={() => {
                                      setPasswordChangeMethod('email')
                                      setPasswordData(prev => ({ ...prev, currentPassword: '' }))
                                    }}
                                    className={`flex-1 px-4 py-2 rounded-md text-sm font-medium transition-colors ${
                                      passwordChangeMethod === 'email'
                                        ? 'bg-blue-500 text-white'
                                        : 'bg-gray-200 dark:bg-gray-700 text-gray-700 dark:text-gray-300 hover:bg-gray-300 dark:hover:bg-gray-600'
                                    }`}
                                  >
                                    使用邮箱验证码
                                  </button>
                                </div>
                              </div>

                              {/* 根据选择的方式显示不同的输入 */}
                              {passwordChangeMethod === 'password' ? (
                                <Input
                                  label="当前密码"
                                  type={showCurrentPassword ? 'text' : 'password'}
                                  value={passwordData.currentPassword}
                                  onChange={(e) => setPasswordData(prev => ({ ...prev, currentPassword: e.target.value }))}
                                  leftIcon={<Lock className="w-4 h-4" />}
                                  rightIcon={
                                    <button
                                      type="button"
                                      onClick={() => setShowCurrentPassword(!showCurrentPassword)}
                                      className="text-gray-400 hover:text-gray-600 dark:hover:text-gray-300"
                                    >
                                      {showCurrentPassword ? <EyeOff className="w-4 h-4" /> : <Eye className="w-4 h-4" />}
                                    </button>
                                  }
                                  placeholder="请输入当前密码"
                                />
                              ) : (
                                <div className="space-y-2">
                                  <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
                                    邮箱验证码
                                  </label>
                                  <div className="flex space-x-2">
                                    <div className="flex-1">
                                      <Input
                                        label=""
                                        type="text"
                                        value={emailCode}
                                        onChange={(e) => setEmailCode(e.target.value)}
                                        leftIcon={<Mail className="w-4 h-4" />}
                                        placeholder="请输入邮箱验证码"
                                      />
                                    </div>
                                    <Button
                                      variant="outline"
                                      size="sm"
                                      onClick={handleSendEmailCode}
                                      disabled={isSendingCode || countdown > 0}
                                      className="whitespace-nowrap self-end"
                                    >
                                      {countdown > 0 ? `${countdown}秒` : isSendingCode ? '发送中...' : '发送验证码'}
                                    </Button>
                                  </div>
                                  <p className="text-xs text-gray-500 dark:text-gray-400">
                                    验证码将发送到 {user?.email}
                                  </p>
                                </div>
                              )}
                              
                              <Input
                                label="新密码"
                                type={showNewPassword ? 'text' : 'password'}
                                value={passwordData.newPassword}
                                onChange={(e) => setPasswordData(prev => ({ ...prev, newPassword: e.target.value }))}
                                leftIcon={<Lock className="w-4 h-4" />}
                                rightIcon={
                                  <button
                                    type="button"
                                    onClick={() => setShowNewPassword(!showNewPassword)}
                                    className="text-gray-400 hover:text-gray-600 dark:hover:text-gray-300"
                                  >
                                    {showNewPassword ? <EyeOff className="w-4 h-4" /> : <Eye className="w-4 h-4" />}
                                  </button>
                                }
                                placeholder="请输入新密码"
                              />
                              
                              <Input
                                label="确认新密码"
                                type={showConfirmPassword ? 'text' : 'password'}
                                value={passwordData.confirmPassword}
                                onChange={(e) => setPasswordData(prev => ({ ...prev, confirmPassword: e.target.value }))}
                                leftIcon={<Lock className="w-4 h-4" />}
                                rightIcon={
                                  <button
                                    type="button"
                                    onClick={() => setShowConfirmPassword(!showConfirmPassword)}
                                    className="text-gray-400 hover:text-gray-600 dark:hover:text-gray-300"
                                  >
                                    {showConfirmPassword ? <EyeOff className="w-4 h-4" /> : <Eye className="w-4 h-4" />}
                                  </button>
                                }
                                placeholder="请再次输入新密码"
                              />
                              
                              <div className="flex space-x-3">
                                <Button
                                  variant="outline"
                                  onClick={() => {
                                    setIsChangingPassword(false)
                                    setPasswordData({ currentPassword: '', newPassword: '', confirmPassword: '' })
                                    setEmailCode('')
                                    setPasswordChangeMethod('password')
                                    setCountdown(0)
                                  }}
                                  disabled={isLoading}
                                >
                                  取消
                                </Button>
                                <Button
                                  variant="primary"
                                  onClick={handlePasswordChange}
                                  disabled={isLoading}
                                >
                                  {isLoading ? '修改中...' : '确认修改'}
                                </Button>
                              </div>
                            </motion.div>
                          )}
                        </AnimatePresence>

                        <div className={`flex items-center justify-between p-4 rounded-lg border ${
                          user?.twoFactorEnabled 
                            ? 'bg-green-50 dark:bg-green-900/20 border-green-200 dark:border-green-800' 
                            : 'bg-gray-50 dark:bg-gray-800'
                        }`}>
                          <div className="flex items-center space-x-3">
                            <div className={`p-2 rounded-lg ${
                              user?.twoFactorEnabled 
                                ? 'bg-green-100 dark:bg-green-900/30' 
                                : 'bg-gray-100 dark:bg-gray-700'
                            }`}>
                              <Shield className={`w-5 h-5 ${
                                user?.twoFactorEnabled 
                                  ? 'text-green-600 dark:text-green-400' 
                                  : 'text-gray-600 dark:text-gray-400'
                              }`} />
                            </div>
                            <div>
                              <h4 className="font-medium text-gray-900 dark:text-white">两步验证</h4>
                              <p className="text-sm text-gray-600 dark:text-gray-400">
                                {user?.twoFactorEnabled ? '已启用 - 为您的账户提供额外的安全保护' : '为您的账户添加额外的安全保护'}
                              </p>
                            </div>
                          </div>
                          <div className="flex items-center space-x-2">
                            {user?.twoFactorEnabled ? (
                              <Button 
                                variant="destructive" 
                                size="sm"
                                onClick={() => setShowTwoFactorDisable(true)}
                                disabled={isTwoFactorLoading}
                              >
                                禁用
                              </Button>
                            ) : (
                              <Button 
                                variant="outline" 
                                size="sm"
                                onClick={handleTwoFactorSetup}
                                disabled={isTwoFactorLoading}
                              >
                                {isTwoFactorLoading ? '设置中...' : '启用'}
                              </Button>
                            )}
                          </div>
                        </div>

                        {/* 设备管理 */}
                        <div className="mt-6">
                          <div className="flex items-center justify-between mb-4">
                            <h4 className="text-md font-semibold text-gray-900 dark:text-white">设备管理</h4>
                            <Button
                              variant="outline"
                              size="sm"
                              onClick={loadDevices}
                              disabled={isLoadingDevices}
                            >
                              {isLoadingDevices ? '加载中...' : '刷新'}
                            </Button>
                          </div>
                          
                          {isLoadingDevices ? (
                            <div className="flex items-center justify-center py-8">
                              <LoadingAnimation type="spinner" size="md" />
                              <span className="ml-2 text-gray-600 dark:text-gray-400">加载中...</span>
                            </div>
                          ) : devices.length === 0 ? (
                            <div className="text-center py-8 bg-gray-50 dark:bg-gray-800 rounded-lg">
                              <Settings className="w-12 h-12 text-gray-400 mx-auto mb-4" />
                              <p className="text-gray-500 dark:text-gray-400">暂无设备记录</p>
                            </div>
                          ) : (
                            <div className="space-y-3 max-h-96 overflow-y-auto">
                              {devices.map((device) => (
                                <motion.div
                                  key={device.id}
                                  initial={{ opacity: 0, y: 20 }}
                                  animate={{ opacity: 1, y: 0 }}
                                  className="flex items-center justify-between p-4 bg-gray-50 dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700"
                                >
                                  <div className="flex items-center space-x-4 flex-1">
                                    <div className={`p-2 rounded-lg ${
                                      device.isTrusted 
                                        ? 'bg-green-100 dark:bg-green-900/30' 
                                        : 'bg-gray-100 dark:bg-gray-700'
                                    }`}>
                                      <Settings className={`w-5 h-5 ${
                                        device.isTrusted 
                                          ? 'text-green-600 dark:text-green-400' 
                                          : 'text-gray-600 dark:text-gray-400'
                                      }`} />
                                    </div>
                                    <div className="flex-1">
                                      <div className="flex items-center space-x-2">
                                        <h5 className="text-sm font-medium text-gray-900 dark:text-white">
                                          {device.deviceName || `${device.browser} on ${device.os}`}
                                        </h5>
                                        {device.isTrusted && (
                                          <Badge variant="success" className="text-xs">已信任</Badge>
                                        )}
                                      </div>
                                      <p className="text-xs text-gray-600 dark:text-gray-400 mt-1">
                                        {device.os} • {device.browser}
                                      </p>
                                      <p className="text-xs text-gray-500 dark:text-gray-500 mt-1">
                                        {device.location || device.ipAddress} • 最后使用: {new Date(device.lastUsedAt).toLocaleString('zh-CN')}
                                      </p>
                                    </div>
                                  </div>
                                  <div className="flex items-center space-x-2">
                                    {!device.isTrusted ? (
                                      <Button
                                        variant="outline"
                                        size="sm"
                                        onClick={() => handleTrustDevice(device.deviceId)}
                                        disabled={isLoading}
                                      >
                                        信任
                                      </Button>
                                    ) : (
                                      <Button
                                        variant="outline"
                                        size="sm"
                                        onClick={() => handleUntrustDevice(device.deviceId)}
                                        disabled={isLoading}
                                      >
                                        取消信任
                                      </Button>
                                    )}
                                    <Button
                                      variant="destructive"
                                      size="sm"
                                      onClick={() => handleDeleteDevice(device.deviceId)}
                                      disabled={isLoading}
                                    >
                                      删除
                                    </Button>
                                  </div>
                                </motion.div>
                              ))}
                            </div>
                          )}
                        </div>
                      </div>
                    </div>
                  </Card>
                </TabsContent>

                {/* 活动记录标签页 */}
                <TabsContent value="activity" className="mt-6">
                  <Card>
                    <div className="p-6">
                      <div className="flex items-center justify-between mb-4">
                        <h3 className="text-lg font-semibold text-gray-900 dark:text-white">最近活动</h3>
                        <Button
                          variant="outline"
                          size="sm"
                          onClick={() => loadActivities(1)}
                          disabled={isLoadingActivities}
                        >
                          {isLoadingActivities ? '加载中...' : '刷新'}
                        </Button>
                      </div>
                      
                      {isLoadingActivities ? (
                        <div className="flex items-center justify-center py-8">
                          <LoadingAnimation type="spinner" size="md" />
                          <span className="ml-2 text-gray-600 dark:text-gray-400">加载中...</span>
                        </div>
                      ) : activities.length === 0 ? (
                        <div className="text-center py-8">
                          <Settings className="w-12 h-12 text-gray-400 mx-auto mb-4" />
                          <p className="text-gray-500 dark:text-gray-400">暂无活动记录</p>
                        </div>
                      ) : (
                        <div className="space-y-4 max-h-96 overflow-y-auto">
                          {activities.map((activity) => (
                            <motion.div
                              key={activity.id}
                              initial={{ opacity: 0, y: 20 }}
                              animate={{ opacity: 1, y: 0 }}
                              className="flex items-center space-x-4 p-4 bg-gray-50 dark:bg-gray-800 rounded-lg"
                            >
                              <div className={`p-2 ${getActivityBgColor(activity.action)} rounded-lg`}>
                                {getActivityIcon(activity.action)}
                              </div>
                              <div className="flex-1">
                                <p className="text-sm font-medium text-gray-900 dark:text-white">
                                  {formatActivityDescription(activity)}
                                </p>
                                <p className="text-xs text-gray-600 dark:text-gray-400">
                                  {formatTimeAgo(activity.createdAt)} • {activity.browser} • {activity.location}
                                </p>
                                {activity.details && (
                                  <p className="text-xs text-gray-500 dark:text-gray-500 mt-1">
                                    {activity.details}
                                  </p>
                                )}
                              </div>
                            </motion.div>
                          ))}
                          
                          {/* 分页 */}
                          {activityTotalPages > 1 && (
                            <div className="flex items-center justify-center space-x-2 mt-6">
                              <Button
                                variant="outline"
                                size="sm"
                                onClick={() => loadActivities(activityPage - 1)}
                                disabled={activityPage <= 1 || isLoadingActivities}
                              >
                                上一页
                              </Button>
                              <span className="text-sm text-gray-600 dark:text-gray-400">
                                第 {activityPage} 页，共 {activityTotalPages} 页
                              </span>
                              <Button
                                variant="outline"
                                size="sm"
                                onClick={() => loadActivities(activityPage + 1)}
                                disabled={activityPage >= activityTotalPages || isLoadingActivities}
                              >
                                下一页
                              </Button>
                            </div>
                          )}
                        </div>
                      )}
                    </div>
                  </Card>
                </TabsContent>
              </Tabs>
            </FadeIn>
          </div>
        </div>
      </div>

      {/* 两步验证设置模态框 */}
      {showTwoFactorSetup && twoFactorSetup && (
        <div className="fixed inset-0 z-50 overflow-y-auto">
          <div className="flex min-h-screen items-center justify-center p-4">
            <div className="fixed inset-0 bg-black bg-opacity-50" onClick={() => setShowTwoFactorSetup(false)}></div>
            <div className="relative bg-white dark:bg-gray-800 rounded-lg shadow-xl max-w-md w-full p-6">
              <div className="flex items-center justify-between mb-4">
                <h3 className="text-lg font-semibold text-gray-900 dark:text-white">设置两步验证</h3>
                <button
                  onClick={() => setShowTwoFactorSetup(false)}
                  className="text-gray-400 hover:text-gray-600 dark:hover:text-gray-300"
                >
                  <X className="w-5 h-5" />
                </button>
              </div>
              
              <div className="space-y-4">
                <p className="text-sm text-gray-600 dark:text-gray-400">
                  请使用您的身份验证器应用扫描下面的二维码，然后输入生成的验证码。
                </p>
                
                <div className="flex justify-center p-4 bg-white rounded-lg border">
                  <img 
                    src={twoFactorSetup.qrCode} 
                    alt="Two-Factor Authentication QR Code"
                    className="w-48 h-48"
                  />
                </div>
                
                <div className="space-y-2">
                  <label className="block text-sm font-medium text-gray-700 dark:text-gray-300">
                    验证码
                  </label>
                  <input
                    type="text"
                    value={twoFactorCode}
                    onChange={(e) => setTwoFactorCode(e.target.value)}
                    placeholder="输入6位验证码"
                    className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500 dark:bg-gray-700 dark:text-white"
                    maxLength={6}
                  />
                </div>
                
                <div className="flex justify-end space-x-3">
                  <Button
                    variant="outline"
                    onClick={() => setShowTwoFactorSetup(false)}
                    disabled={isTwoFactorLoading}
                  >
                    取消
                  </Button>
                  <Button
                    onClick={handleTwoFactorEnable}
                    disabled={isTwoFactorLoading || !twoFactorCode.trim()}
                  >
                    {isTwoFactorLoading ? '启用中...' : '启用'}
                  </Button>
                </div>
              </div>
            </div>
          </div>
        </div>
      )}

      {/* 两步验证禁用模态框 */}
      {showTwoFactorDisable && (
        <div className="fixed inset-0 z-50 overflow-y-auto">
          <div className="flex min-h-screen items-center justify-center p-4">
            <div className="fixed inset-0 bg-black bg-opacity-50" onClick={() => setShowTwoFactorDisable(false)}></div>
            <div className="relative bg-white dark:bg-gray-800 rounded-lg shadow-xl max-w-md w-full p-6">
              <div className="flex items-center justify-between mb-4">
                <h3 className="text-lg font-semibold text-gray-900 dark:text-white">禁用两步验证</h3>
                <button
                  onClick={() => setShowTwoFactorDisable(false)}
                  className="text-gray-400 hover:text-gray-600 dark:hover:text-gray-300"
                >
                  <X className="w-5 h-5" />
                </button>
              </div>
              
              <div className="space-y-4">
                <p className="text-sm text-gray-600 dark:text-gray-400">
                  为了安全起见，请输入您的身份验证器应用生成的验证码来禁用两步验证。
                </p>
                
                <div className="space-y-2">
                  <label className="block text-sm font-medium text-gray-700 dark:text-gray-300">
                    验证码
                  </label>
                  <input
                    type="text"
                    value={twoFactorCode}
                    onChange={(e) => setTwoFactorCode(e.target.value)}
                    placeholder="输入6位验证码"
                    className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-md focus:outline-none focus:ring-2 focus:ring-red-500 dark:bg-gray-700 dark:text-white"
                    maxLength={6}
                  />
                </div>
                
                <div className="flex justify-end space-x-3">
                  <Button
                    variant="outline"
                    onClick={() => setShowTwoFactorDisable(false)}
                    disabled={isTwoFactorLoading}
                  >
                    取消
                  </Button>
                  <Button
                    variant="destructive"
                    onClick={handleTwoFactorDisable}
                    disabled={isTwoFactorLoading || !twoFactorCode.trim()}
                  >
                    {isTwoFactorLoading ? '禁用中...' : '禁用'}
                  </Button>
                </div>
              </div>
            </div>
          </div>
        </div>
      )}

      {/* 确认对话框 */}
      <ConfirmDialog
        isOpen={confirmDialog.isOpen}
        onClose={() => setConfirmDialog(prev => ({ ...prev, isOpen: false }))}
        onConfirm={confirmDialog.onConfirm}
        title={confirmDialog.title}
        message={confirmDialog.message}
        type={confirmDialog.type}
        loading={isLoading}
      />
    </div>
  )
}

export default Profile
