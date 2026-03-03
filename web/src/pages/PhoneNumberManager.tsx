import { useState, useEffect } from 'react'
import { motion } from 'framer-motion'
import { 
  Plus, 
  Phone, 
  Star, 
  Edit, 
  Trash2,
  Link as LinkIcon,
  Unlink,
  Info
} from 'lucide-react'
import Button from '@/components/UI/Button'
import Modal from '@/components/UI/Modal'
import { showAlert } from '@/utils/notification'
import { 
  getPhoneNumbers, 
  deletePhoneNumber, 
  setPrimaryPhoneNumber,
  bindScheme,
  unbindScheme
} from '@/api/phoneNumber'
import { getSchemes } from '@/api/scheme'
import type { PhoneNumber } from '@/types/phoneNumber'
import type { Scheme } from '@/types/scheme'
import PhoneNumberForm from '@/components/PhoneNumber/PhoneNumberForm'
import PhoneNumberCard from '@/components/PhoneNumber/PhoneNumberCard'
import ForwardGuide from '@/components/PhoneNumber/ForwardGuide'

const PhoneNumberManager = () => {
  const [numbers, setNumbers] = useState<PhoneNumber[]>([])
  const [schemes, setSchemes] = useState<Scheme[]>([])
  const [loading, setLoading] = useState(false)
  const [showForm, setShowForm] = useState(false)
  const [showGuide, setShowGuide] = useState(false)
  const [editingNumber, setEditingNumber] = useState<PhoneNumber | null>(null)
  const [deleteConfirm, setDeleteConfirm] = useState<number | null>(null)
  const [bindingNumber, setBindingNumber] = useState<PhoneNumber | null>(null)

  useEffect(() => {
    loadNumbers()
    loadSchemes()
  }, [])

  const loadNumbers = async () => {
    try {
      setLoading(true)
      const res = await getPhoneNumbers()
      if (res.code === 200 && res.data) {
        setNumbers(res.data)
      }
    } catch (error: any) {
      showAlert(error.msg || '加载号码失败', 'error')
    } finally {
      setLoading(false)
    }
  }

  const loadSchemes = async () => {
    try {
      const res = await getSchemes()
      if (res.code === 200 && res.data) {
        setSchemes(res.data)
      }
    } catch (error: any) {
      console.error('加载方案失败:', error)
    }
  }

  const handleCreate = () => {
    setEditingNumber(null)
    setShowForm(true)
  }

  const handleEdit = (number: PhoneNumber) => {
    setEditingNumber(number)
    setShowForm(true)
  }

  const handleDelete = async (id: number) => {
    try {
      const res = await deletePhoneNumber(id)
      if (res.code === 200) {
        showAlert('删除成功', 'success')
        loadNumbers()
      } else {
        throw new Error(res.msg)
      }
    } catch (error: any) {
      showAlert(error.msg || '删除失败', 'error')
    } finally {
      setDeleteConfirm(null)
    }
  }

  const handleSetPrimary = async (id: number) => {
    try {
      const res = await setPrimaryPhoneNumber(id)
      if (res.code === 200) {
        showAlert('已设为主号码', 'success')
        loadNumbers()
      } else {
        throw new Error(res.msg)
      }
    } catch (error: any) {
      showAlert(error.msg || '设置失败', 'error')
    }
  }

  const handleBindScheme = async (numberId: number, schemeId: number) => {
    try {
      const res = await bindScheme(numberId, schemeId)
      if (res.code === 200) {
        showAlert('绑定成功', 'success')
        loadNumbers()
        setBindingNumber(null)
      } else {
        throw new Error(res.msg)
      }
    } catch (error: any) {
      showAlert(error.msg || '绑定失败', 'error')
    }
  }

  const handleUnbindScheme = async (numberId: number) => {
    try {
      const res = await unbindScheme(numberId)
      if (res.code === 200) {
        showAlert('已解绑', 'success')
        loadNumbers()
      } else {
        throw new Error(res.msg)
      }
    } catch (error: any) {
      showAlert(error.msg || '解绑失败', 'error')
    }
  }

  const handleFormSuccess = () => {
    setShowForm(false)
    setEditingNumber(null)
    loadNumbers()
  }

  return (
    <div className="p-6 max-w-7xl mx-auto">
      {/* Header */}
      <div className="mb-6">
        <div className="flex items-center justify-between mb-2">
          <div>
            <h1 className="text-3xl font-bold text-foreground">
              号码管理
            </h1>
            <p className="text-muted-foreground mt-1">
              管理您的手机号码，设置呼叫转移和绑定代接方案
            </p>
          </div>
          <div className="flex gap-3">
            <Button
              variant="outline"
              onClick={() => setShowGuide(true)}
              leftIcon={<Info className="w-4 h-4" />}
            >
              呼叫转移指引
            </Button>
            <Button
              onClick={handleCreate}
              leftIcon={<Plus className="w-4 h-4" />}
              size="lg"
            >
              添加号码
            </Button>
          </div>
        </div>
      </div>

      {/* Numbers Grid */}
      {loading ? (
        <div className="text-center py-12">
          <div className="inline-block w-8 h-8 border-4 border-primary border-t-transparent rounded-full animate-spin" />
          <p className="mt-4 text-muted-foreground">加载中...</p>
        </div>
      ) : numbers.length === 0 ? (
        <motion.div
          initial={{ opacity: 0, scale: 0.95 }}
          animate={{ opacity: 1, scale: 1 }}
          className="text-center py-12 bg-card border border-border rounded-lg"
        >
          <Phone className="w-16 h-16 mx-auto text-muted-foreground mb-4" />
          <h3 className="text-lg font-medium text-foreground mb-2">
            还没有添加号码
          </h3>
          <p className="text-muted-foreground mb-6">
            添加您的手机号码，开始使用AI代接服务
          </p>
          <Button onClick={handleCreate} leftIcon={<Plus className="w-4 h-4" />}>
            添加号码
          </Button>
        </motion.div>
      ) : (
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
          {numbers.map((number, index) => (
            <motion.div
              key={number.id}
              initial={{ opacity: 0, y: 20 }}
              animate={{ opacity: 1, y: 0 }}
              transition={{ delay: index * 0.1 }}
            >
              <PhoneNumberCard
                number={number}
                onEdit={() => handleEdit(number)}
                onDelete={() => setDeleteConfirm(number.id)}
                onSetPrimary={() => handleSetPrimary(number.id)}
                onBind={() => setBindingNumber(number)}
                onUnbind={() => handleUnbindScheme(number.id)}
              />
            </motion.div>
          ))}
        </div>
      )}

      {/* Add/Edit Form Modal */}
      <Modal
        isOpen={showForm}
        onClose={() => {
          setShowForm(false)
          setEditingNumber(null)
        }}
        title={editingNumber ? '编辑号码' : '添加号码'}
        size="md"
      >
        <PhoneNumberForm
          number={editingNumber}
          onSuccess={handleFormSuccess}
          onCancel={() => {
            setShowForm(false)
            setEditingNumber(null)
          }}
        />
      </Modal>

      {/* Forward Guide Modal */}
      <Modal
        isOpen={showGuide}
        onClose={() => setShowGuide(false)}
        title="呼叫转移设置指引"
        size="lg"
      >
        <ForwardGuide />
      </Modal>

      {/* Bind Scheme Modal */}
      <Modal
        isOpen={bindingNumber !== null}
        onClose={() => setBindingNumber(null)}
        title="绑定代接方案"
        size="md"
      >
        <div className="space-y-4">
          <p className="text-sm text-muted-foreground">
            为号码 <span className="font-medium text-foreground">{bindingNumber?.phoneNumber}</span> 选择一个代接方案
          </p>
          
          {schemes.length === 0 ? (
            <div className="text-center py-8 text-muted-foreground">
              暂无可用方案，请先创建方案
            </div>
          ) : (
            <div className="space-y-2">
              {schemes.map((scheme) => (
                <motion.button
                  key={scheme.id}
                  whileHover={{ scale: 1.02 }}
                  whileTap={{ scale: 0.98 }}
                  onClick={() => bindingNumber && handleBindScheme(bindingNumber.id, scheme.id)}
                  className="w-full p-4 text-left border border-border rounded-lg hover:border-primary hover:bg-muted/50 transition-all"
                >
                  <div className="flex items-center justify-between">
                    <div>
                      <div className="font-medium text-foreground">{scheme.name}</div>
                      {scheme.description && (
                        <div className="text-sm text-muted-foreground mt-1">
                          {scheme.description}
                        </div>
                      )}
                    </div>
                    {scheme.isActive && (
                      <span className="px-2 py-1 bg-green-100 text-green-700 dark:bg-green-900 dark:text-green-300 rounded-full text-xs">
                        激活中
                      </span>
                    )}
                  </div>
                </motion.button>
              ))}
            </div>
          )}
        </div>
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
            确定要删除这个号码吗？此操作无法撤销。
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

export default PhoneNumberManager
