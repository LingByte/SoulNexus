import { useCallback, useEffect, useState } from 'react'
import { Trash2 } from 'lucide-react'
import { useI18nStore } from '@/stores/i18nStore'
import Button from '@/components/UI/Button'
import Badge from '@/components/UI/Badge'
import LoadingAnimation from '@/components/Animations/LoadingAnimation'
import { showAlert } from '@/utils/notification'
import {
  ACD_ROUTE_TYPES,
  ACD_SIP_SOURCES,
  ACD_WORK_STATES,
  createACDPoolTarget,
  deleteACDPoolTarget,
  fetchSIPUsersForSelect,
  listACDPoolTargets,
  updateACDPoolTarget,
  type ACDPoolTargetRow,
  type ACDSipSource,
  type SIPUserRow,
} from '@/api/sipContactCenter'

function acdTrunkGatewayCell(r: ACDPoolTargetRow): string {
  if (r.routeType !== 'sip' || (r.sipSource || '').toLowerCase() !== 'trunk') return '—'
  const h = (r.sipTrunkHost || '').trim()
  if (!h) return '—'
  const p = r.sipTrunkPort != null && r.sipTrunkPort > 0 ? r.sipTrunkPort : 5060
  const base = `${h}:${p}`
  const sig = (r.sipTrunkSignalingAddr || '').trim()
  return sig ? `${base} → ${sig}` : base
}

function acdCallerCell(r: ACDPoolTargetRow): string {
  if (r.routeType !== 'sip') return '—'
  const id = (r.sipCallerId || '').trim()
  if (!id) return '—'
  const d = (r.sipCallerDisplayName || '').trim()
  return d ? `${id} · ${d}` : id
}

type FormState = {
  name: string
  routeType: string
  sipSource: ACDSipSource
  targetValue: string
  sipTrunkHost: string
  sipTrunkPort: number
  sipTrunkSignalingAddr: string
  sipCallerId: string
  sipCallerDisplayName: string
  weight: number
  workState: string
}

const defaultForm = (): FormState => ({
  name: '',
  routeType: 'sip',
  sipSource: 'internal',
  targetValue: '',
  sipTrunkHost: '',
  sipTrunkPort: 5060,
  sipTrunkSignalingAddr: '',
  sipCallerId: '',
  sipCallerDisplayName: '',
  weight: 10,
  workState: 'offline',
})

