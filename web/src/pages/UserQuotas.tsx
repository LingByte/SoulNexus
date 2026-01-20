import React, { useEffect, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import {
  getUserQuotas,
  createUserQuota,
  updateUserQuota,
  deleteUserQuota,
  type UserQuota,
  getQuotaTypeLabel,
  formatQuotaValue
} from '@/api/quota';
import { showAlert } from '@/utils/notification';
import { Plus, Edit, Trash2, Database, X } from 'lucide-react';
import Button from '@/components/UI/Button';
import QuotaModal from '@/components/Quota/QuotaModal';

const UserQuotas: React.FC = () => {
  const navigate = useNavigate();
  const [quotas, setQuotas] = useState<UserQuota[]>([]);
  const [loading, setLoading] = useState(false);
  const [showQuotaModal, setShowQuotaModal] = useState(false);
  const [editingQuota, setEditingQuota] = useState<UserQuota | null>(null);

  const fetchQuotas = async () => {
    try {
      setLoading(true);
      const res = await getUserQuotas();
      setQuotas(res.data || []);
    } catch (err: any) {
      showAlert(err?.msg || '获取配额列表失败', 'error');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchQuotas();
  }, []);

  return (
    <div className="min-h-screen bg-gray-50 dark:bg-neutral-900 flex flex-col">
      <div className="max-w-7xl w-full mx-auto px-4 sm:px-6 lg:px-8 pt-8 pb-8 flex flex-col">
        <div className="flex items-center justify-between mb-6">
          <div>
            <h1 className="text-3xl font-bold text-gray-900 dark:text-gray-100 mb-1">配额管理</h1>
            <p className="text-sm text-gray-500 dark:text-gray-400">配置和管理您的使用配额</p>
          </div>
          <Button
            onClick={() => {
              setEditingQuota(null);
              setShowQuotaModal(true);
            }}
            variant="primary"
            size="md"
            leftIcon={<Plus className="w-4 h-4" />}
          >
            添加配额
          </Button>
        </div>

        {loading ? (
          <div className="bg-white dark:bg-neutral-800 border border-gray-200 dark:border-neutral-700 rounded-lg p-16 text-center">
            <div className="text-gray-400 dark:text-gray-500">加载中...</div>
          </div>
        ) : quotas.length === 0 ? (
          <div className="bg-white dark:bg-neutral-800 border border-gray-200 dark:border-neutral-700 rounded-lg p-16 text-center">
            <Database className="w-12 h-12 mx-auto mb-4 text-gray-400 dark:text-gray-500" />
            <p className="text-gray-500 dark:text-gray-400 mb-4">还没有配置配额</p>
            <p className="text-sm text-gray-500 dark:text-gray-500 mb-4">
              配额用于限制您的使用量，当使用率达到阈值时会触发告警
            </p>
            <Button
              onClick={() => setShowQuotaModal(true)}
              variant="primary"
              size="md"
              leftIcon={<Plus className="w-4 h-4" />}
            >
              创建第一个配额
            </Button>
          </div>
        ) : (
          <div className="grid grid-cols-1 lg:grid-cols-2 gap-4">
            {quotas.map(quota => {
              const percentage = quota.totalQuota > 0 
                ? ((quota.usedQuota / quota.totalQuota) * 100).toFixed(2)
                : '0';
              return (
                <div
                  key={quota.id}
                  className="bg-white dark:bg-neutral-800 border border-gray-200 dark:border-neutral-700 rounded-lg p-6 hover:border-purple-400 dark:hover:border-purple-500 hover:shadow-md transition-all"
                >
                  <div className="flex items-start justify-between mb-4">
                    <div className="flex-1">
                      <div className="flex items-center gap-3 mb-2">
                        <h3 className="font-semibold text-gray-900 dark:text-gray-100">
                          {getQuotaTypeLabel(quota.quotaType)}
                        </h3>
                        <span className="px-2 py-0.5 rounded text-xs bg-gray-100 dark:bg-neutral-700 text-gray-600 dark:text-gray-400">
                          {quota.period === 'lifetime' ? '永久' : quota.period === 'monthly' ? '按月' : '按年'}
                        </span>
                      </div>
                      <div className="space-y-2">
                        <div className="flex items-center justify-between text-sm">
                          <span className="text-gray-600 dark:text-gray-400">已使用：</span>
                          <span className="font-medium">{formatQuotaValue(quota.quotaType, quota.usedQuota)}</span>
                        </div>
                        <div className="flex items-center justify-between text-sm">
                          <span className="text-gray-600 dark:text-gray-400">总配额：</span>
                          <span className="font-medium">{quota.totalQuota === 0 ? '无限制' : formatQuotaValue(quota.quotaType, quota.totalQuota)}</span>
                        </div>
                        {quota.totalQuota > 0 && (
                          <div className="mt-3">
                            <div className="flex items-center justify-between text-xs text-gray-500 dark:text-gray-400 mb-1">
                              <span>使用率</span>
                              <span>{percentage}%</span>
                            </div>
                            <div className="w-full bg-gray-200 dark:bg-neutral-700 rounded-full h-2">
                              <div
                                className={`h-2 rounded-full transition-all ${
                                  parseFloat(percentage) >= 90
                                    ? 'bg-red-500'
                                    : parseFloat(percentage) >= 75
                                    ? 'bg-orange-500'
                                    : parseFloat(percentage) >= 50
                                    ? 'bg-yellow-500'
                                    : 'bg-green-500'
                                }`}
                                style={{ width: `${Math.min(parseFloat(percentage), 100)}%` }}
                              />
                            </div>
                          </div>
                        )}
                        {quota.description && (
                          <p className="text-xs text-gray-500 dark:text-gray-400 mt-2">{quota.description}</p>
                        )}
                      </div>
                    </div>
                    <div className="flex items-center gap-2 ml-4">
                      <Button
                        onClick={() => {
                          setEditingQuota(quota);
                          setShowQuotaModal(true);
                        }}
                        variant="ghost"
                        size="sm"
                        leftIcon={<Edit className="w-4 h-4" />}
                      >
                        编辑
                      </Button>
                      <Button
                        onClick={async () => {
                          if (!confirm('确定要删除这个配额配置吗？')) return;
                          try {
                            await deleteUserQuota(quota.quotaType);
                            showAlert('删除成功', 'success');
                            fetchQuotas();
                          } catch (err: any) {
                            showAlert(err?.msg || '删除失败', 'error');
                          }
                        }}
                        variant="ghost"
                        size="sm"
                        leftIcon={<Trash2 className="w-4 h-4" />}
                      >
                        删除
                      </Button>
                    </div>
                  </div>
                </div>
              );
            })}
          </div>
        )}
      </div>

      {/* 配额管理弹窗 */}
      {showQuotaModal && (
        <UserQuotaModal
          isOpen={showQuotaModal}
          onClose={() => {
            setShowQuotaModal(false);
            setEditingQuota(null);
          }}
          quota={editingQuota}
          onSuccess={() => {
            fetchQuotas();
          }}
        />
      )}
    </div>
  );
};

// 用户配额模态框组件
interface UserQuotaModalProps {
  isOpen: boolean;
  onClose: () => void;
  quota?: UserQuota | null;
  onSuccess: () => void;
}

const UserQuotaModal: React.FC<UserQuotaModalProps> = ({ isOpen, onClose, quota, onSuccess }) => {
  const [loading, setLoading] = useState(false);
  const [formData, setFormData] = useState({
    quotaType: '' as any,
    totalQuota: '',
    period: 'lifetime' as any,
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
        await updateUserQuota(quota.quotaType, data);
        showAlert('更新成功', 'success');
      } else {
        await createUserQuota(data);
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

  const quotaTypes = [
    'llm_tokens',
    'llm_calls',
    'api_calls',
    'call_duration',
    'call_count',
    'asr_duration',
    'asr_count',
    'tts_duration',
    'tts_count',
  ] as const;

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
                onChange={(e) => setFormData({ ...formData, quotaType: e.target.value })}
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
              onChange={(e) => setFormData({ ...formData, period: e.target.value })}
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

export default UserQuotas;

