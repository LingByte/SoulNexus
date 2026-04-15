import { useState, useEffect } from 'react'
import { Save, Bell, Shield, Globe, Mail, Eye, EyeOff, QrCode, Key, RefreshCw } from 'lucide-react'
import AdminLayout from '@/components/Layout/AdminLayout'
import Card from '@/components/UI/Card'
import Button from '@/components/UI/Button'
import Input from '@/components/UI/Input'
import { Switch } from '@/components/UI/Switch'
import { getConfig, createConfig, updateConfig, getTwoFactorStatus, setupTwoFactor, enableTwoFactor, disableTwoFactor } from '@/services/adminApi'
import { showAlert } from '@/utils/notification'

// 站点配置 key 列表
const SITE_KEYS = ['SITE_NAME', 'SITE_DESCRIPTION', 'SITE_URL', 'SITE_LOGO_URL'] as const
type SiteKey = typeof SITE_KEYS[number]

const SITE_LABELS: Record<SiteKey, string> = {
  SITE_NAME: '站点名称',
  SITE_DESCRIPTION: '站点描述',
  SITE_URL: '站点地址',
  SITE_LOGO_URL: 'Logo URL',
}

const Settings = () => {
  const [settings, setSettings] = useState({
    siteName: 'LingStorage',
    siteDescription: 'Ling Global File Storage Gateway',
    emailNotifications: true,
    pushNotifications: false,
    twoFactorAuth: false,
    language: 'zh-CN',
    timezone: 'Asia/Shanghai',
  })

  // 站点配置
  const [siteConfig, setSiteConfig] = useState<Record<SiteKey, string>>({
    SITE_NAME: '',
    SITE_DESCRIPTION: '',
    SITE_URL: '',
    SITE_LOGO_URL: '',
  })
  const [siteConfigLoading, setSiteConfigLoading] = useState(false)
  const [siteConfigSaving, setSiteConfigSaving] = useState(false)
  const [mailConfig, setMailConfig] = useState<{
    host?: string
    username?: string
    password?: string
    port?: string
    from?: string
  } | null>(null)
  const [mailFormData, setMailFormData] = useState({
    host: '',
    username: '',
    password: '',
    port: '587',
    from: '',
  })
  const [showPassword, setShowPassword] = useState(false)
  const [loading, setLoading] = useState(false)
  const [savingMail, setSavingMail] = useState(false)
  
  // 2FA states
  const [twoFactorStatus, setTwoFactorStatus] = useState<{
    enabled: boolean
    hasSecret: boolean
  } | null>(null)
  const [twoFactorSetup, setTwoFactorSetup] = useState<{
    secret: string
    qrCode: string
    url: string
  } | null>(null)
  const [twoFactorCode, setTwoFactorCode] = useState('')
  const [loading2FA, setLoading2FA] = useState(false)

  // 加载站点配置
  useEffect(() => {
    const loadSiteConfig = async () => {
      setSiteConfigLoading(true)
      try {
        const results = await Promise.all(
          SITE_KEYS.map(key => getConfig(key).catch(() => null))
        )
        const cfg = { ...siteConfig }
        SITE_KEYS.forEach((key, i) => {
          if (results[i]?.value !== undefined) cfg[key] = results[i]!.value
        })
        setSiteConfig(cfg)
      } catch (e) {
        // ignore
      } finally {
        setSiteConfigLoading(false)
      }
    }
    loadSiteConfig()
  }, [])

  const handleSaveSiteConfig = async () => {
    setSiteConfigSaving(true)
    try {
      for (const key of SITE_KEYS) {
        const value = siteConfig[key]
        try {
          await getConfig(key)
          await updateConfig(key, { value, desc: SITE_LABELS[key], autoload: true, public: true })
        } catch {
          await createConfig({ key, value, desc: SITE_LABELS[key], format: 'text', autoload: true, public: true })
        }
      }
      showAlert('站点配置保存成功', 'success')
    } catch (error: any) {
      showAlert('保存失败', 'error', error?.msg || error?.message)
    } finally {
      setSiteConfigSaving(false)
    }
  }

  // 加载邮件配置
  useEffect(() => {
    const loadMailConfig = async () => {
      if (!settings.emailNotifications) {
        setMailConfig(null)
        return
      }
      try {
        setLoading(true)
        const [host, username, password, port, from] = await Promise.all([
          getConfig('MAIL_HOST').catch(() => null),
          getConfig('MAIL_USERNAME').catch(() => null),
          getConfig('MAIL_PASSWORD').catch(() => null),
          getConfig('MAIL_PORT').catch(() => null),
          getConfig('MAIL_FROM').catch(() => null),
        ])
        
        const config: any = {}
        // 检查配置是否存在（即使 value 为空，只要配置对象存在就说明已配置）
        if (host) {
          if (host.value) {
            config.host = host.value
            setMailFormData(prev => ({ ...prev, host: host.value }))
          } else {
            // 配置存在但值为空（可能是非公开配置），显示占位符
            config.host = '****'
            setMailFormData(prev => ({ ...prev, host: '****' }))
          }
        }
        if (username) {
          if (username.value) {
            config.username = username.value
            setMailFormData(prev => ({ ...prev, username: username.value }))
          } else {
            // 配置存在但值为空（可能是非公开配置），显示占位符
            config.username = '****'
            setMailFormData(prev => ({ ...prev, username: '****' }))
          }
        }
        if (password) {
          // 密码字段始终显示占位符，不显示真实值
          config.password = '****'
          setMailFormData(prev => ({ ...prev, password: '****' }))
        }
        if (port) {
          if (port.value) {
            config.port = port.value
            setMailFormData(prev => ({ ...prev, port: port.value }))
          } else {
            // 配置存在但值为空（可能是非公开配置），显示占位符
            config.port = '****'
            setMailFormData(prev => ({ ...prev, port: '****' }))
          }
        }
        if (from) {
          if (from.value) {
            config.from = from.value
            setMailFormData(prev => ({ ...prev, from: from.value }))
          } else {
            // 配置存在但值为空（可能是非公开配置），显示占位符
            config.from = '****'
            setMailFormData(prev => ({ ...prev, from: '****' }))
          }
        }
        
        if (Object.keys(config).length > 0) {
          setMailConfig(config)
        } else {
          setMailConfig(null)
        }
      } catch (error) {
        console.error('加载邮件配置失败:', error)
        setMailConfig(null)
      } finally {
        setLoading(false)
      }
    }
    loadMailConfig()
  }, [settings.emailNotifications])

  // 加载2FA状态
  useEffect(() => {
    const load2FAStatus = async () => {
      try {
        const status = await getTwoFactorStatus()
        setTwoFactorStatus(status)
        setSettings(prev => ({ ...prev, twoFactorAuth: status.enabled }))
      } catch (error) {
        console.error('加载2FA状态失败:', error)
      }
    }
    load2FAStatus()
  }, [])

  // 保存邮件配置
  const handleSaveMailConfig = async () => {
    // 检查必填字段（排除占位符）
    const hasHost = mailFormData.host && mailFormData.host !== '****'
    const hasUsername = mailFormData.username && mailFormData.username !== '****'
    const hasFrom = mailFormData.from && mailFormData.from !== '****'
    
    if (!hasHost || !hasUsername || !hasFrom) {
      showAlert('请填写完整的邮件配置信息', 'error')
      return
    }
    
    // 检查是否有实际值需要更新（排除占位符）
    const shouldUpdateHost = mailFormData.host && mailFormData.host !== '****'
    const shouldUpdateUsername = mailFormData.username && mailFormData.username !== '****'
    const shouldUpdatePort = mailFormData.port && mailFormData.port !== '****'
    const shouldUpdateFrom = mailFormData.from && mailFormData.from !== '****'
    const shouldUpdatePassword = mailFormData.password && mailFormData.password !== '****'
    
    // 如果是新配置，密码是必需的
    if (!mailConfig && !shouldUpdatePassword) {
      showAlert('请填写SMTP密码', 'error')
      return
    }
    
    try {
      setSavingMail(true)
      const configs: Array<{ key: string; value: string; desc: string; format: 'text' | 'int'; autoload: boolean; public: boolean }> = []
      
      // 只有在需要更新时才添加配置
      if (shouldUpdateHost) {
        configs.push({ key: 'MAIL_HOST', value: mailFormData.host, desc: 'Mail SMTP Host', format: 'text' as const, autoload: true, public: false })
      }
      if (shouldUpdateUsername) {
        configs.push({ key: 'MAIL_USERNAME', value: mailFormData.username, desc: 'Mail SMTP Username', format: 'text' as const, autoload: true, public: false })
      }
      if (shouldUpdatePort) {
        configs.push({ key: 'MAIL_PORT', value: mailFormData.port, desc: 'Mail SMTP Port', format: 'int' as const, autoload: true, public: false })
      }
      if (shouldUpdateFrom) {
        configs.push({ key: 'MAIL_FROM', value: mailFormData.from, desc: 'Mail From Address', format: 'text' as const, autoload: true, public: false })
      }
      if (shouldUpdatePassword) {
        configs.push({ key: 'MAIL_PASSWORD', value: mailFormData.password, desc: 'Mail SMTP Password', format: 'text' as const, autoload: true, public: false })
      }
      
      for (const cfg of configs) {
        try {
          await getConfig(cfg.key)
          // 存在则更新
          await updateConfig(cfg.key, { value: cfg.value })
        } catch {
          // 不存在则创建
          await createConfig(cfg)
        }
      }
      
      showAlert('邮件配置保存成功', 'success')
      // 重新加载配置以获取最新值
      const [host, username, password, port, from] = await Promise.all([
        getConfig('MAIL_HOST').catch(() => null),
        getConfig('MAIL_USERNAME').catch(() => null),
        getConfig('MAIL_PASSWORD').catch(() => null),
        getConfig('MAIL_PORT').catch(() => null),
        getConfig('MAIL_FROM').catch(() => null),
      ])
      
      const updatedConfig: any = {}
      if (host?.value) {
        updatedConfig.host = host.value
        setMailFormData(prev => ({ ...prev, host: host.value }))
      }
      if (username?.value) {
        updatedConfig.username = username.value
        setMailFormData(prev => ({ ...prev, username: username.value }))
      }
      if (password?.value) {
        updatedConfig.password = password.value
        setMailFormData(prev => ({ ...prev, password: '****' }))
      }
      if (port?.value) {
        updatedConfig.port = port.value
        setMailFormData(prev => ({ ...prev, port: port.value }))
      }
      if (from?.value) {
        updatedConfig.from = from.value
        setMailFormData(prev => ({ ...prev, from: from.value }))
      }
      setMailConfig(updatedConfig)
    } catch (error: any) {
      showAlert('保存邮件配置失败', 'error', error?.msg || error?.message)
    } finally {
      setSavingMail(false)
    }
  }

  // 设置2FA
  const handleSetup2FA = async () => {
    try {
      setLoading2FA(true)
      const setup = await setupTwoFactor()
      setTwoFactorSetup(setup)
      setTwoFactorStatus(prev => prev ? { ...prev, hasSecret: true } : { enabled: false, hasSecret: true })
    } catch (error: any) {
      showAlert('设置2FA失败', 'error', error?.message || error?.msg)
    } finally {
      setLoading2FA(false)
    }
  }

  // 启用2FA
  const handleEnable2FA = async () => {
    if (!twoFactorCode) {
      showAlert('请输入验证码', 'error')
      return
    }
    try {
      setLoading2FA(true)
      await enableTwoFactor(twoFactorCode)
      showAlert('2FA已启用', 'success')
      setTwoFactorStatus(prev => prev ? { ...prev, enabled: true } : { enabled: true, hasSecret: true })
      setSettings(prev => ({ ...prev, twoFactorAuth: true }))
      setTwoFactorSetup(null)
      setTwoFactorCode('')
    } catch (error: any) {
      showAlert('启用2FA失败', 'error', error?.message || error?.msg)
    } finally {
      setLoading2FA(false)
    }
  }

  // 禁用2FA
  const handleDisable2FA = async () => {
    if (!twoFactorCode) {
      showAlert('请输入验证码', 'error')
      return
    }
    try {
      setLoading2FA(true)
      await disableTwoFactor(twoFactorCode)
      showAlert('2FA已禁用', 'success')
      setTwoFactorStatus(prev => prev ? { ...prev, enabled: false, hasSecret: false } : { enabled: false, hasSecret: false })
      setSettings(prev => ({ ...prev, twoFactorAuth: false }))
      setTwoFactorSetup(null)
      setTwoFactorCode('')
    } catch (error: any) {
      showAlert('禁用2FA失败', 'error', error?.message || error?.msg)
    } finally {
      setLoading2FA(false)
    }
  }

  return (
    <AdminLayout
      title="系统设置"
      description="管理系统配置和偏好设置"
    >
      <div className="space-y-6">
        {/* 基本设置 */}
        <Card className="p-6">
          <div className="flex items-center gap-3 mb-6">
            <div className="w-10 h-10 rounded-lg bg-blue-100 dark:bg-blue-900/20 flex items-center justify-center">
              <Globe className="w-5 h-5 text-blue-600 dark:text-blue-400" />
            </div>
            <div>
              <h3 className="text-lg font-semibold text-slate-900 dark:text-white">
                基本设置
              </h3>
              <p className="text-sm text-slate-500 dark:text-slate-400">
                配置系统基本信息
              </p>
            </div>
          </div>

          <div className="space-y-4">
            {siteConfigLoading ? (
              <div className="flex items-center gap-2 text-sm text-slate-500">
                <RefreshCw className="w-4 h-4 animate-spin" /> 加载中...
              </div>
            ) : (
              <>
                {SITE_KEYS.map(key => (
                  <div key={key}>
                    <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">
                      {SITE_LABELS[key]}
                    </label>
                    <Input
                      value={siteConfig[key]}
                      onChange={(e) => setSiteConfig({ ...siteConfig, [key]: e.target.value })}
                      placeholder={`请输入${SITE_LABELS[key]}`}
                    />
                  </div>
                ))}
                <div className="flex justify-end pt-2">
                  <Button
                    variant="primary"
                    onClick={handleSaveSiteConfig}
                    disabled={siteConfigSaving}
                    leftIcon={siteConfigSaving ? <RefreshCw className="w-4 h-4 animate-spin" /> : <Save className="w-4 h-4" />}
                  >
                    {siteConfigSaving ? '保存中...' : '保存'}
                  </Button>
                </div>
              </>
            )}
          </div>
        </Card>

        {/* 通知设置 */}
        <Card className="p-6">
          <div className="flex items-center gap-3 mb-6">
            <div className="w-10 h-10 rounded-lg bg-green-100 dark:bg-green-900/20 flex items-center justify-center">
              <Bell className="w-5 h-5 text-green-600 dark:text-green-400" />
            </div>
            <div>
              <h3 className="text-lg font-semibold text-slate-900 dark:text-white">
                通知设置
              </h3>
              <p className="text-sm text-slate-500 dark:text-slate-400">
                管理通知偏好
              </p>
            </div>
          </div>

          <div className="space-y-4">
            <div className="flex items-center justify-between">
              <div>
                <p className="text-sm font-medium text-slate-900 dark:text-white">
                  邮件通知
                </p>
                <p className="text-xs text-slate-500 dark:text-slate-400">
                  接收重要事件的邮件通知
                </p>
              </div>
              <Switch
                checked={settings.emailNotifications}
                onCheckedChange={(checked) =>
                  setSettings({ ...settings, emailNotifications: checked })
                }
              />
            </div>
            
            {settings.emailNotifications && loading && (
              <div className="mt-4 text-xs text-slate-500 dark:text-slate-400">
                加载邮件配置中...
              </div>
            )}
            {/* 邮件配置表单（始终显示，可编辑） */}
            {settings.emailNotifications && !loading && (
              <div className="mt-4 p-4 bg-slate-50 dark:bg-slate-800/50 rounded-lg border border-slate-200 dark:border-slate-700">
                <div className="flex items-center gap-2 mb-4">
                  <Mail className="w-4 h-4 text-slate-600 dark:text-slate-400" />
                  <p className="text-sm font-medium text-slate-900 dark:text-white">
                    {mailConfig ? '邮件服务器配置' : '配置邮件服务器'}
                  </p>
                </div>
                <div className="space-y-3">
                  <Input
                    label="SMTP 服务器 *"
                    value={mailFormData.host}
                    onChange={(e) => setMailFormData({ ...mailFormData, host: e.target.value })}
                    placeholder="smtp.example.com"
                    required
                  />
                  <div className="grid grid-cols-2 gap-3">
                    <Input
                      label="端口 *"
                      value={mailFormData.port}
                      onChange={(e) => setMailFormData({ ...mailFormData, port: e.target.value })}
                      placeholder="587"
                      type="number"
                      required
                    />
                    <Input
                      label="用户名 *"
                      value={mailFormData.username}
                      onChange={(e) => setMailFormData({ ...mailFormData, username: e.target.value })}
                      placeholder="user@example.com"
                      required
                    />
                  </div>
                  <Input
                    label="密码 *"
                    value={mailFormData.password}
                    onChange={(e) => {
                      // 如果当前是占位符，清空让用户输入新密码
                      const newValue = e.target.value === '****' ? '' : e.target.value
                      setMailFormData({ ...mailFormData, password: newValue })
                    }}
                    onFocus={(e) => {
                      // 如果当前是占位符，聚焦时清空
                      if (e.target.value === '****') {
                        setMailFormData({ ...mailFormData, password: '' })
                      }
                    }}
                    type={showPassword ? 'text' : 'password'}
                    placeholder={mailConfig && mailConfig.password ? '留空不修改密码' : '请输入SMTP密码'}
                    required={!mailConfig || !mailConfig.password}
                    rightIcon={
                      <button
                        type="button"
                        onClick={() => setShowPassword(!showPassword)}
                        className="text-slate-400 hover:text-slate-600 dark:hover:text-slate-300"
                      >
                        {showPassword ? <EyeOff className="w-4 h-4" /> : <Eye className="w-4 h-4" />}
                      </button>
                    }
                  />
                  {mailConfig && mailConfig.password && mailFormData.password === '****' && (
                    <p className="text-xs text-slate-500 dark:text-slate-400">
                      密码已配置，留空不修改，输入新值将更新密码
                    </p>
                  )}
                  <Input
                    label="发件人邮箱 *"
                    value={mailFormData.from}
                    onChange={(e) => {
                      const newValue = e.target.value === '****' ? '' : e.target.value
                      setMailFormData({ ...mailFormData, from: newValue })
                    }}
                    onFocus={(e) => {
                      if (e.target.value === '****') {
                        setMailFormData({ ...mailFormData, from: '' })
                      }
                    }}
                    placeholder="noreply@example.com"
                    type="email"
                    required
                  />
                  <div className="flex justify-end pt-2">
                    <Button
                      variant="primary"
                      size="sm"
                      onClick={handleSaveMailConfig}
                      disabled={savingMail}
                      leftIcon={<Save className="w-4 h-4" />}
                    >
                      {savingMail ? '保存中...' : '保存配置'}
                    </Button>
                  </div>
                </div>
              </div>
            )}
          </div>
        </Card>

        {/* 安全设置 */}
        <Card className="p-6">
          <div className="flex items-center gap-3 mb-6">
            <div className="w-10 h-10 rounded-lg bg-red-100 dark:bg-red-900/20 flex items-center justify-center">
              <Shield className="w-5 h-5 text-red-600 dark:text-red-400" />
            </div>
            <div>
              <h3 className="text-lg font-semibold text-slate-900 dark:text-white">
                安全设置
              </h3>
              <p className="text-sm text-slate-500 dark:text-slate-400">
                管理账户安全选项
              </p>
            </div>
          </div>

          <div className="space-y-4">
            <div>
              <p className="text-sm font-medium text-slate-900 dark:text-white">
                双因素认证
              </p>
              <p className="text-xs text-slate-500 dark:text-slate-400">
                为账户添加额外的安全层
              </p>
            </div>

            {/* 2FA设置流程 */}
            {!twoFactorStatus?.enabled && !twoFactorSetup && (
              <div className="mt-4 p-4 bg-slate-50 dark:bg-slate-800/50 rounded-lg border border-slate-200 dark:border-slate-700">
                <p className="text-sm text-slate-600 dark:text-slate-400 mb-3">
                  双因素认证未启用。点击下方按钮开始设置。
                </p>
                <Button
                  variant="primary"
                  size="sm"
                  onClick={handleSetup2FA}
                  disabled={loading2FA}
                  leftIcon={<Key className="w-4 h-4" />}
                >
                  {loading2FA ? '设置中...' : '设置2FA'}
                </Button>
              </div>
            )}

            {/* 显示QR码和验证码输入 */}
            {twoFactorSetup && !twoFactorStatus?.enabled && (
              <div className="mt-4 p-4 bg-slate-50 dark:bg-slate-800/50 rounded-lg border border-slate-200 dark:border-slate-700">
                <div className="flex items-center gap-2 mb-3">
                  <QrCode className="w-4 h-4 text-slate-600 dark:text-slate-400" />
                  <p className="text-sm font-medium text-slate-900 dark:text-white">
                    扫描二维码完成设置
                  </p>
                </div>
                <div className="space-y-4">
                  <div className="flex justify-center">
                    <img src={twoFactorSetup.qrCode} alt="2FA QR Code" className="border border-slate-200 dark:border-slate-700 rounded-lg" />
                  </div>
                  <div className="text-center">
                    <p className="text-xs text-slate-600 dark:text-slate-400 mb-2">
                      或手动输入密钥：
                    </p>
                    <p className="text-xs font-mono text-slate-900 dark:text-white bg-slate-100 dark:bg-slate-800 p-2 rounded">
                      {twoFactorSetup.secret}
                    </p>
                  </div>
                  <div>
                    <Input
                      label="验证码"
                      value={twoFactorCode}
                      onChange={(e) => setTwoFactorCode(e.target.value)}
                      placeholder="输入6位验证码"
                      maxLength={6}
                    />
                    <p className="text-xs text-slate-500 dark:text-slate-400 mt-1">
                      请使用您的身份验证器应用扫描上方二维码，然后输入生成的6位验证码
                    </p>
                  </div>
                  <div className="flex gap-2">
                    <Button
                      variant="primary"
                      size="sm"
                      onClick={handleEnable2FA}
                      disabled={loading2FA || !twoFactorCode}
                    >
                      {loading2FA ? '启用中...' : '启用2FA'}
                    </Button>
                    <Button
                      variant="outline"
                      size="sm"
                      onClick={() => {
                        setTwoFactorSetup(null)
                        setTwoFactorCode('')
                      }}
                    >
                      取消
                    </Button>
                  </div>
                </div>
              </div>
            )}

            {/* 已启用2FA，显示禁用选项 */}
            {twoFactorStatus?.enabled && (
              <div className="mt-4 p-4 bg-slate-50 dark:bg-slate-800/50 rounded-lg border border-slate-200 dark:border-slate-700">
                <div className="flex items-center gap-2 mb-3">
                  <Shield className="w-4 h-4 text-green-600 dark:text-green-400" />
                  <p className="text-sm font-medium text-green-600 dark:text-green-400">
                    双因素认证已启用
                  </p>
                </div>
                <div className="space-y-3">
                  <Input
                    label="验证码（用于禁用2FA）"
                    value={twoFactorCode}
                    onChange={(e) => setTwoFactorCode(e.target.value)}
                    placeholder="输入6位验证码"
                    maxLength={6}
                  />
                  <Button
                    variant="destructive"
                    size="sm"
                    onClick={handleDisable2FA}
                    disabled={loading2FA || !twoFactorCode}
                  >
                    {loading2FA ? '禁用中...' : '禁用2FA'}
                  </Button>
                </div>
              </div>
            )}
          </div>
        </Card>
      </div>
    </AdminLayout>
  )
}

export default Settings

