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
import CredentialAiBundleEditor, {
  credentialAiBundleFromRow,
  credentialAiBundleToPayload,
  emptyCredentialAiBundle,
  type CredentialAiBundleState,
} from '@/components/credentials/CredentialAiBundleEditor'
import PlatformCredentialLLMTest from '@/components/credentials/PlatformCredentialLLMTest'

type NameExpiresForm = {
  name: string
  expiresAt: string
}

type EditForm = NameExpiresForm & { id: string | null }

const emptyNameExpires = (): NameExpiresForm => ({ name: '', expiresAt: '' })
const emptyEdit = (): EditForm => ({ id: null, ...emptyNameExpires() })

function isPlatformKey(r: CredentialRow): boolean {
  return r.kind === 'platform_bundle' || r.usesTenantAi === true
}

function StatusTag({ status, expiresAt }: { status: CredentialStatus; expiresAt?: string | null }) {
  const { t } = useTranslation()
  const expired = expiresAt ? new Date(expiresAt).getTime() < Date.now() : false
  if (expired) return <Tag color="orangered">{t('common.expired')}</Tag>
  if (status === 'active') return <Tag color="green">{t('common.enabled')}</Tag>
  return <Tag color="red">{t('common.disabled')}</Tag>
}

function fmtTime(v?: string | null) {
  return v ? new Date(v).toLocaleString() : '—'
}

