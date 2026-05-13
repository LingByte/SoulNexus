import { useState, useEffect, useRef } from 'react'
import {
  User, Mail, Camera, Save, Edit3, X, Lock, Eye, EyeOff,
  Phone, Heart,
  Smartphone, Monitor, Laptop, Tablet, MapPin,
} from 'lucide-react'
import { useAuthStore } from '../stores/authStore'
import { useI18nStore } from '../stores/i18nStore'
import { useThemeStore, type ThemeMode } from '../stores/themeStore'
import Button from '../components/UI/Button'
import Input from '../components/UI/Input'
import Card from '../components/UI/Card'
import Badge from '../components/UI/Badge'
import Switch from '../components/UI/Switch'
import { Select, SelectTrigger, SelectContent, SelectItem, SelectValue } from '../components/UI/Select'
import FadeIn from '../components/Animations/FadeIn'
import LoadingAnimation from '../components/Animations/LoadingAnimation'
import { showAlert } from '../utils/notification'
import { getProfile, updateProfile, updatePreferences, changePassword, changePasswordByEmail, uploadAvatar, setupTwoFactor, enableTwoFactor, disableTwoFactor, getUserDevices, deleteUserDevice, trustUserDevice, untrustUserDevice, TwoFactorSetupResponse, UserDevice } from '../api/profile'
import { sendEmailCode, sendEmailVerification } from '../api/auth'
import { motion, AnimatePresence } from 'framer-motion'
import ConfirmDialog from '../components/UI/ConfirmDialog'
import { beginSSOLogin } from '@/utils/sso'
import { Link, useParams, Navigate } from 'react-router-dom'
import Billing from '@/pages/Billing.tsx'
import NotificationCenter from '@/pages/NotificationCenter.tsx'
import CredentialManager from '@/pages/CredentialManager.tsx'
import LLMTokenManager from '@/pages/profile/LLMTokenManager.tsx'
import TeamWorkspacePage from '@/pages/profile/TeamWorkspacePage.tsx'
import ProfileAuditLogPanel from '@/pages/profile/ProfileAuditLogPanel.tsx'
import { resolveDeviceVisualKind, type DeviceVisualKind } from '@/pages/profile/profileDeviceVisual'
import { getUserServiceBaseURL } from '@/config/apiConfig'

const DEVICE_VISUAL: Record<
  DeviceVisualKind,
  { Icon: typeof Monitor; ring: string; icon: string }
> = {
  desktop: {
    Icon: Monitor,
    ring: 'ring-sky-200/80 dark:ring-sky-800/60',
    icon: 'text-sky-600 dark:text-sky-400',
  },
  laptop: {
    Icon: Laptop,
    ring: 'ring-amber-200/80 dark:ring-amber-800/50',
    icon: 'text-amber-700 dark:text-amber-400',
  },
  tablet: {
    Icon: Tablet,
    ring: 'ring-violet-200/80 dark:ring-violet-800/50',
    icon: 'text-violet-600 dark:text-violet-400',
  },
  phone: {
    Icon: Smartphone,
    ring: 'ring-emerald-200/80 dark:ring-emerald-800/50',
    icon: 'text-emerald-600 dark:text-emerald-400',
  },
}

const VALID_SECTIONS = new Set([
  'personal',
  'teams',
  'activity',
  'billing',
  'user-devices',
  'account-security',
  'credential',
  'llm-tokens',
  'notifications',
  'locale',
])

