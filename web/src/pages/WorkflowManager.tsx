import React, { useState, useEffect } from 'react'
import { useNavigate } from 'react-router-dom'
import { Search } from 'lucide-react'
import {
  Card as ArcoCard,
  Input,
  Modal as ArcoModal,
  Select,
  Spin,
  Tag,
  Typography,
} from '@arco-design/web-react'
import { IconDelete, IconEdit, IconPlus, IconRefresh } from '@arco-design/web-react/icon'
import Badge from '@/components/workflow/ui/Badge'
import { Button, Empty } from '@/components/ui'
import { WorkflowForm } from '@/components/workflow/forms/WorkflowForm'
import { showAlert } from '@/utils/notification'
import workflowService, {
  WorkflowDefinition,
  WorkflowGraph,
  CreateWorkflowDefinitionRequest,
  UpdateWorkflowDefinitionRequest,
} from '@/api/workflow'
import BaseLayout from '@/components/Layout/BaseLayout'
import { useTranslation } from '@/i18n'

type WorkflowStatus = 'draft' | 'active' | 'archived'

const WorkflowManager: React.FC = () => {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const [workflows, setWorkflows] = useState<WorkflowDefinition[]>([])
  const [filteredWorkflows, setFilteredWorkflows] = useState<WorkflowDefinition[]>([])
  const [selectedStatus, setSelectedStatus] = useState<string>('all')
  const [searchTerm, setSearchTerm] = useState('')
  const [isCreateModalOpen, setIsCreateModalOpen] = useState(false)
  const [isEditModalOpen, setIsEditModalOpen] = useState(false)
  const [editingWorkflow, setEditingWorkflow] = useState<WorkflowDefinition | null>(null)
  const [viewMode, setViewMode] = useState<'grid' | 'list'>('grid')
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [saving, setSaving] = useState(false)

  useEffect(() => {
    void loadWorkflows()
  }, [selectedStatus, searchTerm])

  const loadWorkflows = async () => {
    setLoading(true)
    setError(null)
    try {
      const params: { status?: WorkflowStatus; keyword?: string } = {}
      if (selectedStatus !== 'all') params.status = selectedStatus as WorkflowStatus
      if (searchTerm) params.keyword = searchTerm

      const response = await workflowService.listDefinitions(params)
      if (response.code === 200) {
        setWorkflows(Array.isArray(response.data) ? response.data : [])
      } else {
        setError(response.msg || t('workflow.messages.loadFailed'))
      }
    } catch (err: unknown) {
      setError((err as { msg?: string }).msg || t('workflow.messages.loadFailed'))
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    let filtered = Array.isArray(workflows) ? workflows : []
    if (selectedStatus !== 'all') {
      filtered = filtered.filter((w) => w.status === selectedStatus)
    }
    if (searchTerm) {
      const term = searchTerm.toLowerCase()
      filtered = filtered.filter(
        (w) =>
          w.name.toLowerCase().includes(term) ||
          w.slug.toLowerCase().includes(term) ||
          (w.description || '').toLowerCase().includes(term) ||
          w.tags?.some((tag) => tag.toLowerCase().includes(term)),
      )
    }
    setFilteredWorkflows(filtered)
  }, [workflows, selectedStatus, searchTerm])

  const openEditor = (workflow: WorkflowDefinition) => {
    navigate(`/workflows/${workflow.id}`)
  }

  const handleDelete = (id: number) => {
    ArcoModal.confirm({
      title: t('common.confirmDelete'),
      content: t('workflow.messages.deleteConfirm'),
      onOk: async () => {
        setLoading(true)
        try {
          const response = await workflowService.deleteDefinition(id)
          if (response.code === 200) {
            setWorkflows((prev) => prev.filter((w) => w.id !== id))
            showAlert(t('workflow.messages.deleteSuccess'), 'success')
          } else {
            showAlert(response.msg || t('workflow.messages.deleteFailed'), 'error')
          }
        } catch (err: unknown) {
          showAlert((err as { msg?: string }).msg || t('workflow.messages.deleteFailed'), 'error')
        } finally {
          setLoading(false)
        }
      },
    })
  }

  const handleSave = async (workflowData: Partial<WorkflowDefinition>) => {
    setSaving(true)
    setError(null)
    try {
      if (editingWorkflow) {
        const updateData: UpdateWorkflowDefinitionRequest = {
          name: workflowData.name,
          description: workflowData.description,
          status: workflowData.status,
          definition: workflowData.definition,
          settings: workflowData.settings,
          tags: workflowData.tags,
          version: editingWorkflow.version,
        }
        const response = await workflowService.updateDefinition(editingWorkflow.id, updateData)
        if (response.code === 200) {
          setWorkflows((prev) => prev.map((w) => (w.id === editingWorkflow.id ? response.data : w)))
          setIsEditModalOpen(false)
          showAlert(t('workflow.messages.saveSuccess'), 'success')
        } else {
          setError(response.msg || t('workflow.messages.saveFailed'))
        }
      } else {
        if (!workflowData.name?.trim()) {
          showAlert(t('workflow.messages.nameRequired'), 'error')
          return
        }
        let finalDefinition: WorkflowGraph
        if (!workflowData.definition?.nodes?.length) {
          const timestamp = Date.now()
          const startId = `start-${timestamp}`
          const endId = `end-${timestamp}`
          finalDefinition = {
            nodes: [
              { id: startId, name: t('workflow.nodes.start'), type: 'start', position: { x: 100, y: 100 } },
              { id: endId, name: t('workflow.nodes.end'), type: 'end', position: { x: 300, y: 100 } },
            ],
            edges: [{ id: `e-${timestamp}`, source: startId, target: endId, type: 'default' }],
          }
        } else {
          finalDefinition = workflowData.definition
        }

        const createData: CreateWorkflowDefinitionRequest = {
          name: workflowData.name.trim(),
          description: workflowData.description,
          status: workflowData.status || 'draft',
          definition: finalDefinition,
          settings: workflowData.settings,
          tags: workflowData.tags,
        }

        const response = await workflowService.createDefinition(createData)
        if (response.code === 200) {
          setWorkflows((prev) => [response.data, ...prev])
          setIsCreateModalOpen(false)
          navigate(`/workflows/${response.data.id}`)
        } else {
          setError(response.msg || t('workflow.messages.saveFailed'))
        }
      }
    } catch (err: unknown) {
      setError((err as { msg?: string }).msg || t('workflow.messages.saveFailed'))
    } finally {
      setSaving(false)
      setEditingWorkflow(null)
    }
  }

  const getStatusBadge = (status: WorkflowStatus) => {
    const variants = {
      draft: { variant: 'muted' as const, label: t('workflow.status.draft') },
      active: { variant: 'success' as const, label: t('workflow.status.active') },
      archived: { variant: 'outline' as const, label: t('workflow.status.archived') },
    }
    const config = variants[status]
    return <Badge variant={config.variant}>{config.label}</Badge>
  }

  const formatDate = (dateString: string) =>
    new Date(dateString).toLocaleDateString('zh-CN', { year: 'numeric', month: 'short', day: 'numeric' })

  const statusOptions = [
    { value: 'all', label: `${t('workflow.filter.all')} (${workflows.length})` },
    { value: 'draft', label: `${t('workflow.status.draft')} (${workflows.filter((w) => w.status === 'draft').length})` },
    { value: 'active', label: `${t('workflow.status.active')} (${workflows.filter((w) => w.status === 'active').length})` },
    { value: 'archived', label: `${t('workflow.status.archived')} (${workflows.filter((w) => w.status === 'archived').length})` },
  ]

  const renderWorkflowCard = (workflow: WorkflowDefinition) => (
    <ArcoCard key={workflow.id} hoverable className="h-full cursor-pointer border border-neutral-100" onClick={() => openEditor(workflow)}>
      <div className="mb-3 flex items-start justify-between gap-2">
        <div className="flex min-w-0 flex-1 items-center gap-3">
          <div className="flex h-10 w-10 shrink-0 items-center justify-center rounded-xl bg-gradient-to-br from-blue-500 to-indigo-600">
            <IconEdit />
          </div>
          <div className="min-w-0">
            <div className="truncate text-sm font-medium text-neutral-900">{workflow.name}</div>
            <div className="mt-0.5 text-xs text-neutral-500">v{workflow.version}</div>
          </div>
        </div>
        {getStatusBadge(workflow.status)}
      </div>
      <p className="mb-3 line-clamp-2 text-xs text-neutral-500">{workflow.description || t('common.noDescription')}</p>
      <div className="mb-3 flex flex-wrap gap-1">
        {workflow.tags?.slice(0, 3).map((tag) => (
          <Tag key={tag} size="small" className="!rounded-full">{tag}</Tag>
        ))}
      </div>
      <div className="flex items-center justify-between text-xs text-neutral-400">
        <span>{formatDate(workflow.updatedAt)}</span>
      </div>
      <div className="mt-3 flex gap-2" onClick={(e) => e.stopPropagation()}>
        <Button type="outline" size="xs" icon={<IconEdit />} onClick={() => { setEditingWorkflow(workflow); setIsEditModalOpen(true) }}>
          {t('common.edit')}
        </Button>
        <Button type="outline" size="xs" status="danger" icon={<IconDelete />} onClick={() => void handleDelete(workflow.id)}>
          {t('common.delete')}
        </Button>
      </div>
    </ArcoCard>
  )

  return (
    <BaseLayout title={t('workflow.title')} description={t('workflow.subtitle')}>
      {error ? (
        <Typography.Text type="error" className="mb-3 block">
          {error}
        </Typography.Text>
      ) : null}

      <div className="mb-4 flex flex-wrap items-center gap-2">
        <Input
          allowClear
          prefix={<Search className="h-4 w-4 text-gray-400" />}
          placeholder={t('workflow.searchPlaceholder')}
          value={searchTerm}
          onChange={setSearchTerm}
          style={{ width: 260 }}
        />
        <Select value={selectedStatus} onChange={setSelectedStatus} options={statusOptions} style={{ width: 160 }} />
        <Select
          value={viewMode}
          onChange={setViewMode}
          options={[
            { value: 'grid', label: t('workflow.viewMode.grid') },
            { value: 'list', label: t('workflow.viewMode.list') },
          ]}
          style={{ width: 120 }}
        />
        <div className="flex-1" />
        <Button type="outline" icon={<IconRefresh />} onClick={() => void loadWorkflows()}>
          {t('common.refresh')}
        </Button>
        <Button type="primary" icon={<IconPlus />} onClick={() => setIsCreateModalOpen(true)}>
          {t('workflow.create')}
        </Button>
      </div>

      {loading && workflows.length === 0 ? (
        <div className="flex justify-center py-16">
          <Spin tip={t('workflow.loading')} />
        </div>
      ) : filteredWorkflows.length === 0 ? (
        <Empty
          preset="no-data"
          description={searchTerm ? t('workflow.noMatch') : t('workflow.noWorkflows')}
        />
      ) : viewMode === 'grid' ? (
        <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4">
          {filteredWorkflows.map(renderWorkflowCard)}
        </div>
      ) : (
        <div className="space-y-2">
          {filteredWorkflows.map((workflow) => (
            <ArcoCard key={workflow.id} hoverable className="cursor-pointer" onClick={() => openEditor(workflow)}>
              <div className="flex items-center justify-between gap-4">
                <div className="min-w-0 flex-1">
                  <Typography.Text bold>{workflow.name}</Typography.Text>
                  <Typography.Text type="secondary" className="!text-xs block truncate">
                    {workflow.description || t('common.noDescription')}
                  </Typography.Text>
                </div>
                {getStatusBadge(workflow.status)}
              </div>
            </ArcoCard>
          ))}
        </div>
      )}

      <ArcoModal
        title={editingWorkflow ? t('workflow.edit') : t('workflow.create')}
        visible={isCreateModalOpen || isEditModalOpen}
        onCancel={() => {
          setIsCreateModalOpen(false)
          setIsEditModalOpen(false)
          setEditingWorkflow(null)
          setError(null)
        }}
        footer={null}
        style={{ width: 560 }}
        unmountOnExit
      >
        <WorkflowForm
          workflow={editingWorkflow}
          onSave={handleSave}
          saving={saving}
          onCancel={() => {
            setIsCreateModalOpen(false)
            setIsEditModalOpen(false)
            setEditingWorkflow(null)
            setError(null)
          }}
        />
      </ArcoModal>
    </BaseLayout>
  )
}

export default WorkflowManager