function toDatetimeLocalValue(iso?: string | null): string {
  if (!iso) return ''
  const d = new Date(iso)
  if (Number.isNaN(d.getTime())) return ''
  const pad = (n: number) => String(n).padStart(2, '0')
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}T${pad(d.getHours())}:${pad(d.getMinutes())}`
}

function toExpiresAtPayload(raw: string): string | null {
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
  const [createForm, setCreateForm] = useState<NameExpiresForm>(emptyNameExpires)
  const [createKeyKind, setCreateKeyKind] = useState<'user_bundle' | 'platform_bundle'>('user_bundle')
  const [createAiBundle, setCreateAiBundle] = useState<CredentialAiBundleState>(emptyCredentialAiBundle)
  const [creating, setCreating] = useState(false)

  const [revealOpen, setRevealOpen] = useState(false)
  const [issued, setIssued] = useState<CredentialCreateResult | null>(null)

  const [editOpen, setEditOpen] = useState(false)
  const [editForm, setEditForm] = useState<EditForm>(emptyEdit)
  const [editAiBundle, setEditAiBundle] = useState<CredentialAiBundleState>(emptyCredentialAiBundle)
  const [editIsPlatform, setEditIsPlatform] = useState(false)
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
  }, [page, statusQ, nameQ, t])

  useEffect(() => {
    void load()
  }, [load])

  const openCreate = () => {
    setCreateForm(emptyNameExpires())
    setCreateKeyKind('user_bundle')
    setCreateAiBundle(emptyCredentialAiBundle())
    setCreateOpen(true)
  }

  const submitCreate = async () => {
    setCreating(true)
    try {
      let expiresAt: string | null
      try {
        expiresAt = toExpiresAtPayload(createForm.expiresAt)
      } catch {
        showAlert(t('common.expireTimeInvalid'), 'error')
        return
      }
      const res = await createCredential({
        name: createForm.name.trim() || undefined,
        expiresAt,
        kind: createKeyKind,
        ...(createKeyKind === 'user_bundle' ? credentialAiBundleToPayload(createAiBundle) : {}),
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
    setEditForm({
      id: r.id,
      name: r.name || '',
      expiresAt: toDatetimeLocalValue(r.expiresAt),
    })
    setEditAiBundle(credentialAiBundleFromRow(r))
    setEditIsPlatform(isPlatformKey(r))
    setEditOpen(true)
  }

  const submitEdit = async () => {
    if (editForm.id == null) return
    const name = editForm.name.trim()
    if (!name) {
      showAlert(t('common.nameRequired'), 'error')
      return
    }
    let expiresAt: string | null
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
        expiresAt,
        ...(editIsPlatform ? {} : credentialAiBundleToPayload(editAiBundle)),
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

  const doRegenerate = (r: CredentialRow) => {
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

  const copyApiKey = async (value: string) => {
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
          <Typography.Text type="secondary" style={{ fontSize: 12 }}>
            {t('common.name')}
          </Typography.Text>
          <Input
            allowClear
            placeholder={t('common.filterPlaceholder')}
            style={{ width: 200 }}
            value={nameQ}
            onChange={setNameQ}
          />
        </Space>
        <Space direction="vertical" size={4}>
          <Typography.Text type="secondary" style={{ fontSize: 12 }}>
            {t('common.status')}
          </Typography.Text>
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
            <div className="w-full overflow-x-auto">
              <table className="w-max min-w-full border-collapse text-sm" style={{ minWidth: 1280 }}>
                <thead style={{ background: 'var(--color-fill-2)' }}>
                  <tr>
                    <th className="whitespace-nowrap px-3 py-3 text-left">{t('common.name')}</th>
                    <th className="whitespace-nowrap px-3 py-3 text-left">{t('credentialAi.keyKind')}</th>
                    <th className="whitespace-nowrap px-3 py-3 text-left">API Key</th>
                    <th className="whitespace-nowrap px-3 py-3 text-left">{t('common.status')}</th>
                    <th className="whitespace-nowrap px-3 py-3 text-left">{t('common.expireTime')}</th>
                    <th className="whitespace-nowrap px-3 py-3 text-left">{t('common.lastUsed')}</th>
                    <th className="whitespace-nowrap px-3 py-3 text-right">{t('common.requestCount')}</th>
                    <th className="whitespace-nowrap px-3 py-3 text-left">{t('common.createTime')}</th>
                    <th className="whitespace-nowrap px-3 py-3 text-right">{t('common.actions')}</th>
                  </tr>
                </thead>
                <tbody>
                  {rows.length === 0 ? (
                    <tr>
                      <td colSpan={9} style={{ padding: 0 }}>
                        <TableEmpty description={t('common.noData')} />
                      </td>
                    </tr>
                  ) : (
                    rows.map((r) => (
                      <tr key={r.id} style={{ borderTop: '1px solid var(--color-border)' }}>
                        <td className="max-w-[220px] whitespace-nowrap px-3 py-3">
                          <div className="truncate font-medium">{r.name || '—'}</div>
                          {r.legacyHmac ? (
                            <div className="text-xs text-[var(--color-danger-6)]">{t('common.legacyAkskHint')}</div>
                          ) : null}
                        </td>
                        <td className="whitespace-nowrap px-3 py-3">
                          <Tag>
                            {isPlatformKey(r) ? t('credentialAi.kindPlatform') : t('credentialAi.kindUser')}
                          </Tag>
                        </td>
                        <td
                          className="max-w-[360px] truncate whitespace-nowrap px-3 py-3 font-mono text-xs"
                          title={r.apiKeyPrefix || r.accessKey}
                        >
                          {r.apiKeyPrefix || r.accessKey || '—'}
                        </td>
                        <td className="whitespace-nowrap px-3 py-3">
                          <StatusTag status={r.status} expiresAt={r.expiresAt} />
                        </td>
                        <td className="whitespace-nowrap px-3 py-3 text-xs text-[var(--color-text-3)]">
                          {r.expiresAt ? fmtTime(r.expiresAt) : t('common.neverExpire')}
                        </td>
                        <td className="whitespace-nowrap px-3 py-3 text-xs text-[var(--color-text-3)]">
                          {fmtTime(r.lastUsedAt)}
                        </td>
                        <td className="whitespace-nowrap px-3 py-3 text-right tabular-nums">
                          {r.requestCount ?? 0}
                        </td>
                        <td className="whitespace-nowrap px-3 py-3 text-xs text-[var(--color-text-3)]">
                          {r.createdAt ? new Date(r.createdAt).toLocaleString() : '—'}
                        </td>
                        <td className="whitespace-nowrap px-3 py-3 text-right">
                          <div className="inline-flex flex-nowrap items-center gap-2">
                            <Button type="outline" size="small" onClick={() => openEdit(r)}>
                              {t('common.edit')}
                            </Button>
                            <Button type="outline" size="small" onClick={() => doRegenerate(r)}>
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
                          </div>
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
              <Typography.Text type="secondary">
                {t('common.total')}: {total}
              </Typography.Text>
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
            <Typography.Text style={{ fontSize: 12 }}>{t('credentialAi.keyKind')}</Typography.Text>
            <Select
              style={{ width: '100%', maxWidth: 360, marginTop: 6 }}
              value={createKeyKind}
              options={[
                { value: 'user_bundle', label: t('credentialAi.kindUser') },
                { value: 'platform_bundle', label: t('credentialAi.kindPlatform') },
              ]}
              onChange={(v) => setCreateKeyKind(v === 'platform_bundle' ? 'platform_bundle' : 'user_bundle')}
            />
            <Typography.Paragraph type="secondary" style={{ margin: '6px 0 0', fontSize: 11 }}>
              {createKeyKind === 'platform_bundle' ? t('credentialAi.kindPlatformHint') : t('credentialAi.kindUserHint')}
            </Typography.Paragraph>
          </div>
          {createKeyKind === 'user_bundle' ? (
            <CredentialAiBundleEditor value={createAiBundle} onChange={setCreateAiBundle} />
          ) : null}
          <div>
            <Typography.Text style={{ fontSize: 12 }}>{t('common.expireTimeOptional')}</Typography.Text>
            <Input
              type="datetime-local"
              value={createForm.expiresAt}
              onChange={(v) => setCreateForm((f) => ({ ...f, expiresAt: v }))}
            />
            <Typography.Text type="secondary" style={{ fontSize: 11 }}>
              {t('common.expireTimeNone')}
            </Typography.Text>
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
        {issued ? (
          <Space direction="vertical" style={{ width: '100%' }} size={16}>
            <div
              style={{
                background: 'rgb(255, 219, 190)',
                border: '1px solid rgb(255, 169, 102)',
                borderRadius: 8,
                padding: '12px 16px',
              }}
            >
              <Typography.Text style={{ fontSize: 13, color: '#b25000' }}>{t('common.apiKeyWarning')}</Typography.Text>
            </div>
            <div>
              <Typography.Text type="secondary" style={{ fontSize: 12 }}>
                {t('common.name')}
              </Typography.Text>
              <div style={{ padding: '6px 0', fontWeight: 500 }}>{issued.name}</div>
            </div>
            <div>
              <Typography.Text type="secondary" style={{ fontSize: 12 }}>
                API Key
              </Typography.Text>
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
                onClick={() => void copyApiKey(issued.apiKey)}
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
        ) : null}
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
            <Input value={editForm.name} onChange={(v) => setEditForm((f) => ({ ...f, name: v }))} />
          </div>
          {editIsPlatform ? (
            <>
              <Typography.Paragraph type="secondary" style={{ margin: 0, fontSize: 12 }}>
                {t('credentialAi.kindPlatformHint')}
              </Typography.Paragraph>
              {editForm.id ? <PlatformCredentialLLMTest credentialId={editForm.id} /> : null}
            </>
          ) : (
            <CredentialAiBundleEditor
              value={editAiBundle}
              onChange={setEditAiBundle}
              credentialId={editForm.id || undefined}
            />
          )}
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
        <Typography.Text>{t('common.deleteCredentialWarning')}</Typography.Text>
        {delTarget ? (
          <Typography.Paragraph style={{ marginTop: 8, fontFamily: 'monospace', fontSize: 12, wordBreak: 'break-all' }}>
            {delTarget.accessKey}
          </Typography.Paragraph>
        ) : null}
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
