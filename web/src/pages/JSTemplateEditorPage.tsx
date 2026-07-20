import { useCallback, useEffect, useRef, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { Spin, Typography } from '@arco-design/web-react'
import { IconArrowLeft, IconSettings } from '@arco-design/web-react/icon'
import BaseLayout from '@/components/Layout/BaseLayout'
import { Button } from '@/components/ui'
import { JSTemplateEditorSplit, loadEmbedPreviewConfig, saveEmbedPreviewConfig } from '@/components/js-template/JSTemplateEditor'
import JSTemplateSettingsModal from '@/components/js-template/JSTemplateSettingsModal'
import {
  createJSTemplate,
  getJSTemplate,
  isValidJSTemplateId,
  updateJSTemplate,
  uploadJSTemplateAvatar,
} from '@/api/jsTemplates'
import { getApiBaseURL } from '@/config/apiConfig'
import { useSidebar } from '@/contexts/SidebarContext'
import { useTranslation } from '@/i18n'
import { showAlert } from '@/utils/notification'

const DEFAULT_USAGE = '网页嵌入浮层聊天'
const CREATE_PLACEHOLDER = '// Loading default embed.js…\n'

async function fetchDefaultEmbedJS(): Promise<string> {
  const apiBase = getApiBaseURL().replace(/\/$/, '')
  const res = await fetch(`${apiBase}/lingecho/embed/v1/embed.js`)
  if (!res.ok) throw new Error(`HTTP ${res.status}`)
  return res.text()
}

export type JSTemplateEditorPageProps = {
  mode: 'create' | 'edit'
  templateId?: string
}

export default function JSTemplateEditorPage({ mode, templateId }: JSTemplateEditorPageProps) {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const isEdit = mode === 'edit' && isValidJSTemplateId(templateId)
  const editId = isEdit ? String(templateId).trim() : ''
  const { isCollapsed, setIsCollapsed } = useSidebar()
  const sidebarBeforeRef = useRef<boolean | null>(null)

  const [loading, setLoading] = useState(isEdit)
  const [contentLoading, setContentLoading] = useState(!isEdit)
  const [saving, setSaving] = useState(false)
  const [settingsOpen, setSettingsOpen] = useState(false)
  const [name, setName] = useState('')
  const [usage, setUsage] = useState(DEFAULT_USAGE)
  const [status, setStatus] = useState('active')
  const [content, setContent] = useState(isEdit ? '' : CREATE_PLACEHOLDER)
  const [jsSourceId, setJsSourceId] = useState('')
  const [avatarUrl, setAvatarUrl] = useState('')
  const [avatarUploading, setAvatarUploading] = useState(false)
  const [previewConfig, setPreviewConfig] = useState(() =>
    loadEmbedPreviewConfig(getApiBaseURL().replace(/\/$/, '')),
  )

  const handlePreviewConfigChange = useCallback((config: typeof previewConfig) => {
    setPreviewConfig(config)
    saveEmbedPreviewConfig(config)
  }, [])

  useEffect(() => {
    sidebarBeforeRef.current = isCollapsed
    setIsCollapsed(true)
    return () => {
      setIsCollapsed(sidebarBeforeRef.current ?? false)
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps -- collapse on enter, restore previous state on leave
  }, [])

  const loadCreateDefaults = useCallback(async () => {
    setContentLoading(true)
    try {
      setContent(await fetchDefaultEmbedJS())
    } catch {
      setContent('// paste embed JS here\n')
    } finally {
      setContentLoading(false)
    }
  }, [])

  const loadExisting = useCallback(async () => {
    if (!isEdit || !editId) return
    setLoading(true)
    try {
      const res = await getJSTemplate(editId)
      if (res.code !== 200 || !res.data) {
        showAlert(res.msg || t('common.loadFailed'), 'error')
        navigate('/js-templates')
        return
      }
      const row = res.data
      setName(row.name)
      setUsage(row.usage || '')
      setStatus(row.status || 'active')
      setContent(row.content || '')
      setJsSourceId(row.jsSourceId || '')
      setAvatarUrl(row.avatarUrl || '')
    } catch (e: unknown) {
      showAlert((e as { msg?: string })?.msg || t('common.loadFailed'), 'error')
      navigate('/js-templates')
    } finally {
      setLoading(false)
    }
  }, [editId, isEdit, navigate, t])

  useEffect(() => {
    if (isEdit) {
      void loadExisting()
    } else {
      void loadCreateDefaults()
    }
  }, [isEdit, loadExisting, loadCreateDefaults])

  const save = async () => {
    if (!name.trim() || !content.trim()) {
      showAlert(t('jsTemplate.nameContentRequired'), 'error')
      setSettingsOpen(true)
      return
    }
    setSaving(true)
    try {
      const body = {
        name: name.trim(),
        content: content.trim(),
        usage: usage.trim(),
        status,
        avatarUrl: avatarUrl.trim(),
      }
      const res = isEdit && editId
        ? await updateJSTemplate(editId, body)
        : await createJSTemplate(body)
      if (res.code !== 200) {
        showAlert(res.msg || t('common.saveFailed'), 'error')
        return
      }
      showAlert(t('common.saveSuccess'), 'success')
      const createdId = res.data?.id
      if (!isEdit && isValidJSTemplateId(createdId)) {
        navigate(`/js-templates/${createdId}/edit`)
        return
      }
      navigate('/js-templates')
    } catch (e: unknown) {
      showAlert((e as { msg?: string })?.msg || t('common.saveFailed'), 'error')
    } finally {
      setSaving(false)
    }
  }

  return (
    <BaseLayout hideHeader>
      <div className="flex h-[calc(100vh)] w-full min-w-0 flex-col overflow-hidden bg-background">
        <div className="flex h-14 shrink-0 items-center justify-between gap-3 border-b border-border px-4">
          <div className="flex min-w-0 items-center gap-2">
            <Button type="text" icon={<IconArrowLeft />} onClick={() => navigate('/js-templates')}>
              {t('common.back')}
            </Button>
            <div className="min-w-0">
              <Typography.Text bold className="!block truncate">
                {name.trim() || (isEdit ? t('jsTemplate.edit') : t('jsTemplate.create'))}
              </Typography.Text>
              {usage.trim() ? (
                <Typography.Text type="secondary" className="!text-xs truncate !block">
                  {usage.trim()}
                </Typography.Text>
              ) : null}
            </div>
          </div>
          <div className="flex shrink-0 items-center gap-2">
            <Button type="outline" icon={<IconSettings />} onClick={() => setSettingsOpen(true)}>
              {t('jsTemplate.settingsTitle')}
            </Button>
            <Button type="outline" onClick={() => navigate('/js-templates')}>
              {t('common.cancel')}
            </Button>
            <Button type="primary" loading={saving} onClick={() => void save()}>
              {t('common.save')}
            </Button>
          </div>
        </div>

        <div className="min-h-0 flex-1 overflow-hidden">
          {loading ? (
            <div className="flex h-full items-center justify-center">
              <Spin tip={t('common.loading')} />
            </div>
          ) : (
            <JSTemplateEditorSplit
              content={content}
              onChange={setContent}
              previewConfig={previewConfig}
              onPreviewConfigChange={handlePreviewConfigChange}
              contentLoading={contentLoading}
              deferPreview={!isEdit}
            />
          )}
        </div>
      </div>

      <JSTemplateSettingsModal
        visible={settingsOpen}
        name={name}
        usage={usage}
        status={status}
        avatarUrl={avatarUrl || undefined}
        jsSourceId={jsSourceId || undefined}
        avatarUploading={avatarUploading}
        onAvatarPick={
          isEdit && editId
            ? async (file) => {
                setAvatarUploading(true)
                try {
                  const res = await uploadJSTemplateAvatar(editId, file)
                  if (res.code !== 200) {
                    showAlert(res.msg || t('jsTemplate.avatarFailed'), 'error')
                    return
                  }
                  const url = res.data?.avatarUrl || res.data?.template?.avatarUrl || ''
                  if (url) setAvatarUrl(url)
                  showAlert(t('jsTemplate.avatarUpdated'), 'success')
                } catch (e: unknown) {
                  showAlert((e as { msg?: string })?.msg || t('jsTemplate.avatarFailed'), 'error')
                } finally {
                  setAvatarUploading(false)
                }
              }
            : undefined
        }
        onChange={(patch) => {
          if (patch.name !== undefined) setName(patch.name)
          if (patch.usage !== undefined) setUsage(patch.usage)
          if (patch.status !== undefined) setStatus(patch.status)
          if (patch.avatarUrl !== undefined) setAvatarUrl(patch.avatarUrl)
        }}
        onClose={() => setSettingsOpen(false)}
        onConfirm={() => setSettingsOpen(false)}
      />
    </BaseLayout>
  )
}
