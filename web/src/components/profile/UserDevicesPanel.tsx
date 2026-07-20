import { useCallback, useEffect, useState } from 'react'
import { Monitor, Smartphone, Tablet } from 'lucide-react'
import { Pagination } from '@arco-design/web-react'
import { Button } from '@/components/ui'
import { fetchMyDevices, revokeMyDevice, trustMyDevice, deleteMyDevice, type UserDeviceItem } from '@/api/me'
import { useTranslation } from '@/i18n'
import { showAlert } from '@/utils/notification'
import { extractApiErrorMessage } from '@/utils/apiError'
import { cn } from '@/utils/cn'

function DeviceCategoryIcon({ category }: { category: string }) {
  const cat = category === 'mobile' || category === 'tablet' || category === 'desktop' ? category : 'desktop'
  const base = 'flex h-11 w-11 shrink-0 items-center justify-center rounded-xl'
  if (cat === 'mobile') {
    return (
      <div className={cn(base, 'bg-emerald-50 text-emerald-600')}>
        <Smartphone size={22} strokeWidth={1.75} />
      </div>
    )
  }
  if (cat === 'tablet') {
    return (
      <div className={cn(base, 'bg-violet-50 text-violet-600')}>
        <Tablet size={22} strokeWidth={1.75} />
      </div>
    )
  }
  return (
    <div className={cn(base, 'bg-sky-50 text-sky-600')}>
      <Monitor size={22} strokeWidth={1.75} />
    </div>
  )
}

function formatLoginLocation(d: UserDeviceItem) {
  const city = d.lastLoginCity?.trim()
  if (city) return city
  return ''
}

