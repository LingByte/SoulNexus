import { useState, useEffect } from 'react'
import PageHeader from '@/components/Layout/PageHeader'
import { Shield, Globe, QrCode, Key, RefreshCw } from 'lucide-react'
import Card from '@/components/UI/Card'
import Button from '@/components/UI/Button'
import Input from '@/components/UI/Input'
import { getConfig, getTwoFactorStatus, setupTwoFactor, enableTwoFactor, disableTwoFactor } from '@/services/adminApi'
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
  // 站点配置
  const [siteConfig, setSiteConfig] = useState<Record<SiteKey, string>>({
    SITE_NAME: '',
    SITE_DESCRIPTION: '',
    SITE_URL: '',
    SITE_LOGO_URL: '',
  })
  const [siteConfigLoading, setSiteConfigLoading] = useState(false)
  
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
          const val = results[i]?.value ?? results[i]?.Value
          if (val !== undefined) cfg[key] = val
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


  // 加载2FA状态
  useEffect(() => {
    const load2FAStatus = async () => {
      try {
        const status = await getTwoFactorStatus()
        setTwoFactorStatus(status)
      } catch (error) {
        console.error('加载2FA状态失败:', error)
      }
    }
    load2FAStatus()
  }, [])


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
      setTwoFactorSetup(null)
      setTwoFactorCode('')
    } catch (error: any) {
      showAlert('禁用2FA失败', 'error', error?.message || error?.msg)
    } finally {
      setLoading2FA(false)
    }
  }

  return (
    <><PageHeader title="系统设置" description="管理系统配置和偏好设置" />
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
                      disabled
                      placeholder={`请输入${SITE_LABELS[key]}`}
                    />
                  </div>
                ))}
              </>
            )}
          </div>
        </Card>

        {/* 通知设置已迁移到「通知中心 → 渠道供应商 / 邮件模板 / 邮件日志」，此处不再重复展示。 */}

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
    </>
  )
}

export default Settings

