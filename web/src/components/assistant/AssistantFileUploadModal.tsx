import { useRef } from 'react'
import { Modal, Slider, Switch, Typography } from '@arco-design/web-react'
import { IconFile } from '@arco-design/web-react/icon'
import type { FileUploadConfigDraft } from '@/constants/assistantAdvancedConfig'
import { Button } from '@/components/ui'
import AssistantAvatar from '@/components/assistant/AssistantAvatar'

export type AssistantFileUploadModalProps = {
  visible: boolean
  value: FileUploadConfigDraft
  onChange: (next: FileUploadConfigDraft) => void
  onClose: () => void
  onConfirm: () => void
}

export default function AssistantFileUploadModal({
  visible,
  value,
  onChange,
  onClose,
  onConfirm,
}: AssistantFileUploadModalProps) {
  const draft = value
  const set = (patch: Partial<FileUploadConfigDraft>) => onChange({ ...draft, ...patch })

  return (
    <Modal
      visible={visible}
      title={
        <span className="inline-flex items-center gap-2">
          <IconFile style={{ color: '#14b8a6' }} />
          文件上传
        </span>
      }
      onCancel={onClose}
      footer={
        <Button type="primary" onClick={onConfirm}>
          确认
        </Button>
      }
      style={{ width: 480 }}
    >
      <div className="space-y-5 py-2">
        <div className="flex items-center justify-between">
          <div>
            <Typography.Text bold>文档上传</Typography.Text>
            <div className="text-xs text-muted-foreground">支持 PDF、Word、Markdown 等文档解析</div>
          </div>
          <Switch
            checked={!!draft.documentEnabled && !!draft.enabled}
            onChange={(on) => set({ enabled: on || !!draft.imageEnabled, documentEnabled: on })}
          />
        </div>
        {draft.documentEnabled && draft.enabled && (
          <label className="flex items-center gap-2 text-sm text-muted-foreground">
            <input
              type="checkbox"
              checked={!!draft.pdfEnhanced}
              onChange={(e) => set({ pdfEnhanced: e.target.checked })}
            />
            PDF 增强解析（保留表格与换行）
          </label>
        )}
        <div className="flex items-center justify-between">
          <div>
            <Typography.Text bold>图片上传</Typography.Text>
            <div className="text-xs text-muted-foreground">开启后调试对话可发送图片文件</div>
          </div>
          <Switch
            checked={!!draft.imageEnabled && !!draft.enabled}
            onChange={(on) => set({ enabled: on || !!draft.documentEnabled, imageEnabled: on })}
          />
        </div>
        <div>
          <Typography.Text bold>最大文件数量</Typography.Text>
          <Slider
            min={1}
            max={20}
            value={draft.maxFiles ?? 8}
            onChange={(v) => set({ maxFiles: Number(v) })}
            style={{ marginTop: 12 }}
          />
          <div className="mt-1 flex justify-between text-xs text-muted-foreground">
            <span>1</span>
            <span>{draft.maxFiles ?? 8}</span>
            <span>20</span>
          </div>
        </div>
      </div>
    </Modal>
  )
}

export type AssistantFileUploadRowProps = {
  config: FileUploadConfigDraft
  onConfigure: () => void
}

export function AssistantFileUploadRow({ config, onConfigure }: AssistantFileUploadRowProps) {
  const enabled = !!config.enabled && (!!config.documentEnabled || !!config.imageEnabled)
  return (
    <div className="flex items-center justify-between rounded-xl border border-border bg-card p-4">
      <div>
        <Typography.Text bold>文件上传</Typography.Text>
        <div className="text-xs text-muted-foreground">
          {enabled ? '已开启 · 调试模式可发送文件' : '关闭 · 调试模式仅支持文本'}
        </div>
      </div>
      <Button type="outline" size="small" onClick={onConfigure}>
        {enabled ? '配置' : '开启'}
      </Button>
    </div>
  )
}

export function AssistantAvatarPicker({
  avatarUrl,
  onPick,
}: {
  avatarUrl?: string
  onPick: (file: File) => void
}) {
  const inputRef = useRef<HTMLInputElement>(null)
  return (
    <div>
      <button type="button" className="block" onClick={() => inputRef.current?.click()}>
        <AssistantAvatar src={avatarUrl} size="lg" rounded="xl" className="cursor-pointer hover:ring-2 hover:ring-primary/30" />
      </button>
      <input
        ref={inputRef}
        type="file"
        accept="image/*"
        className="hidden"
        onChange={(e) => {
          const f = e.target.files?.[0]
          if (f) onPick(f)
          e.target.value = ''
        }}
      />
    </div>
  )
}
