import { useCallback, useEffect, useState, type ReactNode } from 'react'
import { AnimatePresence, motion } from 'framer-motion'
import { AlertCircle, MicOff, Phone, RefreshCw, Trash2, X } from 'lucide-react'
import { useI18nStore } from '@/stores/i18nStore'
import Button from '@/components/UI/Button'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/UI/Tabs'
import LoadingAnimation from '@/components/Animations/LoadingAnimation'
import { showAlert } from '@/utils/notification'
import {
  listSIPUsers,
  deleteSIPUser,
  getSIPCall,
  listSIPCalls,
  resolveSipRecordingUrl,
  sipAiEndStatusI18nKey,
  type SIPUserRow,
  type SIPCallRow,
} from '@/api/sipContactCenter'
import WebSeatContactTab from '@/pages/ContactCenter/WebSeatContactTab'
import ACDPoolTab from '@/pages/ContactCenter/ACDPoolTab'
import OutboundCampaignTab from '@/pages/ContactCenter/OutboundCampaignTab'
import ScriptManagerTab from '@/pages/ContactCenter/ScriptManagerTab'
import { EllipsisHoverCell } from '@/pages/ContactCenter/EllipsisHoverCell'
import CallAudioPlayer from '@/components/CallAudioPlayer'
import ConfirmDialog from '@/components/UI/ConfirmDialog'