const Profile = () => {
  const { section } = useParams<{ section: string }>()
  const { user, isAuthenticated, updateProfile: updateAuthStore } = useAuthStore()
  const { t } = useI18nStore()
  const [isEditing, setIsEditing] = useState(false)
  const [isChangingPassword, setIsChangingPassword] = useState(false)
  const [isLoading, setIsLoading] = useState(false)
  const [isPageLoading, setIsPageLoading] = useState(true)
  const [showCurrentPassword, setShowCurrentPassword] = useState(false)
  const [showNewPassword, setShowNewPassword] = useState(false)
  const [showConfirmPassword, setShowConfirmPassword] = useState(false)
  
  // 两步验证相关状态
  const [twoFactorSetup, setTwoFactorSetup] = useState<TwoFactorSetupResponse | null>(null)
  const [twoFactorCode, setTwoFactorCode] = useState('')
  const [isTwoFactorLoading, setIsTwoFactorLoading] = useState(false)
  const [showTwoFactorSetup, setShowTwoFactorSetup] = useState(false)
  const [showTwoFactorDisable, setShowTwoFactorDisable] = useState(false)

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
  const [isWechatBinding, setIsWechatBinding] = useState(false)
  const [wechatBindCode, setWechatBindCode] = useState('')
  const [wechatBindExpiresAt, setWechatBindExpiresAt] = useState<number>(0)
  const [wechatBindCountdown, setWechatBindCountdown] = useState(0)
  const [wechatBindStatus, setWechatBindStatus] = useState<'idle' | 'pending' | 'success' | 'failed' | 'expired'>('idle')
  const wechatBindPollRef = useRef<number | null>(null)
  
  const [formData, setFormData] = useState({
    email: user?.email || '',
    phone: user?.phone || '',
    displayName: user?.displayName || '',
    firstName: user?.firstName || '',
    lastName: user?.lastName || '',
    locale: user?.locale || 'zh',
    timezone: user?.timezone || 'Asia/Shanghai',
    themeMode: (user?.themeMode as ThemeMode) || useThemeStore.getState().theme.mode,
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
            locale: response.data.locale || 'zh',
            timezone: response.data.timezone || 'Asia/Shanghai',
            themeMode: response.data.themeMode || useThemeStore.getState().theme.mode,
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

  useEffect(() => {
    if (!isAuthenticated) {
      beginSSOLogin('/profile/personal')
    }
  }, [isAuthenticated])

  useEffect(() => {
    if (!user) return
    setFormData((prev) => ({
      ...prev,
      locale: user.locale || prev.locale,
      timezone: user.timezone || prev.timezone,
      themeMode: (user.themeMode as ThemeMode) || prev.themeMode,
    }))
  }, [user?.locale, user?.timezone, user?.themeMode])

  useEffect(() => {
    const params = new URLSearchParams(window.location.search)
    const bind = params.get('bind')
    if (bind === 'github_ok') {
      showAlert('GitHub 账号绑定成功', 'success', '绑定成功')
      useAuthStore.getState().refreshUserInfo()
      params.delete('bind')
      const next = `${window.location.pathname}${params.toString() ? `?${params.toString()}` : ''}`
      window.history.replaceState({}, '', next)
    } else if (bind === 'github_already_bound') {
      showAlert('当前账号已绑定 GitHub，不能重复绑定其他账号', 'error', '绑定失败')
      params.delete('bind')
      const next = `${window.location.pathname}${params.toString() ? `?${params.toString()}` : ''}`
      window.history.replaceState({}, '', next)
    } else if (bind === 'github_bound_other') {
      showAlert('该 GitHub 账号已被其他用户绑定', 'error', '绑定失败')
      params.delete('bind')
      const next = `${window.location.pathname}${params.toString() ? `?${params.toString()}` : ''}`
      window.history.replaceState({}, '', next)
    } else if (bind === 'wechat_ok') {
      showAlert('微信账号绑定成功', 'success', '绑定成功')
      useAuthStore.getState().refreshUserInfo()
      params.delete('bind')
      const next = `${window.location.pathname}${params.toString() ? `?${params.toString()}` : ''}`
      window.history.replaceState({}, '', next)
    }
  }, [])

  useEffect(() => {
    if (!wechatBindExpiresAt) return
    const updateCountdown = () => {
      const remain = Math.max(0, Math.floor((wechatBindExpiresAt * 1000 - Date.now()) / 1000))
      setWechatBindCountdown(remain)
      if (remain <= 0 && wechatBindStatus === 'pending') {
        setWechatBindStatus('expired')
      }
    }
    updateCountdown()
    const timer = window.setInterval(updateCountdown, 1000)
    return () => window.clearInterval(timer)
  }, [wechatBindExpiresAt, wechatBindStatus])

  useEffect(() => {
    return () => {
      if (wechatBindPollRef.current) {
        window.clearInterval(wechatBindPollRef.current)
        wechatBindPollRef.current = null
      }
    }
  }, [])

  useEffect(() => {
    if (section === 'user-devices' && isAuthenticated) {
      loadDevices()
    }
  }, [section, isAuthenticated])

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

  const handleDeleteDevice = (deviceId: string) => {
    setConfirmDialog({
      isOpen: true,
      title: '删除登录设备',
      message: '确定要删除此设备吗？删除后该设备将无法继续登录。',
      type: 'danger',
      onConfirm: () => performDeleteDevice(deviceId),
    })
  }

  const performDeleteDevice = async (deviceId: string) => {
    setIsLoading(true)
    try {
      const response = await deleteUserDevice(deviceId)
      if (response.code === 200) {
        showAlert('设备删除成功', 'success', '操作成功')
        loadDevices()
      } else {
        throw new Error(response.msg || '删除设备失败')
      }
    } catch (error: any) {
      showAlert(error?.msg || error?.message || '删除设备失败', 'error', '操作失败')
      throw error
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

  const handleSave = async () => {
    setIsLoading(true)
    try {
      const response = await updateProfile(formData)
      if (response.code === 200) {
        await useAuthStore.getState().refreshUserInfo()
        // 资料里保存的外观需立即作用到全局（refresh 不再从服务端覆盖 mode，见 applyAuthUserUIPreferences）
        useThemeStore.getState().setTheme({
          mode: formData.themeMode,
        })
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
      locale: user?.locale || 'zh',
      timezone: user?.timezone || 'Asia/Shanghai',
      themeMode: (user?.themeMode as ThemeMode) || useThemeStore.getState().theme.mode,
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

  const handleBindGithub = () => {
    const target = `${window.location.origin}/profile/personal`
    window.location.href = `${getUserServiceBaseURL()}/auth/github/login?bind=1&redirecturl=${encodeURIComponent(target)}`
  }

  const handleBindWechat = () => {
    const run = async () => {
      setIsWechatBinding(true)
      try {
        const resp = await fetch(`${getUserServiceBaseURL()}/auth/wechat/bind/code`, {
          method: 'GET',
          headers: {
            Authorization: `Bearer ${localStorage.getItem('auth_token') || ''}`,
          },
        })
        const data = await resp.json()
        if (!resp.ok || data.code !== 200 || !data.data?.sessionId || !data.data?.bindCode) {
          throw new Error(data?.msg || '获取微信绑定码失败')
        }
        const sessionId = data.data.sessionId as string
        const bindCode = data.data.bindCode as string
        const expiresAt = Number(data.data.expiresAt || 0)
        setWechatBindCode(bindCode)
        setWechatBindExpiresAt(expiresAt)
        setWechatBindStatus('pending')
        showAlert(`请先扫码关注公众号，然后发送绑定码：${bindCode}`, 'info', '微信绑定')
        if (wechatBindPollRef.current) {
          window.clearInterval(wechatBindPollRef.current)
          wechatBindPollRef.current = null
        }
        const poll = window.setInterval(async () => {
          try {
            const statusResp = await fetch(`${getUserServiceBaseURL()}/auth/wechat/bind/status?sessionId=${encodeURIComponent(sessionId)}`, {
              method: 'GET',
              headers: {
                Authorization: `Bearer ${localStorage.getItem('auth_token') || ''}`,
              },
            })
            const statusData = await statusResp.json()
            const status = statusData?.data?.status
            if (status === 'success') {
              window.clearInterval(poll)
              wechatBindPollRef.current = null
              setIsWechatBinding(false)
              setWechatBindStatus('success')
              showAlert('微信绑定成功', 'success', '绑定成功')
              await useAuthStore.getState().refreshUserInfo()
            } else if (status === 'failed') {
              window.clearInterval(poll)
              wechatBindPollRef.current = null
              setIsWechatBinding(false)
              setWechatBindStatus('failed')
              const reason = statusData?.data?.reason
              if (reason === 'already_bound') {
                showAlert('该微信已绑定其他用户', 'error', '绑定失败')
              } else {
                showAlert('微信绑定失败，请重试', 'error', '绑定失败')
              }
            } else if (status === 'expired') {
              window.clearInterval(poll)
              wechatBindPollRef.current = null
              setIsWechatBinding(false)
              setWechatBindStatus('expired')
              showAlert('绑定码已过期，请重新获取', 'error', '绑定失败')
            }
          } catch {
            // keep polling on transient network errors
          }
        }, 2500)
        wechatBindPollRef.current = poll
      } catch (error: any) {
        showAlert(error?.message || '微信绑定失败，请重试', 'error', '绑定失败')
        setIsWechatBinding(false)
        setWechatBindStatus('failed')
      }
    }
    run()
  }

  if (!isAuthenticated) {
    return null
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

  if (section === 'integrations' || section === 'security') {
    return <Navigate to="/profile/account-security" replace />
  }

  if (!section || !VALID_SECTIONS.has(section)) {
    return <Navigate to="/profile/personal" replace />
  }

  return (
    <>
      <FadeIn direction="right" className={section === 'personal' ? 'flex min-h-0 min-w-0 flex-1 flex-col' : undefined}>
        {section === 'personal' && (
          <div className="flex min-h-0 min-w-0 flex-1 flex-col">
            <div className="flex min-h-0 flex-1 flex-col overflow-hidden rounded-lg border border-slate-200 bg-white dark:border-neutral-800 dark:bg-neutral-950 lg:min-h-[calc(100dvh-8.25rem)]">
              <div className="grid min-h-0 flex-1 grid-cols-1 lg:grid-cols-[15.5rem_minmax(0,1fr)] lg:overflow-hidden">
                <aside className="flex shrink-0 flex-col gap-3 border-b border-slate-200 bg-slate-50/90 px-3 py-4 dark:border-neutral-800 dark:bg-neutral-900/40 sm:px-4 lg:min-h-0 lg:items-stretch lg:justify-between lg:gap-6 lg:self-stretch lg:border-b-0 lg:border-r lg:px-4 lg:py-10">
                  <div className="flex items-center gap-3 lg:flex-col lg:items-center lg:gap-5">
                  <div className="relative shrink-0">
                    <div className="h-16 w-16 overflow-hidden rounded-full bg-slate-200 ring-1 ring-slate-300/80 dark:bg-neutral-700 dark:ring-neutral-600 lg:h-36 lg:w-36">
                      <img
                        src={
                          user?.avatar ||
                          `https://ui-avatars.com/api/?name=${encodeURIComponent(user?.displayName || 'User')}&background=64748b&color=fff&size=128`
                        }
                        alt=""
                        className="h-full w-full object-cover"
                      />
                    </div>
                    <label className="absolute -bottom-0.5 -right-0.5 flex cursor-pointer rounded-full border border-slate-200 bg-white p-1.5 shadow-sm dark:border-neutral-600 dark:bg-neutral-800 lg:bottom-1 lg:right-1 lg:p-2">
                      <Camera className="h-3.5 w-3.5 text-slate-600 dark:text-gray-400 lg:h-4 lg:w-4" />
                      <input
                        type="file"
                        accept="image/jpeg,image/jpg,image/png,image/gif,image/webp"
                        onChange={handleAvatarUpload}
                        className="hidden"
                        disabled={isLoading}
                      />
                    </label>
                    {isLoading && (
                      <div className="absolute inset-0 flex items-center justify-center rounded-full bg-black/40">
                        <LoadingAnimation type="spinner" size="sm" color="#ffffff" />
                      </div>
                    )}
                  </div>
                  <div className="min-w-0 flex-1 lg:w-full lg:flex-none lg:text-center">
                    <p className="truncate text-sm font-medium text-slate-900 dark:text-white lg:text-base">
                      {user?.displayName || user?.email?.split('@')[0] || '—'}
                    </p>
                    <p className="mt-1 line-clamp-2 text-xs text-slate-500 dark:text-slate-400 lg:text-sm">{user?.email || '—'}</p>
                  </div>
                  </div>
                  <div className="w-full space-y-2 border-t border-slate-200 pt-3 dark:border-neutral-700 lg:pt-4">
                    <div className="flex items-center justify-between gap-2 text-xs">
                      <span className="text-slate-500 dark:text-slate-400">微信</span>
                      <Badge variant={user?.wechatOpenId ? 'success' : 'warning'} className="text-[10px]">
                        {user?.wechatOpenId ? '已绑定' : '未绑定'}
                      </Badge>
                    </div>
                    <div className="flex items-center justify-between gap-2 text-xs">
                      <span className="text-slate-500 dark:text-slate-400">GitHub</span>
                      <Badge variant={user?.githubId ? 'success' : 'warning'} className="text-[10px]">
                        {user?.githubId ? '已绑定' : '未绑定'}
                      </Badge>
                    </div>
                    <div className="flex items-center justify-between gap-2 text-xs">
                      <span className="text-slate-500 dark:text-slate-400">微信认证</span>
                      <Badge
                        variant={
                          typeof user?.wechatUnionId === 'string' && user.wechatUnionId.trim() !== ''
                            ? 'success'
                            : 'warning'
                        }
                        className="text-[10px]"
                      >
                        {typeof user?.wechatUnionId === 'string' && user.wechatUnionId.trim() !== ''
                          ? '已通过'
                          : '未通过'}
                      </Badge>
                    </div>
                  </div>
                </aside>

                <div className="flex min-h-0 min-w-0 flex-col lg:overflow-hidden">
                  <div className="flex shrink-0 flex-wrap items-center justify-between gap-2 border-b border-slate-100 px-3 py-3 dark:border-neutral-800 sm:px-4">
                    <span className="text-sm font-medium text-slate-800 dark:text-slate-100">{t('profile.basicInfo')}</span>
                    <div className="flex flex-wrap gap-1.5">
                      {!isEditing ? (
                        <Button
                          variant="outline"
                          size="sm"
                          className="h-8 px-2.5 text-xs"
                          leftIcon={<Edit3 className="h-3.5 w-3.5" />}
                          onClick={() => setIsEditing(true)}
                          disabled={isLoading}
                        >
                          {t('profile.edit')}
                        </Button>
                      ) : (
                        <>
                          <Button
                            variant="outline"
                            size="sm"
                            className="h-8 px-2.5 text-xs"
                            leftIcon={<X className="h-3.5 w-3.5" />}
                            onClick={handleCancel}
                            disabled={isLoading}
                          >
                            {t('profile.cancel')}
                          </Button>
                          <Button
                            variant="primary"
                            size="sm"
                            className="h-8 px-2.5 text-xs"
                            leftIcon={<Save className="h-3.5 w-3.5" />}
                            onClick={handleSave}
                            disabled={isLoading}
                          >
                            {isLoading ? t('profile.saving') : t('profile.save')}
                          </Button>
                        </>
                      )}
                    </div>
                  </div>

                  <div className="min-h-0 flex-1 overflow-y-auto overscroll-y-contain p-3 sm:p-4 lg:max-h-full lg:overflow-y-auto lg:p-6">
                    <div className="grid grid-cols-1 gap-3 md:grid-cols-3 md:gap-x-4 md:gap-y-4">
                      <Input
                        size="sm"
                        label={t('profile.displayName')}
                        value={formData.displayName}
                        onChange={(e) => setFormData((prev) => ({ ...prev, displayName: e.target.value }))}
                        disabled={!isEditing}
                        leftIcon={<User className="h-3.5 w-3.5" />}
                        placeholder={t('profile.displayNamePlaceholder')}
                      />
                      <Input
                        size="sm"
                        label={t('profile.firstName')}
                        value={formData.firstName}
                        onChange={(e) => setFormData((prev) => ({ ...prev, firstName: e.target.value }))}
                        disabled={!isEditing}
                        leftIcon={<User className="h-3.5 w-3.5" />}
                        placeholder={t('profile.firstNamePlaceholder')}
                      />
                      <Input
                        size="sm"
                        label={t('profile.lastName')}
                        value={formData.lastName}
                        onChange={(e) => setFormData((prev) => ({ ...prev, lastName: e.target.value }))}
                        disabled={!isEditing}
                        leftIcon={<User className="h-3.5 w-3.5" />}
                        placeholder={t('profile.lastNamePlaceholder')}
                      />
                      <div>
                        <label className="mb-1 block text-xs font-medium text-slate-600 dark:text-gray-400">
                          {t('profile.gender')}
                        </label>
                        <Select
                          value={formData.gender}
                          onValueChange={(v) => setFormData((prev) => ({ ...prev, gender: v }))}
                          disabled={!isEditing}
                        >
                          <SelectTrigger className="h-9 w-full text-sm">
                            <SelectValue placeholder={t('profile.genderSelect')} />
                          </SelectTrigger>
                          <SelectContent>
                            <SelectItem value="">{t('profile.genderSelect')}</SelectItem>
                            <SelectItem value="male">{t('profile.gender.male')}</SelectItem>
                            <SelectItem value="female">{t('profile.gender.female')}</SelectItem>
                            <SelectItem value="other">{t('profile.gender.other')}</SelectItem>
                          </SelectContent>
                        </Select>
                      </div>
                      <Input
                        size="sm"
                        label={t('profile.email')}
                        type="email"
                        value={formData.email}
                        onChange={(e) => setFormData((prev) => ({ ...prev, email: e.target.value }))}
                        disabled={!isEditing}
                        leftIcon={<Mail className="h-3.5 w-3.5" />}
                        placeholder={t('profile.emailPlaceholder')}
                      />
                      <Input
                        size="sm"
                        label={t('profile.phone')}
                        value={formData.phone}
                        onChange={(e) => setFormData((prev) => ({ ...prev, phone: e.target.value }))}
                        disabled={!isEditing}
                        leftIcon={<Phone className="h-3.5 w-3.5" />}
                        placeholder={t('profile.phonePlaceholder')}
                      />
                      <Input
                        size="sm"
                        label="城市"
                        value={formData.city}
                        onChange={(e) => setFormData((prev) => ({ ...prev, city: e.target.value }))}
                        disabled={!isEditing}
                        placeholder="请输入所在城市"
                      />
                      <Input
                        size="sm"
                        label="地区"
                        value={formData.region}
                        onChange={(e) => setFormData((prev) => ({ ...prev, region: e.target.value }))}
                        disabled={!isEditing}
                        placeholder="请输入所在地区"
                      />
                      <div className="col-span-full">
                        <Input
                          size="sm"
                          label={t('profile.bio')}
                          value={formData.extra}
                          onChange={(e) => setFormData((prev) => ({ ...prev, extra: e.target.value }))}
                          disabled={!isEditing}
                          leftIcon={<Heart className="h-3.5 w-3.5" />}
                          placeholder={t('profile.bioPlaceholder')}
                        />
                      </div>
                    </div>
                  </div>
                </div>
              </div>
            </div>
          </div>
        )}

                {section === 'locale' && (
                  <Card>
                    <div className="p-6 space-y-6">
                      <div>
                        <h3 className="text-lg font-semibold text-gray-900 dark:text-white">语言、时区与外观</h3>
                        <p className="text-sm text-gray-500 dark:text-gray-400 mt-1">
                          设置界面语言、默认时区与外观模式；保存后写入账户。外观模式保存后立即应用到本机。
                        </p>
                      </div>
                      <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                        <div>
                          <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
                            {t('profile.timezone')}
                          </label>
                          <Select
                            value={formData.timezone}
                            onValueChange={(v) => setFormData((prev) => ({ ...prev, timezone: v }))}
                          >
                            <SelectTrigger className="w-full">
                              <SelectValue placeholder={t('profile.timezone')} />
                            </SelectTrigger>
                            <SelectContent>
                              <SelectItem value="Asia/Shanghai">Asia/Shanghai</SelectItem>
                              <SelectItem value="Asia/Tokyo">Asia/Tokyo</SelectItem>
                              <SelectItem value="America/New_York">America/New_York</SelectItem>
                              <SelectItem value="Europe/London">Europe/London</SelectItem>
                              <SelectItem value="UTC">UTC</SelectItem>
                            </SelectContent>
                          </Select>
                        </div>
                        <div>
                          <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
                            {t('profile.language')}
                          </label>
                          <Select
                            value={formData.locale}
                            onValueChange={(v) => setFormData((prev) => ({ ...prev, locale: v }))}
                          >
                            <SelectTrigger className="w-full">
                              <SelectValue placeholder={t('profile.language')} />
                            </SelectTrigger>
                            <SelectContent>
                              <SelectItem value="zh">简体中文</SelectItem>
                              <SelectItem value="zh-TW">繁體中文</SelectItem>
                              <SelectItem value="en">English</SelectItem>
                              <SelectItem value="ja">日本語</SelectItem>
                              <SelectItem value="fr">Français</SelectItem>
                            </SelectContent>
                          </Select>
                        </div>
                      </div>
                      <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                        <div>
                          <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
                            外观模式
                          </label>
                          <Select
                            value={formData.themeMode}
                            onValueChange={(v) =>
                              setFormData((prev) => ({ ...prev, themeMode: v as ThemeMode }))
                            }
                          >
                            <SelectTrigger className="w-full">
                              <SelectValue placeholder="外观模式" />
                            </SelectTrigger>
                            <SelectContent>
                              <SelectItem value="system">跟随系统</SelectItem>
                              <SelectItem value="light">浅色</SelectItem>
                              <SelectItem value="dark">深色</SelectItem>
                            </SelectContent>
                          </Select>
                        </div>
                      </div>
                      <Button variant="primary" size="sm" onClick={() => void handleSave()} disabled={isLoading} leftIcon={<Save className="w-4 h-4" />}>
                        {isLoading ? t('profile.saving') : t('profile.save')}
                      </Button>
                    </div>
                  </Card>
                )}

                {/* 账号、第三方绑定与安全 — 各块标题与分栏样式统一 */}
                {section === 'account-security' && (
                  <div className="overflow-hidden rounded-lg border border-slate-200 bg-white dark:border-neutral-800 dark:bg-neutral-950">
                    <div className="border-b border-slate-100 px-4 py-2.5 dark:border-neutral-800 sm:px-5">
                      <h3 className="text-xs font-semibold uppercase tracking-wide text-slate-500 dark:text-gray-400">
                        {t('profile.accountBindingsTitle')}
                      </h3>
                    </div>
                    <div className="divide-y divide-slate-100 dark:divide-neutral-800">
                      <div className="flex flex-wrap items-center justify-between gap-2 px-4 py-2.5 sm:px-5">
                        <span className="text-sm text-slate-800 dark:text-gray-100">微信认证</span>
                        <div className="flex items-center gap-2">
                          <Badge variant={user?.wechatOpenId ? 'success' : 'warning'} className="text-xs">
                            {user?.wechatOpenId ? '已绑定' : '未绑定'}
                          </Badge>
                          {!user?.wechatOpenId && (
                            <button
                              type="button"
                              onClick={handleBindWechat}
                              disabled={isWechatBinding}
                              className="text-xs font-medium text-sky-600 underline decoration-sky-600/40 underline-offset-2 hover:text-sky-700 disabled:opacity-50 dark:text-sky-400"
                            >
                              {isWechatBinding ? '绑定中...' : '立即绑定'}
                            </button>
                          )}
                        </div>
                      </div>
                      {!!wechatBindCode && (
                        <div className="space-y-2 px-4 py-3 sm:px-5">
                          <div className="overflow-hidden rounded-md border border-slate-200 bg-white dark:border-neutral-700">
                            <img
                              src="/qrcode_official_account.jpg"
                              alt="微信公众号二维码"
                              className="max-h-40 w-full object-contain"
                            />
                          </div>
                          <div className="text-xs text-slate-600 dark:text-gray-400">
                            <span className="font-medium text-slate-700 dark:text-gray-300">微信绑定码：</span>
                            <span className="font-mono tracking-wider">{wechatBindCode}</span>
                          </div>
                          <div className="text-xs text-slate-500 dark:text-gray-500">
                            {wechatBindStatus === 'pending' && `请在公众号发送该绑定码，剩余 ${wechatBindCountdown} 秒`}
                            {wechatBindStatus === 'success' && '绑定成功'}
                            {wechatBindStatus === 'failed' && '绑定失败，请重新获取绑定码'}
                            {wechatBindStatus === 'expired' && '绑定码已过期，请重新获取'}
                          </div>
                        </div>
                      )}
                      <div className="flex flex-wrap items-center justify-between gap-2 px-4 py-2.5 sm:px-5">
                        <span className="text-sm text-slate-800 dark:text-gray-100">GitHub 认证</span>
                        <div className="flex items-center gap-2">
                          <Badge variant={user?.githubId ? 'success' : 'warning'} className="text-xs">
                            {user?.githubId ? '已绑定' : '未绑定'}
                          </Badge>
                          {!user?.githubId && (
                            <button
                              type="button"
                              onClick={handleBindGithub}
                              className="text-xs font-medium text-sky-600 underline decoration-sky-600/40 underline-offset-2 hover:text-sky-700 dark:text-sky-400"
                            >
                              立即绑定
                            </button>
                          )}
                        </div>
                      </div>
                      <div className="flex flex-wrap items-center justify-between gap-2 px-4 py-2.5 sm:px-5">
                        <span className="text-sm text-slate-800 dark:text-gray-100">邮箱状态</span>
                        <div className="flex items-center gap-2">
                          <Badge variant={user?.emailVerified ? 'success' : 'warning'} className="text-xs">
                            {user?.emailVerified ? '已验证' : '未验证'}
                          </Badge>
                          {!user?.emailVerified && (
                            <button
                              type="button"
                              onClick={handleSendEmailVerification}
                              disabled={isSendingEmailVerification}
                              className="text-xs font-medium text-sky-600 underline decoration-sky-600/40 underline-offset-2 hover:text-sky-700 disabled:opacity-50 dark:text-sky-400"
                            >
                              {isSendingEmailVerification ? '发送中...' : '验证'}
                            </button>
                          )}
                        </div>
                      </div>
                    </div>

                    <div className="border-t border-slate-100 dark:border-neutral-800">
                      <div className="border-b border-slate-100 px-4 py-2.5 dark:border-neutral-800 sm:px-5">
                        <h3 className="text-xs font-semibold uppercase tracking-wide text-slate-500 dark:text-gray-400">
                          {t('profile.notificationPreferences')}
                        </h3>
                      </div>
                      <div className="divide-y divide-slate-100 dark:divide-neutral-800">
                        <div className="flex flex-wrap items-center justify-between gap-3 px-4 py-2.5 sm:px-5">
                          <div className="min-w-0">
                            <p className="text-sm font-medium text-slate-800 dark:text-gray-100">{t('profile.emailNotifications')}</p>
                            <p className="text-xs text-slate-500 dark:text-gray-400">{t('profile.emailNotificationsDesc')}</p>
                          </div>
                          <Switch
                            checked={user?.emailNotifications || false}
                            onCheckedChange={async (checked) => {
                              updateAuthStore({ emailNotifications: checked })
                              try {
                                const response = await updatePreferences({ emailNotifications: checked })
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
                        <div className="flex flex-wrap items-center justify-between gap-3 px-4 py-2.5 sm:px-5">
                          <div className="min-w-0">
                            <p className="text-sm font-medium text-slate-800 dark:text-gray-100">{t('profile.pushNotifications')}</p>
                            <p className="text-xs text-slate-500 dark:text-gray-400">{t('profile.pushNotificationsDesc')}</p>
                          </div>
                          <Switch
                            checked={user?.pushNotifications || false}
                            onCheckedChange={async (checked) => {
                              updateAuthStore({ pushNotifications: checked })
                              try {
                                const response = await updatePreferences({ pushNotifications: checked })
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
                      </div>
                    </div>

                    <div className="border-t border-slate-100 dark:border-neutral-800">
                      <div className="border-b border-slate-100 px-4 py-2.5 dark:border-neutral-800 sm:px-5">
                        <h3 className="text-xs font-semibold uppercase tracking-wide text-slate-500 dark:text-gray-400">
                          密码与安全
                        </h3>
                      </div>
                      <div className="divide-y divide-slate-100 dark:divide-neutral-800">
                        <div className="flex flex-wrap items-center justify-between gap-3 px-4 py-2.5 sm:px-5">
                          <div className="min-w-0">
                            <p className="text-sm font-medium text-slate-800 dark:text-gray-100">更改密码</p>
                            <p className="text-xs text-slate-500 dark:text-gray-400">定期更新登录密码</p>
                          </div>
                          <Button
                            variant="outline"
                            size="sm"
                            className="h-8 shrink-0 text-xs"
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
                              className="overflow-hidden border-t border-slate-100 bg-slate-50/60 px-4 py-3 dark:border-neutral-800 dark:bg-neutral-900/40 sm:px-5"
                            >
                              <div className="mb-3">
                                <label className="mb-1.5 block text-xs font-medium text-slate-600 dark:text-gray-400">更改密码方式</label>
                                <div className="flex gap-2">
                                  <button
                                    type="button"
                                    onClick={() => {
                                      setPasswordChangeMethod('password')
                                      setEmailCode('')
                                    }}
                                    className={`flex-1 rounded-md px-3 py-1.5 text-xs font-medium transition-colors ${
                                      passwordChangeMethod === 'password'
                                        ? 'bg-slate-900 text-white dark:bg-gray-100 dark:text-gray-900'
                                        : 'bg-white text-slate-700 ring-1 ring-slate-200 dark:bg-neutral-800 dark:text-gray-300 dark:ring-neutral-600'
                                    }`}
                                  >
                                    当前密码
                                  </button>
                                  <button
                                    type="button"
                                    onClick={() => {
                                      setPasswordChangeMethod('email')
                                      setPasswordData((prev) => ({ ...prev, currentPassword: '' }))
                                    }}
                                    className={`flex-1 rounded-md px-3 py-1.5 text-xs font-medium transition-colors ${
                                      passwordChangeMethod === 'email'
                                        ? 'bg-slate-900 text-white dark:bg-gray-100 dark:text-gray-900'
                                        : 'bg-white text-slate-700 ring-1 ring-slate-200 dark:bg-neutral-800 dark:text-gray-300 dark:ring-neutral-600'
                                    }`}
                                  >
                                    邮箱验证码
                                  </button>
                                </div>
                              </div>
                              {passwordChangeMethod === 'password' ? (
                                <Input
                                  size="sm"
                                  label="当前密码"
                                  type={showCurrentPassword ? 'text' : 'password'}
                                  value={passwordData.currentPassword}
                                  onChange={(e) =>
                                    setPasswordData((prev) => ({ ...prev, currentPassword: e.target.value }))
                                  }
                                  leftIcon={<Lock className="h-3.5 w-3.5" />}
                                  rightIcon={
                                    <button
                                      type="button"
                                      onClick={() => setShowCurrentPassword(!showCurrentPassword)}
                                      className="text-slate-400 hover:text-slate-600 dark:hover:text-gray-300"
                                    >
                                      {showCurrentPassword ? <EyeOff className="h-4 w-4" /> : <Eye className="h-4 w-4" />}
                                    </button>
                                  }
                                  placeholder="请输入当前密码"
                                />
                              ) : (
                                <div className="space-y-2">
                                  <label className="block text-xs font-medium text-slate-600 dark:text-gray-400">邮箱验证码</label>
                                  <div className="flex gap-2">
                                    <div className="min-w-0 flex-1">
                                      <Input
                                        size="sm"
                                        label=""
                                        type="text"
                                        value={emailCode}
                                        onChange={(e) => setEmailCode(e.target.value)}
                                        leftIcon={<Mail className="h-3.5 w-3.5" />}
                                        placeholder="请输入邮箱验证码"
                                      />
                                    </div>
                                    <Button
                                      variant="outline"
                                      size="sm"
                                      className="h-9 shrink-0 self-end text-xs"
                                      onClick={handleSendEmailCode}
                                      disabled={isSendingCode || countdown > 0}
                                    >
                                      {countdown > 0 ? `${countdown}秒` : isSendingCode ? '发送中...' : '发送验证码'}
                                    </Button>
                                  </div>
                                  <p className="text-xs text-slate-500 dark:text-gray-500">验证码将发送到 {user?.email}</p>
                                </div>
                              )}
                              <div className="mt-2 space-y-2">
                                <Input
                                  size="sm"
                                  label="新密码"
                                  type={showNewPassword ? 'text' : 'password'}
                                  value={passwordData.newPassword}
                                  onChange={(e) => setPasswordData((prev) => ({ ...prev, newPassword: e.target.value }))}
                                  leftIcon={<Lock className="h-3.5 w-3.5" />}
                                  rightIcon={
                                    <button
                                      type="button"
                                      onClick={() => setShowNewPassword(!showNewPassword)}
                                      className="text-slate-400 hover:text-slate-600 dark:hover:text-gray-300"
                                    >
                                      {showNewPassword ? <EyeOff className="h-4 w-4" /> : <Eye className="h-4 w-4" />}
                                    </button>
                                  }
                                  placeholder="请输入新密码"
                                />
                                <Input
                                  size="sm"
                                  label="确认新密码"
                                  type={showConfirmPassword ? 'text' : 'password'}
                                  value={passwordData.confirmPassword}
                                  onChange={(e) =>
                                    setPasswordData((prev) => ({ ...prev, confirmPassword: e.target.value }))
                                  }
                                  leftIcon={<Lock className="h-3.5 w-3.5" />}
                                  rightIcon={
                                    <button
                                      type="button"
                                      onClick={() => setShowConfirmPassword(!showConfirmPassword)}
                                      className="text-slate-400 hover:text-slate-600 dark:hover:text-gray-300"
                                    >
                                      {showConfirmPassword ? <EyeOff className="h-4 w-4" /> : <Eye className="h-4 w-4" />}
                                    </button>
                                  }
                                  placeholder="请再次输入新密码"
                                />
                              </div>
                              <div className="mt-3 flex gap-2">
                                <Button
                                  variant="outline"
                                  size="sm"
                                  className="h-8 text-xs"
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
                                  size="sm"
                                  className="h-8 text-xs"
                                  onClick={handlePasswordChange}
                                  disabled={isLoading}
                                >
                                  {isLoading ? '修改中...' : '确认修改'}
                                </Button>
                              </div>
                            </motion.div>
                          )}
                        </AnimatePresence>
                        <div className="flex flex-wrap items-center justify-between gap-3 px-4 py-2.5 sm:px-5">
                          <div className="min-w-0">
                            <p className="text-sm font-medium text-slate-800 dark:text-gray-100">两步验证</p>
                            <p className="text-xs text-slate-500 dark:text-gray-400">
                              {user?.twoFactorEnabled ? '已为账户启用额外保护' : '为账户添加额外保护'}
                            </p>
                          </div>
                          <div className="flex shrink-0 gap-2">
                            {user?.twoFactorEnabled ? (
                              <Button
                                variant="outline"
                                size="sm"
                                className="h-8 text-xs text-red-600 ring-red-200 hover:bg-red-50 dark:text-red-400 dark:ring-red-900"
                                onClick={() => setShowTwoFactorDisable(true)}
                                disabled={isTwoFactorLoading}
                              >
                                禁用
                              </Button>
                            ) : (
                              <Button
                                variant="outline"
                                size="sm"
                                className="h-8 text-xs"
                                onClick={handleTwoFactorSetup}
                                disabled={isTwoFactorLoading}
                              >
                                {isTwoFactorLoading ? '设置中...' : '启用'}
                              </Button>
                            )}
                          </div>
                        </div>
                        <div className="flex flex-wrap items-center justify-between gap-3 px-4 py-2.5 sm:px-5">
                          <div className="min-w-0">
                            <p className="text-sm font-medium text-slate-800 dark:text-gray-100">注销账号</p>
                            <p className="text-xs text-slate-500 dark:text-gray-400">冷静期内无法使用，可随时撤销</p>
                          </div>
                          <Link
                            to="/account-deletion/request"
                            className="inline-flex h-8 shrink-0 items-center justify-center rounded-md bg-slate-900 px-3 text-xs font-medium text-white hover:bg-slate-800 dark:bg-gray-100 dark:text-gray-900 dark:hover:bg-white"
                          >
                            前往注销
                          </Link>
                        </div>
                      </div>
                    </div>
                  </div>
                )}

                {section === 'user-devices' && (
                  <div className="mx-auto w-full max-w-3xl space-y-4">
                    <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
                      <div>
                        <h3 className="text-lg font-semibold text-slate-900 dark:text-white">登录设备</h3>
                        <p className="mt-1 text-sm text-slate-500 dark:text-slate-400">
                          当前账号下的会话与浏览器；可信任常用设备或移除不再使用的条目。
                        </p>
                      </div>
                      <Button
                        variant="outline"
                        size="sm"
                        className="shrink-0 self-start sm:self-center"
                        onClick={loadDevices}
                        disabled={isLoadingDevices}
                      >
                        {isLoadingDevices ? '加载中...' : '刷新'}
                      </Button>
                    </div>

                    {isLoadingDevices ? (
                      <div className="flex items-center justify-center rounded-xl border border-slate-200 bg-slate-50/50 py-16 dark:border-neutral-800 dark:bg-neutral-900/40">
                        <LoadingAnimation type="spinner" size="md" />
                        <span className="ml-2 text-sm text-slate-600 dark:text-slate-400">加载中...</span>
                      </div>
                    ) : devices.length === 0 ? (
                      <div className="rounded-xl border border-dashed border-slate-200 bg-slate-50/50 py-14 text-center dark:border-neutral-800 dark:bg-neutral-900/30">
                        <Monitor className="mx-auto h-10 w-10 text-slate-300 dark:text-slate-600" aria-hidden />
                        <p className="mt-3 text-sm text-slate-500 dark:text-slate-400">暂无设备记录</p>
                      </div>
                    ) : (
                      <ul
                        role="list"
                        className="max-h-[min(520px,72vh)] divide-y divide-slate-200 overflow-y-auto overscroll-y-contain rounded-xl border border-slate-200 bg-white shadow-sm dark:divide-neutral-800 dark:border-neutral-800 dark:bg-neutral-950"
                      >
                        {devices.map((device) => {
                          const kind = resolveDeviceVisualKind(device)
                          const { Icon, ring, icon } = DEVICE_VISUAL[kind]
                          const title = device.deviceName || `${device.browser} · ${device.os}`
                          return (
                            <li
                              key={device.id}
                              className="flex flex-col gap-4 px-4 py-4 transition-colors hover:bg-slate-50/90 sm:flex-row sm:items-center sm:gap-4 sm:px-5 dark:hover:bg-neutral-900/60"
                            >
                              <div className="flex min-w-0 flex-1 items-start gap-4">
                                <div
                                  className={`flex h-12 w-12 shrink-0 items-center justify-center rounded-xl bg-slate-50 ring-2 ${ring} dark:bg-neutral-900/80`}
                                  aria-hidden
                                >
                                  <Icon className={`h-6 w-6 ${icon}`} />
                                </div>
                                <div className="min-w-0 flex-1">
                                  <div className="flex flex-wrap items-center gap-2">
                                    <span className="font-medium text-slate-900 dark:text-white">{title}</span>
                                    {device.isTrusted && (
                                      <Badge variant="success" className="text-[11px] font-medium">
                                        已信任
                                      </Badge>
                                    )}
                                  </div>
                                  <p className="mt-1 text-sm text-slate-600 dark:text-slate-400">
                                    {device.os} · {device.browser}
                                  </p>
                                  <p className="mt-1 flex flex-wrap items-center gap-x-3 gap-y-0.5 text-xs text-slate-500 dark:text-slate-500">
                                    <span className="inline-flex items-center gap-1">
                                      <MapPin className="h-3.5 w-3.5 shrink-0 opacity-70" aria-hidden />
                                      {device.location || device.ipAddress || '—'}
                                    </span>
                                    <span className="text-slate-400 dark:text-slate-600" aria-hidden>
                                      ·
                                    </span>
                                    <span>
                                      最后使用 {new Date(device.lastUsedAt).toLocaleString('zh-CN')}
                                    </span>
                                  </p>
                                </div>
                              </div>
                              <div className="flex shrink-0 flex-wrap items-center gap-2 border-t border-slate-100 pt-3 sm:border-t-0 sm:pt-0 dark:border-neutral-800">
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
                            </li>
                          )
                        })}
                      </ul>
                    )}
                  </div>
                )}

                {section === 'activity' && (
                  <ProfileAuditLogPanel
                    userId={
                      user?.ID != null
                        ? Number(user.ID)
                        : user?.id != null
                          ? Number(user.id)
                          : undefined
                    }
                  />
                )}

                {section === 'teams' && <TeamWorkspacePage />}

                {section === 'billing' && <Billing />}

                {section === 'notifications' && <NotificationCenter />}

                {section === 'credential' && <CredentialManager />}

                {section === 'llm-tokens' && <LLMTokenManager />}
            </FadeIn>

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
    </>
  )
}

export default Profile
