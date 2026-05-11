import { useState, useEffect } from 'react'
import { motion } from 'framer-motion'
import {
  Settings,
  Plus,
  Search,
  Edit,
  Trash2,
  RefreshCw,
  Save,
  X,
  CheckCircle2,
  Eye,
  EyeOff,
} from 'lucide-react'
import Card from '@/components/UI/Card'
import Button from '@/components/UI/Button'
import Input from '@/components/UI/Input'
import Badge from '@/components/UI/Badge'
import EmptyState from '@/components/UI/EmptyState'
import Modal from '@/components/UI/Modal'
import ConfirmDialog from '@/components/UI/ConfirmDialog'
import { Select, SelectTrigger, SelectContent, SelectItem, SelectValue } from '@/components/UI/Select'
import { showAlert } from '@/utils/notification'
import {
  listConfigs,
  createConfig,
  updateConfig,
  deleteConfig,
  type Config,
  type ListConfigsParams,
} from '@/services/adminApi'
import { cn } from '@/utils/cn'

const FORMATS = [
  { value: 'text', label: 'Text' },
  { value: 'json', label: 'JSON' },
  { value: 'yaml', label: 'YAML' },
  { value: 'int', label: 'Integer' },
  { value: 'float', label: 'Float' },
  { value: 'bool', label: 'Boolean' },
]

const STORAGE_TYPES = [
  { value: 'qiniu', label: '七牛云' },
  { value: 'cos', label: '腾讯云COS' },
  { value: 'oss', label: '阿里云OSS' },
  { value: 'minio', label: 'MinIO' },
  { value: 'local', label: '本地存储' },
]

