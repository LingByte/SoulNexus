import { useCallback, useEffect, useState } from 'react'
import { Drawer, Pagination } from '@arco-design/web-react'
import { CheckCircle, XCircle, Globe, Monitor, Smartphone, Tablet, Clock, Mail, MapPin, User } from 'lucide-react'
import dayjs from 'dayjs'
import { Button } from '@/components/ui'
import { fetchMyLoginHistory, type LoginHistoryRow } from '@/api/loginHistory'
import { useTranslation } from '@/i18n'
import { showAlert } from '@/utils/notification'
import { extractApiErrorMessage } from '@/utils/apiError'
import { cn } from '@/utils/cn'

function fmtTime(iso?: string) {
  if (!iso) return '—'
  const d = dayjs(iso)
  return d.isValid() ? d.format('YYYY-MM-DD HH:mm:ss') : '—'
}

function fmtTimeRelative(iso?: string, t?: (key: string, params?: Record<string, string | number>) => string) {
  if (!iso) return '—'
  const d = dayjs(iso)
  if (!d.isValid()) return '—'
  const now = dayjs()
  const diffMinutes = now.diff(d, 'minute')
  if (diffMinutes < 1) return t?.('profile.loginHistoryJustNow') ?? '刚刚'
  if (diffMinutes < 60) return t?.('profile.loginHistoryMinutesAgo', { count: diffMinutes }) ?? `${diffMinutes} 分钟前`
  const diffHours = now.diff(d, 'hour')
  if (diffHours < 24) return t?.('profile.loginHistoryHoursAgo', { count: diffHours }) ?? `${diffHours} 小时前`
  const diffDays = now.diff(d, 'day')
  if (diffDays < 30) return t?.('profile.loginHistoryDaysAgo', { count: diffDays }) ?? `${diffDays} 天前`
  return d.format('YYYY-MM-DD')
}

function methodLabel(method: string | undefined, t: (k: string) => string) {
  const m = String(method || '').toLowerCase()
  if (m === 'password') return t('profile.loginMethodPassword')
  if (m === 'email_code') return t('profile.loginMethodEmailCode')
  if (m === 'oauth_github') return t('profile.loginMethodOAuth')
  if (m === 'register') return t('profile.loginMethodRegister')
  return method || '—'
}

function failureReasonLabel(reason: string | undefined, t: (k: string) => string) {
  const r = String(reason || '').toLowerCase()
  if (!r) return '—'
  const key = `profile.loginFailure.${r}` as const
  const translated = t(key)
  return translated !== key ? translated : reason || '—'
}

type DeviceType = 'desktop' | 'mobile' | 'tablet' | 'unknown'

function parseUserAgent(ua?: string): { browser: string; os: string; deviceType: DeviceType } {
  if (!ua) return { browser: '—', os: '—', deviceType: 'unknown' }
  
  let browser = '—'
  if (ua.includes('Firefox')) browser = 'Firefox'
  else if (ua.includes('Edg')) browser = 'Edge'
  else if (ua.includes('Chrome')) browser = 'Chrome'
  else if (ua.includes('Safari')) browser = 'Safari'
  else if (ua.includes('Opera') || ua.includes('OPR')) browser = 'Opera'
  
  let os = '—'
  if (ua.includes('Windows')) os = 'Windows'
  else if (ua.includes('Mac OS')) os = 'macOS'
  else if (ua.includes('Linux')) os = 'Linux'
  else if (ua.includes('Android')) os = 'Android'
  else if (ua.includes('iOS') || ua.includes('iPhone') || ua.includes('iPad')) os = 'iOS'
  
  let deviceType: DeviceType = 'desktop'
  if (ua.includes('Mobile') || ua.includes('Android')) deviceType = 'mobile'
  else if (ua.includes('iPad') || ua.includes('Tablet')) deviceType = 'tablet'
  
  return { browser, os, deviceType }
}

