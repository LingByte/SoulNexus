import React, { useEffect, useState } from 'react'
import { Globe, Zap, Calendar, Webhook } from 'lucide-react'
import { Button } from '@/components/ui'
import { Input, Switch, Typography } from '@arco-design/web-react'
import { useTranslation } from '@/i18n'
import { getApiBaseURL } from '@/config/apiConfig'
import type { WorkflowDefinition, WorkflowTriggerConfig } from '@/api/workflow'
import {
  buildDefaultApiTestBody,
  buildWorkflowExecuteCurl,
  getWorkflowStartParameterNames,
  normalizeApiTestBody,
} from '@/utils/workflowApiTest'

// 触发器配置面板组件
export interface TriggerConfigPanelProps {
  workflow: WorkflowDefinition | null
  triggerConfig: WorkflowTriggerConfig | undefined
  onUpdate: (config: WorkflowTriggerConfig) => void
  onSave: () => Promise<void>
  onCancel: () => void
  saving: boolean
}

export const TriggerConfigPanel: React.FC<TriggerConfigPanelProps> = ({
  workflow,
  triggerConfig,
  onUpdate,
  onSave,
  onCancel,
  saving
}) => {
  const { t } = useTranslation()
  const [copiedKey, setCopiedKey] = useState(false)
  const [copiedCurl, setCopiedCurl] = useState(false)
  const [apiKeyVisible, setApiKeyVisible] = useState(false)
  const [apiExampleBody, setApiExampleBody] = useState('{}')

  const safeTriggerConfig: WorkflowTriggerConfig = triggerConfig || {}
  const startParamNames = getWorkflowStartParameterNames(workflow)

  useEffect(() => {
    if (workflow?.id) {
      setApiExampleBody(buildDefaultApiTestBody(workflow))
    }
  }, [workflow?.id])

  const normalizedExampleBody = normalizeApiTestBody(apiExampleBody)
  const curlCommand = workflow?.slug
    ? buildWorkflowExecuteCurl({
        slug: workflow.slug,
        apiKey: safeTriggerConfig.api?.apiKey,
        body: normalizedExampleBody.error ? '{"parameters":{}}' : normalizedExampleBody.body,
      })
    : ''

  const copyCurl = () => {
    if (!curlCommand) return
    navigator.clipboard.writeText(curlCommand)
    setCopiedCurl(true)
    setTimeout(() => setCopiedCurl(false), 2000)
  }

  const generateAPIKey = () => {
    const key = Array.from(crypto.getRandomValues(new Uint8Array(32)))
      .map(b => b.toString(16).padStart(2, '0'))
      .join('')
    
    onUpdate({
      ...safeTriggerConfig,
      api: {
        ...safeTriggerConfig.api,
        enabled: true,
        apiKey: key,
        public: safeTriggerConfig.api?.public ?? false
      }
    })
  }

  const copyAPIKey = () => {
    if (safeTriggerConfig.api?.apiKey) {
      navigator.clipboard.writeText(safeTriggerConfig.api.apiKey)
      setCopiedKey(true)
      setTimeout(() => setCopiedKey(false), 2000)
    }
  }

  const getWebhookURL = () => {
    if (!workflow) return ''
    const baseURL = window.location.origin
    return `${baseURL}/api/public/workflows/webhook/${workflow.slug}`
  }

  return (
    <div className="space-y-6 max-h-[70vh] overflow-y-auto">
      {/* API 触发 */}
      <div className="border border-gray-200 dark:border-gray-700 rounded-lg p-4">
        <div className="flex items-center justify-between mb-4">
          <div className="flex items-center gap-2">
            <Globe className="w-5 h-5 text-blue-500" />
            <h3 className="text-base font-semibold text-gray-900 dark:text-white">API 触发</h3>
          </div>
          <Switch
            checked={safeTriggerConfig.api?.enabled || false}
            onChange={(checked) => onUpdate({
              ...safeTriggerConfig,
              api: {
                ...safeTriggerConfig.api,
                enabled: checked,
                public: safeTriggerConfig.api?.public ?? false,
                apiKey: safeTriggerConfig.api?.apiKey,
              },
            })}
          />
        </div>
        
        {safeTriggerConfig.api?.enabled && (
          <div className="space-y-4 mt-4">
            <div className="flex items-center justify-between">
              <Typography.Text className="!text-sm">公开 API（不需要认证）</Typography.Text>
              <Switch
                checked={safeTriggerConfig.api?.public || false}
                onChange={(checked) => onUpdate({
                  ...safeTriggerConfig,
                  api: {
                    ...safeTriggerConfig.api,
                    enabled: true,
                    public: checked,
                    apiKey: safeTriggerConfig.api?.apiKey,
                  },
                })}
              />
            </div>
            
            {safeTriggerConfig.api?.public && (
              <div className="space-y-2">
                <div className="flex items-center gap-2">
                  <Input size="large" className="!h-10 !text-base ![&::placeholder]:text-base" type={apiKeyVisible ? 'text' : 'password'}
                    value={safeTriggerConfig.api?.apiKey || ''}
                    onChange={(val) => onUpdate({
                      ...safeTriggerConfig,
                      api: {
                        ...safeTriggerConfig.api,
                        enabled: true,
                        public: true,
                        apiKey: val
                      }
                    })}
                    placeholder="API 密钥"
                  />
                  <Button
                    variant="outline"
                    size="sm"
                    onClick={() => setApiKeyVisible(!apiKeyVisible)}
                    title={apiKeyVisible ? '隐藏密钥' : '显示密钥'}
                  >
                    {apiKeyVisible ? '隐藏' : '显示'}
                  </Button>
                  <Button
                    variant="outline"
                    size="sm"
                    onClick={generateAPIKey}
                  >
                    生成
                  </Button>
                  {safeTriggerConfig.api?.apiKey && (
                    <Button
                      variant="outline"
                      size="sm"
                      onClick={copyAPIKey}
                    >
                      {copiedKey ? '已复制' : '复制'}
                    </Button>
                  )}
                </div>
                <Typography.Text type="secondary" className="!text-xs block">
                  POST {getApiBaseURL()}/public/workflows/{workflow?.slug}/execute
                </Typography.Text>
                <div className="mt-3 space-y-2 rounded-lg border border-[var(--color-border-2)] p-3">
                  {workflow?.status !== 'active' ? (
                    <div className="rounded-md bg-[var(--color-warning-light-1)] px-3 py-2 text-xs text-[rgb(var(--warning-6))]">
                      当前工作流为“{t(`workflow.status.${workflow?.status || 'draft'}`)}”状态。公开 API 仅执行已激活的工作流，请先修改状态并保存。
                    </div>
                  ) : null}
                  <Typography.Text bold className="!text-xs">
                    {t('workflow.apiCurlExample')}
                  </Typography.Text>
                  {startParamNames.length > 0 ? (
                    <Typography.Text type="secondary" className="!text-xs block">
                      {t('workflow.apiTestParamsHint', { params: startParamNames.join(', ') })}
                    </Typography.Text>
                  ) : null}
                  <Typography.Text type="secondary" className="!text-xs block">
                    {t('workflow.apiCurlBodyHint')}
                  </Typography.Text>
                  <Input.TextArea
                    rows={4}
                    value={apiExampleBody}
                    onChange={setApiExampleBody}
                    placeholder={'{\n  "parameters": {\n    "city": "成都"\n  }\n}'}
                  />
                  {normalizedExampleBody.error === 'invalid_json' ? (
                    <Typography.Text type="error" className="!text-xs block">
                      {t('workflow.apiTestInvalidJson')}
                    </Typography.Text>
                  ) : null}
                  <div className="flex items-center justify-between gap-2">
                    <Typography.Text type="secondary" className="!text-xs">
                      {t('workflow.apiCurlHint')}
                    </Typography.Text>
                    <Button variant="outline" size="sm" onClick={copyCurl} disabled={!curlCommand}>
                      {copiedCurl ? t('workflow.apiCurlCopied') : t('workflow.apiCurlCopy')}
                    </Button>
                  </div>
                  <Input.TextArea
                    rows={8}
                    value={curlCommand}
                    readOnly
                    className="font-mono !text-xs"
                  />
                </div>
              </div>
            )}
          </div>
        )}
      </div>

      {/* 事件触发 */}
      <div className="border border-gray-200 dark:border-gray-700 rounded-lg p-4">
        <div className="flex items-center justify-between mb-4">
          <div className="flex items-center gap-2">
            <Zap className="w-5 h-5 text-yellow-500" />
            <h3 className="text-base font-semibold text-gray-900 dark:text-white">事件触发</h3>
          </div>
          <Switch
            checked={safeTriggerConfig.event?.enabled || false}
            onChange={(checked) => onUpdate({
              ...safeTriggerConfig,
              event: {
                ...safeTriggerConfig.event,
                enabled: checked,
                events: safeTriggerConfig.event?.events || [],
              },
            })}
          />
        </div>
        
        {safeTriggerConfig.event?.enabled && (
          <div className="space-y-4 mt-4">
            <div>
              <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
                监听的事件类型（每行一个，支持通配符 *）
              </label>
              <Input.TextArea
                rows={4}
                value={safeTriggerConfig.event?.events?.join('\n') || ''}
                onChange={(val) => onUpdate({
                  ...safeTriggerConfig,
                  event: {
                    ...safeTriggerConfig.event,
                    enabled: true,
                    events: val.split('\n').filter((s) => s.trim()),
                  },
                })}
                placeholder={'user.created\norder.paid\n*'}
              />
              <p className="text-xs text-gray-500 dark:text-gray-400 mt-1">
                示例: user.created, order.paid, * (监听所有事件)
              </p>
            </div>
          </div>
        )}
      </div>

      {/* 定时触发 */}
      <div className="border border-gray-200 dark:border-gray-700 rounded-lg p-4">
        <div className="flex items-center justify-between mb-4">
          <div className="flex items-center gap-2">
            <Calendar className="w-5 h-5 text-green-500" />
            <h3 className="text-base font-semibold text-gray-900 dark:text-white">定时触发</h3>
          </div>
          <Switch
            checked={safeTriggerConfig.schedule?.enabled || false}
            onChange={(checked) => onUpdate({
              ...safeTriggerConfig,
              schedule: {
                ...safeTriggerConfig.schedule,
                enabled: checked,
                cronExpr: safeTriggerConfig.schedule?.cronExpr || '0 0 * * *',
              },
            })}
          />
        </div>
        
        {safeTriggerConfig.schedule?.enabled && (
          <div className="space-y-4 mt-4">
            <div>
              <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
                Cron 表达式
              </label>
              <Input size="large" className="!h-10 !text-base ![&::placeholder]:text-base" value={safeTriggerConfig.schedule?.cronExpr || ''}
                onChange={(val) => onUpdate({
                  ...safeTriggerConfig,
                  schedule: {
                    ...safeTriggerConfig.schedule,
                    enabled: true,
                    cronExpr: val
                  }
                })}
                placeholder="0 0 * * *"
              />
              <p className="text-xs text-gray-500 dark:text-gray-400 mt-1">
                格式: 秒 分 时 日 月 星期。示例: 0 0 * * * (每天0点), 0 */30 * * * * (每30分钟)
              </p>
            </div>
          </div>
        )}
      </div>

      {/* Webhook 触发 */}
      <div className="border border-gray-200 dark:border-gray-700 rounded-lg p-4">
        <div className="flex items-center justify-between mb-4">
          <div className="flex items-center gap-2">
            <Webhook className="w-5 h-5 text-purple-500" />
            <h3 className="text-base font-semibold text-gray-900 dark:text-white">Webhook 触发</h3>
          </div>
          <Switch
            checked={safeTriggerConfig.webhook?.enabled || false}
            onChange={(checked) => onUpdate({
              ...safeTriggerConfig,
              webhook: {
                ...safeTriggerConfig.webhook,
                enabled: checked,
                secret: safeTriggerConfig.webhook?.secret,
              },
            })}
          />
        </div>
        
        {safeTriggerConfig.webhook?.enabled && (
          <div className="space-y-4 mt-4">
            <div>
              <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
                Webhook URL
              </label>
              <div className="flex items-center gap-2">
                <Input size="large" className="!h-10 !text-base ![&::placeholder]:text-base flex-1" value={getWebhookURL()}
                  readOnly
                  
                />
                <Button
                  variant="outline"
                  size="sm"
                  onClick={() => {
                    navigator.clipboard.writeText(getWebhookURL())
                    showAlert('Webhook URL 已复制到剪贴板', 'success', '已复制')
                  }}
                >
                  复制
                </Button>
              </div>
            </div>
            <div>
              <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
                Webhook 密钥（可选，用于验证）
              </label>
              <Input size="large" className="!h-10 !text-base ![&::placeholder]:text-base" type="password"
                value={safeTriggerConfig.webhook?.secret || ''}
                onChange={(val) => onUpdate({
                  ...safeTriggerConfig,
                  webhook: {
                    ...safeTriggerConfig.webhook,
                    enabled: true,
                    secret: val
                  }
                })}
                placeholder="留空则不验证"
              />
            </div>
          </div>
        )}
      </div>

      {/* 保存按钮 */}
      <div className="flex justify-end gap-2 pt-4 border-t border-gray-200 dark:border-gray-700">
        <Button type="outline" disabled={saving} onClick={onCancel}>
          {t('common.cancel')}
        </Button>
        <Button type="primary" onClick={onSave} loading={saving} disabled={saving}>
          {t('common.save')}
        </Button>
      </div>
    </div>
  )
}