export default function ContactCenterPage() {
  const { t } = useI18nStore()
  const [tab, setTab] = useState<'users' | 'calls' | 'acd' | 'campaign' | 'scripts' | 'agent'>('calls')
  const [acdRefreshNonce, setAcdRefreshNonce] = useState(0)

  const [users, setUsers] = useState<SIPUserRow[]>([])
  const [usersTotal, setUsersTotal] = useState(0)
  const [usersPage, setUsersPage] = useState(1)

  const [calls, setCalls] = useState<SIPCallRow[]>([])
  const [callsTotal, setCallsTotal] = useState(0)
  const [callsPage, setCallsPage] = useState(1)
  const [callFilter, setCallFilter] = useState('')

  const [callsSearchNonce, setCallsSearchNonce] = useState(0)

  const [callDetailDrawerId, setCallDetailDrawerId] = useState<number | null>(null)
  const [callDetailDrawerData, setCallDetailDrawerData] = useState<SIPCallRow | null>(null)
  const [callDetailDrawerLoading, setCallDetailDrawerLoading] = useState(false)
  const [callDetailDrawerFailed, setCallDetailDrawerFailed] = useState(false)

  const [loading, setLoading] = useState(false)
  const [sipUserDeleteOpen, setSipUserDeleteOpen] = useState(false)
  const [sipUserDeleteId, setSipUserDeleteId] = useState<number | null>(null)
  const pageSize = 20

  const loadUsers = useCallback(async () => {
    setLoading(true)
    try {
      const res = await listSIPUsers(usersPage, pageSize)
      if (res.code === 200 && res.data) {
        setUsers(res.data.list || [])
        setUsersTotal(res.data.total || 0)
      }
    } catch (e: unknown) {
      const err = e as { msg?: string }
      showAlert(err?.msg || t('common.failed'), 'error')
    } finally {
      setLoading(false)
    }
  }, [usersPage, t])

  const loadCalls = useCallback(async () => {
    setLoading(true)
    try {
      const res = await listSIPCalls(callsPage, pageSize, {
        callId: callFilter.trim() || undefined,
      })
      if (res.code === 200 && res.data) {
        setCalls(res.data.list || [])
        setCallsTotal(res.data.total || 0)
      }
    } catch (e: unknown) {
      const err = e as { msg?: string }
      showAlert(err?.msg || t('common.failed'), 'error')
    } finally {
      setLoading(false)
    }
  }, [callsPage, callFilter, t])

  useEffect(() => {
    if (tab === 'users') void loadUsers()
  }, [tab, loadUsers])

  useEffect(() => {
    if (tab === 'calls') void loadCalls()
  }, [tab, loadCalls, callsSearchNonce])

  useEffect(() => {
    const onAcdRefresh = () => setAcdRefreshNonce((n) => n + 1)
    window.addEventListener('soulnexus-acd-refresh', onAcdRefresh)
    return () => window.removeEventListener('soulnexus-acd-refresh', onAcdRefresh)
  }, [])

  const closeCallDetailDrawer = () => {
    setCallDetailDrawerId(null)
    setCallDetailDrawerData(null)
    setCallDetailDrawerLoading(false)
    setCallDetailDrawerFailed(false)
  }

  const openCallDetailDrawer = async (id: number) => {
    setCallDetailDrawerId(id)
    setCallDetailDrawerData(null)
    setCallDetailDrawerLoading(true)
    setCallDetailDrawerFailed(false)
    try {
      const res = await getSIPCall(id)
      if (res.code === 200 && res.data) {
        setCallDetailDrawerData(res.data)
      } else {
        setCallDetailDrawerFailed(true)
      }
    } catch {
      setCallDetailDrawerFailed(true)
    } finally {
      setCallDetailDrawerLoading(false)
    }
  }

  const openSipUserDelete = (id: number) => {
    setSipUserDeleteId(id)
    setSipUserDeleteOpen(true)
  }

  const confirmSipUserDelete = async () => {
    if (sipUserDeleteId == null) return
    const id = sipUserDeleteId
    try {
      await deleteSIPUser(id)
      showAlert(t('contactCenter.toast.deleteSipUserOk'), 'success')
      void loadUsers()
    } catch (e: unknown) {
      const err = e as { msg?: string }
      showAlert(err?.msg || t('common.failed'), 'error')
      throw e
    }
  }

  const fmt = (s?: string) => (s ? new Date(s).toLocaleString() : '—')

  return (
    <div className="min-h-screen bg-gray-50 dark:bg-neutral-900 flex flex-col">
      <div className="max-w-7xl w-full mx-auto px-4 sm:px-6 lg:px-8 pt-8 pb-8 flex flex-col gap-6">
        <div className="relative pl-4 flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4">
          <div className="flex items-center gap-3">
            <motion.div
              layoutId="pageTitleIndicator"
              className="absolute left-0 top-1/2 -translate-y-1/2 w-1 h-8 bg-primary rounded-r-full"
              transition={{ type: 'spring', bounce: 0.2, duration: 0.3 }}
            />
            <Phone className="w-8 h-8 text-primary" />
            <div>
              <h1 className="text-2xl font-bold text-gray-900 dark:text-gray-100">
                {t('contactCenter.title')}
              </h1>
              <p className="text-sm text-gray-500 dark:text-gray-400 mt-1">
                SIP {t('contactCenter.subtitle')}
              </p>
            </div>
          </div>
          <Button
            variant="outline"
            size="sm"
            onClick={() => {
              if (tab === 'users') void loadUsers()
              if (tab === 'calls') void loadCalls()
              if (tab === 'acd') setAcdRefreshNonce((n) => n + 1)
              /* agent tab: logs live in WebSeatProvider */
            }}
          >
            <RefreshCw className="w-4 h-4 mr-1" />
            {t('common.refresh')}
          </Button>
        </div>

        <Tabs
          value={tab}
          onValueChange={(v) => {
            const next = v as typeof tab
            setTab(next)
            if (next === 'acd') setAcdRefreshNonce((n) => n + 1)
          }}
          className="w-full"
        >
          <TabsList className="grid w-full max-w-6xl grid-cols-2 sm:grid-cols-6 gap-1 h-auto p-1">
            <TabsTrigger value="users">{t('contactCenter.tab.users')}</TabsTrigger>
            <TabsTrigger value="calls">{t('contactCenter.tab.calls')}</TabsTrigger>
            <TabsTrigger value="acd">{t('contactCenter.tab.acdPool')}</TabsTrigger>
            <TabsTrigger value="campaign">{t('contactCenter.tab.campaign')}</TabsTrigger>
            <TabsTrigger value="scripts">{t('contactCenter.tab.scripts')}</TabsTrigger>
            <TabsTrigger value="agent">{t('contactCenter.tab.agent')}</TabsTrigger>
          </TabsList>

          <TabsContent value="users" className="mt-4 space-y-3">
            <p className="text-xs text-muted-foreground leading-relaxed rounded-lg border border-border bg-muted/30 px-3 py-2.5">
              {t('contactCenter.users.sipRegisterHint')}
            </p>
            {loading ? (
              <LoadingAnimation />
            ) : (
              <div className="overflow-x-auto rounded-lg border border-border bg-card">
                <table className="min-w-[1100px] w-full text-sm">
                  <thead className="bg-muted/50">
                    <tr>
                      <th className="text-left p-3 whitespace-nowrap">ID</th>
                      <th className="text-left p-3 whitespace-nowrap">AOR</th>
                      <th className="text-left p-3 whitespace-nowrap min-w-[140px]">
                        {t('contactCenter.users.contactUri')}
                      </th>
                      <th className="text-left p-3 whitespace-nowrap">
                        {t('contactCenter.users.signalAddr')}
                      </th>
                      <th className="text-left p-3 min-w-[160px]">{t('contactCenter.users.userAgent')}</th>
                      <th className="text-left p-3 whitespace-nowrap">{t('common.status')}</th>
                      <th className="text-left p-3 whitespace-nowrap">
                        {t('contactCenter.users.expiresAt')}
                      </th>
                      <th className="text-left p-3 whitespace-nowrap">
                        {t('contactCenter.users.lastSeen')}
                      </th>
                      <th className="text-left p-3 whitespace-nowrap">{t('common.createdAt')}</th>
                      <th className="text-right p-3 whitespace-nowrap">{t('common.delete')}</th>
                    </tr>
                  </thead>
                  <tbody>
                    {users.length === 0 ? (
                      <tr>
                        <td colSpan={10} className="p-6 text-center text-muted-foreground">
                          {t('common.noData')}
                        </td>
                      </tr>
                    ) : (
                      users.map((u) => (
                        <tr key={u.id} className="border-t border-border">
                          <td className="p-3 whitespace-nowrap">{u.id}</td>
                          <td className="p-3 max-w-[200px] align-top">
                            <EllipsisHoverCell
                              text={`${u.username}@${u.domain}`}
                              lines={2}
                              className="text-sm"
                            />
                          </td>
                          <td className="p-3 max-w-[220px] align-top">
                            <EllipsisHoverCell text={u.contactUri} lines={2} className="text-xs" />
                          </td>
                          <td className="p-3 max-w-[160px] align-top">
                            <EllipsisHoverCell
                              text={
                                u.remoteIp
                                  ? `${u.remoteIp}${u.remotePort != null ? `:${u.remotePort}` : ''}`
                                  : ''
                              }
                              lines={2}
                              mono
                            />
                          </td>
                          <td className="p-3 max-w-[260px] align-top">
                            <EllipsisHoverCell text={u.userAgent} lines={2} className="text-xs" />
                          </td>
                          <td className="p-3 whitespace-nowrap">{u.online ? 'online' : 'offline'}</td>
                          <td className="p-3 whitespace-nowrap text-xs">{fmt(u.expiresAt)}</td>
                          <td className="p-3 whitespace-nowrap text-xs">{fmt(u.lastSeenAt)}</td>
                          <td className="p-3 whitespace-nowrap text-xs">{fmt(u.createdAt)}</td>
                          <td className="p-3 text-right">
                            <Button
                              variant="ghost"
                              size="sm"
                              onClick={() => openSipUserDelete(u.id)}
                              aria-label="delete"
                            >
                              <Trash2 className="w-4 h-4 text-destructive" />
                            </Button>
                          </td>
                        </tr>
                      ))
                    )}
                  </tbody>
                </table>
                <div className="flex items-center justify-between p-3 border-t border-border text-sm">
                  <span className="text-muted-foreground">
                    {t('common.total')}: {usersTotal}
                  </span>
                  <div className="flex gap-2">
                    <Button
                      variant="outline"
                      size="sm"
                      disabled={usersPage <= 1}
                      onClick={() => setUsersPage((p) => Math.max(1, p - 1))}
                    >
                      {t('common.prevPage')}
                    </Button>
                    <Button
                      variant="outline"
                      size="sm"
                      disabled={usersPage * pageSize >= usersTotal}
                      onClick={() => setUsersPage((p) => p + 1)}
                    >
                      {t('common.nextPage')}
                    </Button>
                  </div>
                </div>
              </div>
            )}
          </TabsContent>

          <TabsContent value="calls" className="mt-4 space-y-3">
            <div className="flex flex-wrap gap-2 items-center">
              <input
                className="border border-border rounded-md px-3 py-1.5 text-sm bg-background max-w-xs"
                placeholder="Call-ID"
                value={callFilter}
                onChange={(e) => setCallFilter(e.target.value)}
                onKeyDown={(e) => {
                  if (e.key === 'Enter') {
                    setCallsPage(1)
                    setCallsSearchNonce((n) => n + 1)
                  }
                }}
              />
              <Button
                size="sm"
                onClick={() => {
                  setCallsPage(1)
                  setCallsSearchNonce((n) => n + 1)
                }}
              >
                {t('common.search')}
              </Button>
            </div>
            <p className="text-xs text-muted-foreground max-w-2xl">{t('contactCenter.calls.detailHint')}</p>
            {loading ? (
              <LoadingAnimation />
            ) : (
              <div className="overflow-x-auto rounded-lg border border-border bg-card">
                <table className="min-w-[1480px] w-full text-sm">
                  <thead className="bg-muted/50">
                    <tr>
                      <th className="text-left p-3 whitespace-nowrap">ID</th>
                      <th className="text-left p-3 whitespace-nowrap min-w-[180px]">Call-ID</th>
                      <th className="text-left p-3 whitespace-nowrap">{t('common.status')}</th>
                      <th className="text-left p-3 whitespace-nowrap">Dir</th>
                      <th className="text-left p-3 min-w-[140px]">{t('contactCenter.calls.from')}</th>
                      <th className="text-left p-3 min-w-[140px]">{t('contactCenter.calls.to')}</th>
                      <th className="text-left p-3 whitespace-nowrap text-xs">
                        {t('contactCenter.calls.remoteSig')}
                      </th>
                      <th className="text-left p-3 whitespace-nowrap text-xs font-mono">
                        {t('contactCenter.calls.remoteRtp')}
                      </th>
                      <th className="text-left p-3 whitespace-nowrap text-xs font-mono">
                        {t('contactCenter.calls.localRtp')}
                      </th>
                      <th className="text-left p-3 whitespace-nowrap">Codec</th>
                      <th className="text-left p-3 whitespace-nowrap text-xs">
                        {t('contactCenter.calls.payload')}
                      </th>
                      <th className="text-left p-3 whitespace-nowrap">Dur(s)</th>
                      <th className="text-left p-3 whitespace-nowrap min-w-[120px]">
                        {t('contactCenter.calls.callEnd')}
                      </th>
                      <th className="text-left p-3 whitespace-nowrap">{t('common.time')}</th>
                      <th className="text-left p-3 whitespace-nowrap min-w-[72px]">
                        {t('contactCenter.ai.turnCount')}
                      </th>
                      <th className="text-left p-3 whitespace-nowrap min-w-[72px]">
                        {t('contactCenter.calls.recording')}
                      </th>
                      <th className="text-left p-3 min-w-[120px]">{t('contactCenter.calls.failure')}</th>
                      <th className="text-right p-3 whitespace-nowrap min-w-[100px]">
                        {t('contactCenter.ai.actions')}
                      </th>
                    </tr>
                  </thead>
                  <tbody>
                    {calls.length === 0 ? (
                      <tr>
                        <td colSpan={18} className="p-6 text-center text-muted-foreground">
                          {t('common.noData')}
                        </td>
                      </tr>
                    ) : (
                      calls.map((c) => {
                        const hasRec = Boolean(c.recordingUrl && c.recordingUrl.trim())
                        return (
                          <tr key={c.id} className="border-t border-border align-top">
                            <td className="p-3 whitespace-nowrap">{c.id}</td>
                            <td className="p-3 max-w-[240px] align-top">
                              <EllipsisHoverCell text={c.callId} lines={2} mono />
                            </td>
                            <td className="p-3 whitespace-nowrap">{c.state || '—'}</td>
                            <td className="p-3 whitespace-nowrap">{c.direction || '—'}</td>
                            <td className="p-3 max-w-[200px] align-top">
                              <EllipsisHoverCell text={c.fromHeader} lines={2} className="text-xs" />
                            </td>
                            <td className="p-3 max-w-[200px] align-top">
                              <EllipsisHoverCell text={c.toHeader} lines={2} className="text-xs" />
                            </td>
                            <td className="p-3 max-w-[180px] align-top">
                              <EllipsisHoverCell text={c.remoteAddr} lines={2} mono />
                            </td>
                            <td className="p-3 max-w-[180px] align-top">
                              <EllipsisHoverCell text={c.remoteRtpAddr} lines={2} mono />
                            </td>
                            <td className="p-3 max-w-[180px] align-top">
                              <EllipsisHoverCell text={c.localRtpAddr} lines={2} mono />
                            </td>
                            <td className="p-3 whitespace-nowrap">{c.codec || '—'}</td>
                            <td className="p-3 text-xs whitespace-nowrap">
                              {c.payloadType != null || c.clockRate
                                ? `${c.payloadType ?? '—'} / ${c.clockRate ?? '—'}`
                                : '—'}
                            </td>
                            <td className="p-3 whitespace-nowrap">{c.durationSec ?? '—'}</td>
                            <td className="p-3 text-xs max-w-[140px] align-top">
                              <EllipsisHoverCell
                                text={
                                  c.endStatus
                                    ? t(sipAiEndStatusI18nKey(c.endStatus))
                                    : '—'
                                }
                                lines={2}
                              />
                            </td>
                            <td className="p-3 whitespace-nowrap text-xs">
                              {fmt(c.endedAt || c.byeAt || c.updatedAt)}
                            </td>
                            <td className="p-3 whitespace-nowrap text-xs">
                              {c.turnCount != null && c.turnCount > 0 ? c.turnCount : '—'}
                            </td>
                            <td className="p-3 whitespace-nowrap text-xs">
                              {hasRec ? (
                                <span className="text-primary font-medium">{t('contactCenter.calls.hasRecording')}</span>
                              ) : (
                                <span className="text-muted-foreground">—</span>
                              )}
                            </td>
                            <td className="p-3 max-w-[200px] align-top">
                              <EllipsisHoverCell text={c.failureReason} lines={2} className="text-xs" />
                            </td>
                            <td className="p-3 text-right whitespace-nowrap">
                              <Button
                                variant="outline"
                                size="sm"
                                className="text-xs"
                                onClick={() => void openCallDetailDrawer(c.id)}
                              >
                                {t('contactCenter.calls.viewDetail')}
                              </Button>
                            </td>
                          </tr>
                        )
                      })
                    )}
                  </tbody>
                </table>
                <div className="flex items-center justify-between p-3 border-t border-border text-sm">
                  <span className="text-muted-foreground">
                    {t('common.total')}: {callsTotal}
                  </span>
                  <div className="flex gap-2">
                    <Button
                      variant="outline"
                      size="sm"
                      disabled={callsPage <= 1}
                      onClick={() => setCallsPage((p) => Math.max(1, p - 1))}
                    >
                      {t('common.prevPage')}
                    </Button>
                    <Button
                      variant="outline"
                      size="sm"
                      disabled={callsPage * pageSize >= callsTotal}
                      onClick={() => setCallsPage((p) => p + 1)}
                    >
                      {t('common.nextPage')}
                    </Button>
                  </div>
                </div>
              </div>
            )}
          </TabsContent>

          <TabsContent value="acd" className="mt-0">
            <ACDPoolTab active={tab === 'acd'} refreshNonce={acdRefreshNonce} />
          </TabsContent>

          <TabsContent value="campaign" className="mt-0">
            <OutboundCampaignTab />
          </TabsContent>

          <TabsContent value="scripts" className="mt-0">
            <ScriptManagerTab />
          </TabsContent>

          <TabsContent value="agent" className="mt-4">
            <WebSeatContactTab />
          </TabsContent>
        </Tabs>

        <AnimatePresence>
          {callDetailDrawerId != null && (
            <>
              <motion.button
                key="sip-drawer-overlay"
                type="button"
                aria-label={t('common.close')}
                className="fixed inset-0 z-[100] bg-black/40"
                initial={{ opacity: 0 }}
                animate={{ opacity: 1 }}
                exit={{ opacity: 0 }}
                transition={{ duration: 0.2 }}
                onClick={closeCallDetailDrawer}
              />
              <motion.aside
                key="sip-drawer-panel"
                className="fixed top-0 right-0 z-[101] flex h-full w-full max-w-lg flex-col border-l border-border bg-card shadow-2xl"
                initial={{ x: '100%' }}
                animate={{ x: 0 }}
                exit={{ x: '100%' }}
                transition={{ type: 'spring', damping: 30, stiffness: 320 }}
              >
                {(() => {
                  const d =
                    callDetailDrawerData ??
                    calls.find((c) => c.id === callDetailDrawerId) ??
                    null
                  if (!d) {
                    return (
                      <div className="flex flex-1 items-center justify-center p-6">
                        <LoadingAnimation />
                      </div>
                    )
                  }
                  const turns = callDetailDrawerData?.turns
                  const recUrlResolved =
                    callDetailDrawerData?.recordingUrl?.trim() &&
                    !callDetailDrawerFailed &&
                    !callDetailDrawerLoading
                      ? resolveSipRecordingUrl(callDetailDrawerData.recordingUrl)
                      : ''
                  const detailField = (label: string, value: ReactNode) => (
                    <div className="rounded-md border border-border bg-background/80 p-2.5 text-sm">
                      <div className="mb-0.5 text-[11px] text-muted-foreground">{label}</div>
                      <div className="break-words font-medium">{value ?? '—'}</div>
                    </div>
                  )
                  return (
                    <>
                      <div className="flex shrink-0 items-start justify-between gap-2 border-b border-border px-4 py-3">
                        <div className="min-w-0">
                          <h2 className="text-lg font-semibold leading-tight">
                            {t('contactCenter.calls.detailTitle')}
                          </h2>
                          <p className="mt-1 font-mono text-xs text-muted-foreground break-all">
                            {d.callId}
                          </p>
                        </div>
                        <Button
                          type="button"
                          variant="ghost"
                          size="sm"
                          className="shrink-0 h-9 w-9 p-0"
                          onClick={closeCallDetailDrawer}
                          aria-label={t('common.close')}
                        >
                          <X className="h-5 w-5" />
                        </Button>
                      </div>
                      <div className="min-h-0 flex-1 overflow-y-auto px-4 py-4 space-y-5">
                        <div className="grid grid-cols-2 gap-2 text-xs sm:text-sm">
                          {detailField('ID', d.id)}
                          {detailField(t('common.status'), d.state || '—')}
                          {detailField('Dir', d.direction || '—')}
                          {detailField('Codec', d.codec || '—')}
                          {detailField(
                            t('contactCenter.calls.payload'),
                            d.payloadType != null || d.clockRate
                              ? `${d.payloadType ?? '—'} / ${d.clockRate ?? '—'}`
                              : '—'
                          )}
                          {detailField('Dur(s)', d.durationSec ?? '—')}
                          {detailField(
                            t('contactCenter.calls.callEnd'),
                            d.endStatus ? t(sipAiEndStatusI18nKey(d.endStatus)) : '—'
                          )}
                          {detailField(
                            t('contactCenter.ai.turnCount'),
                            d.turnCount != null && d.turnCount > 0 ? d.turnCount : '—'
                          )}
                          {detailField(t('contactCenter.calls.from'), d.fromHeader || '—')}
                          {detailField(t('contactCenter.calls.to'), d.toHeader || '—')}
                          {detailField(t('contactCenter.calls.remoteSig'), d.remoteAddr || '—')}
                          {detailField(t('contactCenter.calls.remoteRtp'), d.remoteRtpAddr || '—')}
                          {detailField(t('contactCenter.calls.localRtp'), d.localRtpAddr || '—')}
                          {detailField('CSeq', d.cseqInvite || '—')}
                          {detailField(t('common.createdAt'), fmt(d.createdAt))}
                          {detailField('INVITE', fmt(d.inviteAt))}
                          {detailField('ACK', fmt(d.ackAt))}
                          {detailField('BYE', fmt(d.byeAt))}
                          {detailField(t('contactCenter.ai.endedAt'), fmt(d.endedAt))}
                          {detailField(
                            t('contactCenter.ai.transfer'),
                            [d.hadSipTransfer && 'SIP', d.hadWebSeat && 'WebSeat']
                              .filter(Boolean)
                              .join(' · ') || '—'
                          )}
                          <div className="col-span-2">
                            {detailField(t('contactCenter.calls.failure'), d.failureReason || '—')}
                          </div>
                        </div>

                        <div className="rounded-lg border border-border bg-muted/20 p-3">
                          <p className="mb-2 text-sm font-medium text-foreground">
                            {t('contactCenter.calls.recording')}
                          </p>
                          {callDetailDrawerLoading ? (
                            <div className="flex flex-col items-center justify-center gap-3 py-10 text-muted-foreground">
                              <LoadingAnimation />
                              <span className="text-xs">{t('contactCenter.calls.recordingLoading')}</span>
                            </div>
                          ) : callDetailDrawerFailed ? (
                            <div className="flex flex-col items-center gap-3 rounded-md border border-dashed border-destructive/40 bg-destructive/5 px-4 py-8 text-center">
                              <AlertCircle className="h-8 w-8 shrink-0 text-destructive/80" />
                              <p className="text-sm text-foreground">{t('contactCenter.calls.recordingLoadFailed')}</p>
                              <Button
                                variant="outline"
                                size="sm"
                                type="button"
                                onClick={() =>
                                  callDetailDrawerId != null && void openCallDetailDrawer(callDetailDrawerId)
                                }
                              >
                                {t('contactCenter.calls.detailRetry')}
                              </Button>
                            </div>
                          ) : !callDetailDrawerData?.recordingUrl?.trim() ? (
                            <div className="flex flex-col items-center gap-2 rounded-md border border-dashed border-border bg-background/60 px-4 py-8 text-center">
                              <MicOff className="h-7 w-7 shrink-0 text-muted-foreground" />
                              <p className="text-sm text-muted-foreground">{t('contactCenter.calls.recordingEmpty')}</p>
                            </div>
                          ) : (
                            <>
                              <p className="mb-3 text-[11px] leading-snug text-muted-foreground">
                                {t('contactCenter.calls.recordingHint')}
                              </p>
                              <CallAudioPlayer
                                callId={d.callId || `sip-call-${d.id}`}
                                audioUrl={recUrlResolved}
                                hasAudio
                                durationSeconds={
                                  callDetailDrawerData?.durationSec != null &&
                                  callDetailDrawerData.durationSec > 0
                                    ? callDetailDrawerData.durationSec
                                    : null
                                }
                              />
                              <a
                                href={recUrlResolved}
                                target="_blank"
                                rel="noreferrer"
                                className="mt-2 inline-block text-xs text-primary underline"
                              >
                                {t('contactCenter.calls.openRecording')}
                              </a>
                            </>
                          )}
                        </div>

                        <div>
                          <p className="mb-2 text-sm font-medium">{t('contactCenter.ai.detail')}</p>
                          {callDetailDrawerLoading ? (
                            <div className="flex flex-col items-center justify-center gap-2 rounded-md border border-dashed border-border bg-muted/20 px-4 py-8 text-center text-muted-foreground">
                              <LoadingAnimation />
                              <span className="text-xs">{t('contactCenter.calls.dialogLoading')}</span>
                            </div>
                          ) : callDetailDrawerFailed ? (
                            <div className="flex flex-col items-center gap-3 rounded-md border border-dashed border-destructive/40 bg-destructive/5 px-4 py-8 text-center">
                              <AlertCircle className="h-8 w-8 shrink-0 text-destructive/80" />
                              <p className="text-sm text-foreground">{t('contactCenter.calls.dialogLoadFailed')}</p>
                              <Button
                                variant="outline"
                                size="sm"
                                type="button"
                                onClick={() =>
                                  callDetailDrawerId != null && void openCallDetailDrawer(callDetailDrawerId)
                                }
                              >
                                {t('contactCenter.calls.detailRetry')}
                              </Button>
                            </div>
                          ) : !turns || turns.length === 0 ? (
                            <p className="text-xs text-muted-foreground">—</p>
                          ) : (
                            <ul className="space-y-3 rounded-md border border-border bg-background/80 p-3">
                              {turns.map((turn, i) => (
                                <li
                                  key={i}
                                  className="space-y-1 border-l-2 border-primary/40 pl-3 text-sm"
                                >
                                  <div>
                                    <span className="text-xs text-muted-foreground">
                                      {t('contactCenter.ai.user')}{' '}
                                    </span>
                                    {turn.asrText || '—'}
                                  </div>
                                  <div>
                                    <span className="text-xs text-muted-foreground">
                                      {t('contactCenter.ai.assistant')}{' '}
                                    </span>
                                    {turn.llmText || '—'}
                                  </div>
                                  {turn.at ? (
                                    <div className="text-[11px] text-muted-foreground">{fmt(turn.at)}</div>
                                  ) : null}
                                </li>
                              ))}
                            </ul>
                          )}
                        </div>
                      </div>
                    </>
                  )
                })()}
              </motion.aside>
            </>
          )}
        </AnimatePresence>
      </div>

      <ConfirmDialog
        isOpen={sipUserDeleteOpen}
        onClose={() => {
          setSipUserDeleteOpen(false)
          setSipUserDeleteId(null)
        }}
        onConfirm={confirmSipUserDelete}
        title={t('contactCenter.confirm.deleteSipUserTitle')}
        message={t('contactCenter.confirm.deleteSipUserMessage')}
        confirmText={t('contactCenter.confirm.confirmDelete')}
        cancelText={t('common.cancel')}
        type="danger"
      />
    </div>
  )
}