function DeviceIcon({ deviceType }: { deviceType: DeviceType }) {
  const base = 'flex h-10 w-10 shrink-0 items-center justify-center rounded-xl'
  if (deviceType === 'mobile') {
    return (
      <div className={cn(base, 'bg-emerald-50 text-emerald-600')}>
        <Smartphone size={20} strokeWidth={1.75} />
      </div>
    )
  }
  if (deviceType === 'tablet') {
    return (
      <div className={cn(base, 'bg-violet-50 text-violet-600')}>
        <Tablet size={20} strokeWidth={1.75} />
      </div>
    )
  }
  return (
    <div className={cn(base, 'bg-sky-50 text-sky-600')}>
      <Monitor size={20} strokeWidth={1.75} />
    </div>
  )
}

export default function LoginHistoryPanel() {
  const { t } = useTranslation()
  const [loading, setLoading] = useState(true)
  const [rows, setRows] = useState<LoginHistoryRow[]>([])
  const [total, setTotal] = useState(0)
  const [page, setPage] = useState(1)
  const pageSize = 15
  const [detail, setDetail] = useState<LoginHistoryRow | null>(null)

  const load = useCallback(async () => {
    setLoading(true)
    try {
      const res = await fetchMyLoginHistory(page, pageSize)
      if (res.code !== 200) {
        showAlert(res.msg || t('profile.loginHistoryLoadFailed'), 'error')
        return
      }
      setRows(res.data?.list ?? [])
      setTotal(res.data?.total ?? 0)
    } catch (e: unknown) {
      showAlert(extractApiErrorMessage(e, t('profile.loginHistoryLoadFailed')), 'error')
    } finally {
      setLoading(false)
    }
  }, [page, t])

  useEffect(() => {
    void load()
  }, [load])

  return (
    <div className="rounded-xl border border-border bg-card px-5">
      <div className="border-b border-neutral-100 py-4">
        <div className="text-base font-medium text-neutral-900">{t('profile.navLoginHistory')}</div>
        <p className="mt-1 text-sm text-neutral-500">{t('profile.loginHistoryDesc')}</p>
      </div>
      
      {loading ? (
        <div className="py-8 text-center text-sm text-neutral-400">{t('common.loading')}</div>
      ) : rows.length === 0 ? (
        <div className="py-8 text-center text-sm text-neutral-400">{t('profile.loginHistoryEmpty')}</div>
      ) : (
        <>
          {rows.map((row) => {
            const uaInfo = parseUserAgent(row.userAgent)
            return (
              <div key={row.id} className="flex items-center justify-between gap-4 border-b border-neutral-100 py-4 last:border-b-0">
                <div className="flex min-w-0 items-center gap-3">
                  <DeviceIcon deviceType={uaInfo.deviceType} />
                  <div className="min-w-0">
                    <div className="flex flex-wrap items-center gap-2">
                      <span className="font-medium text-neutral-900">
                        {methodLabel(row.loginMethod, t)}
                      </span>
                      {row.success ? (
                        <span className="rounded-full bg-emerald-50 px-2 py-0.5 text-xs font-medium text-emerald-700">
                          {t('profile.loginHistorySuccess')}
                        </span>
                      ) : (
                        <span className="rounded-full bg-red-50 px-2 py-0.5 text-xs font-medium text-red-700">
                          {t('profile.loginHistoryFailed')}
                        </span>
                      )}
                    </div>
                    <div className="mt-1 flex flex-wrap items-center gap-x-3 text-sm text-neutral-500">
                      <span className="flex items-center gap-1">
                        <Clock size={14} className="text-neutral-400" />
                        {fmtTimeRelative(row.createdAt, t)}
                      </span>
                      {row.clientIp && (
                        <span className="flex items-center gap-1">
                          <Globe size={14} className="text-neutral-400" />
                          {row.clientIp}
                        </span>
                      )}
                      {(row.city || row.location) && (
                        <span className="flex items-center gap-1">
                          <MapPin size={14} className="text-neutral-400" />
                          {row.city || row.location}
                        </span>
                      )}
                    </div>
                  </div>
                </div>
                <Button
                  type="outline"
                  size="sm"
                  onClick={() => setDetail(row)}
                >
                  {t('profile.loginHistoryDetail')}
                </Button>
              </div>
            )
          })}
          
          {total > pageSize && (
            <div className="flex justify-end py-4">
              <Pagination
                total={total}
                current={page}
                pageSize={pageSize}
                onChange={(p) => setPage(p)}
                size="small"
              />
            </div>
          )}
        </>
      )}

      <Drawer
        width={480}
        title={t('profile.loginHistoryDetailTitle')}
        visible={detail != null}
        onCancel={() => setDetail(null)}
        footer={null}
      >
        {detail ? (
          <div className="space-y-6">
            <div className="flex items-center gap-4">
              <DeviceIcon deviceType={parseUserAgent(detail.userAgent).deviceType} />
              <div>
                <div className="font-medium text-neutral-900">
                  {methodLabel(detail.loginMethod, t)}
                </div>
                <div className="text-sm text-neutral-500">
                  {fmtTime(detail.createdAt)}
                </div>
              </div>
              {detail.success ? (
                <span className="ml-auto rounded-full bg-emerald-50 px-3 py-1 text-sm font-medium text-emerald-700">
                  <CheckCircle size={16} className="mr-1 inline" />
                  {t('profile.loginHistorySuccess')}
                </span>
              ) : (
                <span className="ml-auto rounded-full bg-red-50 px-3 py-1 text-sm font-medium text-red-700">
                  <XCircle size={16} className="mr-1 inline" />
                  {t('profile.loginHistoryFailed')}
                </span>
              )}
            </div>

            <div className="rounded-xl border border-neutral-100 p-4">
              <h3 className="mb-3 text-sm font-medium text-neutral-900">{t('profile.loginHistorySectionLoginInfo')}</h3>
              <div className="space-y-3">
                <div className="flex items-center gap-3">
                  <Clock size={16} className="text-neutral-400" />
                  <div>
                    <div className="text-sm text-neutral-500">{t('profile.loginHistoryTime')}</div>
                    <div className="text-sm text-neutral-900">{fmtTime(detail.createdAt)}</div>
                  </div>
                </div>
                <div className="flex items-center gap-3">
                  <Globe size={16} className="text-neutral-400" />
                  <div>
                    <div className="text-sm text-neutral-500">{t('profile.loginHistoryIp')}</div>
                    <div className="text-sm text-neutral-900">{detail.clientIp || '—'}</div>
                  </div>
                </div>
                <div className="flex items-center gap-3">
                  <MapPin size={16} className="text-neutral-400" />
                  <div>
                    <div className="text-sm text-neutral-500">{t('profile.loginHistoryLocation')}</div>
                    <div className="text-sm text-neutral-900">
                      {detail.city || detail.location || '—'}
                    </div>
                  </div>
                </div>
                {detail.email && (
                  <div className="flex items-center gap-3">
                    <Mail size={16} className="text-neutral-400" />
                    <div>
                      <div className="text-sm text-neutral-500">{t('profile.loginHistoryEmail')}</div>
                      <div className="text-sm text-neutral-900">{detail.email}</div>
                    </div>
                  </div>
                )}
              </div>
            </div>

            <div className="rounded-xl border border-neutral-100 p-4">
              <h3 className="mb-3 text-sm font-medium text-neutral-900">{t('profile.loginHistorySectionDeviceInfo')}</h3>
              <div className="space-y-3">
                <div className="flex items-center gap-3">
                  <Monitor size={16} className="text-neutral-400" />
                  <div>
                    <div className="text-sm text-neutral-500">{t('profile.loginHistoryDevice')}</div>
                    <div className="text-sm text-neutral-900">{detail.deviceKey || '—'}</div>
                  </div>
                </div>
                <div className="flex items-center gap-3">
                  <User size={16} className="text-neutral-400" />
                  <div>
                    <div className="text-sm text-neutral-500">{t('profile.loginHistoryUserAgent')}</div>
                    <div className="text-sm text-neutral-900 break-all">{detail.userAgent || '—'}</div>
                  </div>
                </div>
              </div>
            </div>

            {detail.failureReason && (
              <div className="rounded-xl border border-red-100 bg-red-50 p-4">
                <h3 className="mb-2 text-sm font-medium text-red-800">{t('profile.loginHistoryFailureReason')}</h3>
                <p className="text-sm text-red-700">{failureReasonLabel(detail.failureReason, t)}</p>
              </div>
            )}
          </div>
        ) : null}
      </Drawer>
    </div>
  )
}