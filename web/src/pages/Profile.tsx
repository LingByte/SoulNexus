import { useEffect, useMemo, useState } from 'react'
import {
  Avatar,
  Card,
  Form,
  Menu,
  Space,
  Tag,
  Upload,
} from '@arco-design/web-react'
import { Button, Input } from '@/components/ui'
import { showAlert } from '@/utils/notification'
import dayjs from 'dayjs'
import { UserCircle } from 'lucide-react'
import { useNavigate, useParams, Navigate } from 'react-router-dom'
import BaseLayout from '@/components/Layout/BaseLayout'
import AccountSecurityPanel from '@/components/profile/AccountSecurityPanel'
import UserDevicesPanel from '@/components/profile/UserDevicesPanel'
import InboxPanel from '@/components/profile/InboxPanel'
import AIReportsPanel from '@/components/profile/AIReportsPanel'
import LoginHistoryPanel from '@/components/profile/LoginHistoryPanel'
import NotificationChannelsPanel from '@/components/profile/NotificationChannelsPanel'
import OperationLogs from '@/pages/OperationLogs'
import AIInvocationLogs from '@/pages/AIInvocationLogs'
import AccessKeys from '@/pages/AccessKeys'
import { fetchMe, updateMe as updateMeApi, uploadMyAvatar } from '@/api/me'
import { useAuthStore } from '@/stores/authStore'
import { useSiteConfig } from '@/contexts/siteConfig'
import { useTranslation } from '@/i18n'
import { extractApiErrorMessage } from '@/utils/apiError'
import { Link } from '@/components/ui'

const FormItem = Form.Item

type ProfileSection = 'info' | 'security' | 'devices' | 'inbox' | 'reports' | 'logs' | 'ai-invocations' | 'access-keys' | 'login-history' | 'notifications'

const SECTION_ALIASES: Record<string, ProfileSection> = {
  info: 'info',
  profile: 'info',
  security: 'security',
  account: 'security',
  devices: 'devices',
  inbox: 'inbox',
  reports: 'reports',
  'ai-reports': 'reports',
  logs: 'logs',
  'operation-logs': 'logs',
  'ai-invocations': 'ai-invocations',
  aiInvocations: 'ai-invocations',
  'access-keys': 'access-keys',
  accessKeys: 'access-keys',
  'login-history': 'login-history',
  loginHistory: 'login-history',
  notifications: 'notifications',
  // legacy aliases → merged notifications page
  webhooks: 'notifications',
  'im-channels': 'notifications',
  imChannels: 'notifications',
}

function canViewWebhooks(me: any): boolean {
  if (me?.principal === 'platform') return false
  const codes = me?.permissionCodes as string[] | undefined
  if (!codes?.length) return false
  return codes.includes('api.webhooks.read') || codes.includes('api.webhooks.write') || codes.includes('*')
}

function canViewAccessKeys(me: any, deploymentMode?: string): boolean {
  if (me?.principal === 'platform') return false
  if (deploymentMode === 'community') return true
  const codes = me?.permissionCodes as string[] | undefined
  if (!codes?.length) return false
  return codes.includes('menu.acc.keys') || codes.includes('api.credentials.read') || codes.includes('*')
}

function canViewAIReports(me: any): boolean {
  if (me?.principal === 'platform') return false
  const codes = me?.permissionCodes as string[] | undefined
  if (!codes?.length) return false
  return (
    codes.includes('api.reports.read') ||
    codes.includes('menu.profile.reports') ||
    codes.includes('*')
  )
}

function canViewOperationLogs(me: any): boolean {
  if (me?.principal === 'platform') return false
  const codes = me?.permissionCodes as string[] | undefined
  if (!codes?.length) return true
  return (
    codes.includes('api.operation_logs.read') ||
    codes.includes('menu.profile.logs') ||
    codes.includes('*')
  )
}

function fmtLastLogin(iso?: string) {
  if (!iso) return '-'
  const d = dayjs(iso)
  return d.isValid() ? d.format('YYYY-MM-DD HH:mm:ss') : '-'
}

