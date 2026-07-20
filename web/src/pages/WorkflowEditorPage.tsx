import React, { useState, useEffect, useRef } from 'react'
import { useNavigate, useParams } from 'react-router-dom'
import { motion, AnimatePresence } from 'framer-motion'
import { X, PanelRightOpen } from 'lucide-react'
import { Button, Empty } from '@/components/ui'
import {
  Input,
  Modal as ArcoModal,
  Select,
  Spin,
  Switch,
  Tabs,
  Tag,
  Typography,
} from '@arco-design/web-react'
import Modal from '@/components/workflow/ui/Modal'
import Badge from '@/components/workflow/ui/Badge'
import WorkflowEditor from '@/components/workflow/WorkflowEditor'
import Terminal, { TerminalLog } from '@/components/workflow/Terminal'
import { showAlert } from '@/utils/notification'
import workflowService, {
  WorkflowDefinition,
  UpdateWorkflowDefinitionRequest,
  WorkflowVersion,
  WorkflowVersionCompareResponse,
} from '@/api/workflow'
import BaseLayout from '@/components/Layout/BaseLayout'
import { useTranslation } from '@/i18n'
import { useSidebar } from '@/contexts/SidebarContext'
import { useWorkflowRun } from '@/hooks/useWorkflowRun'
import { toEditorWorkflow, editorWorkflowToGraph } from '@/utils/workflowTransform'
import { TriggerConfigPanel } from '@/components/workflow/panels/TriggerConfigPanel'
import WorkflowInvocationLogsPanel from '@/components/workflow/WorkflowInvocationLogsPanel'

type WorkflowStatus = 'draft' | 'active' | 'archived'
type EditorView = 'canvas' | 'logs'

