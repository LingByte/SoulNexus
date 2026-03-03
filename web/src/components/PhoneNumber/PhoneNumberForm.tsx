import { useState, useEffect } from 'react'
import { Save } from 'lucide-react'
import Button from '@/components/UI/Button'
import Input from '@/components/UI/Input'
import { showAlert } from '@/utils/notification'
import { createPhoneNumber, updatePhoneNumber } from '@/api/phoneNumber'
import type { PhoneNumber, CreatePhoneNumberRequest } from '@/types/phoneNumber'

interface PhoneNumberFormProps {
  number?: PhoneNumber | null
  onSuccess: () => void
  onCancel: () => void
}

const PhoneNumberForm = ({ number, onSuccess, onCancel }: PhoneNumberFormProps) => {
  const [loading, setLoading] = useState(false)
  const [formData, setFormData] = useState<CreatePhoneNumberRequest>({
    phoneNumber: '',
    displayName: '',
    carrier: ''
  })

  useEffect(() => {
    if (number) {
      setFormData({
        phoneNumber: number.phoneNumber,
        displayName: number.displayName || '',
        carrier: number.carrier || ''
      })
    }
  }, [number])

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()

    if (!formData.phoneNumber.trim()) {
      showAlert('请输入手机号码', 'warning')
      return
    }

    // 简单的手机号验证
    const phoneRegex = /^1[3-9]\d{9}$/
    if (!phoneRegex.test(formData.phoneNumber)) {
      showAlert('请输入有效的手机号码', 'warning')
      return
    }

    try {
      setLoading(true)
      const res = number
        ? await updatePhoneNumber(number.id, formData)
        : await createPhoneNumber(formData)

      if (res.code === 200) {
        showAlert(number ? '更新成功' : '添加成功', 'success')
        onSuccess()
      } else {
        throw new Error(res.msg)
      }
    } catch (error: any) {
      showAlert(error.msg || '操作失败', 'error')
    } finally {
      setLoading(false)
    }
  }

  return (
    <form onSubmit={handleSubmit} className="space-y-4">
      <Input
        label="手机号码"
        type="tel"
        placeholder="请输入11位手机号码"
        value={formData.phoneNumber}
        onChange={(e) => setFormData({ ...formData, phoneNumber: e.target.value })}
        disabled={!!number}
        required
        maxLength={11}
      />

      <Input
        label="显示名称"
        placeholder="例如：我的手机、工作号码"
        value={formData.displayName}
        onChange={(e) => setFormData({ ...formData, displayName: e.target.value })}
      />

      <div>
        <label className="block text-sm font-medium text-foreground mb-1.5">
          运营商
        </label>
        <select
          className="w-full px-3.5 py-2.5 border border-input rounded-md bg-background text-foreground focus:ring-2 focus:ring-ring focus:border-transparent"
          value={formData.carrier}
          onChange={(e) => setFormData({ ...formData, carrier: e.target.value })}
        >
          <option value="">请选择</option>
          <option value="中国移动">中国移动</option>
          <option value="中国联通">中国联通</option>
          <option value="中国电信">中国电信</option>
        </select>
      </div>

      <div className="flex justify-end gap-3 pt-4">
        <Button
          type="button"
          variant="ghost"
          onClick={onCancel}
          disabled={loading}
        >
          取消
        </Button>
        <Button
          type="submit"
          loading={loading}
          leftIcon={<Save className="w-4 h-4" />}
        >
          {number ? '保存' : '添加'}
        </Button>
      </div>
    </form>
  )
}

export default PhoneNumberForm