function fmtStatus(s?: string, t?: (key: string) => string) {
  const v = String(s || '').toLowerCase()
  if (v === 'active') return t?.('profile.statusActive') ?? '正常'
  if (v === 'disabled') return t?.('profile.statusDisabled') ?? '已停用'
  if (v === 'pending') return t?.('profile.statusPending') ?? '待激活'
  return s || '-'
}

function canViewAIInvocations(me: any): boolean {
  if (me?.principal === 'platform') return false
  const codes = me?.permissionCodes as string[] | undefined
  if (!codes?.length) return true
  return (
    codes.includes('api.ai_invocations.read') ||
    codes.includes('menu.profile.ai_invocations') ||
    codes.includes('*')
  )
}

export default function Profile() {
  const { t } = useTranslation()
  const { config: siteConfig } = useSiteConfig()
  const [loading, setLoading] = useState(false)
  const [me, setMe] = useState<any>(null)
  const [profileForm] = Form.useForm()
  const [editingProfile, setEditingProfile] = useState(false)
  const updateLocalProfile = useAuthStore((s) => s.updateProfile)
  const logout = useAuthStore((s) => s.logout)
  const navigate = useNavigate()
  const { section: sectionParam } = useParams<{ section?: string }>()
  const activeSection: ProfileSection = SECTION_ALIASES[String(sectionParam || 'info')] || 'info'

  const loadMe = async () => {
    setLoading(true)
    try {
      const res = await fetchMe()
      if (res.code !== 200 || !res.data) {
        showAlert(res.msg || t('profile.loadFailed'), 'error')
        return
      }
      setMe(res.data)
      const d = res.data
      if (d.principal === 'platform' && d.platformAdmin) {
        profileForm.setFieldsValue({
          displayName: d.platformAdmin.displayName || '',
          username: '',
          phone: '',
        })
        updateLocalProfile({
          id: d.platformAdmin.id,
          email: d.platformAdmin.email,
          displayName: d.platformAdmin.displayName,
          isPlatformAdmin: true,
          principal: 'platform',
        })
      } else if (d.principal === 'tenant' && d.user) {
        profileForm.setFieldsValue({
          displayName: d.user.displayName || '',
          username: d.user.username || '',
          phone: d.user.phone || '',
        })
        updateLocalProfile({
          ...d.user,
          tenantSlug: d.tenant?.slug,
          tenantName: d.tenant?.name,
          principal: 'tenant',
          permissionCodes: d.permissionCodes ?? [],
        })
      }
    } catch (e: any) {
      showAlert(extractApiErrorMessage(e, t('profile.loadFailed')), 'error')
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    void loadMe()
  }, [])

  const isPlatform = me?.principal === 'platform'
  const showAccessKeys = canViewAccessKeys(me, siteConfig.deploymentMode)
  const showWebhooks = canViewWebhooks(me)
  const showAIReports = canViewAIReports(me)
  const showOperationLogs = canViewOperationLogs(me)
  const showAIInvocations = canViewAIInvocations(me)

  const navItems = useMemo(() => {
    const items: { key: ProfileSection; label: string }[] = [
      { key: 'info', label: t('profile.navProfile') },
      { key: 'security', label: t('profile.navAccountSecurity') },
    ]
    if (!isPlatform) {
      items.push({ key: 'devices', label: t('profile.navDeviceManagement') })
    }
    if (showAccessKeys) {
      items.push({ key: 'access-keys', label: t('profile.navAccessKeys') })
    }
    if (showWebhooks) {
      items.push({ key: 'notifications', label: t('profile.navNotifications') })
    }
    items.push({ key: 'login-history', label: t('profile.navLoginHistory') })
    items.push({ key: 'inbox', label: t('profile.navInbox') })
    if (showAIReports) {
      items.push({ key: 'reports', label: t('profile.navAiReports') })
    }
    if (showOperationLogs) {
      items.push({ key: 'logs', label: t('profile.navOperationLogs') })
    }
    if (showAIInvocations) {
      items.push({ key: 'ai-invocations', label: t('profile.navAiInvocations') })
    }
    return items
  }, [isPlatform, showAccessKeys, showWebhooks, showAIReports, showOperationLogs, showAIInvocations, t])

  const goSection = (key: ProfileSection) => {
    navigate(`/profile/${key}`, { replace: true })
  }

  const resetProfileFormFromMe = () => {
    if (isPlatform && me?.platformAdmin) {
      profileForm.setFieldsValue({
        displayName: me.platformAdmin.displayName || '',
        username: '',
        phone: '',
      })
      return
    }
    if (me?.principal === 'tenant' && me?.user) {
      profileForm.setFieldsValue({
        displayName: me.user.displayName || '',
        username: me.user.username || '',
        phone: me.user.phone || '',
      })
    }
  }

  const heroTitle = () => {
    if (isPlatform) return String(me?.platformAdmin?.displayName || me?.platformAdmin?.email || t('profile.adminFallback'))
    const u = me?.user
    return String(u?.displayName?.trim() || u?.username?.trim() || u?.email || t('profile.userFallback'))
  }

  const heroSubtitle = () => {
    if (isPlatform) return String(me?.platformAdmin?.email || '')
    return String(me?.user?.email || '')
  }

  const avatarSrc = () => (!isPlatform ? String(me?.user?.avatarUrl || '').trim() : '')

  const detailGridStyle = {
    display: 'grid',
    gridTemplateColumns: 'minmax(88px, auto) 1fr minmax(88px, auto) 1fr',
    columnGap: 16,
    rowGap: 14,
    alignItems: 'center' as const,
  }

  const profilePanel = (
    <Space direction="vertical" size={16} style={{ width: '100%' }}>
      <Card
        loading={loading}
        bordered={false}
        className="overflow-hidden rounded-xl border border-border bg-gradient-to-br from-primary/10 via-card to-card"
        bodyStyle={{ padding: 20 }}
      >
        <div style={{ display: 'flex', flexWrap: 'wrap', gap: 20, alignItems: 'flex-start' }}>
          <Avatar size={72} className="shrink-0 bg-muted">
            {!isPlatform && avatarSrc() ? (
              <img alt="" src={avatarSrc()} className="h-full w-full object-cover" />
            ) : (
              <UserCircle size={40} strokeWidth={1.5} className="text-muted-foreground" />
            )}
          </Avatar>
          <div style={{ flex: '1 1 220px', minWidth: 0 }}>
            <div style={{ display: 'flex', flexWrap: 'wrap', alignItems: 'center', gap: 10 }}>
              <span className="text-[22px] font-bold text-foreground">{heroTitle()}</span>
              {isPlatform ? (
                <Tag color="orangered">{t('profile.platformAdmin')}</Tag>
              ) : (
                <Tag color="arcoblue">{t('profile.tenantMember')}</Tag>
              )}
              <Tag color={isPlatform || me?.user?.status === 'active' ? 'green' : 'gray'}>
                {isPlatform ? t('profile.active') : fmtStatus(me?.user?.status, t)}
              </Tag>
            </div>
            <div className="mt-2 break-all text-sm text-muted-foreground">{heroSubtitle()}</div>
            <Space style={{ marginTop: 12 }} wrap>
              <Tag color="green" size="small">
                {t('profile.emailVerified')}
              </Tag>
              {!isPlatform && me?.user?.phone ? (
                <Tag color="green" size="small">
                  {t('profile.phoneRegistered')}
                </Tag>
              ) : (
                !isPlatform && (
                  <Tag size="small" className="text-muted-foreground">
                    {t('profile.phoneNotRegistered')}
                  </Tag>
                )
              )}
              {!isPlatform && (
                <Upload
                  accept="image/png,image/jpeg,image/gif,image/webp"
                  showUploadList={false}
                  beforeUpload={async (file: File) => {
                    try {
                      const res = await uploadMyAvatar(file)
                      if (res.code !== 200 || !res.data?.user) {
                        showAlert(res.msg || t('profile.uploadFailed'), 'error')
                        return false
                      }
                      showAlert(t('profile.avatarUpdated'), 'success')
                      updateLocalProfile(res.data.user as never)
                      await loadMe()
                    } catch (e: any) {
                      showAlert(extractApiErrorMessage(e, t('profile.uploadFailed')), 'error')
                    }
                    return false
                  }}
                >
                  <Button size="mini" type="outline">
                    {t('profile.changeAvatar')}
                  </Button>
                </Upload>
              )}
            </Space>
          </div>
        </div>
      </Card>

      <Card title={t('profile.accountDetails')} bordered={false} className="rounded-xl border border-border"
        extra={
          <Button
            type="text"
            size="small"
            onClick={() => {
              if (editingProfile) {
                resetProfileFormFromMe()
                setEditingProfile(false)
              } else {
                setEditingProfile(true)
              }
            }}
          >
            {editingProfile ? t('common.cancel') : t('common.edit')}
          </Button>
        }
      >
        {isPlatform ? (
          <Form
            form={profileForm}
            layout="vertical"
            requiredSymbol={false}
            onSubmit={async (v) => {
              try {
                const res = await updateMeApi({
                  displayName: String(v.displayName || '').trim(),
                })
                if (res.code !== 200 || !res.data) {
                  showAlert(res.msg || t('profile.updateFailed'), 'error')
                  return
                }
                showAlert(t('profile.profileUpdated'), 'success')
                const pdata = res.data as { id?: number; email?: string; displayName?: string }
                updateLocalProfile({
                  ...pdata,
                  isPlatformAdmin: true,
                  principal: 'platform',
                })
                setEditingProfile(false)
                await loadMe()
              } catch (e: any) {
                showAlert(extractApiErrorMessage(e, t('profile.updateFailed')), 'error')
              }
            }}
          >
            <div style={detailGridStyle}>
              <div className="text-[13px] text-muted-foreground whitespace-nowrap">{t('profile.accountId')}</div>
              <div className="text-[13px] text-foreground break-words">{me?.platformAdmin?.id ?? '—'}</div>
              <div className="text-[13px] text-muted-foreground whitespace-nowrap">{t('profile.email')}</div>
              <div className="text-[13px] text-foreground break-words">{me?.platformAdmin?.email || '—'}</div>
              <div className="text-[13px] text-muted-foreground whitespace-nowrap">{t('profile.displayName')}</div>
              {editingProfile ? (
                <FormItem field="displayName" noStyle>
                  <Input placeholder={t('profile.displayName')} />
                </FormItem>
              ) : (
                <div className="text-[13px] text-foreground break-words">{me?.platformAdmin?.displayName || '—'}</div>
              )}
              <div className="text-[13px] text-muted-foreground whitespace-nowrap">{t('profile.status')}</div>
              <div className="text-[13px] text-foreground break-words">{fmtStatus(me?.platformAdmin?.status, t)}</div>
              <div className="text-[13px] text-muted-foreground whitespace-nowrap">{t('profile.twoFactor')}</div>
              <div className="text-[13px] text-foreground break-words">{me?.platformAdmin?.totpEnabled ? t('profile.twoFactorOn') : t('profile.twoFactorOff')}</div>
              <div style={{ gridColumn: '1 / -1', fontSize: 12 }}>
                <Link to="/platform-admins">{t('profile.managePlatformAdmins')}</Link>
              </div>
              {editingProfile && (
                <div style={{ gridColumn: '1 / -1' }}>
                  <Button type="primary" htmlType="submit" size="small">
                    {t('common.save')}
                  </Button>
                </div>
              )}
            </div>
          </Form>
        ) : (
          <Form
            form={profileForm}
            layout="vertical"
            requiredSymbol={false}
            onSubmit={async (v) => {
              try {
                const res = await updateMeApi({
                  displayName: String(v.displayName || '').trim(),
                  username: String(v.username || '').trim(),
                  phone: String(v.phone || '').trim(),
                })
                if (res.code !== 200 || !res.data) {
                  showAlert(res.msg || t('profile.updateFailed'), 'error')
                  return
                }
                showAlert(t('profile.profileUpdated'), 'success')
                updateLocalProfile(res.data as never)
                setEditingProfile(false)
                await loadMe()
              } catch (e: any) {
                showAlert(extractApiErrorMessage(e, t('profile.updateFailed')), 'error')
              }
            }}
          >
            <div style={detailGridStyle}>
              <div className="text-[13px] text-muted-foreground whitespace-nowrap">{t('profile.accountId')}</div>
              <div className="text-[13px] text-foreground break-words">{me?.user?.id ?? '—'}</div>
              <div className="text-[13px] text-muted-foreground whitespace-nowrap">{t('profile.email')}</div>
              <div className="text-[13px] text-foreground break-words">{me?.user?.email || '—'}</div>
              <div className="text-[13px] text-muted-foreground whitespace-nowrap">{t('profile.displayName')}</div>
              {editingProfile ? (
                <FormItem field="displayName" noStyle>
                  <Input placeholder={t('profile.displayNamePlaceholder')} />
                </FormItem>
              ) : (
                <div className="text-[13px] text-foreground break-words">{me?.user?.displayName || '—'}</div>
              )}
              <div className="text-[13px] text-muted-foreground whitespace-nowrap">{t('profile.username')}</div>
              {editingProfile ? (
                <FormItem field="username" noStyle>
                  <Input placeholder={t('profile.usernamePlaceholder')} />
                </FormItem>
              ) : (
                <div className="text-[13px] text-foreground break-words">{me?.user?.username || '—'}</div>
              )}
              <div className="text-[13px] text-muted-foreground whitespace-nowrap">{t('profile.phone')}</div>
              {editingProfile ? (
                <FormItem field="phone" noStyle>
                  <Input placeholder={t('profile.phonePlaceholder')} />
                </FormItem>
              ) : (
                <div className="text-[13px] text-foreground break-words">{me?.user?.phone || '—'}</div>
              )}
              <div className="text-[13px] text-muted-foreground whitespace-nowrap">{t('profile.loginCount')}</div>
              <div className="text-[13px] text-foreground break-words">{me?.user?.loginCount ?? 0}</div>
              <div className="text-[13px] text-muted-foreground whitespace-nowrap">{t('profile.lastLogin')}</div>
              <div className="text-[13px] text-foreground break-words">{fmtLastLogin(me?.user?.lastLogin)}</div>
              <div className="text-[13px] text-muted-foreground whitespace-nowrap">{t('profile.organization')}</div>
              <div className="text-[13px] text-foreground break-words">{me?.tenant?.name || '—'}</div>
              <div className="text-[13px] text-muted-foreground whitespace-nowrap">{t('profile.organizationSlug')}</div>
              <div className="text-[13px] text-foreground break-words">{me?.tenant?.slug || '—'}</div>
              <div className="text-[13px] text-muted-foreground whitespace-nowrap">{t('profile.department')}</div>
              <div className="text-[13px] text-foreground break-words">
                {Array.isArray(me?.user?.tenantGroups) && me.user.tenantGroups.length
                  ? me.user.tenantGroups.map((g: { name?: string }) => g.name).join('、')
                  : me?.user?.tenantGroup?.name || t('profile.unassigned')}
              </div>
              <div className="text-[13px] text-muted-foreground whitespace-nowrap">{t('profile.accountStatus')}</div>
              <div className="text-[13px] text-foreground break-words">{fmtStatus(me?.user?.status, t)}</div>
              <div className="text-[13px] text-muted-foreground whitespace-nowrap">{t('profile.lastLoginIp')}</div>
              <div className="text-[13px] text-foreground break-words">{me?.user?.lastLoginIp || '—'}</div>
              {me?.user?.source ? (
                <>
                  <div className="text-[13px] text-muted-foreground whitespace-nowrap">{t('profile.accountSource')}</div>
                  <div className="text-[13px] text-foreground break-words">{me.user.source}</div>
                </>
              ) : null}
              <div className="text-[13px] text-muted-foreground whitespace-nowrap">{t('profile.registeredAt')}</div>
              <div className="text-[13px] text-foreground break-words">
                {me?.user?.createdAt ? dayjs(me.user.createdAt).format('YYYY/MM/DD HH:mm:ss') : '—'}
              </div>
              <div className="text-[13px] text-muted-foreground whitespace-nowrap">{t('profile.twoFactor')}</div>
              <div className="text-[13px] text-foreground break-words">{me?.user?.totpEnabled ? t('profile.twoFactorOn') : t('profile.twoFactorOff')}</div>
              <div className="col-span-full mt-1 text-[13px] text-muted-foreground">{t('profile.role')}</div>
              <div className="col-span-full text-[13px] text-foreground break-words">
                {Array.isArray(me?.user?.roles) && me.user.roles.length
                  ? me.user.roles.map((r: { name?: string }) => r.name).join('、')
                  : '—'}
              </div>
              {editingProfile && (
                <div style={{ gridColumn: '1 / -1' }}>
                  <Button type="primary" htmlType="submit" size="small">
                    {t('common.save')}
                  </Button>
                </div>
              )}
            </div>
          </Form>
        )}
      </Card>
    </Space>
  )

  const handleLogout = async () => {
    await logout()
    navigate('/login', { replace: true })
  }

  if (sectionParam && !SECTION_ALIASES[sectionParam]) {
    return <Navigate to="/profile/info" replace />
  }

  if (activeSection === 'devices' && isPlatform) {
    return <Navigate to="/profile/info" replace />
  }
  if (activeSection === 'access-keys' && !showAccessKeys) {
    return <Navigate to="/profile/info" replace />
  }
  if (activeSection === 'notifications' && !showWebhooks) {
    return <Navigate to="/profile/info" replace />
  }
  if (activeSection === 'reports' && !showAIReports) {
    return <Navigate to="/profile/info" replace />
  }
  if (activeSection === 'logs' && !showOperationLogs) {
    return <Navigate to="/profile/info" replace />
  }
  if (activeSection === 'ai-invocations' && !showAIInvocations) {
    return <Navigate to="/profile/info" replace />
  }

  return (
    <BaseLayout title={t('profile.title')} description="">
      <div className="flex min-h-[calc(100vh-96px)] items-stretch gap-3">
        <aside className="flex w-[168px] shrink-0 flex-col overflow-hidden rounded-lg border border-border bg-card">
          <div className="px-3 pb-1 pt-3 text-xs text-muted-foreground">{t('profile.navSection')}</div>
          <Menu
            className="!flex-1 !border-none !bg-transparent"
            style={{ width: '100%', padding: '4px 6px 8px' }}
            selectedKeys={[activeSection]}
            onClickMenuItem={(key) => goSection(key as ProfileSection)}
          >
            {navItems.map((item) => (
              <Menu.Item key={item.key} style={{ borderRadius: 8, lineHeight: '44px', height: 44, marginBottom: 4 }}>
                {item.label}
              </Menu.Item>
            ))}
          </Menu>
          <div className="mt-auto shrink-0 border-t border-border p-3">
            <Button long status="warning" onClick={() => void handleLogout()}>
              {t('profile.logout')}
            </Button>
          </div>
        </aside>

        <div className="min-w-0 max-h-[calc(100vh-96px)] flex-1 basis-[360px] overflow-y-auto">
          {activeSection === 'info' && profilePanel}
          {activeSection === 'security' && (
            <AccountSecurityPanel
              me={me}
              boundEmail={me?.principal === 'platform' ? me?.platformAdmin?.email : me?.user?.email}
              voiceprintEnabled={Boolean(siteConfig.VOICEPRINT_PROVIDER?.trim())}
              onReload={loadMe}
              onTotpUpdated={(user) => updateLocalProfile(user as never)}
            />
          )}
          {activeSection === 'devices' && !isPlatform && <UserDevicesPanel />}
          {activeSection === 'access-keys' && showAccessKeys && <AccessKeys embedded />}
          {activeSection === 'inbox' && <InboxPanel />}
          {activeSection === 'reports' && showAIReports && <AIReportsPanel />}
          {activeSection === 'logs' && showOperationLogs && <OperationLogs embedded />}
          {activeSection === 'ai-invocations' && showAIInvocations && <AIInvocationLogs embedded />}
          {activeSection === 'login-history' && <LoginHistoryPanel />}
          {activeSection === 'notifications' && showWebhooks && <NotificationChannelsPanel />}
        </div>
      </div>
    </BaseLayout>
  )
}
