import { useCallback, useEffect, useState } from 'react'
import {
  Drawer,
  Modal,
  Space,
  Tag,
  Typography,
} from '@arco-design/web-react'
import { Button, Input, Select, Card, TableEmpty } from '@/components/ui'
import { Loading } from '@/components/ui/loading'
import { IconCopy, IconDelete } from '@arco-design/web-react/icon'
import BaseLayout from '@/components/Layout/BaseLayout'
import {
  createCredential,
  deleteCredential,
  disableCredential,
  enableCredential,
  listCredentials,
  regenerateCredential,
  updateCredential,
  type CredentialCreateResult,
  type CredentialRow,
  type CredentialStatus,
} from '@/api/credentials'
import { showAlert } from '@/utils/notification'
import { extractApiErrorMessage } from '@/utils/apiError'
import { useTranslation } from '@/i18n'
import { getCredentialAkskRouteCatalog, type AkskRouteCatalogGroup } from '@/api/akskRoutePolicy'
import AkskRoutePicker from '@/components/platform/AkskRoutePicker'

type EditState = {
  id: string | null
  name: string
  allowIp: string
  permissionCodesJson: string
  allowedRouteIds: string[]
  expiresAt: string
}

const defaultEdit = (): EditState => ({
  id: null,
  name: '',
  allowIp: '',
  permissionCodesJson: '[]',
  allowedRouteIds: [],
  expiresAt: '',
})

function permissionsFromRouteSelection(groups: AkskRouteCatalogGroup[], routeIds: string[]): string[] {
  const allow = new Set(routeIds)
  const perms = new Set<string>()
  for (const g of groups) {
    for (const e of g.entries) {
      if (allow.has(e.id) && e.permission) perms.add(e.permission)
    }
  }
  return perms.size > 0 ? Array.from(perms) : ['*']
}

const StatusTag = ({ status, expiresAt }: { status: CredentialStatus; expiresAt?: string | null }) => {
  const { t: tt } = useTranslation()
  const expired = expiresAt ? new Date(expiresAt).getTime() < Date.now() : false
if (expired) return <Tag color="orangered">{tt('common.expired')}</Tag>
if (status === 'active') return <Tag color="green">{tt('common.enabled')}</Tag>
return <Tag color="red">{tt('common.disabled')}</Tag>
}

const fmtTime = (v?: string | null) => (v ? new Date(v).toLocaleString() : '—')