const Configs = () => {
  const [configs, setConfigs] = useState<Config[]>([])
  const [total, setTotal] = useState(0)
  const [loading, setLoading] = useState(false)
  const [search, setSearch] = useState('')
  const [autoloadFilter, setAutoloadFilter] = useState<string>('')
  const [publicFilter, setPublicFilter] = useState<string>('')
  const [currentPage, setCurrentPage] = useState(1)
  const [pageSize] = useState(20)
  const [showCreateModal, setShowCreateModal] = useState(false)
  const [showEditModal, setShowEditModal] = useState(false)
  const [showDeleteConfirm, setShowDeleteConfirm] = useState(false)
  const [selectedConfig, setSelectedConfig] = useState<Config | null>(null)
  const [editingConfig, setEditingConfig] = useState<Config | null>(null)
  const [showValue, setShowValue] = useState<{ [key: string]: boolean }>({})

  // Form states
  const [formData, setFormData] = useState({
    key: '',
    desc: '',
    value: '',
    format: 'text' as Config['format'],
    autoload: false,
    public: false,
  })

  // 获取配置列表
  const fetchConfigs = async () => {
    try {
      setLoading(true)
      const params: ListConfigsParams = {
        page: currentPage,
        page_size: pageSize,
      }
      if (search) params.search = search
      if (autoloadFilter === 'true') params.autoload = true
      else if (autoloadFilter === 'false') params.autoload = false
      if (publicFilter === 'true') params.public = true
      else if (publicFilter === 'false') params.public = false

      const response = await listConfigs(params)
      // Normalize value field (backend may return Value with capital V)
      const normalizedConfigs = (response.configs || []).map(config => ({
        ...config,
        value: config.value || (config as any).Value || '',
      }))
      setConfigs(normalizedConfigs)
      setTotal(response.total || 0)
    } catch (error: any) {
      showAlert('获取配置列表失败', 'error', error?.msg || error?.message)
    } finally {
      setLoading(false)
    }
  }

  // Reset to first page when filters change
  useEffect(() => {
    if (currentPage === 1) {
      fetchConfigs()
    } else {
      setCurrentPage(1)
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [search, autoloadFilter, publicFilter])

  // Fetch when page changes
  useEffect(() => {
    fetchConfigs()
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [currentPage])

  // 创建配置
  const handleCreate = () => {
    setFormData({
      key: '',
      desc: '',
      value: '',
      format: 'text',
      autoload: false,
      public: false,
    })
    setShowCreateModal(true)
  }

  const handleCreateSubmit = async () => {
    if (!formData.key.trim()) {
      showAlert('请输入配置键', 'error')
      return
    }

    try {
      setLoading(true)
      await createConfig(formData)
      
      showAlert('创建配置成功', 'success')
      setShowCreateModal(false)
      await fetchConfigs()
    } catch (error: any) {
      showAlert('创建配置失败', 'error', error?.msg || error?.message)
    } finally {
      setLoading(false)
    }
  }

  // 编辑配置
  const handleEdit = (config: Config) => {
    setEditingConfig(config)
    setFormData({
      key: config.key,
      desc: config.desc || '',
      value: config.value || config.Value || '',
      format: config.format || 'text',
      autoload: config.autoload || false,
      public: config.public || false,
    })
    setShowEditModal(true)
  }

  const handleEditSubmit = async () => {
    if (!editingConfig) return

    try {
      setLoading(true)
      await updateConfig(editingConfig.key, {
        desc: formData.desc,
        value: formData.value,
        format: formData.format,
        autoload: formData.autoload,
        public: formData.public,
      })
      
      showAlert('更新配置成功', 'success')
      setShowEditModal(false)
      setEditingConfig(null)
      await fetchConfigs()
    } catch (error: any) {
      showAlert('更新配置失败', 'error', error?.msg || error?.message)
    } finally {
      setLoading(false)
    }
  }

  // 删除配置
  const handleDelete = (config: Config) => {
    setSelectedConfig(config)
    setShowDeleteConfirm(true)
  }

  const confirmDelete = async () => {
    if (!selectedConfig) return

    try {
      setLoading(true)
      await deleteConfig(selectedConfig.key)
      
      showAlert('删除配置成功', 'success')
      setShowDeleteConfirm(false)
      setSelectedConfig(null)
      await fetchConfigs()
    } catch (error: any) {
      showAlert('删除配置失败', 'error', error?.msg || error?.message)
    } finally {
      setLoading(false)
    }
  }

  // 切换显示/隐藏值
  const toggleShowValue = (key: string) => {
    setShowValue((prev) => ({ ...prev, [key]: !prev[key] }))
  }

  // 格式化值显示
  const formatValue = (value: string, format: string) => {
    if (!value) return ''
    if (format === 'json' || format === 'yaml') {
      try {
        return JSON.stringify(JSON.parse(value), null, 2)
      } catch {
        return value
      }
    }
    return value
  }

  const totalPages = Math.ceil(total / pageSize)

  return (
    <>
      <div className="space-y-6">
        {/* 搜索和过滤 */}
        <Card>
          <div className="flex flex-col sm:flex-row gap-4">
            <div className="flex-1 relative">
              <Search className="absolute left-3 top-1/2 transform -translate-y-1/2 w-4 h-4 text-muted-foreground" />
              <Input
                placeholder="搜索配置键或描述..."
                value={search}
                onChange={(e) => setSearch(e.target.value)}
                className="pl-10"
              />
            </div>
            <Select value={autoloadFilter} onValueChange={setAutoloadFilter}>
              <SelectTrigger className="w-full sm:w-[180px]">
                <SelectValue placeholder="自动加载">
                  {autoloadFilter === '' ? '自动加载: 全部' : autoloadFilter === 'true' ? '自动加载: 是' : '自动加载: 否'}
                </SelectValue>
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="">全部</SelectItem>
                <SelectItem value="true">是</SelectItem>
                <SelectItem value="false">否</SelectItem>
              </SelectContent>
            </Select>
            <Select value={publicFilter} onValueChange={setPublicFilter}>
              <SelectTrigger className="w-full sm:w-[180px]">
                <SelectValue placeholder="公开">
                  {publicFilter === '' ? '公开: 全部' : publicFilter === 'true' ? '公开: 是' : '公开: 否'}
                </SelectValue>
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="">全部</SelectItem>
                <SelectItem value="true">是</SelectItem>
                <SelectItem value="false">否</SelectItem>
              </SelectContent>
            </Select>
            <Button
              variant="outline"
              onClick={fetchConfigs}
              disabled={loading}
              leftIcon={<RefreshCw className={cn('w-4 h-4', loading && 'animate-spin')} />}
            >
              刷新
            </Button>
            <Button 
              onClick={handleCreate}
              leftIcon={<Plus className="w-4 h-4" />}
            >
              新建配置
            </Button>
          </div>
        </Card>

        {/* 配置列表 */}
        <Card>
          {loading && configs.length === 0 ? (
            <div className="flex items-center justify-center py-12">
              <RefreshCw className="w-6 h-6 animate-spin text-muted-foreground" />
            </div>
          ) : configs.length === 0 ? (
            <EmptyState
              icon={Settings}
              title="暂无配置"
              description="点击上方按钮创建第一个配置"
            />
          ) : (
            <div className="overflow-x-auto">
              <table className="w-full">
                <thead>
                  <tr className="border-b border-border">
                    <th className="text-left p-4 font-medium text-sm w-[15%]">键</th>
                    <th className="text-left p-4 font-medium text-sm w-[15%]">描述</th>
                    <th className="text-left p-4 font-medium text-sm w-[35%]">值</th>
                    <th className="text-left p-4 font-medium text-sm w-[8%]">格式</th>
                    <th className="text-center p-4 font-medium text-sm w-[8%]">自动加载</th>
                    <th className="text-center p-4 font-medium text-sm w-[8%]">公开</th>
                    <th className="text-right p-4 font-medium text-sm w-[11%]">操作</th>
                  </tr>
                </thead>
                <tbody>
                  {configs.map((config) => (
                    <motion.tr
                      key={config.id}
                      initial={{ opacity: 0, y: 10 }}
                      animate={{ opacity: 1, y: 0 }}
                      className="border-b border-border hover:bg-accent/50 transition-colors"
                    >
                      <td className="p-4">
                        <code className="text-sm font-mono bg-muted px-2 py-1 rounded">
                          {config.key}
                        </code>
                      </td>
                      <td className="p-4 text-sm text-muted-foreground">
                        {config.public ? (config.desc || '-') : '-'}
                      </td>
                      <td className="p-4">
                        {config.public ? (
                          <div className="flex items-center gap-2">
                            <code className="text-xs font-mono bg-muted px-2 py-1 rounded flex-1 min-w-0 break-words">
                              {showValue[config.key]
                                ? formatValue(config.value || config.Value || '', config.format)
                                : '••••••••'}
                            </code>
                            <button
                              onClick={() => toggleShowValue(config.key)}
                              className="text-muted-foreground hover:text-foreground transition-colors flex-shrink-0"
                            >
                              {showValue[config.key] ? (
                                <EyeOff className="w-4 h-4" />
                              ) : (
                                <Eye className="w-4 h-4" />
                              )}
                            </button>
                          </div>
                        ) : (
                          <span className="text-muted-foreground text-sm">-</span>
                        )}
                      </td>
                      <td className="p-4">
                        <Badge variant="outline">{config.format || 'text'}</Badge>
                      </td>
                      <td className="p-4 text-center">
                        {config.autoload ? (
                          <CheckCircle2 className="w-5 h-5 text-green-500 mx-auto" />
                        ) : (
                          <X className="w-5 h-5 text-muted-foreground mx-auto" />
                        )}
                      </td>
                      <td className="p-4 text-center">
                        {config.public ? (
                          <CheckCircle2 className="w-5 h-5 text-green-500 mx-auto" />
                        ) : (
                          <X className="w-5 h-5 text-muted-foreground mx-auto" />
                        )}
                      </td>
                      <td className="p-4">
                        <div className="flex items-center justify-end gap-2">
                          <Button
                            variant="ghost"
                            size="sm"
                            onClick={() => handleEdit(config)}
                            leftIcon={<Edit className="w-4 h-4" />}
                          >
                            编辑
                          </Button>
                          <Button
                            variant="ghost"
                            size="sm"
                            onClick={() => handleDelete(config)}
                            className="text-destructive hover:text-destructive"
                            leftIcon={<Trash2 className="w-4 h-4" />}
                          >
                            删除
                          </Button>
                        </div>
                      </td>
                    </motion.tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}

          {/* 分页 */}
          {totalPages > 1 && (
            <div className="flex items-center justify-between mt-4 pt-4 border-t border-border">
              <div className="text-sm text-muted-foreground">
                共 {total} 条，第 {currentPage} / {totalPages} 页
              </div>
              <div className="flex gap-2">
                <Button
                  variant="outline"
                  size="sm"
                  onClick={() => setCurrentPage((p) => Math.max(1, p - 1))}
                  disabled={currentPage === 1}
                >
                  上一页
                </Button>
                <Button
                  variant="outline"
                  size="sm"
                  onClick={() => setCurrentPage((p) => Math.min(totalPages, p + 1))}
                  disabled={currentPage === totalPages}
                >
                  下一页
                </Button>
              </div>
            </div>
          )}
        </Card>

        {/* 创建配置模态框 */}
        <Modal
          isOpen={showCreateModal}
          onClose={() => setShowCreateModal(false)}
          title="创建配置"
          size="lg"
        >
          <div className="space-y-4">
            <div>
              <label className="block text-sm font-medium mb-2">配置键 *</label>
              <Input
                value={formData.key}
                onChange={(e) => setFormData({ ...formData, key: e.target.value.toUpperCase() })}
                placeholder="例如: KEY_SITE_NAME"
              />
            </div>
            <div>
              <label className="block text-sm font-medium mb-2">描述</label>
              <Input
                value={formData.desc}
                onChange={(e) => setFormData({ ...formData, desc: e.target.value })}
                placeholder="配置项描述"
              />
            </div>
            <div>
              <label className="block text-sm font-medium mb-2">值 *</label>
              {formData.key === 'STORAGE_KIND' ? (
                <Select
                  value={formData.value}
                  onValueChange={(value) => setFormData({ ...formData, value })}
                >
                  <SelectTrigger>
                    <SelectValue placeholder="选择存储类型" />
                  </SelectTrigger>
                  <SelectContent>
                    {STORAGE_TYPES.map((type) => (
                      <SelectItem key={type.value} value={type.value}>
                        {type.label}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              ) : (
                <textarea
                  value={formData.value}
                  onChange={(e) => setFormData({ ...formData, value: e.target.value })}
                  placeholder="配置值"
                  className="w-full min-h-[100px] px-3 py-2 border border-input rounded-md bg-background text-sm"
                  rows={4}
                />
              )}
            </div>
            <div>
              <label className="block text-sm font-medium mb-2">格式</label>
              <Select
                value={formData.format}
                onValueChange={(value) =>
                  setFormData({ ...formData, format: value as Config['format'] })
                }
              >
                <SelectTrigger>
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  {FORMATS.map((format) => (
                    <SelectItem key={format.value} value={format.value}>
                      {format.label}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
            <div className="flex items-center gap-4">
              <label className="flex items-center gap-2 cursor-pointer">
                <input
                  type="checkbox"
                  checked={formData.autoload}
                  onChange={(e) => setFormData({ ...formData, autoload: e.target.checked })}
                  className="w-4 h-4"
                />
                <span className="text-sm">自动加载</span>
              </label>
              <label className="flex items-center gap-2 cursor-pointer">
                <input
                  type="checkbox"
                  checked={formData.public}
                  onChange={(e) => setFormData({ ...formData, public: e.target.checked })}
                  className="w-4 h-4"
                />
                <span className="text-sm">公开</span>
              </label>
            </div>
            <div className="flex justify-end gap-2 pt-4">
              <Button variant="outline" onClick={() => setShowCreateModal(false)}>
                取消
              </Button>
              <Button 
                onClick={handleCreateSubmit} 
                disabled={loading}
                leftIcon={<Save className="w-4 h-4" />}
              >
                创建
              </Button>
            </div>
          </div>
        </Modal>

        {/* 编辑配置模态框 */}
        <Modal
          isOpen={showEditModal}
          onClose={() => {
            setShowEditModal(false)
            setEditingConfig(null)
          }}
          title="编辑配置"
          size="lg"
        >
          <div className="space-y-4">
            <div>
              <label className="block text-sm font-medium mb-2">配置键</label>
              <Input value={formData.key} disabled className="bg-muted" />
            </div>
            <div>
              <label className="block text-sm font-medium mb-2">描述</label>
              <Input
                value={formData.desc}
                onChange={(e) => setFormData({ ...formData, desc: e.target.value })}
                placeholder="配置项描述"
              />
            </div>
            <div>
              <label className="block text-sm font-medium mb-2">值 *</label>
              {formData.key === 'STORAGE_KIND' ? (
                <Select
                  value={formData.value}
                  onValueChange={(value) => setFormData({ ...formData, value })}
                >
                  <SelectTrigger>
                    <SelectValue placeholder="选择存储类型" />
                  </SelectTrigger>
                  <SelectContent>
                    {STORAGE_TYPES.map((type) => (
                      <SelectItem key={type.value} value={type.value}>
                        {type.label}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              ) : (
                <textarea
                  value={formData.value}
                  onChange={(e) => setFormData({ ...formData, value: e.target.value })}
                  placeholder="配置值"
                  className="w-full min-h-[100px] px-3 py-2 border border-input rounded-md bg-background text-sm"
                  rows={4}
                />
              )}
            </div>
            <div>
              <label className="block text-sm font-medium mb-2">格式</label>
              <Select
                value={formData.format}
                onValueChange={(value) =>
                  setFormData({ ...formData, format: value as Config['format'] })
                }
              >
                <SelectTrigger>
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  {FORMATS.map((format) => (
                    <SelectItem key={format.value} value={format.value}>
                      {format.label}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
            <div className="flex items-center gap-4">
              <label className="flex items-center gap-2 cursor-pointer">
                <input
                  type="checkbox"
                  checked={formData.autoload}
                  onChange={(e) => setFormData({ ...formData, autoload: e.target.checked })}
                  className="w-4 h-4"
                />
                <span className="text-sm">自动加载</span>
              </label>
              <label className="flex items-center gap-2 cursor-pointer">
                <input
                  type="checkbox"
                  checked={formData.public}
                  onChange={(e) => setFormData({ ...formData, public: e.target.checked })}
                  className="w-4 h-4"
                />
                <span className="text-sm">公开</span>
              </label>
            </div>
            <div className="flex justify-end gap-2 pt-4">
              <Button variant="outline" onClick={() => {
                setShowEditModal(false)
                setEditingConfig(null)
              }}>
                取消
              </Button>
              <Button 
                onClick={handleEditSubmit} 
                disabled={loading}
                leftIcon={<Save className="w-4 h-4" />}
              >
                保存
              </Button>
            </div>
          </div>
        </Modal>

        {/* 删除确认对话框 */}
        <ConfirmDialog
          isOpen={showDeleteConfirm}
          onClose={() => {
            setShowDeleteConfirm(false)
            setSelectedConfig(null)
          }}
          onConfirm={confirmDelete}
          title="删除配置"
          message={`确定要删除配置 "${selectedConfig?.key}" 吗？此操作不可恢复。`}
          confirmText="删除"
          cancelText="取消"
          variant="danger"
          loading={loading}
        />
      </div>
    </>
  )
}

export default Configs