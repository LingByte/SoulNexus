import { useCallback, useEffect, useMemo, useState } from 'react'
import { Modal, Spin, Typography } from '@arco-design/web-react'
import { IconClose, IconRobot } from '@arco-design/web-react/icon'
import { Button, Input, Loading } from '@/components/ui'
import { listTenantUsers, type TenantUserRow } from '@/api/tenantUsers'
import {
  addAssistantMembers,
  listAssistantMembers,
  patchAssistantSettings,
  removeAssistantMember,
  uploadAssistantAvatar,
} from '@/api/assistants'
import { useTranslation } from '@/i18n'
import { showAlert } from '@/utils/notification'
import { AssistantAvatarPicker } from '@/components/assistant/AssistantFileUploadModal'

export type AssistantSettingsModalProps = {
  visible: boolean
  assistantId: string
  name: string
  description: string
  avatarUrl?: string
  onClose: () => void
  onSaved: (patch: { name: string; description: string; avatarUrl?: string }) => void
}

function useDelayedLoading(loading: boolean, delayMs = 200) {
  const [visible, setVisible] = useState(false)
  useEffect(() => {
    if (!loading) {
      setVisible(false)
      return
    }
    const timer = window.setTimeout(() => setVisible(true), delayMs)
    return () => window.clearTimeout(timer)
  }, [loading, delayMs])
  return visible
}

function memberErrorMessage(msg: string | undefined, duplicateFallback: string, genericFallback: string) {
  const text = (msg || '').trim()
  if (!text) return genericFallback
  if (/duplicate entry|unique constraint|idx_assistant_member/i.test(text)) return duplicateFallback
  return text
}