const toDatetimeLocalValue = (iso?: string | null): string => {
  if (!iso) return ''
  const d = new Date(iso)
  if (Number.isNaN(d.getTime())) return ''
  const pad = (n: number) => String(n).padStart(2, '0')
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}T${pad(d.getHours())}:${pad(d.getMinutes())}`
}

const toExpiresAtPayload = (raw: string): string | null | undefined => {
  const s = raw.trim()
  if (!s) return null
  const d = new Date(s)
  if (Number.isNaN(d.getTime())) throw new Error('invalid date')
  return d.toISOString()
}

const AccessKeys = ({ embedded = false }: { embedded?: boolean }) => {
  const { t } = useTranslation()
  const statusOptions: { label: string; value: '' | CredentialStatus }[] = [
    { label: t('common.all'), value: '' },
    { label: t('common.enable'), value: 'active' },
    { label: t('common.disable'), value: 'disabled' },
  ]
  const [rows, setRows] = useState<CredentialRow[]>([])
  const [total, setTotal] = useState(0)
  const [page, setPage] = useState(1)
  const [statusQ, setStatusQ] = useState<'' | CredentialStatus>('')
  const [nameQ, setNameQ] = useState('')
  const [loading, setLoading] = useState(false)

  const [createOpen, setCreateOpen] = useState(false)
  const [createForm, setCreateForm] = useState({
    name: '',
    allowIp: '',
    permissionCodesJson: '[]',
    allowedRouteIds: [] as string[],
    expiresAt: '',
  })
  const [routeCatalogGroups, setRouteCatalogGroups] = useState<AkskRouteCatalogGroup[]>([])
  const [routeCatalogLoading, setRouteCatalogLoading] = useState(false)
  const [creating, setCreating] = useState(false)

  const [revealOpen, setRevealOpen] = useState(false)
  const [issued, setIssued] = useState<CredentialCreateResult | null>(null)

  const [editOpen, setEditOpen] = useState(false)
  const [editForm, setEditForm] = useState<EditState>(defaultEdit)
  const [editing, setEditing] = useState(false)

  const [delTarget, setDelTarget] = useState<CredentialRow | null>(null)
  const [delLoading, setDelLoading] = useState(false)

  const pageSize = 20

  const load = useCallback(async () => {
    setLoading(true)
    try {
      const res = await listCredentials({
        page,
        size: pageSize,
        status: statusQ || undefined,
        name: nameQ.trim() || undefined,
      })
      if (res.code === 200 && res.data) {
        setRows(res.data.list || [])
        setTotal(res.data.total || 0)
      } else {
        showAlert(res.msg || t('common.loadFailed'), 'error')
      }
    } catch (e: unknown) {
      showAlert(extractApiErrorMessage(e, t('common.loadFailed')), 'error')
    } finally {
      setLoading(false)
    }
  }, [page, statusQ, nameQ])

  useEffect(() => {
    void load()
  }, [load])

  const loadRouteCatalog = useCallback(async () => {
    setRouteCatalogLoading(true)
    try {
      const res = await getCredentialAkskRouteCatalog()
      if (res.code === 200 && res.data) {
        setRouteCatalogGroups(res.data.groups || [])
      }
    } catch {
      setRouteCatalogGroups([])
    } finally {
      setRouteCatalogLoading(false)
    }
  }, [])

  const applyRouteSelection = (routeIds: string[], syncPerms = true) => {
    const perms = syncPerms ? permissionsFromRouteSelection(routeCatalogGroups, routeIds) : undefined
    return {
      allowedRouteIds: routeIds,
      ...(syncPerms ? { permissionCodesJson: JSON.stringify(perms) } : {}),
    }
  }

  const openCreate = () => {
    setCreateForm({ name: '', allowIp: '', permissionCodesJson: '[]', allowedRouteIds: [], expiresAt: '' })
    void loadRouteCatalog()
    setCreateOpen(true)
  }

  const parsePermissionCodes = (raw: string): string[] | undefined => {
    const s = raw.trim()
    if (!s) return undefined
    const parsed = JSON.parse(s) as unknown
    if (!Array.isArray(parsed) || !parsed.every((x) => typeof x === 'string')) {
      throw new Error('权限码须为非空 JSON 字符串数组')
    }
    return parsed
  }

  const submitCreate = async () => {
    if (createForm.allowedRouteIds.length === 0) {
      showAlert(t('akskRoutePolicy.needSelection'), 'error')
      return
    }
    setCreating(true)
    try {
      let permissionCodes: string[] | undefined
      try {
        permissionCodes = parsePermissionCodes(createForm.permissionCodesJson)
      } catch {
        showAlert(t('common.permissionCodeInvalid'), 'error')
        setCreating(false)
        return
      }
      let expiresAt: string | null | undefined
      try {
        expiresAt = toExpiresAtPayload(createForm.expiresAt)
      } catch {
        showAlert(t('common.expireTimeInvalid'), 'error')
        setCreating(false)
        return
      }
      const res = await createCredential({
        name: createForm.name.trim() || undefined,
        allowIp: createForm.allowIp.trim() || undefined,
        permissionCodes,
        allowedRouteIds: createForm.allowedRouteIds,
        expiresAt,
      })
      if (res.code === 200 && res.data) {
        setIssued(res.data)
        setRevealOpen(true)
        setCreateOpen(false)
        void load()
      } else {
        showAlert(res.msg || t('common.createFailed'), 'error')
      }
    } catch (e: unknown) {
      showAlert(extractApiErrorMessage(e, t('common.createFailed')), 'error')
    } finally {
      setCreating(false)
    }
  }

  const openEdit = (r: CredentialRow) => {
    const pc = Array.isArray(r.permissionCodes) ? r.permissionCodes : []
    setEditForm({
      id: r.id,
      name: r.name || '',
      allowIp: r.allowIp || '',
      permissionCodesJson: JSON.stringify(pc),
      allowedRouteIds: Array.isArray(r.allowedRouteIds) ? r.allowedRouteIds : [],
      expiresAt: toDatetimeLocalValue(r.expiresAt),
    })
    void loadRouteCatalog()
    setEditOpen(true)
  }

  const submitEdit = async () => {
    if (editForm.id == null) return
    if (editForm.allowedRouteIds.length === 0) {
      showAlert(t('akskRoutePolicy.needSelection'), 'error')
      return
    }
    const name = editForm.name.trim()
    if (!name) {
      showAlert(t('common.nameRequired'), 'error')
      return
    }
    let permissionCodes: string[]
    try {
      const parsed = parsePermissionCodes(editForm.permissionCodesJson)
      permissionCodes = parsed ?? []
    } catch {
      showAlert(t('common.permissionCodeInvalidShort'), 'error')
      return
    }
    let expiresAt: string | null | undefined
    try {
      expiresAt = toExpiresAtPayload(editForm.expiresAt)
    } catch {
      showAlert(t('common.expireTimeInvalid'), 'error')
      return
    }
    setEditing(true)
    try {
      const res = await updateCredential(editForm.id, {
        name,
        allowIp: editForm.allowIp.trim(),
        permissionCodes,
        allowedRouteIds: editForm.allowedRouteIds,
        expiresAt,
      })
      if (res.code === 200) {
        showAlert(t('common.saveSuccess'), 'success')
        setEditOpen(false)
        void load()
      } else {
        showAlert(res.msg || t('common.saveFailed'), 'error')
      }
    } catch (e: unknown) {
      showAlert(extractApiErrorMessage(e, t('common.saveFailed')), 'error')
    } finally {
      setEditing(false)
    }
  }

  const toggleStatus = async (r: CredentialRow) => {
    try {
      const res = r.status === 'active' ? await disableCredential(r.id) : await enableCredential(r.id)
      if (res.code === 200) {
        showAlert(r.status === 'active' ? t('common.disableSuccess') : t('common.enableSuccess'), 'success')
        void load()
      } else {
        showAlert(res.msg || t('common.failed'), 'error')
      }
    } catch (e: unknown) {
      showAlert(extractApiErrorMessage(e, t('common.failed')), 'error')
    }
  }

  const doRegenerate = async (r: CredentialRow) => {
    Modal.confirm({
      title: t('common.regenerateApiKeyTitle'),
      content: t('common.regenerateApiKeyHint'),
      okText: t('common.confirm'),
      cancelText: t('common.cancel'),
      onOk: async () => {
        try {
          const res = await regenerateCredential(r.id)
          if (res.code === 200 && res.data) {
            setIssued(res.data)
            setRevealOpen(true)
            void load()
          } else {
            showAlert(res.msg || t('common.failed'), 'error')
          }
        } catch (e: unknown) {
          showAlert(extractApiErrorMessage(e, t('common.failed')), 'error')
        }
      },
    })
  }

  const confirmDelete = async () => {
    if (!delTarget) return
    setDelLoading(true)
    try {
      const res = await deleteCredential(delTarget.id)
      if (res.code === 200) {
        showAlert(t('common.deleteSuccess'), 'success')
        setDelTarget(null)
        void load()
      } else {
        showAlert(res.msg || t('common.deleteFailed'), 'error')
      }
    } catch (e: unknown) {
      showAlert(extractApiErrorMessage(e, t('common.deleteFailed')), 'error')
    } finally {
      setDelLoading(false)
    }
  }

  const copy = async (_label: string, value: string) => {
    try {
      await navigator.clipboard.writeText(value)
      showAlert(t('common.copySuccess'), 'success')
    } catch {
      showAlert(t('common.copyFailed'), 'error')
    }
  }

  const content = (
      <Space direction="vertical" size={16} style={{ width: '100%' }}>
        <Space wrap align="end">
          <Space direction="vertical" size={4}>
            <Typography.Text type="secondary" style={{ fontSize: 12 }}>{t('common.name')}</Typography.Text>
            <Input
              allowClear
              placeholder={t('common.filterPlaceholder')}
              style={{ width: 200 }}
              value={nameQ}
              onChange={setNameQ}
            />
          </Space>
          <Space direction="vertical" size={4}>
            <Typography.Text type="secondary" style={{ fontSize: 12 }}>{t('common.status')}</Typography.Text>
            <Select
              style={{ width: 140 }}
              value={statusQ}
              onChange={(v) => setStatusQ((v as '' | CredentialStatus) ?? '')}
              options={statusOptions}
            />
          </Space>
          <Button
            type="primary"
            onClick={() => {
              setPage(1)
              void load()
            }}
          >
            {t('common.search')}
          </Button>
          <Button type="outline" onClick={openCreate}>
            {t('common.newCredential')}
          </Button>
        </Space>

        <Card bordered={false}>
          {loading ? (
            <Loading block tip={t('common.loading')} />
          ) : (
            <>
              <div style={{ overflowX: 'auto' }}>
                <table style={{ minWidth: 1400, width: '100%', fontSize: 13 }}>
                  <thead style={{ background: 'var(--color-fill-2)' }}>
                    <tr>
                      <th style={{ textAlign: 'left', padding: 12 }}>{t('common.name')}</th>
                      <th style={{ textAlign: 'left', padding: 12 }}>API Key</th>
                      <th style={{ textAlign: 'left', padding: 12 }}>{t('common.status')}</th>
                      <th style={{ textAlign: 'left', padding: 12 }}>{t('common.expireTime')}</th>
                      <th style={{ textAlign: 'left', padding: 12 }}>{t('common.lastUsed')}</th>
                      <th style={{ textAlign: 'right', padding: 12 }}>{t('common.requestCount')}</th>
                      <th style={{ textAlign: 'left', padding: 12 }}>{t('common.permissionCode')}</th>
                      <th style={{ textAlign: 'left', padding: 12 }}>{t('common.allowIp')}</th>
                      <th style={{ textAlign: 'left', padding: 12 }}>{t('common.createTime')}</th>
                      <th style={{ textAlign: 'right', padding: 12 }}>{t('common.actions')}</th>
                    </tr>
                  </thead>
                  <tbody>
                    {rows.length === 0 ? (
                      <tr>
                        <td colSpan={10} style={{ padding: 0 }}>
                          <TableEmpty description={t('common.noData')} />
                        </td>
                      </tr>
                    ) : (
                      rows.map((r) => (
                        <tr key={r.id} style={{ borderTop: '1px solid var(--color-border)' }}>
                          <td style={{ padding: 12, maxWidth: 200 }}>
                            <div style={{ fontWeight: 500 }}>{r.name || '—'}</div>
                            {r.legacyHmac ? (
                              <div style={{ fontSize: 12, color: 'var(--color-danger-6)' }}>{t('common.legacyAkskHint')}</div>
                            ) : null}
                          </td>
                          <td
                            style={{
                              padding: 12,
                              fontFamily: 'monospace',
                              fontSize: 12,
                              overflow: 'hidden',
                              textOverflow: 'ellipsis',
                              whiteSpace: 'nowrap',
                              maxWidth: 320,
                            }}
                            title={r.apiKeyPrefix || r.accessKey}
                          >
                            <Space size={6}>
                              <span>{r.apiKeyPrefix || r.accessKey || '—'}</span>
                            </Space>
                          </td>
                          <td style={{ padding: 12 }}>
                            <StatusTag status={r.status} expiresAt={r.expiresAt} />
                          </td>
                          <td style={{ padding: 12, fontSize: 12, color: 'var(--color-text-3)' }}>
                            {r.expiresAt ? fmtTime(r.expiresAt) : t('common.neverExpire')}
                          </td>
                          <td style={{ padding: 12, fontSize: 12, color: 'var(--color-text-3)' }}>
                            {fmtTime(r.lastUsedAt)}
                          </td>
                          <td style={{ padding: 12, textAlign: 'right', fontVariantNumeric: 'tabular-nums' }}>
                            {r.requestCount ?? 0}
                          </td>
                          <td
                            style={{
                              padding: 12,
                              fontFamily: 'monospace',
                              fontSize: 11,
                              overflow: 'hidden',
                              textOverflow: 'ellipsis',
                              whiteSpace: 'nowrap',
                              maxWidth: 160,
                              color: 'var(--color-text-2)',
                            }}
                            title={r.permissionCodes?.join(', ') || '—'}
                          >
                            {r.permissionCodes && r.permissionCodes.length
                              ? r.permissionCodes.join(', ')
                              : '—'}
                          </td>
                          <td
                            style={{
                              padding: 12,
                              fontFamily: 'monospace',
                              fontSize: 12,
                              overflow: 'hidden',
                              textOverflow: 'ellipsis',
                              whiteSpace: 'nowrap',
                              maxWidth: 220,
                            }}
                            title={r.allowIp || '不限制'}
                          >
                            {r.allowIp || t('common.unlimited')}
                          </td>
                          <td style={{ padding: 12, fontSize: 12, color: 'var(--color-text-3)' }}>
                            {r.createdAt ? new Date(r.createdAt).toLocaleString() : '—'}
                          </td>
                          <td style={{ padding: 12, textAlign: 'right' }}>
                            <Space>
                              <Button type="outline" size="small" onClick={() => openEdit(r)}>
                                {t('common.edit')}
                              </Button>
                              <Button type="outline" size="small" onClick={() => void doRegenerate(r)}>
                                {t('common.regenerateApiKey')}
                              </Button>
                              <Button
                                type="outline"
                                size="small"
                                status={r.status === 'active' ? 'warning' : 'success'}
                                onClick={() => void toggleStatus(r)}
                              >
                                {r.status === 'active' ? t('common.disable') : t('common.enable')}
                              </Button>
                              <Button
                                type="outline"
                                status="danger"
                                size="small"
                                icon={<IconDelete />}
                                onClick={() => setDelTarget(r)}
                              >
                                {t('common.delete')}
                              </Button>
                            </Space>
                          </td>
                        </tr>
                      ))
                    )}
                  </tbody>
                </table>
              </div>
              <div
                style={{
                  display: 'flex',
                  justifyContent: 'space-between',
                  marginTop: 12,
                  paddingTop: 12,
                  borderTop: '1px solid var(--color-border)',
                }}
              >
                <Typography.Text type="secondary">{t('common.total')}: {total}</Typography.Text>
                <Space>
                  <Button size="small" disabled={page <= 1} onClick={() => setPage((p) => Math.max(1, p - 1))}>
                    {t('common.previous')}
                  </Button>
                  <Button size="small" disabled={page * pageSize >= total} onClick={() => setPage((p) => p + 1)}>
                    {t('common.next')}
                  </Button>
                </Space>
              </div>
            </>
          )}
        </Card>

        <Drawer
          title={t('common.newCredential')}
          visible={createOpen}
          placement="right"
          width={720}
          onCancel={() => {
            if (!creating) setCreateOpen(false)
          }}
          footer={
            <Space>
              <Button onClick={() => setCreateOpen(false)} disabled={creating}>
                {t('common.cancel')}
              </Button>
              <Button type="primary" loading={creating} onClick={() => void submitCreate()}>
                {creating ? t('common.createWithProgress') : t('common.create')}
              </Button>
            </Space>
          }
        >
          <Space direction="vertical" style={{ width: '100%' }} size={12}>
            <div>
              <Typography.Text style={{ fontSize: 12 }}>{t('common.name')}</Typography.Text>
              <Input
                placeholder={t('common.credentialNameExample')}
                value={createForm.name}
                onChange={(v) => setCreateForm((f) => ({ ...f, name: v }))}
              />
            </div>
            <div>
              <Typography.Text style={{ fontSize: 12 }}>{t('common.allowIpLabel')}</Typography.Text>
              <Input
                placeholder={t('common.ipPlaceholder')}
                value={createForm.allowIp}
                onChange={(v) => setCreateForm((f) => ({ ...f, allowIp: v }))}
              />
            </div>
            <div>
              <Typography.Text style={{ fontSize: 12 }}>{t('akskRoutePolicy.credentialScope')} *</Typography.Text>
              <Typography.Paragraph type="secondary" style={{ margin: '4px 0 8px', fontSize: 11 }}>
                {routeCatalogGroups.length === 0 && !routeCatalogLoading
                  ? t('akskRoutePolicy.platformClosed')
                  : t('akskRoutePolicy.credentialScopeHint')}
              </Typography.Paragraph>
              {routeCatalogLoading ? (
                <Loading />
              ) : (
                <AkskRoutePicker
                  groups={routeCatalogGroups}
                  selectedIds={createForm.allowedRouteIds}
                  defaultExpanded={false}
                  onChange={(ids) => setCreateForm((f) => ({ ...f, ...applyRouteSelection(ids) }))}
                />
              )}
            </div>
            <div>
              <Typography.Text style={{ fontSize: 12 }}>{t('common.permissionCodeJsonHint')}</Typography.Text>
              <Input.TextArea
                placeholder={t('common.permissionCodePlaceholder')}
                autoSize={{ minRows: 2, maxRows: 6 }}
                value={createForm.permissionCodesJson}
                onChange={(v) => setCreateForm((f) => ({ ...f, permissionCodesJson: v }))}
                style={{ fontFamily: 'monospace', fontSize: 12 }}
              />
            </div>
            <div>
              <Typography.Text style={{ fontSize: 12 }}>{t('common.expireTimeOptional')}</Typography.Text>
              <Input
                type="datetime-local"
                value={createForm.expiresAt}
                onChange={(v) => setCreateForm((f) => ({ ...f, expiresAt: v }))}
              />
              <Typography.Text type="secondary" style={{ fontSize: 11 }}>{t('common.expireTimeNone')}</Typography.Text>
            </div>
            <Typography.Paragraph type="warning" style={{ margin: 0, fontSize: 12 }}>
              {t('common.createTimeHint')}
            </Typography.Paragraph>
          </Space>
        </Drawer>

        <Modal
          title={t('common.credentialCreated')}
          visible={revealOpen}
          maskClosable={false}
          onCancel={() => {
            setRevealOpen(false)
            setIssued(null)
          }}
          footer={
            <Button
              type="primary"
              onClick={() => {
                setRevealOpen(false)
                setIssued(null)
              }}
            >
              {t('common.iSaved')}
            </Button>
          }
        >
          {issued && (
            <Space direction="vertical" style={{ width: '100%' }} size={16}>
              <div
                style={{
                  background: 'rgb(255, 219, 190)',
                  border: '1px solid rgb(255, 169, 102)',
                  borderRadius: 8,
                  padding: '12px 16px',
                }}
              >
                <Typography.Text style={{ fontSize: 13, color: '#b25000' }}>
                  {t('common.apiKeyWarning')}
                </Typography.Text>
              </div>
              <div>
                <Typography.Text type="secondary" style={{ fontSize: 12 }}>{t('common.name')}</Typography.Text>
                <div style={{ padding: '6px 0', fontWeight: 500 }}>{issued.name}</div>
              </div>
              <div>
                <Typography.Text type="secondary" style={{ fontSize: 12 }}>API Key</Typography.Text>
                <div
                  style={{
                    marginTop: 6,
                    background: 'var(--color-fill-2)',
                    border: '1px solid var(--color-border)',
                    borderRadius: 6,
                    padding: '10px 12px',
                    fontFamily: 'monospace',
                    fontSize: 12,
                    wordBreak: 'break-all',
                    lineHeight: 1.6,
                  }}
                >
                  {issued.apiKey}
                </div>
                <Button
                  type="primary"
                  icon={<IconCopy />}
                  style={{ marginTop: 8 }}
                  onClick={() => void copy('API Key', issued.apiKey)}
                >
                  {t('common.copy')}
                </Button>
              </div>
              <div
                style={{
                  background: 'var(--color-fill-2)',
                  borderRadius: 6,
                  padding: '10px 12px',
                  fontSize: 12,
                  color: 'var(--color-text-3)',
                  lineHeight: 1.6,
                }}
              >
                {t('common.apiKeyAuthHint')}
              </div>
            </Space>
          )}
        </Modal>

        <Drawer
          title={t('common.editCredential')}
          visible={editOpen}
          placement="right"
          width={720}
          onCancel={() => {
            if (!editing) setEditOpen(false)
          }}
          footer={
            <Space>
              <Button onClick={() => setEditOpen(false)} disabled={editing}>
                {t('common.cancel')}
              </Button>
              <Button type="primary" loading={editing} onClick={() => void submitEdit()}>
                {editing ? t('common.saveWithProgress') : t('common.save')}
              </Button>
            </Space>
          }
        >
          <Space direction="vertical" style={{ width: '100%' }} size={12}>
            <div>
              <Typography.Text style={{ fontSize: 12 }}>{t('common.name')} *</Typography.Text>
              <Input
                value={editForm.name}
                onChange={(v) => setEditForm((f) => ({ ...f, name: v }))}
              />
            </div>
            <div>
              <Typography.Text style={{ fontSize: 12 }}>{t('common.allowIpLabel')}</Typography.Text>
              <Input
                value={editForm.allowIp}
                onChange={(v) => setEditForm((f) => ({ ...f, allowIp: v }))}
              />
            </div>
            <div>
              <Typography.Text style={{ fontSize: 12 }}>{t('akskRoutePolicy.credentialScope')} *</Typography.Text>
              <Typography.Paragraph type="secondary" style={{ margin: '4px 0 8px', fontSize: 11 }}>
                {routeCatalogGroups.length === 0 && !routeCatalogLoading
                  ? t('akskRoutePolicy.platformClosed')
                  : t('akskRoutePolicy.credentialScopeHint')}
              </Typography.Paragraph>
              {routeCatalogLoading ? (
                <Loading />
              ) : (
                <AkskRoutePicker
                  groups={routeCatalogGroups}
                  selectedIds={editForm.allowedRouteIds}
                  onChange={(ids) => setEditForm((f) => ({ ...f, ...applyRouteSelection(ids) }))}
                />
              )}
            </div>
            <div>
              <Typography.Text style={{ fontSize: 12 }}>{t('common.permissionCodeJsonHint')}</Typography.Text>
              <Input.TextArea
                autoSize={{ minRows: 2, maxRows: 6 }}
                value={editForm.permissionCodesJson}
                onChange={(v) => setEditForm((f) => ({ ...f, permissionCodesJson: v }))}
                style={{ fontFamily: 'monospace', fontSize: 12 }}
              />
            </div>
            <div>
              <Typography.Text style={{ fontSize: 12 }}>{t('common.expireTimeEditLabel')}</Typography.Text>
              <Input
                type="datetime-local"
                value={editForm.expiresAt}
                onChange={(v) => setEditForm((f) => ({ ...f, expiresAt: v }))}
              />
            </div>
            <Typography.Paragraph type="secondary" style={{ margin: 0, fontSize: 12 }}>
              {t('common.accessKeyNoEdit')}
            </Typography.Paragraph>
          </Space>
        </Drawer>

        <Modal
          title={t('common.confirmDeleteCredential')}
          visible={!!delTarget}
          maskClosable={false}
          onCancel={() => {
            if (!delLoading) setDelTarget(null)
          }}
          footer={
            <Space>
              <Button onClick={() => setDelTarget(null)} disabled={delLoading}>
                {t('common.cancel')}
              </Button>
              <Button status="danger" loading={delLoading} onClick={() => void confirmDelete()}>
                {t('common.confirmDelete')}
              </Button>
            </Space>
          }
        >
          <Typography.Text>
            {t('common.deleteCredentialWarning')}
          </Typography.Text>
          {delTarget && (
            <Typography.Paragraph
              style={{ marginTop: 8, fontFamily: 'monospace', fontSize: 12, wordBreak: 'break-all' }}
            >
              {delTarget.accessKey}
            </Typography.Paragraph>
          )}
        </Modal>
      </Space>
  )

  if (embedded) {
    return (
      <div>
        <div className="mb-5">
          <h2 className="text-base font-medium text-foreground">{t('pages.accessKeys.title')}</h2>
          <p className="mt-1 text-sm text-muted-foreground">{t('pages.accessKeys.description')}</p>
        </div>
        {content}
      </div>
    )
  }

  return (
    <BaseLayout title={t('pages.accessKeys.title')} description={t('pages.accessKeys.description')}>
      {content}
    </BaseLayout>
  )
}

export default AccessKeys
