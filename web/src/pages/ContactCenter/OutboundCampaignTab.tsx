import { useEffect, useMemo, useState } from 'react'
import Button from '@/components/UI/Button'
import LoadingAnimation from '@/components/Animations/LoadingAnimation'
import Modal from '@/components/UI/Modal'
import { useI18nStore } from '@/stores/i18nStore'
import { showAlert } from '@/utils/notification'
import {
  createOutboundCampaign,
  deleteOutboundCampaign,
  enqueueOutboundCampaignContacts,
  listOutboundCampaignContacts,
  resetOutboundCampaignSuppressedContacts,
  getOutboundCampaignLogs,
  getOutboundCampaignMetrics,
  listOutboundCampaigns,
  listSIPScriptTemplates,
  pauseOutboundCampaign,
  resumeOutboundCampaign,
  startOutboundCampaign,
  stopOutboundCampaign,
  type OutboundCampaignRow,
  type OutboundCampaignLogRow,
  type OutboundCampaignContactRow,
  type OutboundCampaignMetrics,
  type SIPScriptTemplateRow,
} from '@/api/sipContactCenter'

export default function OutboundCampaignTab() {
  const { t } = useI18nStore()
  const [campaigns, setCampaigns] = useState<OutboundCampaignRow[]>([])
  const [campaignsTotal, setCampaignsTotal] = useState(0)
  const [campaignsPage, setCampaignsPage] = useState(1)
  const [campaignsLoading, setCampaignsLoading] = useState(false)
  const [creating, setCreating] = useState(false)
  const [opBusyId, setOpBusyId] = useState<number | null>(null)
  const [deletingId, setDeletingId] = useState<number | null>(null)
  const [submittingContacts, setSubmittingContacts] = useState(false)
  const [resetSuppressedBusy, setResetSuppressedBusy] = useState(false)
  const [contactsLoading, setContactsLoading] = useState(false)
  const [contactsRows, setContactsRows] = useState<OutboundCampaignContactRow[]>([])
  const [metricsLoading, setMetricsLoading] = useState(false)
  const [logsLoading, setLogsLoading] = useState(false)
  const [metrics, setMetrics] = useState<OutboundCampaignMetrics | null>(null)
  const [logs, setLogs] = useState<OutboundCampaignLogRow[]>([])
  const [scripts, setScripts] = useState<SIPScriptTemplateRow[]>([])
  const [selectedScriptId, setSelectedScriptId] = useState('')
  const [createModalOpen, setCreateModalOpen] = useState(false)
  const [detailModalOpen, setDetailModalOpen] = useState(false)
  const [detailCampaignId, setDetailCampaignId] = useState<number | null>(null)

  const [name, setName] = useState('')
  const [systemPrompt, setSystemPrompt] = useState('你是电话回访助手，先核验身份，再按流程提问，最后礼貌结束。')
  const [openingMessage, setOpeningMessage] = useState('您好，我是回访助手。')
  const [closingMessage, setClosingMessage] = useState('感谢您的时间，祝您生活愉快。')
  const [outboundHost, setOutboundHost] = useState('')
  const [outboundPort, setOutboundPort] = useState(5060)
  const [signalingAddr, setSignalingAddr] = useState('')
  const [contactsText, setContactsText] = useState('1001\n1002')
  const pageSize = 10

  const detailCampaign = useMemo(() => campaigns.find((c) => c.id === detailCampaignId) || null, [campaigns, detailCampaignId])
  const queueView = useMemo(() => {
    const waiting = contactsRows
      .filter((row) => ['ready', 'retrying'].includes(String(row.status || '').toLowerCase()))
      .slice()
      .sort((a, b) => {
        const ta = a.nextRunAt ? new Date(a.nextRunAt).getTime() : Number.MAX_SAFE_INTEGER
        const tb = b.nextRunAt ? new Date(b.nextRunAt).getTime() : Number.MAX_SAFE_INTEGER
        if (ta !== tb) return ta - tb
        const ca = a.createdAt ? new Date(a.createdAt).getTime() : 0
        const cb = b.createdAt ? new Date(b.createdAt).getTime() : 0
        if (ca !== cb) return ca - cb
        return a.id - b.id
      })
    const positionById = new Map<number, number>()
    waiting.forEach((row, idx) => positionById.set(row.id, idx + 1))
    return {
      total: contactsRows.length,
      waiting: waiting.length,
      dialing: contactsRows.filter((r) => String(r.status || '').toLowerCase() === 'dialing').length,
      active: contactsRows.filter((r) => ['dialing', 'retrying', 'ready'].includes(String(r.status || '').toLowerCase())).length,
      positionById,
    }
  }, [contactsRows])

  useEffect(() => {
    void (async () => {
      try {
        const res = await listSIPScriptTemplates(1, 200)
        if (res.code === 200 && res.data?.list) {
          setScripts(res.data.list.filter((x) => x.enabled))
        }
      } catch {
        setScripts([])
      }
    })()
  }, [])

  const loadCampaigns = async () => {
    setCampaignsLoading(true)
    try {
      const res = await listOutboundCampaigns(campaignsPage, pageSize)
      if (res.code === 200 && res.data) {
        setCampaigns(res.data.list || [])
        setCampaignsTotal(res.data.total || 0)
      }
    } catch (e: unknown) {
      const err = e as { msg?: string }
      showAlert(err?.msg || t('common.failed'), 'error')
    } finally {
      setCampaignsLoading(false)
    }
  }

  useEffect(() => {
    void loadCampaigns()
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [campaignsPage])

  const resetCreateForm = () => {
    setName('')
    setSelectedScriptId('')
    setSystemPrompt('你是电话回访助手，先核验身份，再按流程提问，最后礼貌结束。')
    setOpeningMessage('您好，我是回访助手。')
    setClosingMessage('感谢您的时间，祝您生活愉快。')
    setOutboundHost('')
    setOutboundPort(5060)
    setSignalingAddr('')
  }

  const createCampaign = async () => {
    if (!name.trim()) {
      showAlert(t('contactCenter.campaign.nameRequired'), 'error')
      return
    }
    if (!outboundHost.trim()) {
      showAlert(t('contactCenter.campaign.hostRequired'), 'error')
      return
    }
    setCreating(true)
    try {
      const selected = scripts.find((s) => String(s.id) === selectedScriptId)
      const scriptSpec =
        selected?.scriptSpec != null
          ? typeof selected.scriptSpec === 'string'
            ? selected.scriptSpec
            : JSON.stringify(selected.scriptSpec)
          : JSON.stringify({
              id: 'followup-v1',
              version: '2026-04-06',
              start_id: 'begin',
              steps: [
                { id: 'begin', type: 'say', prompt: '你好，这里是云联络中心回访。', next_id: 'end' },
                { id: 'end', type: 'end' },
              ],
            })
      const res = await createOutboundCampaign({
        name: name.trim(),
        scenario: 'campaign',
        media_profile: 'script',
        script_id: selected?.scriptId || 'followup-v1',
        script_version: '',
        script_spec: scriptSpec,
        system_prompt: systemPrompt.trim(),
        opening_message: openingMessage.trim(),
        closing_message: closingMessage.trim(),
        outbound_host: outboundHost.trim(),
        outbound_port: Number(outboundPort) || 5060,
        signaling_addr: signalingAddr.trim(),
      })
      if (res.code === 200 && res.data?.id) {
        showAlert(t('common.success'), 'success')
        setCreateModalOpen(false)
        resetCreateForm()
        await loadCampaigns()
      } else {
        showAlert(res.msg || t('common.failed'), 'error')
      }
    } catch (e: unknown) {
      const err = e as { msg?: string }
      showAlert(err?.msg || t('common.failed'), 'error')
    } finally {
      setCreating(false)
    }
  }

  const submitContacts = async () => {
    if (!detailCampaignId) {
      showAlert(t('contactCenter.campaign.createFirst'), 'error')
      return
    }
    const contacts = contactsText
      .split('\n')
      .map((x) => x.trim())
      .filter(Boolean)
      .map((phone, idx) => ({ phone, priority: Math.max(1, 10 - idx) }))
    if (!contacts.length) {
      showAlert(t('contactCenter.campaign.contactsRequired'), 'error')
      return
    }
    setSubmittingContacts(true)
    try {
      const res = await enqueueOutboundCampaignContacts(detailCampaignId, contacts)
      if (res.code === 200) {
        showAlert(t('contactCenter.campaign.contactsImported', { count: String(res.data?.accepted || 0) }), 'success')
        void refreshLogs(true)
        void loadContacts(true)
      } else {
        showAlert(res.msg || t('common.failed'), 'error')
      }
    } catch (e: unknown) {
      const err = e as { msg?: string }
      showAlert(err?.msg || t('common.failed'), 'error')
    } finally {
      setSubmittingContacts(false)
    }
  }

  const doCampaignOp = async (campaignId: number, op: 'start' | 'pause' | 'resume' | 'stop') => {
    setOpBusyId(campaignId)
    try {
      const res =
        op === 'start'
          ? await startOutboundCampaign(campaignId)
          : op === 'pause'
            ? await pauseOutboundCampaign(campaignId)
            : op === 'stop'
              ? await stopOutboundCampaign(campaignId)
              : await resumeOutboundCampaign(campaignId)
      if (res.code === 200) showAlert(t('common.success'), 'success')
      else showAlert(res.msg || t('common.failed'), 'error')
      await loadCampaigns()
      if (detailCampaignId === campaignId) {
        void refreshLogs(true)
      }
    } catch (e: unknown) {
      const err = e as { msg?: string }
      showAlert(err?.msg || t('common.failed'), 'error')
    } finally {
      setOpBusyId(null)
    }
  }

  const removeCampaign = async (campaign: OutboundCampaignRow) => {
    if (campaign.status === 'running') {
      showAlert('运行中任务不可删除，请先中断或停止', 'error')
      return
    }
    setDeletingId(campaign.id)
    try {
      const res = await deleteOutboundCampaign(campaign.id)
      if (res.code === 200) {
        showAlert(t('common.success'), 'success')
        if (detailCampaignId === campaign.id) {
          setDetailModalOpen(false)
          setDetailCampaignId(null)
        }
        await loadCampaigns()
      } else {
        showAlert(res.msg || t('common.failed'), 'error')
      }
    } catch (e: unknown) {
      const err = e as { msg?: string }
      showAlert(err?.msg || t('common.failed'), 'error')
    } finally {
      setDeletingId(null)
    }
  }

  const refreshMetrics = async () => {
    setMetricsLoading(true)
    try {
      const res = await getOutboundCampaignMetrics()
      if (res.code === 200 && res.data) setMetrics(res.data)
      else showAlert(res.msg || t('common.failed'), 'error')
    } catch (e: unknown) {
      const err = e as { msg?: string }
      showAlert(err?.msg || t('common.failed'), 'error')
    } finally {
      setMetricsLoading(false)
    }
  }

  const refreshLogs = async (silent = false) => {
    if (!detailCampaignId) {
      setLogs([])
      return
    }
    if (!silent) setLogsLoading(true)
    try {
      const res = await getOutboundCampaignLogs(detailCampaignId, 120)
      if (res.code === 200 && res.data?.list) setLogs(res.data.list)
      else if (!silent) showAlert(res.msg || t('common.failed'), 'error')
    } catch (e: unknown) {
      if (!silent) {
        const err = e as { msg?: string }
        showAlert(err?.msg || t('common.failed'), 'error')
      }
    } finally {
      if (!silent) setLogsLoading(false)
    }
  }

  const loadContacts = async (silent = false) => {
    if (!detailCampaignId) {
      setContactsRows([])
      return
    }
    if (!silent) setContactsLoading(true)
    try {
      const res = await listOutboundCampaignContacts(detailCampaignId, 1, 500)
      if (res.code === 200 && res.data?.list) setContactsRows(res.data.list)
      else if (!silent) showAlert(res.msg || t('common.failed'), 'error')
    } catch (e: unknown) {
      if (!silent) {
        const err = e as { msg?: string }
        showAlert(err?.msg || t('common.failed'), 'error')
      }
    } finally {
      if (!silent) setContactsLoading(false)
    }
  }

  const resetSuppressedContacts = async () => {
    if (!detailCampaignId) return
    setResetSuppressedBusy(true)
    try {
      const res = await resetOutboundCampaignSuppressedContacts(detailCampaignId)
      if (res.code === 200) {
        showAlert(`已重置 ${res.data?.updated || 0} 条 suppressed 联系人`, 'success')
        await loadContacts(true)
        await refreshLogs(true)
      } else {
        showAlert(res.msg || t('common.failed'), 'error')
      }
    } catch (e: unknown) {
      const err = e as { msg?: string }
      showAlert(err?.msg || t('common.failed'), 'error')
    } finally {
      setResetSuppressedBusy(false)
    }
  }

  useEffect(() => {
    if (!detailModalOpen) return
    void refreshLogs()
    void loadContacts()
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [detailCampaignId, detailModalOpen])

  useEffect(() => {
    if (!detailCampaignId || !detailModalOpen) return
    const timer = window.setInterval(() => {
      void refreshLogs(true)
    }, 3000)
    return () => window.clearInterval(timer)
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [detailCampaignId, detailModalOpen])

  return (
    <div className="mt-4 space-y-4">
      <div className="flex items-center justify-between gap-3">
        <p className="text-xs text-muted-foreground leading-relaxed rounded-lg border border-border bg-muted/30 px-3 py-2.5 flex-1">
          {t('contactCenter.campaign.hint')}
        </p>
        <Button
          onClick={() => {
            resetCreateForm()
            setCreateModalOpen(true)
          }}
        >
          {t('contactCenter.campaign.create')}
        </Button>
      </div>

      <div className="rounded-lg border border-border bg-card p-3 space-y-3">
        <div className="flex items-center justify-between gap-2">
          <h3 className="text-sm font-semibold">任务管理列表</h3>
          <Button size="sm" variant="outline" onClick={() => void loadCampaigns()}>
            {t('common.refresh')}
          </Button>
        </div>
        {campaignsLoading ? (
          <LoadingAnimation />
        ) : (
          <div className="max-h-[520px] overflow-auto rounded border border-border">
            <table className="w-full text-xs">
              <thead className="bg-muted/50">
                <tr>
                  <th className="text-left p-2">ID</th>
                  <th className="text-left p-2">{t('contactCenter.campaign.name')}</th>
                  <th className="text-left p-2">{t('contactCenter.script.scriptId')}</th>
                  <th className="text-left p-2">{t('common.status')}</th>
                  <th className="text-left p-2">{t('common.updatedAt')}</th>
                  <th className="text-left p-2">操作</th>
                </tr>
              </thead>
              <tbody>
                {campaigns.map((c) => (
                  <tr key={c.id} className="border-t">
                    <td className="p-2">{c.id}</td>
                    <td className="p-2">{c.name}</td>
                    <td className="p-2 font-mono">{c.scriptId || '—'}</td>
                    <td className="p-2">{c.status || '—'}</td>
                    <td className="p-2">{c.updatedAt ? new Date(c.updatedAt).toLocaleString() : '—'}</td>
                    <td className="p-2">
                      <div className="flex flex-wrap gap-1">
                        <Button
                          size="sm"
                          variant="outline"
                          disabled={opBusyId === c.id}
                          onClick={() => {
                            setDetailCampaignId(c.id)
                            setDetailModalOpen(true)
                          }}
                        >
                          详情
                        </Button>
                        <Button size="sm" disabled={opBusyId === c.id} onClick={() => void doCampaignOp(c.id, 'start')}>
                          启动
                        </Button>
                        <Button size="sm" variant="outline" disabled={opBusyId === c.id} onClick={() => void doCampaignOp(c.id, 'pause')}>
                          中断
                        </Button>
                        <Button size="sm" variant="outline" disabled={opBusyId === c.id} onClick={() => void doCampaignOp(c.id, 'stop')}>
                          停止
                        </Button>
                        <Button
                          size="sm"
                          variant="outline"
                          disabled={deletingId === c.id || c.status === 'running'}
                          onClick={() => void removeCampaign(c)}
                        >
                          删除
                        </Button>
                      </div>
                    </td>
                  </tr>
                ))}
                {campaigns.length === 0 && (
                  <tr>
                    <td colSpan={6} className="p-3 text-center text-muted-foreground">{t('common.noData')}</td>
                  </tr>
                )}
              </tbody>
            </table>
          </div>
        )}
        <div className="flex gap-2">
          <Button variant="outline" size="sm" disabled={campaignsPage <= 1} onClick={() => setCampaignsPage((p) => Math.max(1, p - 1))}>
            {t('common.prevPage')}
          </Button>
          <Button variant="outline" size="sm" disabled={campaignsPage * pageSize >= campaignsTotal} onClick={() => setCampaignsPage((p) => p + 1)}>
            {t('common.nextPage')}
          </Button>
        </div>
      </div>

      <Modal
        isOpen={createModalOpen}
        onClose={() => setCreateModalOpen(false)}
        title={t('contactCenter.campaign.create')}
        size="lg"
      >
        <div className="space-y-3">
          <div className="grid grid-cols-1 md:grid-cols-2 gap-3">
            <div className="space-y-2 md:col-span-2">
              <label className="text-xs text-muted-foreground">{t('contactCenter.script.scriptId')}</label>
              <select
                className="border border-border rounded-md px-3 py-2 bg-background w-full text-sm"
                value={selectedScriptId}
                onChange={(e) => setSelectedScriptId(e.target.value)}
              >
                <option value="">{t('common.none')}</option>
                {scripts.map((s) => (
                  <option key={s.id} value={String(s.id)}>
                    {s.name} ({s.scriptId})
                  </option>
                ))}
              </select>
            </div>
            <div className="space-y-2 md:col-span-2">
              <label className="text-xs text-muted-foreground">{t('contactCenter.campaign.name')}</label>
              <input className="border border-border rounded-md px-3 py-2 bg-background w-full" value={name} onChange={(e) => setName(e.target.value)} />
            </div>
            <div className="space-y-2">
              <label className="text-xs text-muted-foreground">{t('contactCenter.campaign.outboundHost')}</label>
              <input className="border border-border rounded-md px-3 py-2 bg-background w-full" value={outboundHost} onChange={(e) => setOutboundHost(e.target.value)} placeholder="10.0.0.8" />
            </div>
            <div className="space-y-2">
              <label className="text-xs text-muted-foreground">{t('contactCenter.campaign.outboundPort')}</label>
              <input type="number" min={1} max={65535} className="border border-border rounded-md px-3 py-2 bg-background w-full" value={outboundPort} onChange={(e) => setOutboundPort(parseInt(e.target.value, 10) || 5060)} />
            </div>
            <div className="space-y-2 md:col-span-2">
              <label className="text-xs text-muted-foreground">{t('contactCenter.campaign.signalingAddr')}</label>
              <input className="border border-border rounded-md px-3 py-2 bg-background w-full" value={signalingAddr} onChange={(e) => setSignalingAddr(e.target.value)} placeholder="10.0.0.8:5060" />
            </div>
            <div className="space-y-2 md:col-span-2">
              <label className="text-xs text-muted-foreground">{t('contactCenter.campaign.systemPrompt')}</label>
              <textarea className="border border-border rounded-md px-3 py-2 bg-background w-full h-20" value={systemPrompt} onChange={(e) => setSystemPrompt(e.target.value)} />
            </div>
            <div className="space-y-2">
              <label className="text-xs text-muted-foreground">{t('contactCenter.campaign.opening')}</label>
              <input className="border border-border rounded-md px-3 py-2 bg-background w-full" value={openingMessage} onChange={(e) => setOpeningMessage(e.target.value)} />
            </div>
            <div className="space-y-2">
              <label className="text-xs text-muted-foreground">{t('contactCenter.campaign.closing')}</label>
              <input className="border border-border rounded-md px-3 py-2 bg-background w-full" value={closingMessage} onChange={(e) => setClosingMessage(e.target.value)} />
            </div>
          </div>
          <div className="flex justify-end gap-2">
            <Button variant="outline" onClick={() => setCreateModalOpen(false)} disabled={creating}>
              {t('common.cancel')}
            </Button>
            <Button onClick={() => void createCampaign()} disabled={creating}>
              {creating ? t('common.loading') : t('contactCenter.campaign.create')}
            </Button>
          </div>
        </div>
      </Modal>

      <Modal
        isOpen={detailModalOpen}
        onClose={() => setDetailModalOpen(false)}
        title={detailCampaign ? `任务详情 #${detailCampaign.id} - ${detailCampaign.name}` : '任务详情'}
        size="xl"
      >
        <div className="space-y-3">
          <div className="flex flex-wrap gap-2">
            <Button
              size="sm"
              onClick={() => detailCampaignId && void doCampaignOp(detailCampaignId, 'start')}
              disabled={!detailCampaignId || opBusyId === detailCampaignId}
            >
              启动
            </Button>
            <Button
              size="sm"
              variant="outline"
              onClick={() => detailCampaignId && void doCampaignOp(detailCampaignId, 'pause')}
              disabled={!detailCampaignId || opBusyId === detailCampaignId}
            >
              中断
            </Button>
            <Button
              size="sm"
              variant="outline"
              onClick={() => detailCampaignId && void doCampaignOp(detailCampaignId, 'stop')}
              disabled={!detailCampaignId || opBusyId === detailCampaignId}
            >
              停止
            </Button>
          </div>

          <div className="space-y-2">
            <label className="text-xs text-muted-foreground">{t('contactCenter.campaign.contacts')}（导入）</label>
            <textarea className="border border-border rounded-md px-3 py-2 bg-background w-full h-24 font-mono text-xs" value={contactsText} onChange={(e) => setContactsText(e.target.value)} />
            <Button size="sm" variant="outline" onClick={() => void submitContacts()} disabled={submittingContacts || !detailCampaignId}>
              {submittingContacts ? t('common.loading') : t('contactCenter.campaign.importContacts')}
            </Button>
            <Button size="sm" variant="outline" onClick={() => void resetSuppressedContacts()} disabled={!detailCampaignId || resetSuppressedBusy}>
              {resetSuppressedBusy ? t('common.loading') : '重置被抑制号码'}
            </Button>
          </div>

          <div className="rounded-lg border border-border bg-card p-3 space-y-2">
            <div className="flex items-center justify-between">
              <h3 className="text-sm font-semibold">已导入联系人</h3>
              <Button size="sm" variant="outline" onClick={() => void loadContacts()} disabled={!detailCampaignId || contactsLoading}>
                {contactsLoading ? t('common.loading') : t('common.refresh')}
              </Button>
            </div>
            <div className="grid grid-cols-2 md:grid-cols-4 gap-2 text-xs">
              <div className="rounded border border-border p-2">总联系人: {queueView.total}</div>
              <div className="rounded border border-border p-2">队列中: {queueView.waiting}</div>
              <div className="rounded border border-border p-2">拨号中: {queueView.dialing}</div>
              <div className="rounded border border-border p-2">活跃任务: {queueView.active}</div>
            </div>
            <div className="max-h-52 overflow-auto rounded border border-border">
              <table className="w-full text-xs">
                <thead className="bg-muted/50">
                  <tr>
                    <th className="text-left p-2">号码</th>
                    <th className="text-left p-2">队列位置</th>
                    <th className="text-left p-2">状态</th>
                    <th className="text-left p-2">尝试</th>
                    <th className="text-left p-2">失败原因</th>
                    <th className="text-left p-2">下次重试</th>
                  </tr>
                </thead>
                <tbody>
                  {contactsRows.map((row) => (
                    <tr key={row.id} className="border-t">
                      <td className="p-2 font-mono">{row.phone}</td>
                      <td className="p-2">{queueView.positionById.get(row.id) || '—'}</td>
                      <td className="p-2">{row.status || '—'}</td>
                      <td className="p-2">{`${row.attemptCount ?? 0}/${row.maxAttempts ?? 0}`}</td>
                      <td className="p-2">{row.failureReason || '—'}</td>
                      <td className="p-2">{row.nextRunAt ? new Date(row.nextRunAt).toLocaleString() : '—'}</td>
                    </tr>
                  ))}
                  {contactsRows.length === 0 && (
                    <tr>
                      <td colSpan={6} className="p-3 text-center text-muted-foreground">暂无联系人</td>
                    </tr>
                  )}
                </tbody>
              </table>
            </div>
          </div>

          <div className="rounded-lg border border-border bg-card p-3 space-y-2">
            <div className="flex items-center justify-between">
              <h3 className="text-sm font-semibold">{t('contactCenter.campaign.metrics')}</h3>
              <Button size="sm" variant="outline" onClick={() => void refreshMetrics()} disabled={metricsLoading}>
                {metricsLoading ? t('common.loading') : t('common.refresh')}
              </Button>
            </div>
            <div className="grid grid-cols-2 md:grid-cols-5 gap-2 text-xs">
              <div className="rounded border border-border p-2">invited: {metrics?.invited_total ?? 0}</div>
              <div className="rounded border border-border p-2">answered: {metrics?.answered_total ?? 0}</div>
              <div className="rounded border border-border p-2">failed: {metrics?.failed_total ?? 0}</div>
              <div className="rounded border border-border p-2">retrying: {metrics?.retrying_total ?? 0}</div>
              <div className="rounded border border-border p-2">suppressed: {metrics?.suppressed_total ?? 0}</div>
            </div>
          </div>

          <div className="rounded-lg border border-border bg-card p-3 space-y-2">
            <div className="flex items-center justify-between">
              <h3 className="text-sm font-semibold">执行日志终端</h3>
              <Button size="sm" variant="outline" onClick={() => void refreshLogs()} disabled={!detailCampaignId || logsLoading}>
                {logsLoading ? t('common.loading') : t('common.refresh')}
              </Button>
            </div>
            <div className="rounded border border-border bg-black text-green-300 text-xs font-mono p-2 h-64 overflow-auto">
              {!detailCampaignId && <div className="text-zinc-400">请选择任务后查看日志</div>}
              {detailCampaignId && logs.length === 0 && <div className="text-zinc-400">暂无执行日志</div>}
              {logs.map((row) => (
                <div key={`${row.type}-${row.id}-${row.at}`} className="leading-5 break-all">
                  <span className="text-zinc-400">[{new Date(row.at).toLocaleString()}]</span>{' '}
                  <span className={row.level === 'error' ? 'text-red-300' : 'text-cyan-300'}>{row.type.toUpperCase()}</span>{' '}
                  {row.phone ? <span className="text-yellow-200">phone={row.phone} </span> : null}
                  {row.callId ? <span className="text-yellow-200">call={row.callId} </span> : null}
                  <span>{row.message}</span>
                </div>
              ))}
            </div>
          </div>
        </div>
      </Modal>
    </div>
  )
}