export default function ACDPoolTab({
  active,
  refreshNonce = 0,
}: {
  active: boolean
  refreshNonce?: number
}) {
  const { t } = useI18nStore()
  const [rows, setRows] = useState<ACDPoolTargetRow[]>([])
  const [total, setTotal] = useState(0)
  const [page, setPage] = useState(1)
  const [routeTypeFilter, setRouteTypeFilter] = useState('')
  const [loading, setLoading] = useState(false)
  const [modalOpen, setModalOpen] = useState(false)
  const [editingId, setEditingId] = useState<number | null>(null)
  const [form, setForm] = useState<FormState>(defaultForm)
  const [saving, setSaving] = useState(false)
  const [sipUsersPick, setSipUsersPick] = useState<SIPUserRow[]>([])
  const pageSize = 20
  const load = useCallback(async () => {
    if (!active) return
    setLoading(true)
    try {
      const res = await listACDPoolTargets(page, pageSize, {
        routeType: routeTypeFilter.trim() || undefined,
      })
      if (res.code === 200 && res.data) {
        setRows(res.data.list || [])
        setTotal(res.data.total || 0)
      }
    } catch (e: unknown) {
      const err = e as { msg?: string }
      showAlert(err?.msg || t('common.failed'), 'error')
    } finally {
      setLoading(false)
    }
  }, [active, page, routeTypeFilter, t])

  useEffect(() => {
    void load()
  }, [load, refreshNonce])

  useEffect(() => {
    if (!modalOpen || !active) return
    if (editingId != null && form.routeType !== 'sip') return
    let cancelled = false
    void (async () => {
      try {
        const list = await fetchSIPUsersForSelect(500)
        if (!cancelled) setSipUsersPick(list)
      } catch {
        if (!cancelled) setSipUsersPick([])
      }
    })()
    return () => {
      cancelled = true
    }
  }, [modalOpen, active, editingId, form.routeType])

  const openCreate = () => {
    setEditingId(null)
    setForm(defaultForm())
    setModalOpen(true)
  }

  const openEdit = (r: ACDPoolTargetRow) => {
    setEditingId(r.id)
    const src = (r.sipSource || '').toLowerCase() === 'trunk' ? 'trunk' : 'internal'
    setForm({
      name: r.name || '',
      routeType: r.routeType || 'sip',
      sipSource: r.routeType === 'web' ? 'internal' : src,
      targetValue: r.targetValue || '',
      sipTrunkHost: r.sipTrunkHost || '',
      sipTrunkPort: r.sipTrunkPort != null && r.sipTrunkPort > 0 ? r.sipTrunkPort : 5060,
      sipTrunkSignalingAddr: r.sipTrunkSignalingAddr || '',
      sipCallerId: r.sipCallerId || '',
      sipCallerDisplayName: r.sipCallerDisplayName || '',
      weight: r.weight ?? 0,
      workState: r.workState || 'offline',
    })
    setModalOpen(true)
  }

  const closeModal = () => {
    setModalOpen(false)
    setEditingId(null)
  }

  const save = async () => {
    setSaving(true)
    try {
      const routeType = editingId == null ? 'sip' : form.routeType
      const tv = routeType === 'sip' ? form.targetValue.trim() : ''
      if (routeType === 'sip' && !tv) {
        showAlert(t('contactCenter.acd.sipTargetRequired'), 'error')
        setSaving(false)
        return
      }
      if (routeType === 'sip' && form.sipSource === 'trunk' && !form.sipTrunkHost.trim()) {
        showAlert(t('contactCenter.acd.sipTrunkHostRequired'), 'error')
        setSaving(false)
        return
      }
      const trunkPort = Number(form.sipTrunkPort) || 5060
      const body = {
        name: form.name.trim(),
        routeType,
        sipSource: routeType === 'sip' ? form.sipSource : '',
        targetValue: tv,
        sipTrunkHost:
          routeType === 'sip' && form.sipSource === 'trunk' ? form.sipTrunkHost.trim() : '',
        sipTrunkPort: routeType === 'sip' && form.sipSource === 'trunk' ? trunkPort : 0,
        sipTrunkSignalingAddr:
          routeType === 'sip' && form.sipSource === 'trunk' ? form.sipTrunkSignalingAddr.trim() : '',
        sipCallerId: routeType === 'sip' ? form.sipCallerId.trim() : '',
        sipCallerDisplayName: routeType === 'sip' ? form.sipCallerDisplayName.trim() : '',
        weight: Number(form.weight) || 0,
        workState: form.workState,
      }
      const res =
        editingId == null
          ? await createACDPoolTarget(body)
          : await updateACDPoolTarget(editingId, body)
      if (res.code === 200) {
        showAlert(t('common.success'), 'success')
        closeModal()
        void load()
      } else {
        showAlert(res.msg || t('common.failed'), 'error')
      }
    } catch (e: unknown) {
      const err = e as { msg?: string }
      showAlert(err?.msg || t('common.failed'), 'error')
    } finally {
      setSaving(false)
    }
  }

  const onDelete = async (id: number) => {
    if (!window.confirm(t('common.confirm'))) return
    try {
      const res = await deleteACDPoolTarget(id)
      if (res.code === 200) {
        showAlert(t('common.success'), 'success')
        void load()
      } else {
        showAlert(res.msg || t('common.failed'), 'error')
      }
    } catch (e: unknown) {
      const err = e as { msg?: string }
      showAlert(err?.msg || t('common.failed'), 'error')
    }
  }

  const workStateLabel = (s: string) => t(`contactCenter.acd.workState.${s}`)

  return (
    <div className="mt-4 space-y-3">
      <p className="text-xs text-muted-foreground leading-relaxed rounded-lg border border-border bg-primary/5 px-3 py-2.5">
        {t('contactCenter.acd.transferPickHint')}
      </p>
      <p className="text-xs text-muted-foreground leading-relaxed rounded-lg border border-border bg-muted/30 px-3 py-2.5">
        {t('contactCenter.acd.sipRegisterNotAcdHint')}
      </p>
      <div className="flex flex-wrap gap-2 items-end">
        <div className="flex flex-col gap-1">
          <label className="text-xs text-muted-foreground">{t('contactCenter.acd.routeType')}</label>
          <select
            className="border border-border rounded-md px-3 py-1.5 text-sm bg-background w-28"
            value={routeTypeFilter}
            onChange={(e) => setRouteTypeFilter(e.target.value)}
          >
            <option value="">{t('common.all')}</option>
            {ACD_ROUTE_TYPES.map((rt) => (
              <option key={rt} value={rt}>
                {rt}
              </option>
            ))}
          </select>
        </div>
        <Button
          size="sm"
          onClick={() => {
            setPage(1)
            void load()
          }}
        >
          {t('common.search')}
        </Button>
        <Button size="sm" variant="outline" onClick={openCreate}>
          {t('contactCenter.acd.addSip')}
        </Button>
      </div>

      {loading ? (
        <LoadingAnimation />
      ) : (
        <div className="overflow-x-auto rounded-lg border border-border bg-card">
          <table className="min-w-[1240px] w-full text-sm">
            <thead className="bg-muted/50">
              <tr>
                <th className="text-left p-3 whitespace-nowrap">ID</th>
                <th className="text-left p-3 whitespace-nowrap">{t('contactCenter.acd.name')}</th>
                <th className="text-left p-3 whitespace-nowrap">{t('contactCenter.acd.routeType')}</th>
                <th className="text-left p-3 whitespace-nowrap min-w-[100px]">{t('contactCenter.acd.sipSourceColumn')}</th>
                <th className="text-left p-3 min-w-[140px]">{t('contactCenter.acd.targetColumn')}</th>
                <th className="text-left p-3 min-w-[160px]">{t('contactCenter.acd.sipTrunkGatewayColumn')}</th>
                <th className="text-left p-3 min-w-[120px]">{t('contactCenter.acd.sipCallerIdColumn')}</th>
                <th className="text-left p-3 whitespace-nowrap">{t('contactCenter.acd.weight')}</th>
                <th className="text-left p-3 min-w-[200px]">{t('contactCenter.acd.status')}</th>
                <th className="text-left p-3 whitespace-nowrap text-xs">{t('contactCenter.acd.workStateAt')}</th>
                <th className="text-right p-3 whitespace-nowrap sticky right-0 z-[2] border-l border-border bg-muted/50">
                  {t('contactCenter.ai.actions')}
                </th>
              </tr>
            </thead>
            <tbody>
              {rows.length === 0 ? (
                <tr>
                  <td colSpan={11} className="p-6 text-center text-muted-foreground">
                    {t('common.noData')}
                  </td>
                </tr>
              ) : (
                rows.map((r) => (
                  <tr key={r.id} className="border-t border-border align-top">
                    <td className="p-3 whitespace-nowrap">{r.id}</td>
                    <td className="p-3 max-w-[160px] truncate">{r.name || '—'}</td>
                    <td className="p-3 whitespace-nowrap">{r.routeType}</td>
                    <td className="p-3 whitespace-nowrap text-xs">
                      {r.routeType === 'sip' ? (
                        (r.sipSource || '').toLowerCase() === 'trunk' ? (
                          <Badge variant="outline" size="xs" shape="pill">
                            {t('contactCenter.acd.sipSourceTrunk')}
                          </Badge>
                        ) : (
                          <Badge variant="secondary" size="xs" shape="pill">
                            {t('contactCenter.acd.sipSourceInternal')}
                          </Badge>
                        )
                      ) : (
                        '—'
                      )}
                    </td>
                    <td className="p-3 font-mono text-xs max-w-[200px] break-all text-muted-foreground">
                      {r.routeType === 'sip' ? r.targetValue || '—' : '—'}
                    </td>
                    <td className="p-3 font-mono text-xs max-w-[220px] break-all text-muted-foreground">
                      {acdTrunkGatewayCell(r)}
                    </td>
                    <td className="p-3 font-mono text-xs max-w-[180px] break-all text-muted-foreground">
                      {acdCallerCell(r)}
                    </td>
                    <td className="p-3 whitespace-nowrap">{r.weight}</td>
                    <td className="p-3 align-top">
                      <div className="flex flex-wrap gap-1.5">
                        <Badge variant="outline" size="xs" shape="pill">
                          {workStateLabel(r.workState)}
                        </Badge>
                        {r.routeType === 'sip' &&
                          ((r.sipSource || '').toLowerCase() === 'trunk' ? (
                            <Badge variant="outline" size="xs" shape="pill">
                              {t('contactCenter.acd.sipTrunkRegHint')}
                            </Badge>
                          ) : (
                            <Badge
                              variant={r.liveLineOnline ? 'success' : 'muted'}
                              size="xs"
                              shape="pill"
                            >
                              {r.liveLineOnline
                                ? t('contactCenter.acd.liveSipOnline')
                                : t('contactCenter.acd.liveSipOffline')}
                            </Badge>
                          ))}
                        {r.routeType === 'web' && (
                          <Badge variant={r.liveLineOnline ? 'success' : 'muted'} size="xs" shape="pill">
                            {r.liveLineOnline
                              ? t('contactCenter.acd.liveWebOnline')
                              : t('contactCenter.acd.liveWebOffline')}
                          </Badge>
                        )}
                      </div>
                    </td>
                    <td className="p-3 whitespace-nowrap text-xs text-muted-foreground">
                      {r.workStateAt ? new Date(r.workStateAt).toLocaleString() : '—'}
                    </td>
                    <td className="p-3 text-right sticky right-0 z-[2] border-l border-border bg-card">
                      <div className="flex flex-wrap items-center justify-end gap-1">
                        <Button variant="outline" size="sm" className="text-xs" onClick={() => openEdit(r)}>
                          {t('common.edit')}
                        </Button>
                        <Button
                          variant="outline"
                          size="sm"
                          className="text-xs text-destructive border-destructive/40 hover:bg-destructive/10"
                          onClick={() => void onDelete(r.id)}
                        >
                          <Trash2 className="h-3.5 w-3.5 sm:mr-1" aria-hidden />
                          <span className="hidden sm:inline">{t('common.delete')}</span>
                        </Button>
                      </div>
                    </td>
                  </tr>
                ))
              )}
            </tbody>
          </table>
          <div className="flex items-center justify-between p-3 border-t border-border text-sm">
            <span className="text-muted-foreground">
              {t('common.total')}: {total}
            </span>
            <div className="flex gap-2">
              <Button variant="outline" size="sm" disabled={page <= 1} onClick={() => setPage((p) => Math.max(1, p - 1))}>
                {t('common.prevPage')}
              </Button>
              <Button
                variant="outline"
                size="sm"
                disabled={page * pageSize >= total}
                onClick={() => setPage((p) => p + 1)}
              >
                {t('common.nextPage')}
              </Button>
            </div>
          </div>
        </div>
      )}

      {modalOpen && (
        <div className="fixed inset-0 z-[110] flex items-center justify-center p-4">
          <button
            type="button"
            className="absolute inset-0 bg-black/50"
            aria-label={t('common.close')}
            onClick={closeModal}
          />
          <div className="relative z-[111] w-full max-w-xl rounded-lg border border-border bg-card p-5 shadow-xl space-y-4 max-h-[90vh] overflow-y-auto">
            <h3 className="text-lg font-semibold">
              {editingId == null ? t('contactCenter.acd.addSip') : t('contactCenter.acd.edit')}
            </h3>
            <div className="space-y-3 text-sm">
              <div className="flex flex-col gap-1">
                <label className="text-xs text-muted-foreground">{t('contactCenter.acd.name')}</label>
                <input
                  className="border border-border rounded-md px-3 py-2 bg-background"
                  value={form.name}
                  onChange={(e) => setForm((f) => ({ ...f, name: e.target.value }))}
                />
              </div>
              {editingId == null ? (
                <p className="text-[11px] text-muted-foreground leading-snug rounded-md border border-border bg-muted/20 px-3 py-2">
                  {t('contactCenter.acd.createSipOnlyHint')}
                </p>
              ) : (
                <div className="flex flex-col gap-1">
                  <label className="text-xs text-muted-foreground">{t('contactCenter.acd.routeType')}</label>
                  <p className="rounded-md border border-border bg-muted/30 px-3 py-2 font-mono text-xs">
                    {form.routeType}
                    {form.routeType === 'web' && (
                      <span className="ml-2 text-muted-foreground font-sans text-[11px]">
                        ({t('contactCenter.acd.routeTypeReadonlyWeb')})
                      </span>
                    )}
                  </p>
                </div>
              )}
              {(editingId == null || form.routeType === 'sip') && (
                <>
                  <div className="flex flex-col gap-1">
                    <label className="text-xs text-muted-foreground">{t('contactCenter.acd.sipSourceField')}</label>
                    <select
                      className="border border-border rounded-md px-3 py-2 bg-background text-sm"
                      value={form.sipSource}
                      onChange={(e) =>
                        setForm((f) => ({
                          ...f,
                          sipSource: e.target.value as ACDSipSource,
                          targetValue: '',
                          sipTrunkHost: '',
                          sipTrunkPort: 5060,
                          sipTrunkSignalingAddr: '',
                        }))
                      }
                    >
                      {ACD_SIP_SOURCES.map((s) => (
                        <option key={s} value={s}>
                          {s === 'internal'
                            ? t('contactCenter.acd.sipSourceInternal')
                            : t('contactCenter.acd.sipSourceTrunk')}
                        </option>
                      ))}
                    </select>
                    <p className="text-[11px] text-muted-foreground leading-snug">{t('contactCenter.acd.sipSourceHint')}</p>
                  </div>
                  {form.sipSource === 'internal' ? (
                    <div className="flex flex-col gap-1">
                      <label className="text-xs text-muted-foreground">{t('contactCenter.acd.sipPickRegisteredUser')}</label>
                      <select
                        className="border border-border rounded-md px-3 py-2 bg-background font-mono text-xs"
                        value={form.targetValue}
                        onChange={(e) => setForm((f) => ({ ...f, targetValue: e.target.value }))}
                      >
                        <option value="">{t('contactCenter.acd.sipPickPlaceholder')}</option>
                        {sipUsersPick.map((u) => (
                          <option key={u.id} value={u.username}>
                            {u.username}@{u.domain}
                            {u.online ? ` · ${t('contactCenter.acd.sipUserOnlineMark')}` : ''}
                          </option>
                        ))}
                      </select>
                      <p className="text-[11px] text-muted-foreground leading-snug">{t('contactCenter.acd.sipDialHint')}</p>
                    </div>
                  ) : (
                    <div className="space-y-3">
                      <div className="flex flex-col gap-1">
                        <label className="text-xs text-muted-foreground">{t('contactCenter.acd.sipTrunkTarget')}</label>
                        <input
                          className="border border-border rounded-md px-3 py-2 bg-background font-mono text-xs"
                          value={form.targetValue}
                          onChange={(e) => setForm((f) => ({ ...f, targetValue: e.target.value }))}
                          placeholder={t('contactCenter.acd.sipTrunkPlaceholder')}
                        />
                      </div>
                      <div className="flex flex-col gap-1">
                        <label className="text-xs text-muted-foreground">{t('contactCenter.acd.sipTrunkHostLabel')}</label>
                        <input
                          className="border border-border rounded-md px-3 py-2 bg-background font-mono text-xs"
                          value={form.sipTrunkHost}
                          onChange={(e) => setForm((f) => ({ ...f, sipTrunkHost: e.target.value }))}
                          placeholder={t('contactCenter.acd.sipTrunkHostPlaceholder')}
                        />
                      </div>
                      <div className="flex flex-col gap-1">
                        <label className="text-xs text-muted-foreground">{t('contactCenter.acd.sipTrunkPortLabel')}</label>
                        <input
                          type="number"
                          min={1}
                          max={65535}
                          className="border border-border rounded-md px-3 py-2 bg-background font-mono text-xs"
                          value={form.sipTrunkPort}
                          onChange={(e) =>
                            setForm((f) => ({ ...f, sipTrunkPort: parseInt(e.target.value, 10) || 5060 }))
                          }
                        />
                        <p className="text-[11px] text-muted-foreground leading-snug">
                          {t('contactCenter.acd.sipTrunkPortHint')}
                        </p>
                      </div>
                      <div className="flex flex-col gap-1">
                        <label className="text-xs text-muted-foreground">{t('contactCenter.acd.sipTrunkSignalingLabel')}</label>
                        <input
                          className="border border-border rounded-md px-3 py-2 bg-background font-mono text-xs"
                          value={form.sipTrunkSignalingAddr}
                          onChange={(e) => setForm((f) => ({ ...f, sipTrunkSignalingAddr: e.target.value }))}
                          placeholder={t('contactCenter.acd.sipTrunkSignalingPlaceholder')}
                        />
                        <p className="text-[11px] text-muted-foreground leading-snug">
                          {t('contactCenter.acd.sipTrunkSignalingHint')}
                        </p>
                      </div>
                      <p className="text-[11px] text-muted-foreground leading-snug rounded-md border border-border bg-muted/20 px-3 py-2">
                        {t('contactCenter.acd.sipTrunkHint')}
                      </p>
                    </div>
                  )}
                  <div className="space-y-3 rounded-md border border-border bg-muted/10 px-3 py-3">
                    <p className="text-xs font-medium text-foreground">{t('contactCenter.acd.sipCallerSection')}</p>
                    <div className="flex flex-col gap-1">
                      <label className="text-xs text-muted-foreground">{t('contactCenter.acd.sipCallerIdLabel')}</label>
                      <input
                        className="border border-border rounded-md px-3 py-2 bg-background font-mono text-xs"
                        value={form.sipCallerId}
                        onChange={(e) => setForm((f) => ({ ...f, sipCallerId: e.target.value }))}
                        placeholder={t('contactCenter.acd.sipCallerIdPlaceholder')}
                      />
                    </div>
                    <div className="flex flex-col gap-1">
                      <label className="text-xs text-muted-foreground">{t('contactCenter.acd.sipCallerDisplayLabel')}</label>
                      <input
                        className="border border-border rounded-md px-3 py-2 bg-background text-xs"
                        value={form.sipCallerDisplayName}
                        onChange={(e) => setForm((f) => ({ ...f, sipCallerDisplayName: e.target.value }))}
                        placeholder={t('contactCenter.acd.sipCallerDisplayPlaceholder')}
                      />
                    </div>
                    <p className="text-[11px] text-muted-foreground leading-snug">{t('contactCenter.acd.sipCallerHint')}</p>
                  </div>
                </>
              )}
              {editingId != null && form.routeType === 'web' && (
                <p className="text-[11px] text-muted-foreground leading-snug rounded-md border border-dashed border-border bg-muted/20 px-3 py-2">
                  {t('contactCenter.acd.webNoTargetHint')}
                </p>
              )}
              <div className="flex flex-col gap-1">
                <label className="text-xs text-muted-foreground">{t('contactCenter.acd.weight')}</label>
                <input
                  type="number"
                  className="border border-border rounded-md px-3 py-2 bg-background"
                  value={form.weight}
                  onChange={(e) => setForm((f) => ({ ...f, weight: parseInt(e.target.value, 10) || 0 }))}
                />
                <p className="text-[11px] text-muted-foreground">{t('contactCenter.acd.weightHint')}</p>
              </div>
              <div className="flex flex-col gap-1">
                <label className="text-xs text-muted-foreground">{t('contactCenter.acd.workState')}</label>
                <select
                  className="border border-border rounded-md px-3 py-2 bg-background"
                  value={form.workState}
                  onChange={(e) => setForm((f) => ({ ...f, workState: e.target.value }))}
                >
                  {ACD_WORK_STATES.map((ws) => (
                    <option key={ws} value={ws}>
                      {workStateLabel(ws)}
                    </option>
                  ))}
                </select>
              </div>
            </div>
            <div className="flex justify-end gap-2 pt-2">
              <Button variant="outline" type="button" onClick={closeModal} disabled={saving}>
                {t('common.cancel')}
              </Button>
              <Button type="button" onClick={() => void save()} disabled={saving}>
                {saving ? t('common.loading') : t('common.save')}
              </Button>
            </div>
          </div>
        </div>
      )}
    </div>
  )
}
