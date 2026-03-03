import React from 'react'
import { Trash2, AlertTriangle, User, Calendar } from 'lucide-react'
import Modal, { ModalContent, ModalFooter } from '@/components/UI/Modal'
import Button from '@/components/UI/Button'
import { motion } from 'framer-motion'

interface VoiceprintDeleteConfirmProps {
  isOpen: boolean
  onClose: () => void
  onConfirm: () => void
  voiceprintName: string
  speakerId?: string
  createdAt?: string
  loading?: boolean
}

const VoiceprintDeleteConfirm: React.FC<VoiceprintDeleteConfirmProps> = ({
  isOpen,
  onClose,
  onConfirm,
  voiceprintName,
  speakerId,
  createdAt,
  loading = false
}) => {
  const handleConfirm = () => {
    onConfirm()
  }

  return (
    <Modal
      isOpen={isOpen}
      onClose={onClose}
      size="md"
      closeOnOverlayClick={!loading}
      closeOnEscape={!loading}
      showCloseButton={false}
    >
      <ModalContent>
        <div className="space-y-6">
          {/* 标题区域 */}
          <div className="flex items-start space-x-4">
            <motion.div
              initial={{ scale: 0 }}
              animate={{ scale: 1 }}
              transition={{ delay: 0.1, type: "spring", stiffness: 200 }}
              className="flex-shrink-0 p-3 bg-red-100 dark:bg-red-900/20 rounded-full"
            >
              <Trash2 className="w-6 h-6 text-red-600 dark:text-red-400" />
            </motion.div>
            <div className="flex-1">
              <h3 className="text-lg font-semibold text-gray-900 dark:text-white">
                删除声纹
              </h3>
              <p className="text-sm text-gray-600 dark:text-gray-400 mt-1">
                此操作将永久删除声纹数据，无法恢复
              </p>
            </div>
          </div>

          {/* 声纹信息卡片 */}
          <div className="bg-gray-50 dark:bg-gray-800/50 rounded-lg p-4 border border-gray-200 dark:border-gray-700">
            <div className="flex items-start gap-3">
              <div className="p-2 bg-purple-100 dark:bg-purple-900/20 rounded-lg">
                <User className="w-5 h-5 text-purple-600 dark:text-purple-400" />
              </div>
              <div className="flex-1">
                <h4 className="font-medium text-gray-900 dark:text-white">
                  {voiceprintName}
                </h4>
                {speakerId && (
                  <p className="text-sm text-gray-600 dark:text-gray-400 mt-1">
                    ID: {speakerId}
                  </p>
                )}
                {createdAt && (
                  <div className="flex items-center gap-1 text-xs text-gray-500 dark:text-gray-500 mt-2">
                    <Calendar className="w-3 h-3" />
                    创建时间: {new Date(createdAt).toLocaleString()}
                  </div>
                )}
              </div>
            </div>
          </div>

          {/* 警告信息 */}
          <div className="bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 rounded-lg p-4">
            <div className="flex items-start gap-3">
              <AlertTriangle className="w-5 h-5 text-red-500 mt-0.5 flex-shrink-0" />
              <div className="space-y-2">
                <p className="text-sm font-medium text-red-800 dark:text-red-200">
                  确定要删除声纹 "{voiceprintName}" 吗？
                </p>
                <div className="text-xs text-red-600 dark:text-red-300 space-y-1">
                  <p>• 此操作不可恢复，删除后将无法找回该声纹数据</p>
                  <p>• 删除声纹后，相关的识别记录也将被清除</p>
                  <p>• 如需重新使用，需要重新注册该说话人的声纹</p>
                </div>
              </div>
            </div>
          </div>
        </div>
      </ModalContent>
      <ModalFooter>
        <Button
          variant="outline"
          onClick={onClose}
          disabled={loading}
        >
          取消
        </Button>
        <Button
          variant="destructive"
          onClick={handleConfirm}
          loading={loading}
          leftIcon={<Trash2 className="w-4 h-4" />}
        >
          {loading ? '删除中...' : '确认删除'}
        </Button>
      </ModalFooter>
    </Modal>
  )
}

export default VoiceprintDeleteConfirm