export default function WorkflowEditorPage() {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const { id: idParam } = useParams<{ id: string }>()
  const workflowId = Number(idParam)
  const { setIsCollapsed, isCollapsed, effectiveSidebarWidth } = useSidebar()
  const sidebarBeforeEditorRef = useRef<boolean | null>(null)

  const [selectedWorkflow, setSelectedWorkflow] = useState<WorkflowDefinition | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [saving, setSaving] = useState(false)
  const [terminalLogs, setTerminalLogs] = useState<TerminalLog[]>([])
  const [isTerminalVisible, setIsTerminalVisible] = useState(false)
  const [currentInstanceId, setCurrentInstanceId] = useState<number | null>(null)
  const [isRunning, setIsRunning] = useState(false)
  const [showTriggerConfig, setShowTriggerConfig] = useState(false)
  const [triggerConfig, setTriggerConfig] = useState<WorkflowDefinition['triggers']>({})
  const [showVersionHistory, setShowVersionHistory] = useState(false)
  const [versions, setVersions] = useState<WorkflowVersion[]>([])
  const [loadingVersions, setLoadingVersions] = useState(false)
  const [showVersionCompare, setShowVersionCompare] = useState(false)
  const [compareData, setCompareData] = useState<WorkflowVersionCompareResponse | null>(null)
  const [selectedVersion1, setSelectedVersion1] = useState<number | null>(null)
  const [selectedVersion2, setSelectedVersion2] = useState<number | null>(null)
  const [changeNote, setChangeNote] = useState('')
  const [publishing, setPublishing] = useState(false)
  const [showPublishModal, setShowPublishModal] = useState(false)
  const [propertyTab, setPropertyTab] = useState<'basic' | 'params'>('basic')
  const [workflowStatus, setWorkflowStatus] = useState<WorkflowStatus>('draft')
  const [inputParameters, setInputParameters] = useState<any[]>([])
  const [outputParameters, setOutputParameters] = useState<any[]>([])
  const [showProperties, setShowProperties] = useState(false)
  const [editorView, setEditorView] = useState<EditorView>('canvas')

  const { runWorkflow } = useWorkflowRun({
    onLogsChange: setTerminalLogs,
    onVisibleChange: setIsTerminalVisible,
    onRunningChange: setIsRunning,
    onInstanceIdChange: setCurrentInstanceId,
  })

  useEffect(() => {
    if (!Number.isFinite(workflowId) || workflowId <= 0) {
      navigate('/workflows', { replace: true })
      return
    }
    setLoading(true)
    workflowService.getDefinition(workflowId).then((response) => {
      if (response.code === 200 && response.data) {
        setSelectedWorkflow(response.data)
        setError(null)
      } else {
        setError(response.msg || t('workflow.messages.loadFailed'))
      }
    }).catch((err: unknown) => {
      setError((err as { msg?: string }).msg || t('workflow.messages.loadFailed'))
    }).finally(() => setLoading(false))
  }, [workflowId, navigate, t])

  useEffect(() => {
    if (!selectedWorkflow) return
    setTriggerConfig(selectedWorkflow.triggers || {})
    setInputParameters(selectedWorkflow.inputParameters || [])
    setOutputParameters(selectedWorkflow.outputParameters || [])
    setWorkflowStatus(selectedWorkflow.status)
  }, [selectedWorkflow])

  useEffect(() => {
    if (!selectedWorkflow) return
    sidebarBeforeEditorRef.current = isCollapsed
    setIsCollapsed(true)
    return () => {
      setIsCollapsed(sidebarBeforeEditorRef.current ?? false)
      sidebarBeforeEditorRef.current = null
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps -- restore sidebar only when leaving editor
  }, [selectedWorkflow?.id, setIsCollapsed])

  const loadVersions = async (defId: number) => {
    setLoadingVersions(true)
    try {
      const response = await workflowService.listVersions(defId)
      if (response.code === 200 && Array.isArray(response.data)) {
        setVersions(response.data)
      }
    } finally {
      setLoadingVersions(false)
    }
  }

  const publishWorkflow = async () => {
    if (!selectedWorkflow) return
    setPublishing(true)
    try {
      const response = await workflowService.publishDefinition(selectedWorkflow.id, changeNote)
      if (response.code === 200) {
        setChangeNote('')
        setShowPublishModal(false)
        showAlert(t('common.publishSuccess'), 'success')
        await loadVersions(selectedWorkflow.id)
      } else {
        showAlert(response.msg || t('common.publishFailed'), 'error')
      }
    } catch (err: unknown) {
      showAlert((err as { msg?: string }).msg || t('common.publishFailed'), 'error')
    } finally {
      setPublishing(false)
    }
  }

  const handleCompareVersions = async (defId: number, v1Id: number, v2Id: number) => {
    try {
      const response = await workflowService.compareVersions(defId, v1Id, v2Id)
      if (response.code === 200 && response.data) {
        setCompareData(response.data)
        setShowVersionHistory(false)
        setShowVersionCompare(true)
        setSelectedVersion1(null)
        setSelectedVersion2(null)
      } else {
        showAlert(response.msg || t('workflow.messages.versionCompareFailed'), 'error')
      }
    } catch {
      showAlert(t('workflow.messages.versionCompareFailed'), 'error')
    }
  }

  const handleRollback = (defId: number, versionId: number) => {
    ArcoModal.confirm({
      title: t('common.confirm'),
      content: t('workflow.messages.rollbackConfirm'),
      onOk: async () => {
        setLoading(true)
        try {
          const response = await workflowService.rollbackVersion(defId, versionId)
          if (response.code === 200) {
            const reload = await workflowService.getDefinition(defId)
            if (reload.code === 200) setSelectedWorkflow(reload.data)
            setShowVersionHistory(false)
            showAlert(t('workflow.messages.rollbackSuccess'), 'success')
          }
        } finally {
          setLoading(false)
        }
      },
    })
  }

  const getStatusBadge = (status: WorkflowStatus) => {
    const variants = {
      draft: { variant: 'warning' as const, label: t('workflow.status.draft') },
      active: { variant: 'success' as const, label: t('workflow.status.active') },
      archived: { variant: 'secondary' as const, label: t('workflow.status.archived') },
    }
    const config = variants[status]
    return <Badge variant={config.variant}>{config.label}</Badge>
  }

  const formatDate = (dateString: string) => {
    return new Date(dateString).toLocaleDateString('zh-CN', { year: 'numeric', month: 'short', day: 'numeric' })
  }

  if (loading) {
    return (
      <BaseLayout hideHeader>
        <div className="flex h-[50vh] items-center justify-center">
          <Spin tip={t('workflow.loading')} />
        </div>
      </BaseLayout>
    )
  }

  if (!selectedWorkflow) {
    return (
      <BaseLayout hideHeader>
        <div className="flex h-[50vh] flex-col items-center justify-center gap-3">
          <Typography.Text type="error">{error || t('workflow.messages.loadFailed')}</Typography.Text>
          <Button type="outline" onClick={() => navigate('/workflows')}>{t('workflow.back')}</Button>
        </div>
      </BaseLayout>
    )
  }

  return (
    <BaseLayout hideHeader>
    <div className="flex h-[calc(100vh)] w-full min-w-0 flex-col overflow-hidden bg-gray-50 dark:bg-gray-900">
      {/* 顶部工具栏 */}
      <div className="h-14 border-b border-gray-200 dark:border-gray-800 dark:bg-gray-800 flex items-center justify-between gap-2 px-3 md:px-4 min-w-0">
        <div className="flex items-center gap-4">
          <Button 
            variant="ghost"
            size="sm"
            onClick={() => navigate('/workflows')}
          >
            {t('workflow.back')}
          </Button>
          <div className="h-6 w-px bg-gray-300 dark:bg-gray-600" />
          <div className="flex rounded-lg border border-gray-200 p-0.5 dark:border-gray-700">
            <Button
              type={editorView === 'canvas' ? 'primary' : 'text'}
              size="sm"
              onClick={() => setEditorView('canvas')}
            >
              编排
            </Button>
            <Button
              type={editorView === 'logs' ? 'primary' : 'text'}
              size="sm"
              onClick={() => setEditorView('logs')}
            >
              调用日志
            </Button>
          </div>
          <div className="h-6 w-px bg-gray-300 dark:bg-gray-600" />
          <div>
            <h1 className="text-sm font-semibold text-gray-900 dark:text-white">
              {selectedWorkflow.name}
            </h1>
            <p className="hidden sm:block text-xs text-gray-500 dark:text-gray-400 truncate">
              {selectedWorkflow.description}
            </p>
          </div>
        </div>
        <div className="flex flex-nowrap items-center justify-end gap-2 shrink-0 overflow-x-auto max-w-[min(100%,520px)]">
          <Tag size="small" color="arcoblue" className="shrink-0">
            v{selectedWorkflow.version}
          </Tag>
          {getStatusBadge(selectedWorkflow.status)}
          <Button
            type="text"
            size="sm"
            onClick={async () => {
              await loadVersions(selectedWorkflow.id)
              setShowVersionHistory(true)
            }}
          >
            {t('workflow.versionHistory')}
          </Button>
          <Button type="outline" size="sm" loading={publishing} onClick={() => setShowPublishModal(true)}>
            {t('common.publishVersion')}
          </Button>
          <Button type="outline" size="sm" onClick={() => setShowTriggerConfig(true)}>
            {t('workflow.triggerConfig')}
          </Button>
          {!showProperties && (
            <Button type="outline" size="sm" icon={<PanelRightOpen className="h-4 w-4" />} onClick={() => setShowProperties(true)}>
              {t('workflow.properties')}
            </Button>
          )}
        </div>
      </div>

      {/* 主内容区域 - 编辑器全屏，使用内部的节点库 */}
      <div className="flex min-h-0 flex-1 w-full overflow-hidden">
        {editorView === 'logs' ? (
          <div className="flex min-h-0 w-full flex-1 overflow-hidden">
            <WorkflowInvocationLogsPanel
              definitionId={selectedWorkflow.id}
              definitionName={selectedWorkflow.name}
            />
          </div>
        ) : (
        <div className="flex min-h-0 flex-1 w-full overflow-hidden">
        <div className="relative min-h-0 min-w-0 flex-1 overflow-hidden">
          <WorkflowEditor
            className="h-full w-full"
            workflowId={selectedWorkflow.id}
            workflow={toEditorWorkflow(selectedWorkflow)}
            onSave={async (workflow: any) => {
              if (!selectedWorkflow) {
                showAlert('没有选中的工作流', 'error', '保存失败')
                return
              }
              
              setSaving(true)
              setError(null)
              
              try {
                const updatedDefinition = editorWorkflowToGraph(workflow)
                
                const updateData: UpdateWorkflowDefinitionRequest = {
                  name: workflow.name,
                  description: workflow.description,
                  status: workflowStatus,
                  definition: updatedDefinition,
                  triggers: triggerConfig,
                  version: selectedWorkflow.version,
                  inputParameters: inputParameters,
                  outputParameters: outputParameters
                }
                
                const response = await workflowService.updateDefinition(selectedWorkflow.id, updateData)
                
                if (response.code === 200) {
                  setSelectedWorkflow(response.data)
                  showAlert(t('workflow.messages.saveSuccess'), 'success')
                } else {
                  setError(response.msg || '更新工作流失败')
                  showAlert(response.msg || '更新工作流失败', 'error', '保存失败')
                  
                  if (response.msg?.includes('version conflict')) {
                    const reloadResponse = await workflowService.getDefinition(selectedWorkflow.id)
                    if (reloadResponse.code === 200) {
                      setSelectedWorkflow(reloadResponse.data)
                    }
                  }
                }
              } catch (error: any) {
                setError(error.msg || error.message || '保存工作流失败')
                showAlert(error.msg || error.message || '保存工作流时发生错误', 'error', '保存失败')
                console.error('Failed to save workflow:', error)
              } finally {
                setSaving(false)
              }
            }}
            onStop={async (instanceId: number) => {
              try {
                const response = await workflowService.stopInstance(instanceId)

                if (response.code === 200) {
                  setIsRunning(false)
                  setCurrentInstanceId(null)
                  const now = new Date()
                  const timestamp = `${now.getHours().toString().padStart(2, '0')}:${now.getMinutes().toString().padStart(2, '0')}:${now.getSeconds().toString().padStart(2, '0')}.${now.getMilliseconds().toString().padStart(3, '0')}`
                  setTerminalLogs(prev => [...prev, {
                    timestamp,
                    level: 'warning',
                    message: '工作流已被用户停止'
                  }])
                  showAlert('工作流已停止', 'success', '停止成功')
                } else {
                  console.error('Failed to stop workflow:', response.msg)
                  showAlert(response.msg || '停止工作流失败', 'error', '停止失败')
                }
              } catch (error: any) {
                console.error('Error stopping workflow:', error)
                showAlert(error.msg || error.message || '停止工作流时发生错误', 'error', '停止失败')
              }
            }}
            isRunning={isRunning}
            currentInstanceId={currentInstanceId}
            onRun={async (workflow, parameters = {}) => {
              if (!selectedWorkflow) return
              await runWorkflow(selectedWorkflow.id, workflow.name, parameters)
            }}
          />
        </div>

        {/* 右侧：属性面板 */}
        <AnimatePresence>
          {showProperties && (
            <motion.div
              initial={{ width: 0, opacity: 0 }}
              animate={{ width: 320, opacity: 1 }}
              exit={{ width: 0, opacity: 0 }}
              transition={{ duration: 0.2 }}
              className="border-l border-gray-200 dark:border-gray-800 bg-white dark:bg-gray-800 overflow-hidden flex flex-col shrink-0 min-h-0"
            >
              <div className="p-3 border-b border-gray-200 dark:border-gray-700">
                <div className="flex items-center justify-between mb-2">
                  <Typography.Text bold>{t('workflow.properties')}</Typography.Text>
                  <Button type="text" size="mini" icon={<X className="h-4 w-4" />} onClick={() => setShowProperties(false)} />
                </div>
                <Tabs
                  activeTab={propertyTab}
                  onChange={(key) => setPropertyTab(key as 'basic' | 'params')}
                  size="small"
                  className="workflow-property-tabs"
                >
                  <Tabs.TabPane key="basic" title={t('workflow.basicInfo')}>
                    <div className="flex-1 overflow-y-auto p-4 space-y-4 max-h-[calc(100vh-220px)]">
                      <div>
                        <Typography.Text type="secondary" className="!text-xs block">
                          「保存」仅更新当前草稿；需要固定快照时请点顶部「{t('common.publishVersion')}」。
                        </Typography.Text>
                      </div>
                      <div>
                        <Typography.Text className="!text-xs">{t('workflow.name')}</Typography.Text>
                        <Input value={selectedWorkflow.name} readOnly />
                      </div>
                      <div>
                        <Typography.Text className="!text-xs">{t('common.description')}</Typography.Text>
                        <Input.TextArea rows={3} value={selectedWorkflow.description} readOnly />
                      </div>
                      <div>
                        <Typography.Text className="!text-xs">{t('common.status')}</Typography.Text>
                        <Select
                          value={workflowStatus}
                          onChange={(v) => setWorkflowStatus(v as WorkflowStatus)}
                          options={[
                            { value: 'draft', label: t('workflow.status.draft') },
                            { value: 'active', label: t('workflow.status.active') },
                            { value: 'archived', label: t('workflow.status.archived') },
                          ]}
                          style={{ width: '100%' }}
                        />
                        <Typography.Text type="secondary" className="!text-xs block mt-1">
                          公开 API 触发需要「激活」状态，修改后请点击画布工具栏「保存」。
                        </Typography.Text>
                      </div>
                      <div>
                        <Typography.Text className="!text-xs">{t('workflow.tags')}</Typography.Text>
                        <div className="flex flex-wrap gap-2 mt-1">
                          {selectedWorkflow.tags?.length ? (
                            selectedWorkflow.tags.map((tag, idx) => (
                              <Badge key={idx} variant="outline" size="xs">{tag}</Badge>
                            ))
                          ) : (
                            <Typography.Text type="secondary" className="!text-xs">—</Typography.Text>
                          )}
                        </div>
                      </div>
                      <div className="pt-4 border-t border-gray-200 dark:border-gray-700 text-xs text-gray-500 dark:text-gray-400 space-y-1">
                        <div className="flex justify-between">
                          <span>创建时间</span>
                          <span>{formatDate(selectedWorkflow.createdAt)}</span>
                        </div>
                        <div className="flex justify-between">
                          <span>更新时间</span>
                          <span>{formatDate(selectedWorkflow.updatedAt)}</span>
                        </div>
                        <div className="flex justify-between">
                          <span>创建者</span>
                          <span>{selectedWorkflow.createdBy}</span>
                        </div>
                      </div>
                    </div>
                  </Tabs.TabPane>
                  <Tabs.TabPane key="params" title={t('workflow.paramDefs')}>
                    <div className="flex-1 overflow-y-auto p-4 space-y-4 max-h-[calc(100vh-220px)]">
                      <div>
                        <div className="flex items-center justify-between mb-3">
                          <Typography.Text bold className="!text-sm">输入参数</Typography.Text>
                          <Button
                            type="outline"
                            size="xs"
                            onClick={() => {
                              setInputParameters([
                                ...inputParameters,
                                { name: '', type: 'string', required: false, description: '' },
                              ])
                            }}
                          >
                            添加
                          </Button>
                        </div>
                        <div className="space-y-2 max-h-48 overflow-y-auto">
                          {inputParameters.map((param, index) => (
                            <div key={index} className="p-3 border border-gray-200 dark:border-gray-700 rounded-lg">
                              <div className="grid grid-cols-2 gap-2 mb-2">
                                <Input placeholder="参数名" value={param.name} onChange={(val) => {
                                  const newParams = [...inputParameters]
                                  newParams[index] = { ...param, name: val }
                                  setInputParameters(newParams)
                                }} />
                                <Select
                                  value={param.type}
                                  onChange={(val) => {
                                    const newParams = [...inputParameters]
                                    newParams[index] = { ...param, type: val }
                                    setInputParameters(newParams)
                                  }}
                                  options={[
                                    { value: 'string', label: '字符串' },
                                    { value: 'number', label: '数字' },
                                    { value: 'boolean', label: '布尔值' },
                                    { value: 'object', label: '对象' },
                                    { value: 'array', label: '数组' },
                                  ]}
                                />
                              </div>
                              <Input
                                placeholder="描述"
                                value={param.description}
                                onChange={(val) => {
                                  const newParams = [...inputParameters]
                                  newParams[index] = { ...param, description: val }
                                  setInputParameters(newParams)
                                }}
                              />
                              <div className="flex items-center justify-between mt-2">
                                <label className="flex items-center text-xs gap-2">
                                  <Switch
                                    size="small"
                                    checked={param.required}
                                    onChange={(checked) => {
                                      const newParams = [...inputParameters]
                                      newParams[index] = { ...param, required: checked }
                                      setInputParameters(newParams)
                                    }}
                                  />
                                  <span>必需</span>
                                </label>
                                <Button type="outline" size="xs" onClick={() => setInputParameters(inputParameters.filter((_, i) => i !== index))}>
                                  删除
                                </Button>
                              </div>
                            </div>
                          ))}
                        </div>
                      </div>
                      <div className="pt-4 border-t border-gray-200 dark:border-gray-700">
                        <div className="flex items-center justify-between mb-3">
                          <Typography.Text bold className="!text-sm">输出参数</Typography.Text>
                          <Button
                            type="outline"
                            size="xs"
                            onClick={() => {
                              setOutputParameters([
                                ...outputParameters,
                                { name: '', type: 'string', required: false, description: '' },
                              ])
                            }}
                          >
                            添加
                          </Button>
                        </div>
                        <div className="space-y-2 max-h-48 overflow-y-auto">
                          {outputParameters.map((param, index) => (
                            <div key={index} className="p-3 border border-gray-200 dark:border-gray-700 rounded-lg">
                              <div className="grid grid-cols-2 gap-2 mb-2">
                                <Input placeholder="参数名" value={param.name} onChange={(val) => {
                                  const newParams = [...outputParameters]
                                  newParams[index] = { ...param, name: val }
                                  setOutputParameters(newParams)
                                }} />
                                <Select
                                  value={param.type}
                                  onChange={(val) => {
                                    const newParams = [...outputParameters]
                                    newParams[index] = { ...param, type: val }
                                    setOutputParameters(newParams)
                                  }}
                                  options={[
                                    { value: 'string', label: '字符串' },
                                    { value: 'number', label: '数字' },
                                    { value: 'boolean', label: '布尔值' },
                                    { value: 'object', label: '对象' },
                                    { value: 'array', label: '数组' },
                                  ]}
                                />
                              </div>
                              <Input
                                placeholder="描述"
                                value={param.description}
                                onChange={(val) => {
                                  const newParams = [...outputParameters]
                                  newParams[index] = { ...param, description: val }
                                  setOutputParameters(newParams)
                                }}
                              />
                              <div className="flex items-center justify-between mt-2">
                                <label className="flex items-center text-xs gap-2">
                                  <Switch
                                    size="small"
                                    checked={param.required}
                                    onChange={(checked) => {
                                      const newParams = [...outputParameters]
                                      newParams[index] = { ...param, required: checked }
                                      setOutputParameters(newParams)
                                    }}
                                  />
                                  <span>必需</span>
                                </label>
                                <Button type="outline" size="xs" onClick={() => setOutputParameters(outputParameters.filter((_, i) => i !== index))}>
                                  删除
                                </Button>
                              </div>
                            </div>
                          ))}
                        </div>
                      </div>
                    </div>
                  </Tabs.TabPane>
                </Tabs>
              </div>
            </motion.div>
          )}
        </AnimatePresence>
        </div>
        )}
      </div>
      
      {/* 终端组件 */}
      <Terminal
        logs={terminalLogs}
        isVisible={isTerminalVisible}
        onClose={() => setIsTerminalVisible(false)}
        onClear={() => setTerminalLogs([])}
        leftOffset={effectiveSidebarWidth}
        title={t('workflow.editor.terminalTitle')}
        waitingText={t('workflow.editor.terminalWaiting')}
      />

      {/* 触发器配置模态框 */}
      <Modal
        isOpen={showTriggerConfig}
        onClose={() => setShowTriggerConfig(false)}
        title="触发器配置"
        size="xl"
      >
        <TriggerConfigPanel
          workflow={selectedWorkflow}
          triggerConfig={triggerConfig}
          onUpdate={(config) => setTriggerConfig(config)}
          onCancel={() => setShowTriggerConfig(false)}
          onSave={async () => {
            if (!selectedWorkflow) return
            
            setSaving(true)
            try {
              const updateData: UpdateWorkflowDefinitionRequest = {
                triggers: triggerConfig,
                version: selectedWorkflow.version,
              }
              
              const response = await workflowService.updateDefinition(selectedWorkflow.id, updateData)
              
              if (response.code === 200) {
                setSelectedWorkflow(response.data)
                setShowTriggerConfig(false)
                
                showAlert('触发器配置已保存', 'success', '保存成功')
                if (
                  triggerConfig?.api?.enabled &&
                  triggerConfig?.api?.public &&
                  response.data.status !== 'active'
                ) {
                  showAlert('公开 API 仅对「激活」状态的工作流生效，请在属性面板将状态改为激活并保存工作流。', 'warning')
                }
              } else {
                showAlert(response.msg || '保存触发器配置失败', 'error', '保存失败')
              }
            } catch (error: any) {
              showAlert(error.msg || error.message || '保存触发器配置时发生错误', 'error', '保存失败')
            } finally {
              setSaving(false)
            }
          }}
          saving={saving}
        />
      </Modal>

      <Modal
        isOpen={showPublishModal}
        onClose={() => setShowPublishModal(false)}
        title={t('common.publishVersion')}
        size="md"
      >
        <div className="space-y-4">
          <Typography.Text type="secondary" className="!text-sm block">
            发布会将当前草稿冻结为不可变版本，供版本历史查看与回滚。「保存」不会创建版本记录。
          </Typography.Text>
          <div>
            <Typography.Text className="!text-xs">{t('workflow.changeNote')}（可选）</Typography.Text>
            <Input.TextArea
              rows={3}
              placeholder={t('workflow.editor.changeNotePlaceholder')}
              value={changeNote}
              onChange={setChangeNote}
            />
          </div>
          <div className="flex justify-end gap-2">
            <Button type="outline" onClick={() => setShowPublishModal(false)}>
              {t('common.cancel')}
            </Button>
            <Button type="primary" loading={publishing} onClick={() => void publishWorkflow()}>
              {t('common.publishVersion')}
            </Button>
          </div>
        </div>
      </Modal>

      {/* 版本历史模态框 */}
      <Modal
        isOpen={showVersionHistory}
        onClose={() => {
          setShowVersionHistory(false)
          setVersions([])
        }}
        title="版本历史"
        size="xl"
      >
        <div className="space-y-4">
          {loadingVersions ? (
            <div className="text-center py-8 text-gray-500 dark:text-gray-400">加载中...</div>
          ) : versions.length === 0 ? (
            <div className="text-center py-8 text-gray-500 dark:text-gray-400">暂无版本历史</div>
          ) : (
            <div className="space-y-2 max-h-96 overflow-y-auto">
              {(selectedVersion1 || selectedVersion2) && (
                <Typography.Text type="secondary" className="!text-xs block mb-2">
                  {t('workflow.versionCompareHint')}
                  {selectedVersion1 ? ` v#${selectedVersion1}` : ''}
                  {selectedVersion2 ? ` vs v#${selectedVersion2}` : ''}
                </Typography.Text>
              )}
              {versions.map((version) => {
                const selected = selectedVersion1 === version.id || selectedVersion2 === version.id
                return (
                <div
                  key={version.id}
                  className={`p-4 border rounded-lg transition-colors ${selected ? 'border-[rgb(var(--primary-6))] bg-[var(--color-primary-light-1)]' : 'border-gray-200 dark:border-gray-700 hover:bg-gray-50 dark:hover:bg-gray-800'}`}
                >
                  <div className="flex items-start justify-between">
                    <div className="flex-1">
                      <div className="flex items-center gap-2 mb-2">
                        <Tag size="small">v{version.version}</Tag>
                        {version.changeNote && (
                          <Typography.Text type="secondary" className="!text-xs">
                            {version.changeNote}
                          </Typography.Text>
                        )}
                      </div>
                      <Typography.Text type="secondary" className="!text-xs block">
                        {version.updatedBy} · {new Date(version.createdAt).toLocaleString()}
                      </Typography.Text>
                    </div>
                    <div className="flex items-center gap-2">
                      <Button
                        type="text"
                        size="mini"
                        onClick={() => {
                          if (selectedVersion1 === version.id) {
                            setSelectedVersion1(null)
                            return
                          }
                          if (selectedVersion2 === version.id) {
                            setSelectedVersion2(null)
                            return
                          }
                          if (selectedVersion1 === null) {
                            setSelectedVersion1(version.id)
                          } else if (selectedVersion2 === null && selectedVersion1 !== version.id) {
                            setSelectedVersion2(version.id)
                          } else {
                            setSelectedVersion1(version.id)
                            setSelectedVersion2(null)
                          }
                        }}
                      >
                        {selected ? t('workflow.versionDeselect') : t('workflow.versionCompare')}
                      </Button>
                      <Button
                        type="text"
                        size="mini"
                        onClick={() => {
                          if (selectedWorkflow) {
                            void handleRollback(selectedWorkflow.id, version.id)
                          }
                        }}
                      >
                        {t('workflow.versionRollback')}
                      </Button>
                    </div>
                  </div>
                </div>
              )})}
            </div>
          )}
          {selectedVersion1 && selectedVersion2 && selectedWorkflow ? (
            <div className="flex justify-end pt-2 border-t border-gray-200 dark:border-gray-700">
              <Button
                type="primary"
                size="sm"
                onClick={() => void handleCompareVersions(selectedWorkflow.id, selectedVersion1, selectedVersion2)}
              >
                {t('workflow.versionCompareAction')}
              </Button>
            </div>
          ) : null}
        </div>
      </Modal>

      {/* 版本对比模态框 */}
      <Modal
        isOpen={showVersionCompare}
        onClose={() => {
          setShowVersionCompare(false)
          setCompareData(null)
          setSelectedVersion1(null)
          setSelectedVersion2(null)
        }}
        title="版本对比"
        size="xl"
      >
        {compareData && (
          <div className="space-y-4">
            <div className="grid grid-cols-2 gap-4 mb-4">
              <div className="p-3 bg-blue-50 dark:bg-blue-900/20 rounded-lg">
                <div className="text-sm font-semibold text-blue-900 dark:text-blue-300 mb-1">
                  版本 {compareData.version1.version}
                </div>
                <div className="text-xs text-blue-700 dark:text-blue-400">
                  {new Date(compareData.version1.createdAt).toLocaleString('zh-CN')}
                </div>
              </div>
              <div className="p-3 bg-green-50 dark:bg-green-900/20 rounded-lg">
                <div className="text-sm font-semibold text-green-900 dark:text-green-300 mb-1">
                  版本 {compareData.version2.version}
                </div>
                <div className="text-xs text-green-700 dark:text-green-400">
                  {new Date(compareData.version2.createdAt).toLocaleString('zh-CN')}
                </div>
              </div>
            </div>

            <div className="grid grid-cols-2 gap-3 text-sm">
              <div className="rounded-lg border border-gray-200 p-3 dark:border-gray-700">
                <div className="text-xs text-gray-500">节点数量</div>
                <div className="mt-1 font-medium">
                  {compareData.version1.definition.nodes.length}
                  <span className="mx-2 text-gray-400">→</span>
                  {compareData.version2.definition.nodes.length}
                </div>
              </div>
              <div className="rounded-lg border border-gray-200 p-3 dark:border-gray-700">
                <div className="text-xs text-gray-500">连接数量</div>
                <div className="mt-1 font-medium">
                  {compareData.version1.definition.edges.length}
                  <span className="mx-2 text-gray-400">→</span>
                  {compareData.version2.definition.edges.length}
                </div>
              </div>
            </div>

            <div className="space-y-4 max-h-96 overflow-y-auto">
              {Object.keys(compareData.diff).length === 0 ? (
                <Empty preset="no-data" description={t('workflow.compareNoDiff')} />
              ) : null}
              {compareData.diff.name && (
                <div className="p-3 border border-gray-200 dark:border-gray-700 rounded-lg">
                  <div className="text-sm font-semibold mb-2">名称变更</div>
                  <div className="text-sm">
                    <div className="text-red-600 dark:text-red-400">- {compareData.diff.name.old}</div>
                    <div className="text-green-600 dark:text-green-400">+ {compareData.diff.name.new}</div>
                  </div>
                </div>
              )}

              {compareData.diff.description && (
                <div className="p-3 border border-gray-200 dark:border-gray-700 rounded-lg">
                  <div className="text-sm font-semibold mb-2">描述变更</div>
                  <div className="text-sm">
                    <div className="text-red-600 dark:text-red-400">- {compareData.diff.description.old}</div>
                    <div className="text-green-600 dark:text-green-400">+ {compareData.diff.description.new}</div>
                  </div>
                </div>
              )}

              {compareData.diff.nodes && (
                <div className="p-3 border border-gray-200 dark:border-gray-700 rounded-lg">
                  <div className="text-sm font-semibold mb-2">节点变更</div>
                  {compareData.diff.nodes.added && compareData.diff.nodes.added.length > 0 && (
                    <div className="mb-2">
                      <div className="text-xs font-medium text-green-600 dark:text-green-400 mb-1">新增节点:</div>
                      {compareData.diff.nodes.added.map((node) => (
                        <div key={node.id} className="text-sm text-green-600 dark:text-green-400 ml-2">
                          + {node.name} ({node.type})
                        </div>
                      ))}
                    </div>
                  )}
                  {compareData.diff.nodes.removed && compareData.diff.nodes.removed.length > 0 && (
                    <div className="mb-2">
                      <div className="text-xs font-medium text-red-600 dark:text-red-400 mb-1">删除节点:</div>
                      {compareData.diff.nodes.removed.map((node) => (
                        <div key={node.id} className="text-sm text-red-600 dark:text-red-400 ml-2">
                          - {node.name} ({node.type})
                        </div>
                      ))}
                    </div>
                  )}
                  {compareData.diff.nodes.modified && compareData.diff.nodes.modified.length > 0 && (
                    <div>
                      <div className="text-xs font-medium text-yellow-600 dark:text-yellow-400 mb-1">修改节点:</div>
                      {compareData.diff.nodes.modified.map((item) => (
                        <div key={item.id} className="text-sm ml-2">
                          <div className="text-yellow-600 dark:text-yellow-400">
                            ~ {item.old.name} → {item.new.name}
                          </div>
                        </div>
                      ))}
                    </div>
                  )}
                </div>
              )}

              {compareData.diff.edges && (
                <div className="p-3 border border-gray-200 dark:border-gray-700 rounded-lg">
                  <div className="text-sm font-semibold mb-2">边变更</div>
                  {compareData.diff.edges.added && compareData.diff.edges.added.length > 0 && (
                    <div className="mb-2">
                      <div className="text-xs font-medium text-green-600 dark:text-green-400 mb-1">新增边:</div>
                      {compareData.diff.edges.added.map((edge) => (
                        <div key={edge.id} className="text-sm text-green-600 dark:text-green-400 ml-2">
                          + {edge.source} → {edge.target}
                        </div>
                      ))}
                    </div>
                  )}
                  {compareData.diff.edges.removed && compareData.diff.edges.removed.length > 0 && (
                    <div className="mb-2">
                      <div className="text-xs font-medium text-red-600 dark:text-red-400 mb-1">删除边:</div>
                      {compareData.diff.edges.removed.map((edge) => (
                        <div key={edge.id} className="text-sm text-red-600 dark:text-red-400 ml-2">
                          - {edge.source} → {edge.target}
                        </div>
                      ))}
                    </div>
                  )}
                  {compareData.diff.edges.modified && compareData.diff.edges.modified.length > 0 && (
                    <div>
                      <div className="text-xs font-medium text-yellow-600 dark:text-yellow-400 mb-1">修改边:</div>
                      {compareData.diff.edges.modified.map((item) => (
                        <div key={item.id} className="text-sm ml-2 text-yellow-600 dark:text-yellow-400">
                          ~ {item.old.source} → {item.old.target} → {item.new.source} → {item.new.target}
                        </div>
                      ))}
                    </div>
                  )}
                </div>
              )}
              {compareData.diff.status ? (
                <div className="rounded-lg border border-gray-200 p-3 text-sm dark:border-gray-700">
                  <div className="mb-2 font-semibold">状态变更</div>
                  <div className="text-red-600">- {compareData.diff.status.old}</div>
                  <div className="text-green-600">+ {compareData.diff.status.new}</div>
                </div>
              ) : null}
              {compareData.diff.settings ? (
                <div className="rounded-lg border border-gray-200 p-3 dark:border-gray-700">
                  <div className="mb-2 text-sm font-semibold">设置变更</div>
                  <div className="grid grid-cols-2 gap-2">
                    <Input.TextArea rows={6} readOnly value={JSON.stringify(compareData.diff.settings.old, null, 2)} className="font-mono !text-xs" />
                    <Input.TextArea rows={6} readOnly value={JSON.stringify(compareData.diff.settings.new, null, 2)} className="font-mono !text-xs" />
                  </div>
                </div>
              ) : null}
              {compareData.diff.triggers ? (
                <div className="rounded-lg border border-gray-200 p-3 dark:border-gray-700">
                  <div className="mb-2 text-sm font-semibold">触发器变更</div>
                  <div className="grid grid-cols-2 gap-2">
                    <Input.TextArea rows={6} readOnly value={JSON.stringify(compareData.diff.triggers.old, null, 2)} className="font-mono !text-xs" />
                    <Input.TextArea rows={6} readOnly value={JSON.stringify(compareData.diff.triggers.new, null, 2)} className="font-mono !text-xs" />
                  </div>
                </div>
              ) : null}
              <div className="rounded-lg border border-gray-200 p-3 dark:border-gray-700">
                <div className="mb-2 text-sm font-semibold">完整流程定义</div>
                <div className="mb-1 grid grid-cols-2 gap-2 text-xs text-gray-500">
                  <span>版本 {compareData.version1.version}</span>
                  <span>版本 {compareData.version2.version}</span>
                </div>
                <div className="grid grid-cols-2 gap-2">
                  <Input.TextArea rows={10} readOnly value={JSON.stringify(compareData.version1.definition, null, 2)} className="font-mono !text-xs" />
                  <Input.TextArea rows={10} readOnly value={JSON.stringify(compareData.version2.definition, null, 2)} className="font-mono !text-xs" />
                </div>
              </div>
            </div>
          </div>
        )}
      </Modal>
    </div>
    </BaseLayout>
  )
}
