import { useState, useEffect } from 'react'
import { motion } from 'framer-motion'
import { 
  Plus, 
  Settings, 
  Trash2, 
  Edit, 
  Power, 
  PowerOff,
  Phone,
  Mic,
  MessageSquare,
  BarChart3
} from 'lucide-react'
import Button from '@/components/UI/Button'
import Modal from '@/components/UI/Modal'
import { showAlert } from '@/utils/notification'
import { 
  getSchemes, 
  deleteScheme, 
  activateScheme,
  deactivateScheme 
} from '@/api/scheme'
import type { Scheme } from '@/types/scheme'
import SchemeForm from '@/components/Scheme/SchemeForm'
import SchemeCard from '@/components/Scheme/SchemeCard'

const SchemeManager = () => {
  const [schemes, setSchemes] = useState<Scheme[]>([])
  const [loading, setLoading] = useState(false)
  const [showForm, setShowForm] = useState(false)
  const [editingScheme, setEditingScheme] = useState<Scheme | null>(null)
  const [deleteConfirm, setDeleteConfirm] = useState<number | null>(null)

  useEffect(() => {
    loadSchemes()
  }, [])

  const loadSchemes = async () => {
    try {
      setLoading(true)
      const res = await getSchemes()
      if (res.code === 200 && res.data) {
        setSchemes(res.data)
      }
    } catch (error: any) {
      showAlert(error.msg || '加载方案失败', 'error')
    } finally {
      setLoading(false)
    }
  }

  const handleCreate = () => {
    setEditingScheme(null)
    setShowForm(true)
  }

  const handleEdit = (scheme: Scheme) => {
    setEditingScheme(scheme)
    setShowForm(true)
  }

  const handleDelete = async (id: number) => {
    try {
      const res = await deleteScheme(id)
      if (res.code === 200) {
        showAlert('删除成功', 'success')
        loadSchemes()
      } else {
        throw new Error(res.msg)
      }
    } catch (error: any) {
      showAlert(error.msg || '删除失败', 'error')
    } finally {
      setDeleteConfirm(null)
    }
  }

  const handleToggleActive = async (scheme: Scheme) => {
    try {
      const res = scheme.isActive 
        ? await deactivateScheme(scheme.id)
        : await activateScheme(scheme.id)
      
      if (res.code === 200) {
        showAlert(scheme.isActive ? '已停用' : '已激活', 'success')
        loadSchemes()
      } else {
        throw new Error(res.msg)
      }
    } catch (error: any) {
      showAlert(error.msg || '操作失败', 'error')
    }
  }

  const handleFormSuccess = () => {
    setShowForm(false)
    setEditingScheme(null)
    loadSchemes()
  }

  const activeScheme = schemes.find(s => s.isActive)

  return (
    <div className="p-6 max-w-7xl mx-auto">
      {/* Header */}
      <div className="mb-6">
        <div className="flex items-center justify-between mb-2">
          <div>
            <h1 className="text-3xl font-bold text-foreground">
              代接方案管理
            </h1>
            <p className="text-muted-foreground mt-1">
              创建和管理您的AI代接方案，配置个性化的接听策略
            </p>
          </div>
          <Button
            onClick={handleCreate}
            leftIcon={<Plus className="w-4 h-4" />}
            size="lg"
          >
            创建方案
          </Button>
        </div>

        {/* Active Scheme Banner */}
        {activeScheme && (
          <motion.div
            initial={{ opacity: 0, y: -10 }}
            animate={{ opacity: 1, y: 0 }}
            className="mt-4 p-4 bg-primary/10 border border-primary/20 rounded-lg"
          >
            <div className="flex items-center justify-between">
              <div className="flex items-center gap-3">
                <div className="w-10 h-10 rounded-full bg-primary/20 flex items-center justify-center">
                  <Power className="w-5 h-5 text-primary" />
                </div>
                <div>
                  <div className="font-medium text-foreground">
                    当前激活方案
                  </div>
                  <div className="text-sm text-muted-foreground">
                    {activeScheme.name}
                  </div>
                </div>
              </div>
              <div className="flex items-center gap-2">
                <span className="px-3 py-1 bg-green-100 text-green-700 dark:bg-green-900 dark:text-green-300 rounded-full text-sm font-medium">
                  运行中
                </span>
              </div>
            </div>
          </motion.div>
        )}
      </div>

      {/* Schemes Grid */}
      {loading ? (
        <div className="text-center py-12">
          <div className="inline-block w-8 h-8 border-4 border-primary border-t-transparent rounded-full animate-spin" />
          <p className="mt-4 text-muted-foreground">加载中...</p>
        </div>
      ) : schemes.length === 0 ? (
        <motion.div
          initial={{ opacity: 0, scale: 0.95 }}
          animate={{ opacity: 1, scale: 1 }}
          className="text-center py-12 bg-card border border-border rounded-lg"
        >
          <Settings className="w-16 h-16 mx-auto text-muted-foreground mb-4" />
          <h3 className="text-lg font-medium text-foreground mb-2">
            还没有代接方案
          </h3>
          <p className="text-muted-foreground mb-6">
            创建您的第一个AI代接方案，开始智能接听
          </p>
          <Button onClick={handleCreate} leftIcon={<Plus className="w-4 h-4" />}>
            创建方案
          </Button>
        </motion.div>
      ) : (
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
          {schemes.map((scheme, index) => (
            <motion.div
              key={scheme.id}
              initial={{ opacity: 0, y: 20 }}
              animate={{ opacity: 1, y: 0 }}
              transition={{ delay: index * 0.1 }}
            >
              <SchemeCard
                scheme={scheme}
                onEdit={() => handleEdit(scheme)}
                onDelete={() => setDeleteConfirm(scheme.id)}
                onToggleActive={() => handleToggleActive(scheme)}
              />
            </motion.div>
          ))}
        </div>
      )}

      {/* Create/Edit Form Modal */}
      <Modal
        isOpen={showForm}
        onClose={() => {
          setShowForm(false)
          setEditingScheme(null)
        }}
        title={editingScheme ? '编辑方案' : '创建方案'}
        size="xl"
      >
        <SchemeForm
          scheme={editingScheme}
          onSuccess={handleFormSuccess}
          onCancel={() => {
            setShowForm(false)
            setEditingScheme(null)
          }}
        />
      </Modal>

      {/* Delete Confirmation Modal */}
      <Modal
        isOpen={deleteConfirm !== null}
        onClose={() => setDeleteConfirm(null)}
        title="确认删除"
        size="sm"
      >
        <div className="space-y-4">
          <p className="text-muted-foreground">
            确定要删除这个方案吗？此操作无法撤销。
          </p>
          <div className="flex justify-end gap-3">
            <Button
              variant="ghost"
              onClick={() => setDeleteConfirm(null)}
            >
              取消
            </Button>
            <Button
              variant="destructive"
              onClick={() => deleteConfirm && handleDelete(deleteConfirm)}
            >
              删除
            </Button>
          </div>
        </div>
      </Modal>
    </div>
  )
}

export default SchemeManager
