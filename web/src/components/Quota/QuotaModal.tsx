import React, { useState, useEffect } from 'react';
import { createGroupQuota, updateGroupQuota, type GroupQuota, type QuotaType, type QuotaPeriod, getQuotaTypeLabel } from '@/api/quota';
import { showAlert } from '@/utils/notification';
import { X } from 'lucide-react';
import Button from '@/components/UI/Button';

interface QuotaModalProps {
  isOpen: boolean;
  onClose: () => void;
  groupId: number;
  quota?: GroupQuota | null;
  onSuccess: () => void;
}

const QuotaModal: React.FC<QuotaModalProps> = ({ isOpen, onClose, groupId, quota, onSuccess }) => {
  const [loading, setLoading] = useState(false);
  const [formData, setFormData] = useState({
    quotaType: '' as QuotaType | '',
    totalQuota: '',
    period: 'lifetime' as QuotaPeriod,
    description: '',
  });

  useEffect(() => {
    if (quota) {
      setFormData({
        quotaType: quota.quotaType,
        totalQuota: quota.totalQuota.toString(),
        period: quota.period,
        description: quota.description || '',
      });
    } else {
      setFormData({
        quotaType: '',
        totalQuota: '',
        period: 'lifetime',
        description: '',
      });
    }
  }, [quota, isOpen]);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!formData.quotaType) {
      showAlert('请选择配额类型', 'error');
      return;
    }
    if (!formData.totalQuota || parseFloat(formData.totalQuota) < 0) {
      showAlert('请输入有效的总配额', 'error');
      return;
    }

    setLoading(true);
    try {
      const data = {
        quotaType: formData.quotaType,
        totalQuota: parseFloat(formData.totalQuota),
        period: formData.period,
        description: formData.description,
      };

      if (quota) {
        await updateGroupQuota(groupId, quota.quotaType, data);
        showAlert('更新成功', 'success');
      } else {
        await createGroupQuota(groupId, data);
        showAlert('创建成功', 'success');
      }
      onSuccess();
      onClose();
    } catch (err: any) {
      showAlert(err?.msg || err?.message || '操作失败', 'error');
    } finally {
      setLoading(false);
    }
  };

  if (!isOpen) return null;

  return (
    <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50 p-4">
      <div className="bg-white dark:bg-neutral-800 rounded-lg max-w-2xl w-full max-h-[90vh] overflow-y-auto">
        <div className="sticky top-0 bg-white dark:bg-neutral-800 border-b border-gray-200 dark:border-neutral-700 p-6 flex items-center justify-between">
          <h2 className="text-xl font-semibold text-gray-900 dark:text-gray-100">
            {quota ? '编辑配额' : '创建配额'}
          </h2>
          <button
            onClick={onClose}
            className="text-gray-400 hover:text-gray-600 dark:hover:text-gray-300"
          >
            <X className="w-5 h-5" />
          </button>
        </div>

        <form onSubmit={handleSubmit} className="p-6 space-y-6">
          <div>
            <label className="block text-sm font-medium mb-2">
              配额类型 {!quota && '*'}
            </label>
            {quota ? (
              <div className="px-3 py-2 border border-gray-300 dark:border-neutral-700 rounded-lg bg-gray-50 dark:bg-neutral-900 text-gray-700 dark:text-gray-300">
                {getQuotaTypeLabel(quota.quotaType)}
              </div>
            ) : (
              <select
                value={formData.quotaType}
                onChange={(e) => setFormData({ ...formData, quotaType: e.target.value as QuotaType })}
                className="w-full px-3 py-2 border border-gray-300 dark:border-neutral-700 rounded-lg bg-white dark:bg-neutral-800 focus:outline-none focus:ring-2 focus:ring-purple-500"
                required
              >
                <option value="">请选择配额类型</option>
                {quotaTypes.map(type => (
                  <option key={type} value={type}>
                    {getQuotaTypeLabel(type)}
                  </option>
                ))}
              </select>
            )}
          </div>

          <div>
            <label className="block text-sm font-medium mb-2">
              总配额 *
            </label>
            <input
              type="number"
              min="0"
              step="0.01"
              value={formData.totalQuota}
              onChange={(e) => setFormData({ ...formData, totalQuota: e.target.value })}
              className="w-full px-3 py-2 border border-gray-300 dark:border-neutral-700 rounded-lg bg-white dark:bg-neutral-800 focus:outline-none focus:ring-2 focus:ring-purple-500"
              placeholder="0 表示无限制"
              required
            />
            <p className="text-xs text-gray-500 dark:text-gray-400 mt-1">
              输入 0 表示不限制该配额类型的使用量
            </p>
          </div>

          <div>
            <label className="block text-sm font-medium mb-2">
              配额周期
            </label>
            <select
              value={formData.period}
              onChange={(e) => setFormData({ ...formData, period: e.target.value as QuotaPeriod })}
              className="w-full px-3 py-2 border border-gray-300 dark:border-neutral-700 rounded-lg bg-white dark:bg-neutral-800 focus:outline-none focus:ring-2 focus:ring-purple-500"
            >
              <option value="lifetime">永久有效</option>
              <option value="monthly">按月重置</option>
              <option value="yearly">按年重置</option>
            </select>
            <p className="text-xs text-gray-500 dark:text-gray-400 mt-1">
              选择配额的生效周期，到期后会自动重置使用量
            </p>
          </div>

          <div>
            <label className="block text-sm font-medium mb-2">
              描述
            </label>
            <textarea
              value={formData.description}
              onChange={(e) => setFormData({ ...formData, description: e.target.value })}
              rows={3}
              className="w-full px-3 py-2 border border-gray-300 dark:border-neutral-700 rounded-lg bg-white dark:bg-neutral-800 focus:outline-none focus:ring-2 focus:ring-purple-500"
              placeholder="可选：添加配额说明"
            />
          </div>

          <div className="flex items-center justify-end gap-3 pt-4 border-t border-gray-200 dark:border-neutral-700">
            <Button
              type="button"
              onClick={onClose}
              variant="ghost"
            >
              取消
            </Button>
            <Button
              type="submit"
              variant="primary"
              disabled={loading}
            >
              {loading ? '保存中...' : '保存'}
            </Button>
          </div>
        </form>
      </div>
    </div>
  );
};

export default QuotaModal;

