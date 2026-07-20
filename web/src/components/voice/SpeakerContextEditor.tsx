import { useEffect, useState } from 'react'
import { Input, Modal, Select, Space, Typography } from '@arco-design/web-react'
import { Plus, Trash2 } from 'lucide-react'
import { Button } from '@/components/ui'
import {
  getVoiceprintSpeaker,
  upsertVoiceprintSpeaker,
  type SpeakerAttribute,
  type VoiceprintProfile,
} from '@/api/voiceprint'
import { showAlert } from '@/utils/notification'

type Props = {
  profile: VoiceprintProfile | null
  onClose: () => void
  onSaved?: () => void
}

type AttrRow = SpeakerAttribute
type CredDraft = { provider: string; secretRef: string; scopes: string; hasSecret: boolean; clear: boolean }

const ATTR_EXAMPLES = [
  { key: 'role', value: 'teacher', hint: '角色：teacher / student / agent' },
  { key: 'cloudsteps_user', value: '12345', hint: '外部系统用户 ID（给模型看，不是 token）' },
  { key: 'preferred_lang', value: 'zh', hint: '偏好语言' },
]

export default function SpeakerContextEditor({ profile, onClose, onSaved }: Props) {
  const [loading, setLoading] = useState(false)
  const [saving, setSaving] = useState(false)
  const [displayName, setDisplayName] = useState('')
  const [notes, setNotes] = useState('')
  const [attrs, setAttrs] = useState<AttrRow[]>([{ key: 'role', value: 'teacher', visibility: 'llm' }])
  const [creds, setCreds] = useState<CredDraft[]>([
    { provider: 'cloudsteps', secretRef: '', scopes: 'read_schedule,book_lesson', hasSecret: false, clear: false },
  ])

  useEffect(() => {
    if (!profile) return
    setLoading(true)
    void (async () => {
      try {
        const res = await getVoiceprintSpeaker(profile.id)
        if (res.code !== 200 || !res.data) {
          setDisplayName(profile.name)
          return
        }
        const d = res.data
        setDisplayName(d.subject?.displayName || d.name || profile.name)
        setNotes(d.subject?.notes || '')
        setAttrs(
          d.attributes?.length
            ? d.attributes.map((a) => ({
                key: a.key,
                value: a.value,
                visibility: a.visibility || 'llm',
              }))
            : [{ key: 'role', value: 'teacher', visibility: 'llm' }],
        )
        if (d.credentials?.length) {
          setCreds(
            d.credentials.map((c) => ({
              provider: c.provider,
              secretRef: '',
              scopes: c.scopes || '',
              hasSecret: !!c.hasSecret,
              clear: false,
            })),
          )
        } else {
          setCreds([
            {
              provider: 'cloudsteps',
              secretRef: '',
              scopes: 'read_schedule,book_lesson',
              hasSecret: false,
              clear: false,
            },
          ])
        }
      } finally {
        setLoading(false)
      }
    })()
  }, [profile])

  const handleSave = async () => {
    if (!profile) return
    setSaving(true)
    try {
      const res = await upsertVoiceprintSpeaker(profile.id, {
        displayName: displayName.trim(),
        notes: notes.trim(),
        attributes: attrs.filter((a) => a.key.trim()),
        credentials: creds
          .filter((c) => c.provider.trim())
          .map((c) => ({
            provider: c.provider.trim(),
            secretRef: c.clear ? undefined : c.secretRef.trim() || undefined,
            scopes: c.scopes.trim() || undefined,
            clear: c.clear,
          })),
      })
      if (res.code !== 200) {
        showAlert(res.msg || '保存失败', 'error')
        return
      }
      showAlert('说话人上下文已保存', 'success')
      onSaved?.()
      onClose()
    } finally {
      setSaving(false)
    }
  }

  return (
    <Modal
      title={profile ? `说话人上下文 · ${profile.name}` : '说话人上下文'}
      visible={!!profile}
      onCancel={onClose}
      onOk={() => void handleSave()}
      confirmLoading={saving}
      okText="保存"
      cancelText="取消"
      style={{ width: 640 }}
      unmountOnExit
    >
      {loading ? (
        <Typography.Text type="secondary">加载中…</Typography.Text>
      ) : (
        <div className="space-y-4">
          <div className="rounded-lg border border-border bg-neutral-50 px-3 py-2 text-xs text-neutral-600 space-y-1">
            <div>
              <b>显示名</b>：会话里模型看到的姓名，一般填「张老师」这类称呼。
            </div>
            <div>
              <b>LLM 属性</b>：键值对，会写进「本通说话人」卡片。visibility 选 <code>llm</code> 才会给模型看；
              <code>internal</code> / <code>tool</code> 不会进提示词。
            </div>
            <div>
              <b>工具凭证</b>：provider 填业务名（云阶固定 <code>cloudsteps</code>），secret 填该账号的 Bearer
              token；scopes 可选，逗号分隔，仅作备注/权限说明。
            </div>
          </div>

          <Input
            addBefore="显示名"
            value={displayName}
            onChange={setDisplayName}
            placeholder="例：张老师"
          />
          <Input
            addBefore="备注"
            value={notes}
            onChange={setNotes}
            placeholder="例：负责四级陪练排课（可选）"
            allowClear
          />

          <div>
            <div className="mb-1 flex items-center justify-between">
              <Typography.Text bold>LLM 属性</Typography.Text>
              <Button
                size="mini"
                icon={<Plus size={12} />}
                onClick={() => setAttrs((prev) => [...prev, { key: '', value: '', visibility: 'llm' }])}
              >
                添加
              </Button>
            </div>
            <Typography.Paragraph type="secondary" style={{ marginBottom: 8, fontSize: 12 }}>
              常用示例：
              {ATTR_EXAMPLES.map((ex) => (
                <button
                  key={ex.key}
                  type="button"
                  className="ml-2 text-primary underline-offset-2 hover:underline"
                  onClick={() =>
                    setAttrs((prev) => {
                      const exists = prev.some((a) => a.key === ex.key)
                      if (exists) {
                        return prev.map((a) => (a.key === ex.key ? { ...a, value: ex.value, visibility: 'llm' } : a))
                      }
                      return [...prev.filter((a) => a.key.trim()), { key: ex.key, value: ex.value, visibility: 'llm' }]
                    })
                  }
                >
                  {ex.key}={ex.value}
                </button>
              ))}
            </Typography.Paragraph>
            <Space direction="vertical" className="w-full" size="small">
              {attrs.map((a, i) => (
                <div key={i} className="flex flex-wrap items-center gap-2">
                  <Input
                    style={{ width: 130 }}
                    placeholder="key，如 role"
                    value={a.key}
                    onChange={(v) =>
                      setAttrs((prev) => prev.map((x, j) => (j === i ? { ...x, key: v } : x)))
                    }
                  />
                  <Input
                    style={{ flex: 1, minWidth: 120 }}
                    placeholder="value，如 teacher"
                    value={a.value}
                    onChange={(v) =>
                      setAttrs((prev) => prev.map((x, j) => (j === i ? { ...x, value: v } : x)))
                    }
                  />
                  <Select
                    style={{ width: 110 }}
                    value={a.visibility || 'llm'}
                    onChange={(v) =>
                      setAttrs((prev) =>
                        prev.map((x, j) => (j === i ? { ...x, visibility: String(v) } : x)),
                      )
                    }
                    options={[
                      { label: 'llm（给模型）', value: 'llm' },
                      { label: 'internal', value: 'internal' },
                      { label: 'tool', value: 'tool' },
                    ]}
                  />
                  <Button
                    size="mini"
                    status="danger"
                    icon={<Trash2 size={12} />}
                    onClick={() => setAttrs((prev) => prev.filter((_, j) => j !== i))}
                  />
                </div>
              ))}
            </Space>
          </div>

          <div>
            <div className="mb-1 flex items-center justify-between">
              <Typography.Text bold>工具凭证</Typography.Text>
              <Button
                size="mini"
                icon={<Plus size={12} />}
                onClick={() =>
                  setCreds((prev) => [
                    ...prev,
                    { provider: '', secretRef: '', scopes: '', hasSecret: false, clear: false },
                  ])
                }
              >
                添加
              </Button>
            </div>
            <Typography.Paragraph type="secondary" style={{ marginBottom: 8, fontSize: 12 }}>
              云阶示例：provider 填 <code>cloudsteps</code>，secret 填老师账号登录后的 Bearer token（不要带
              “Bearer ”前缀也可以，按你 MCP 工具约定）。scopes 可填{' '}
              <code>read_schedule,book_lesson</code> 作说明，可不填。
            </Typography.Paragraph>
            <Space direction="vertical" className="w-full" size="small">
              {creds.map((c, i) => (
                <div key={i} className="space-y-1 rounded-lg border border-border p-2">
                  <div className="flex flex-wrap gap-2">
                    <Input
                      style={{ width: 140 }}
                      placeholder="cloudsteps"
                      value={c.provider}
                      onChange={(v) =>
                        setCreds((prev) => prev.map((x, j) => (j === i ? { ...x, provider: v } : x)))
                      }
                    />
                    <Input
                      style={{ flex: 1, minWidth: 160 }}
                      placeholder={c.hasSecret ? '已保存（留空不改）' : '粘贴 token / secret'}
                      value={c.secretRef}
                      onChange={(v) =>
                        setCreds((prev) =>
                          prev.map((x, j) => (j === i ? { ...x, secretRef: v, clear: false } : x)),
                        )
                      }
                    />
                  </div>
                  <div className="flex flex-wrap items-center gap-2">
                    <Input
                      style={{ flex: 1 }}
                      placeholder="scopes：read_schedule,book_lesson（可选）"
                      value={c.scopes}
                      onChange={(v) =>
                        setCreds((prev) => prev.map((x, j) => (j === i ? { ...x, scopes: v } : x)))
                      }
                    />
                    {c.hasSecret ? (
                      <Button
                        size="mini"
                        status="danger"
                        type={c.clear ? 'primary' : 'outline'}
                        onClick={() =>
                          setCreds((prev) =>
                            prev.map((x, j) => (j === i ? { ...x, clear: !x.clear } : x)),
                          )
                        }
                      >
                        {c.clear ? '将清除' : '清除凭证'}
                      </Button>
                    ) : null}
                    <Button
                      size="mini"
                      status="danger"
                      icon={<Trash2 size={12} />}
                      onClick={() => setCreds((prev) => prev.filter((_, j) => j !== i))}
                    />
                  </div>
                </div>
              ))}
            </Space>
          </div>
        </div>
      )}
    </Modal>
  )
}