export default function AssistantSettingsModal({
  visible,
  assistantId,
  name: initialName,
  description: initialDescription,
  avatarUrl: initialAvatar,
  onClose,
  onSaved,
}: AssistantSettingsModalProps) {
  const { t } = useTranslation()
  const [name, setName] = useState(initialName)
  const [description, setDescription] = useState(initialDescription)
  const [avatarUrl, setAvatarUrl] = useState(initialAvatar || '')
  const [saving, setSaving] = useState(false)
  const [members, setMembers] = useState<{ userId: string; email?: string; name?: string }[]>([])
  const [membersLoading, setMembersLoading] = useState(false)
  const [tenantUsers, setTenantUsers] = useState<TenantUserRow[]>([])
  const [inviteOpen, setInviteOpen] = useState(false)
  const [inviteLoading, setInviteLoading] = useState(false)
  const [inviteSubmitting, setInviteSubmitting] = useState(false)
  const [selectedIds, setSelectedIds] = useState<string[]>([])
  const [removingId, setRemovingId] = useState<string | null>(null)

  const showMembersLoading = useDelayedLoading(membersLoading)
  const showInviteLoading = useDelayedLoading(inviteLoading)

  const loadMembers = useCallback(async () => {
    setMembersLoading(true)
    try {
      const res = await listAssistantMembers(assistantId)
      if (res.code === 200 && Array.isArray(res.data)) {
        setMembers(res.data)
      }
    } finally {
      setMembersLoading(false)
    }
  }, [assistantId])

  useEffect(() => {
    if (!visible) return
    setName(initialName)
    setDescription(initialDescription)
    setAvatarUrl(initialAvatar || '')
    void loadMembers()
  }, [visible, assistantId, initialName, initialDescription, initialAvatar, loadMembers])

  const memberIdSet = useMemo(() => new Set(members.map((m) => m.userId)), [members])
  const invitableUsers = useMemo(
    () => tenantUsers.filter((u) => !memberIdSet.has(String(u.id))),
    [tenantUsers, memberIdSet],
  )

  const save = async () => {
    if (!name.trim()) {
      showAlert(t('assistant.settings.nameRequired'), 'error')
      return
    }
    setSaving(true)
    try {
      const res = await patchAssistantSettings(assistantId, {
        name: name.trim(),
        description: description.trim(),
      })
      if (res.code !== 200) {
        showAlert(res.msg || t('common.saveFailed'), 'error')
        return
      }
      onSaved({ name: name.trim(), description: description.trim(), avatarUrl: avatarUrl || undefined })
      showAlert(t('assistant.settings.saved'), 'success')
      onClose()
    } catch (e: unknown) {
      showAlert((e as { msg?: string })?.msg || t('common.saveFailed'), 'error')
    } finally {
      setSaving(false)
    }
  }

  const uploadAvatar = async (file: File) => {
    try {
      const res = await uploadAssistantAvatar(assistantId, file)
      if (res.code !== 200) {
        showAlert(res.msg || t('assistant.settings.avatarFailed'), 'error')
        return
      }
      const url = (res.data as { avatarUrl?: string })?.avatarUrl || ''
      if (url) {
        setAvatarUrl(url)
        onSaved({ name: name.trim(), description: description.trim(), avatarUrl: url })
      }
      showAlert(t('assistant.settings.avatarUpdated'), 'success')
    } catch (e: unknown) {
      showAlert((e as { msg?: string })?.msg || t('assistant.settings.avatarFailed'), 'error')
    }
  }

  const openInvite = async () => {
    setInviteOpen(true)
    setSelectedIds([])
    setInviteLoading(true)
    try {
      const res = await listTenantUsers(1, 200)
      if (res.code === 200 && res.data?.list) {
        setTenantUsers(res.data.list)
      }
    } finally {
      setInviteLoading(false)
    }
  }

  const submitInvite = async () => {
    if (selectedIds.length === 0) {
      setInviteOpen(false)
      return
    }
    setInviteSubmitting(true)
    try {
      const res = await addAssistantMembers(assistantId, selectedIds)
      if (res.code !== 200) {
        showAlert(
          memberErrorMessage(res.msg, t('assistant.settings.duplicateMember'), t('assistant.settings.inviteFailed')),
          'error',
        )
        return
      }
      await loadMembers()
      setInviteOpen(false)
      setSelectedIds([])
      showAlert(t('assistant.settings.inviteSuccess'), 'success')
    } catch (e: unknown) {
      const err = e as { msg?: string; message?: string }
      showAlert(
        memberErrorMessage(err.msg || err.message, t('assistant.settings.duplicateMember'), t('assistant.settings.inviteFailed')),
        'error',
      )
    } finally {
      setInviteSubmitting(false)
    }
  }

  const removeMember = async (userId: string) => {
    setRemovingId(userId)
    try {
      const res = await removeAssistantMember(assistantId, userId)
      if (res.code !== 200) {
        showAlert(res.msg || t('assistant.settings.removeFailed'), 'error')
        return
      }
      setMembers((prev) => prev.filter((m) => m.userId !== userId))
      showAlert(t('assistant.settings.removeSuccess'), 'success')
    } catch (e: unknown) {
      showAlert((e as { msg?: string })?.msg || t('assistant.settings.removeFailed'), 'error')
    } finally {
      setRemovingId(null)
    }
  }

  return (
    <>
      <Modal
        visible={visible}
        title={
          <span className="inline-flex items-center gap-2">
            <IconRobot style={{ color: '#2563eb' }} />
            {t('assistant.settings.title')}
          </span>
        }
        onCancel={onClose}
        footer={
          <>
            <Button onClick={onClose}>{t('assistant.settings.close')}</Button>
            <Button type="primary" loading={saving} onClick={() => void save()}>
              {t('common.save')}
            </Button>
          </>
        }
        style={{ width: 520 }}
      >
        <div className="space-y-5 py-2">
          <div>
            <Typography.Text bold style={{ display: 'block', marginBottom: 8 }}>
              {t('assistant.settings.avatarAndName')}
            </Typography.Text>
            <div className="flex items-center gap-3">
              <AssistantAvatarPicker avatarUrl={avatarUrl} onPick={(f) => void uploadAvatar(f)} />
              <Input
                value={name}
                onChange={setName}
                placeholder={t('assistant.settings.appName')}
                style={{ flex: 1 }}
              />
            </div>
          </div>
          <div>
            <Typography.Text bold style={{ display: 'block', marginBottom: 8 }}>
              {t('assistant.settings.intro')}
            </Typography.Text>
            <Input.TextArea
              value={description}
              onChange={setDescription}
              placeholder={t('assistant.settings.introPlaceholder')}
              autoSize={{ minRows: 3, maxRows: 6 }}
            />
          </div>
          <div>
            <div className="mb-2 flex items-center justify-between">
              <Typography.Text bold>{t('assistant.settings.collaborators')}</Typography.Text>
              <Button type="text" size="mini" loading={inviteLoading} onClick={() => void openInvite()}>
                {t('assistant.settings.invite')}
              </Button>
            </div>
            <div className="relative min-h-[72px] rounded-lg border border-border bg-muted/20 px-3 py-4 text-sm text-muted-foreground">
              {showMembersLoading ? (
                <div className="absolute inset-0 flex items-center justify-center">
                  <Loading size="sm" tip={t('common.loading')} />
                </div>
              ) : members.length === 0 ? (
                t('assistant.settings.noCollaborators')
              ) : (
                <ul className="space-y-1">
                  {members.map((m) => (
                    <li key={m.userId} className="flex items-center justify-between gap-2">
                      <span className="truncate">{m.name || m.email || m.userId}</span>
                      <Button
                        type="text"
                        size="mini"
                        loading={removingId === m.userId}
                        disabled={removingId !== null && removingId !== m.userId}
                        icon={<IconClose />}
                        onClick={() => void removeMember(m.userId)}
                      >
                        {t('assistant.settings.remove')}
                      </Button>
                    </li>
                  ))}
                </ul>
              )}
            </div>
          </div>
        </div>
      </Modal>

      <Modal
        visible={inviteOpen}
        title={t('assistant.settings.invite')}
        onCancel={() => setInviteOpen(false)}
        footer={
          <>
            <Button onClick={() => setInviteOpen(false)}>{t('common.cancel')}</Button>
            <Button
              type="primary"
              loading={inviteSubmitting}
              disabled={selectedIds.length === 0}
              onClick={() => void submitInvite()}
            >
              {t('common.confirm')}
            </Button>
          </>
        }
        style={{ width: 480 }}
      >
        <div className="relative max-h-80 min-h-[120px] overflow-y-auto py-2">
          {showInviteLoading ? (
            <div className="flex justify-center py-10">
              <Spin tip={t('common.loading')} />
            </div>
          ) : invitableUsers.length === 0 ? (
            <p className="py-6 text-center text-sm text-muted-foreground">{t('assistant.settings.inviteEmpty')}</p>
          ) : (
            <div className="space-y-2">
              {invitableUsers.map((u) => {
                const id = String(u.id)
                const checked = selectedIds.includes(id)
                return (
                  <label
                    key={id}
                    className="flex cursor-pointer items-center gap-2 rounded-lg px-2 py-1.5 hover:bg-muted/40"
                  >
                    <input
                      type="checkbox"
                      checked={checked}
                      onChange={() => {
                        setSelectedIds((prev) =>
                          checked ? prev.filter((x) => x !== id) : [...prev, id],
                        )
                      }}
                    />
                    <span>{u.displayName || u.email || id}</span>
                  </label>
                )
              })}
            </div>
          )}
        </div>
      </Modal>
    </>
  )
}
