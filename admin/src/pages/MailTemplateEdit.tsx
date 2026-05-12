// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0
//
// 邮件模板独立编辑页：左侧表单 + 右侧 HTML 实时预览。
// 路由：/mail-templates/new、/mail-templates/:id/edit
import { useEffect, useMemo, useState } from 'react'
import { useNavigate, useParams } from 'react-router-dom'
import { Button, Input, Switch, Space, Card, Tabs, Message } from '@arco-design/web-react'
import { ArrowLeft, Save, Eye, Code2 } from 'lucide-react'
import PageHeader from '@/components/Layout/PageHeader'
import {
  getMailTemplate,
  createMailTemplate,
  updateMailTemplate,
  type MailTemplate,
  type MailTemplateUpsertReq,
} from '@/services/notificationsApi'

const TextArea = Input.TextArea
const TabPane = Tabs.TabPane

const empty: MailTemplateUpsertReq = {
  code: '',
  name: '',
  subject: '',
  htmlBody: '',
  description: '',
  variables: '',
  locale: '',
  enabled: true,
}

// 占位符替换：把 {{Name}} 替换为 [Name]（仅用于预览，不实际渲染 go template）。
const previewHtml = (html: string): string => {
  if (!html) return ''
  return html.replace(/\{\{\s*([A-Za-z_][\w]*)\s*\}\}/g, (_, name) => `<span style="background:#fffbe6;border:1px dashed #faad14;padding:0 2px;border-radius:2px;">{{${name}}}</span>`)
}