export default function UserDevicesPanel() {
  const { t } = useTranslation()
  const [loading, setLoading] = useState(true)
  const [devices, setDevices] = useState<UserDeviceItem[]>([])
  const [total, setTotal] = useState(0)
  const [page, setPage] = useState(1)
  const pageSize = 10
  const [actingId, setActingId] = useState<string | null>(null)

  const load = useCallback(async () => {
    setLoading(true)
    try {
      const res = await fetchMyDevices(page, pageSize)
      if (res.code !== 200) {
        showAlert(res.msg || t('profile.devicesLoadFailed'), 'error')
        return
      }
      setDevices(res.data?.list ?? [])
      setTotal(res.data?.total ?? 0)
    } catch (e: unknown) {
      showAlert(extractApiErrorMessage(e, t('profile.devicesLoadFailed')), 'error')
    } finally {
      setLoading(false)
    }
  }, [page, t])

  useEffect(() => {
    void load()
  }, [load])

  const categoryLabel = (c: string) => {
    if (c === 'mobile') return t('profile.deviceMobile')
    if (c === 'desktop') return t('profile.deviceDesktop')
    if (c === 'tablet') return t('profile.deviceTablet')
    return c
  }

  const handleTrust = async (id: string) => {
    setActingId(id)
    try {
      const res = await trustMyDevice(id)
      if (res.code !== 200) {
        showAlert(res.msg || t('profile.deviceTrustFailed'), 'error')
        return
      }
      showAlert(t('profile.deviceTrusted'), 'success')
      await load()
    } catch (e: unknown) {
      showAlert(extractApiErrorMessage(e, t('profile.deviceTrustFailed')), 'error')
    } finally {
      setActingId(null)
    }
  }

  const handleRevoke = async (id: string) => {
    setActingId(id)
    try {
      const res = await revokeMyDevice(id)
      if (res.code !== 200) {
        showAlert(res.msg || t('profile.deviceRevokeFailed'), 'error')
        return
      }
      showAlert(t('profile.deviceRevoked'), 'success')
      await load()
    } catch (e: unknown) {
      showAlert(extractApiErrorMessage(e, t('profile.deviceRevokeFailed')), 'error')
    } finally {
      setActingId(null)
    }
  }

  const handleDelete = async (id: string) => {
    setActingId(id)
    try {
      const res = await deleteMyDevice(id)
      if (res.code !== 200) {
        showAlert(res.msg || t('profile.deviceDeleteFailed'), 'error')
        return
      }
      showAlert(t('profile.deviceDeleted'), 'success')
      await load()
    } catch (e: unknown) {
      showAlert(extractApiErrorMessage(e, t('profile.deviceDeleteFailed')), 'error')
    } finally {
      setActingId(null)
    }
  }

  return (
    <div className="rounded-xl border border-border bg-card px-5">
      <div className="border-b border-neutral-100 py-4">
        <div className="text-base font-medium text-neutral-900">{t('profile.devicesTitle')}</div>
        <p className="mt-1 text-sm text-neutral-500">{t('profile.devicesDesc')}</p>
      </div>
      {loading ? (
        <div className="py-8 text-center text-sm text-neutral-400">{t('common.loading')}</div>
      ) : devices.length === 0 ? (
        <div className="py-8 text-center text-sm text-neutral-400">{t('profile.devicesEmpty')}</div>
      ) : (
        <>
          {devices.map((d) => {
            const location = formatLoginLocation(d)
            return (
              <div key={d.id} className="flex items-center justify-between gap-4 border-b border-neutral-100 py-4 last:border-b-0">
                <div className="flex min-w-0 items-center gap-3">
                  <DeviceCategoryIcon category={d.limitCategory || d.category} />
                  <div className="min-w-0">
                    <div className="flex flex-wrap items-center gap-2">
                      <span className="font-medium text-neutral-900">{d.displayName || t('profile.deviceUnknown')}</span>
                      {d.isCurrent && (
                        <span className="rounded-full bg-[#1671EE]/10 px-2 py-0.5 text-xs font-medium text-[#1671EE]">
                          {t('profile.deviceCurrent')}
                        </span>
                      )}
                      {d.isTrusted && !d.isCurrent && (
                        <span className="rounded-full bg-amber-50 px-2 py-0.5 text-xs text-amber-700">
                          {t('profile.deviceTrustedBadge')}
                        </span>
                      )}
                      {d.sessionActive && !d.isCurrent && (
                        <span className="rounded-full bg-emerald-50 px-2 py-0.5 text-xs text-emerald-700">
                          {t('profile.deviceOnline')}
                        </span>
                      )}
                    </div>
                    <div className="mt-1 text-sm text-neutral-500">
                      {categoryLabel(d.limitCategory || d.category)}
                      {location ? ` · ${t('profile.deviceLoginCity')}: ${location}` : ''}
                      {d.lastIp ? ` · ${d.lastIp}` : ''}
                      {d.lastLoginAt ? ` · ${new Date(d.lastLoginAt).toLocaleString()}` : ''}
                    </div>
                  </div>
                </div>
                {!d.isCurrent && (
                  <div className="flex shrink-0 flex-wrap items-center justify-end gap-2">
                    {d.isTrusted ? (
                      <Button
                        type="outline"
                        loading={actingId === d.id}
                        onClick={() => void handleRevoke(d.id)}
                      >
                        {t('profile.deviceRevoke')}
                      </Button>
                    ) : (
                      <Button
                        type="primary"
                        loading={actingId === d.id}
                        onClick={() => void handleTrust(d.id)}
                      >
                        {t('profile.deviceTrust')}
                      </Button>
                    )}
                    <Button
                      type="outline"
                      status="warning"
                      loading={actingId === d.id}
                      onClick={() => void handleDelete(d.id)}
                    >
                      {t('profile.deviceDelete')}
                    </Button>
                  </div>
                )}
              </div>
            )
          })}
          {total > pageSize ? (
            <div className="flex justify-end py-4">
              <Pagination
                total={total}
                current={page}
                pageSize={pageSize}
                onChange={(p) => setPage(p)}
                size="small"
              />
            </div>
          ) : null}
        </>
      )}
    </div>
  )
}
