import React from 'react';
import { AlertTriangle, X } from 'lucide-react';
import Button from '@/components/UI/Button';
import { useI18nStore } from '@/stores/i18nStore';

interface DeleteConfirmModalProps {
  isOpen: boolean;
  onClose: () => void;
  onConfirm: () => void;
  loading?: boolean;
  quotaType: string;
}

const DeleteConfirmModal: React.FC<DeleteConfirmModalProps> = ({
  isOpen,
  onClose,
  onConfirm,
  loading = false,
  quotaType,
}) => {
  const { t } = useI18nStore();

  if (!isOpen) return null;

  return (
    <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50 p-4">
      <div className="bg-white dark:bg-neutral-800 rounded-lg max-w-md w-full">
        <div className="p-6">
          <div className="flex items-center justify-between mb-4">
            <div className="flex items-center gap-3">
              <div className="w-10 h-10 rounded-full bg-red-100 dark:bg-red-900/30 flex items-center justify-center">
                <AlertTriangle className="w-5 h-5 text-red-600 dark:text-red-400" />
              </div>
              <h2 className="text-lg font-semibold text-gray-900 dark:text-gray-100">
                {t('groupSettings.messages.deleteConfirm')}
              </h2>
            </div>
            <button
              onClick={onClose}
              className="text-gray-400 hover:text-gray-600 dark:hover:text-gray-300"
            >
              <X className="w-5 h-5" />
            </button>
          </div>

          <div className="mb-6">
            <p className="text-gray-600 dark:text-gray-400 text-sm mb-3">
              {t('groupSettings.messages.deleteQuotaWarning')}
            </p>
            <div className="bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 rounded-lg p-3">
              <p className="text-sm font-medium text-red-900 dark:text-red-200">
                {quotaType}
              </p>
            </div>
          </div>

          <div className="flex items-center justify-end gap-3">
            <Button
              type="button"
              onClick={onClose}
              variant="ghost"
              disabled={loading}
            >
              {t('groups.createModal.cancel')}
            </Button>
            <Button
              type="button"
              onClick={onConfirm}
              variant="destructive"
              disabled={loading}
              loading={loading}
            >
              {loading ? t('groupSettings.messages.deleting') : t('alertRules.delete')}
            </Button>
          </div>
        </div>
      </div>
    </div>
  );
};

export default DeleteConfirmModal;