const MailTemplateEditPage = () => {
  const params = useParams<{ id?: string }>()
  const navigate = useNavigate()
  const idNum = params.id ? Number(params.id) : 0
  const isEdit = idNum > 0

  const [loading, setLoading] = useState(false)
  const [saving, setSaving] = useState(false)
  const [form, setForm] = useState<MailTemplateUpsertReq>(empty)
  const [original, setOriginal] = useState<MailTemplate | null>(null)

  useEffect(() => {
    if (!isEdit) return
    setLoading(true)
    getMailTemplate(idNum)
      .then((t) => {
        setOriginal(t)
        setForm({
          code: t.code,
          name: t.name,
          subject: t.subject,
          htmlBody: t.htmlBody,
          description: t.description,
          variables: t.variables,
          locale: t.locale,
          enabled: t.enabled,
        })
      })
      .catch((e) => Message.error(`读取模板失败：${e?.msg || e?.message || e}`))
      .finally(() => setLoading(false))
  }, [idNum, isEdit])

  const previewSrcDoc = useMemo(() => previewHtml(form.htmlBody || ''), [form.htmlBody])

  const updateField = <K extends keyof MailTemplateUpsertReq>(k: K, v: MailTemplateUpsertReq[K]) => {
    setForm((prev) => ({ ...prev, [k]: v }))
  }

  const handleSave = async () => {
    if (!form.name?.trim()) {
      Message.error('请填写模板名称')
      return
    }
    if (!isEdit && !form.code?.trim()) {
      Message.error('请填写模板编码')
      return
    }
    if (!form.htmlBody?.trim()) {
      Message.error('请填写 HTML 正文')
      return
    }
    setSaving(true)
    try {
      if (isEdit) {
        await updateMailTemplate(idNum, form)
        Message.success('更新成功')
      } else {
        await createMailTemplate(form)
        Message.success('创建成功')
      }
      navigate('/mail-templates')
    } catch (e: any) {
      Message.error(`保存失败：${e?.msg || e?.message || e}`)
    } finally {
      setSaving(false)
    }
  }

  return (
    <div className="space-y-4">
      <PageHeader
        title={isEdit ? `编辑模板：${original?.name || ''}` : '新建邮件模板'}
        description="左侧编辑模板内容，右侧实时预览渲染效果。占位符使用 Go template 语法：{{Name}}。"
        actions={
          <Space>
            <Button onClick={() => navigate('/mail-templates')}>
              <span className="inline-flex items-center gap-1"><ArrowLeft size={14} /> 返回列表</span>
            </Button>
            <Button type="primary" loading={saving} onClick={handleSave}>
              <span className="inline-flex items-center gap-1"><Save size={14} /> 保存</span>
            </Button>
          </Space>
        }
      />

      {loading ? (
        <Card>
          <div className="p-8 text-center text-[var(--color-text-3)]">加载中...</div>
        </Card>
      ) : (
        <div className="grid grid-cols-1 lg:grid-cols-2 gap-4">
          {/* 左：编辑 */}
          <Card title="模板内容" bordered>
            <div className="space-y-3">
              <LabeledField label="编码 code" required>
                <Input
                  value={form.code || ''}
                  disabled={isEdit}
                  placeholder="例如：welcome、verification"
                  onChange={(v) => updateField('code', v)}
                />
              </LabeledField>
              <LabeledField label="名称" required>
                <Input
                  value={form.name}
                  placeholder="模板展示名"
                  onChange={(v) => updateField('name', v)}
                />
              </LabeledField>
              <LabeledField label="语言（可选）">
                <Input
                  value={form.locale || ''}
                  placeholder="zh-CN / en-US"
                  onChange={(v) => updateField('locale', v)}
                />
              </LabeledField>
              <LabeledField label="主题 subject">
                <Input
                  value={form.subject || ''}
                  placeholder="支持 {{Name}} 占位符"
                  onChange={(v) => updateField('subject', v)}
                />
              </LabeledField>
              <LabeledField label="说明">
                <Input
                  value={form.description || ''}
                  placeholder="模板用途简介"
                  onChange={(v) => updateField('description', v)}
                />
              </LabeledField>
              <LabeledField label="HTML 正文" required>
                <TextArea
                  value={form.htmlBody}
                  onChange={(v) => updateField('htmlBody', v)}
                  rows={16}
                  style={{ fontFamily: 'ui-monospace, SFMono-Regular, Menlo, monospace', fontSize: 12 }}
                  placeholder="<html>...</html>"
                />
                <p className="text-xs text-[var(--color-text-3)] mt-1">
                  保存时后端自动派生纯文本正文与占位符列表。
                </p>
              </LabeledField>
              <LabeledField label="启用">
                <Switch
                  checked={form.enabled !== false}
                  onChange={(v) => updateField('enabled', v)}
                />
              </LabeledField>
            </div>
          </Card>

          {/* 右：预览 */}
          <Card
            title={
              <span className="inline-flex items-center gap-2">
                <Eye size={14} /> 实时预览
              </span>
            }
            bordered
            bodyStyle={{ padding: 0 }}
          >
            <Tabs defaultActiveTab="render">
              <TabPane key="render" title="渲染">
                <div style={{ height: 600, padding: 12 }}>
                  <iframe
                    title="email-preview"
                    sandbox=""
                    srcDoc={previewSrcDoc}
                    style={{
                      width: '100%',
                      height: '100%',
                      border: '1px solid var(--color-border-2)',
                      borderRadius: 4,
                      background: '#fff',
                    }}
                  />
                </div>
              </TabPane>
              <TabPane
                key="source"
                title={
                  <span className="inline-flex items-center gap-1">
                    <Code2 size={14} /> 源码
                  </span>
                }
              >
                <pre
                  style={{
                    margin: 0,
                    padding: 12,
                    height: 600,
                    overflow: 'auto',
                    fontFamily: 'ui-monospace, SFMono-Regular, Menlo, monospace',
                    fontSize: 12,
                    background: 'var(--color-fill-1)',
                  }}
                >
                  {form.htmlBody || '（空）'}
                </pre>
              </TabPane>
            </Tabs>
          </Card>
        </div>
      )}
    </div>
  )
}

const LabeledField = ({
  label,
  required,
  children,
}: {
  label: string
  required?: boolean
  children: React.ReactNode
}) => (
  <div>
    <label className="block text-sm font-medium text-[var(--color-text-2)] mb-1">
      {label}
      {required && <span className="text-red-500 ml-0.5">*</span>}
    </label>
    {children}
  </div>
)

export default MailTemplateEditPage
