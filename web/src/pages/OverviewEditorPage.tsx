import React, { useEffect, useState, Suspense, lazy, useCallback } from 'react';
import { motion } from 'framer-motion';
import { useParams, useNavigate } from 'react-router-dom';
import { getOverviewConfig, saveOverviewConfig } from '@/api/overview';
import { getGroup } from '@/api/group';
import { OverviewConfig, defaultOverviewConfig } from '@/types/overview';
import { showAlert } from '@/utils/notification';
import { useAuthStore } from '@/stores/authStore';
import { useI18nStore } from '@/stores/i18nStore';
import { overviewCache } from '@/utils/overviewCache';
import { ArrowLeft } from 'lucide-react';
import Button from '@/components/UI/Button';

// 懒加载编辑器组件
const OverviewEditor = lazy(() => import('@/components/Overview/OverviewEditor'));

const OverviewEditorPage: React.FC = () => {
  const { t } = useI18nStore();
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const [overviewConfig, setOverviewConfig] = useState<OverviewConfig | null>(null);
  const [loading, setLoading] = useState(true);
  const [group, setGroup] = useState<any>(null);

  useEffect(() => {
    if (id) {
      loadData();
    }
  }, [id]);

  const loadData = useCallback(async () => {
    if (!id) return;
    
    try {
      setLoading(true);
      
      // 检查缓存
      const cachedConfig = overviewCache.getConfig(Number(id));
      
      // 并行加载组织信息和配置
      const [groupRes, configRes] = await Promise.allSettled([
        getGroup(Number(id)),
        cachedConfig 
          ? Promise.resolve({ code: 200, data: cachedConfig })
          : getOverviewConfig(Number(id))
      ]);
      
      // 处理组织信息
      if (groupRes.status === 'fulfilled' && groupRes.value.code === 200) {
        setGroup(groupRes.value.data);
      }
      
      // 处理配置
      if (configRes.status === 'fulfilled' && configRes.value.code === 200 && configRes.value.data) {
        const backendConfig = configRes.value.data as any;
        const currentGroup = groupRes.status === 'fulfilled' && groupRes.value.code === 200 
          ? groupRes.value.data 
          : group;
        
        const loadedConfig: OverviewConfig = {
          id: backendConfig.id || `config-${Date.now()}`,
          organizationId: backendConfig.organizationId || Number(id),
          name: backendConfig.name || `${currentGroup?.name || ''} - 概览`,
          description: backendConfig.description,
          layout: {
            ...defaultOverviewConfig.layout,
            ...(backendConfig.layout || {})
          },
          widgets: backendConfig.widgets || [],
          theme: {
            ...defaultOverviewConfig.theme,
            ...(backendConfig.theme || {})
          },
          header: backendConfig.header || defaultOverviewConfig.header,
          footer: backendConfig.footer || defaultOverviewConfig.footer,
          createdAt: backendConfig.createdAt,
          updatedAt: backendConfig.updatedAt
        };
        
        // 缓存配置
        if (!cachedConfig) {
          overviewCache.setConfig(Number(id), backendConfig);
        }
        
        setOverviewConfig(loadedConfig);
      } else {
        // 如果没有配置，创建默认配置
        const currentGroup = groupRes.status === 'fulfilled' && groupRes.value.code === 200 
          ? groupRes.value.data 
          : group;
        
        const defaultConfig: OverviewConfig = {
          ...defaultOverviewConfig,
          id: `config-${Date.now()}`,
          organizationId: Number(id),
          name: `${currentGroup?.name || ''} - 概览`,
        };
        setOverviewConfig(defaultConfig);
      }
    } catch (err: any) {
      console.warn('获取概览配置失败:', err);
      // 使用默认配置
      const defaultConfig: OverviewConfig = {
        ...defaultOverviewConfig,
        id: `config-${Date.now()}`,
        organizationId: Number(id),
        name: '概览',
      };
      setOverviewConfig(defaultConfig);
    } finally {
      setLoading(false);
    }
  }, [id, group]);

  const handleSaveOverviewConfig = useCallback(async (newConfig: OverviewConfig) => {
    try {
      const res = await saveOverviewConfig(newConfig);
      if (res.code === 200 && res.data) {
        const backendConfig = res.data as any;
        const frontendConfig: OverviewConfig = {
          id: backendConfig.id || newConfig.id,
          organizationId: backendConfig.organizationId || newConfig.organizationId,
          name: backendConfig.name || newConfig.name,
          description: backendConfig.description || newConfig.description,
          layout: backendConfig.layout || newConfig.layout,
          widgets: backendConfig.widgets || newConfig.widgets,
          theme: backendConfig.theme || newConfig.theme,
          header: backendConfig.header || newConfig.header,
          footer: backendConfig.footer || newConfig.footer,
          createdAt: backendConfig.createdAt || newConfig.createdAt,
          updatedAt: backendConfig.updatedAt || new Date().toISOString()
        };
        setOverviewConfig(frontendConfig);
        
        // 更新缓存
        overviewCache.setConfig(Number(id), backendConfig);
        
        showAlert('概览页面配置保存成功', 'success');
        // 保存成功后返回组织设置页面
        navigate(`/groups/${id}/settings`);
      } else {
        showAlert(res.msg || '保存失败', 'error');
      }
    } catch (error: any) {
      console.error('保存配置失败:', error);
      showAlert(error?.msg || error?.message || '保存失败', 'error');
    }
  }, [id, navigate]);

  const handleCancel = () => {
    navigate(`/groups/${id}/settings`);
  };

  if (loading) {
    return (
      <div className="min-h-screen dark:bg-neutral-900 flex items-center justify-center">
        <div className="text-gray-400">{t('groups.loading')}</div>
      </div>
    );
  }

  if (!overviewConfig) {
    return (
      <div className="min-h-screen dark:bg-neutral-900 flex items-center justify-center">
        <div className="text-center">
          <div className="text-gray-400 text-lg mb-2">无法加载配置</div>
          <Button
            onClick={() => navigate(`/groups/${id}/settings`)}
            variant="primary"
            className="mt-4"
          >
            返回
          </Button>
        </div>
      </div>
    );
  }

  return (
    <div className="min-h-screen dark:bg-neutral-900">
      <div className="max-w-7xl mx-auto pt-6 pb-6 px-4">
        {/* 头部 */}
        <div className="mb-6">
          <Button
            onClick={handleCancel}
            variant="ghost"
            size="sm"
            leftIcon={<ArrowLeft className="w-4 h-4" />}
            className="mb-4"
          >
            返回组织设置
          </Button>
          <div className="relative pl-4">
            <motion.div
              layoutId="pageTitleIndicator"
              className="absolute left-0 top-1/2 -translate-y-1/2 w-1 h-8 bg-primary rounded-r-full"
              transition={{ type: 'spring', bounce: 0.2, duration: 0.3 }}
            />
            <h1 className="text-3xl font-bold text-gray-900 dark:text-gray-100 mb-2">
              {group?.name || '组织'} - 编辑概览页面
            </h1>
            <p className="text-gray-500 dark:text-gray-400">
              自定义组织的概览页面布局、组件和样式
            </p>
          </div>
        </div>

        {/* 编辑器 */}
        <Suspense fallback={
          <div className="flex items-center justify-center py-16">
            <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary"></div>
          </div>
        }>
          <OverviewEditor
            config={overviewConfig}
            onSave={handleSaveOverviewConfig}
            onCancel={handleCancel}
          />
        </Suspense>
      </div>
    </div>
  );
};

export default OverviewEditorPage;

