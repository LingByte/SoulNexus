import { useState } from 'react'
import { Modal } from '@arco-design/web-react'
import Textarea from '@/components/UI/Textarea'
import Button from '@/components/UI/Button'

interface SaveChangeNoteModalProps {
  open: boolean
  saving: boolean
  onCancel: () => void
  onConfirm: (changeNote: string, bumpVersion: boolean) => void
}

export default function SaveChangeNoteModal({ open, saving, onCancel, onConfirm }: SaveChangeNoteModalProps) {
  const [changeNote, setChangeNote] = useState('')
  const [bumpVersion, setBumpVersion] = useState(true)

  const handleOk = () => {
    onConfirm(changeNote.trim(), bumpVersion)
  }

  return (
    <Modal
      visible={open}
      title="保存桌宠项目"
      onCancel={onCancel}
      footer={null}
      unmountOnExit
      afterClose={() => {
        setChangeNote('')
        setBumpVersion(true)
      }}
    >
      <div className="space-y-4 py-1">
        <div>
          <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">变更说明</label>
          <Textarea
            value={changeNote}
            onChange={(e) => setChangeNote(e.target.value)}
            placeholder="描述本次修改（可选，填写后将写入 CHANGELOG 并 bump soulpet.yaml 版本）"
            rows={3}
          />
        </div>
        <label className="flex items-center gap-2 text-sm text-gray-600 dark:text-gray-400 cursor-pointer">
          <input
            type="checkbox"
            checked={bumpVersion}
            onChange={(e) => setBumpVersion(e.target.checked)}
            className="rounded"
          />
          自动 bump 包版本号（semver patch）
        </label>
        <div className="flex justify-end gap-2 pt-2">
          <Button variant="outline" size="sm" onClick={onCancel} disabled={saving}>
            取消
          </Button>
          <Button variant="primary" size="sm" onClick={handleOk} disabled={saving}>
            {saving ? '保存中…' : '保存'}
          </Button>
        </div>
      </div>
    </Modal>
  )
}
