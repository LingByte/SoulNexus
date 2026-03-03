import React, { useEffect, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { getAssistantList, createAssistant, AssistantListItem } from '@/api/assistant';
import AddAssistantModal from '@/components/Voice/AddAssistantModal';
import { showAlert } from '@/utils/notification';
import { useI18nStore } from '@/stores/i18nStore';
import { Bot, MessageCircle, Users, Zap, Circle, Settings, Building2, Plus, Sparkles, TrendingUp, Rocket, Wand2, Network } from 'lucide-react';
import Button from '@/components/UI/Button';
import { motion } from 'framer-motion';

const ICON_MAP = {
  Bot: <Bot className="w-8 h-8" />,
  MessageCircle: <MessageCircle className="w-8 h-8" />,
  Users: <Users className="w-8 h-8" />,
  Zap: <Zap className="w-8 h-8" />,
  Circle: <Circle className="w-8 h-8" />,
};

const ICON_GRADIENTS = {
  Bot: 'from-purple-500 to-pink-500',
  MessageCircle: 'from-blue-500 to-cyan-500',
  Users: 'from-green-500 to-emerald-500',
  Zap: 'from-yellow-500 to-orange-500',
  Circle: 'from-gray-400 to-gray-500',
};

const truncate = (value?: string, max = 16) => {
  if (!value) return '';
  return value.length > max ? value.slice(0, max) + '…' : value;
};

const Assistants: React.FC = () => {
  const { t } = useI18nStore();
  const [assistants, setAssistants] = useState<AssistantListItem[]>([]);
  const [showAddModal, setShowAddModal] = useState(false);
  const navigate = useNavigate();

  const fetchAssistants = async () => {
    try {
      const res = await getAssistantList();
      setAssistants(res.data || []); // 确保始终是数组
    } catch (err) {
      showAlert(t('assistants.messages.fetchFailed'), 'error');
      setAssistants([]); // 错误时设置为空数组
    }
  };

  useEffect(() => {
    fetchAssistants();
  }, []);

  const handleAddAssistant = async (assistant: { name: string; description: string; icon: string; groupId?: number | null }) => {
    try {
      await createAssistant(assistant);
      await fetchAssistants();
      setShowAddModal(false);
      showAlert(t('assistants.messages.createSuccess'), 'success');
    } catch (err: any) {
      showAlert(err?.msg || err?.message || t('assistants.messages.createFailed'), 'error');
    }
  };

  const fmtDate = (iso?: string) => (iso ? iso.slice(0, 10) : '');

  return (
    <div className="min-h-screen dark:bg-neutral-900 flex flex-col">
      <div className="max-w-6xl w-full mx-auto pt-10 pb-4 flex flex-col">
        <div className="flex items-center justify-between mb-7">
          <h1 className="text-2xl font-semibold text-gray-900 dark:text-gray-100 relative pl-4">
            <motion.div
              layoutId="pageTitleIndicator"
              className="absolute left-0 top-1/2 -translate-y-1/2 w-1 h-6 bg-primary rounded-r-full"
              transition={{ type: 'spring', bounce: 0.2, duration: 0.3 }}
            />
            {t('assistants.title')}
          </h1>
          <Button
            onClick={() => setShowAddModal(true)}
            variant="primary"
            size="md"
            leftIcon={<Plus className="w-4 h-4" />}
          >
            {t('assistants.add')}
          </Button>
        </div>
        <div className="w-full grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-4 lg:gap-6">
          {(assistants?.length === 0) && (
            <motion.div 
              className="col-span-full"
              initial={{ opacity: 0, y: 20 }}
              animate={{ opacity: 1, y: 0 }}
              transition={{ duration: 0.5 }}
            >
              <div className="relative max-w-2xl mx-auto py-20 px-6">
                {/* 背景装饰 */}
                <div className="absolute inset-0 bg-gradient-to-br from-purple-50 via-pink-50 to-blue-50 dark:from-purple-900/10 dark:via-pink-900/10 dark:to-blue-900/10 rounded-3xl blur-3xl opacity-50" />
                
                <div className="relative text-center">
                  {/* 主图标容器 */}
                  <motion.div
                    initial={{ scale: 0.8, opacity: 0 }}
                    animate={{ scale: 1, opacity: 1 }}
                    transition={{ delay: 0.2, duration: 0.5, type: "spring" }}
                    className="inline-flex items-center justify-center mb-6"
                  >
                    <div className="relative">
                      {/* 外层光晕 */}
                      <div className="absolute inset-0 bg-gradient-to-r from-purple-400 via-pink-400 to-blue-400 rounded-full blur-2xl opacity-30 animate-pulse" />
                      {/* 中层渐变 */}
                      <div className="absolute inset-0 bg-gradient-to-br from-purple-500 via-pink-500 to-blue-500 rounded-full blur-xl opacity-50" />
                      {/* 内层图标容器 */}
                      <div className="relative w-32 h-32 rounded-full bg-gradient-to-br from-purple-500 via-pink-500 to-blue-500 flex items-center justify-center shadow-2xl">
                        <div className="absolute inset-0 rounded-full bg-gradient-to-br from-white/20 to-transparent" />
                        <Rocket className="w-16 h-16 animate-bounce" style={{ animationDuration: '2s' }} />
                      </div>
                      {/* 装饰星星 */}
                      <motion.div
                        initial={{ scale: 0, rotate: 0 }}
                        animate={{ scale: 1, rotate: 360 }}
                        transition={{ delay: 0.5, duration: 0.8 }}
                        className="absolute -top-2 -right-2"
                      >
                        <Sparkles className="w-8 h-8 text-yellow-400" />
                      </motion.div>
                      <motion.div
                        initial={{ scale: 0, rotate: 0 }}
                        animate={{ scale: 1, rotate: -360 }}
                        transition={{ delay: 0.7, duration: 0.8 }}
                        className="absolute -bottom-2 -left-2"
                      >
                        <Wand2 className="w-6 h-6 text-purple-400" />
                      </motion.div>
                    </div>
                  </motion.div>

                  {/* 标题 */}
                  <motion.h2
                    initial={{ opacity: 0, y: 10 }}
                    animate={{ opacity: 1, y: 0 }}
                    transition={{ delay: 0.4, duration: 0.5 }}
                    className="text-3xl font-bold text-gray-900 dark:text-gray-100 mb-3"
                  >
                    {t('assistants.emptyState.title')}
                  </motion.h2>

                  {/* 描述文字 */}
                  <motion.p
                    initial={{ opacity: 0, y: 10 }}
                    animate={{ opacity: 1, y: 0 }}
                    transition={{ delay: 0.5, duration: 0.5 }}
                    className="text-gray-600 dark:text-gray-400 text-lg mb-8 max-w-md mx-auto leading-relaxed"
                  >
                    {t('assistants.emptyState.description')}
                  </motion.p>

                  {/* 功能特点 */}
                  <motion.div
                    initial={{ opacity: 0, y: 10 }}
                    animate={{ opacity: 1, y: 0 }}
                    transition={{ delay: 0.6, duration: 0.5 }}
                    className="grid grid-cols-1 md:grid-cols-3 gap-4 mb-8 max-w-2xl mx-auto"
                  >
                    {[
                      { icon: Bot, text: t('assistants.emptyState.features.smartDialogue'), color: 'from-purple-500 to-pink-500' },
                      { icon: Zap, text: t('assistants.emptyState.features.fastResponse'), color: 'from-yellow-500 to-orange-500' },
                      { icon: Users, text: t('assistants.emptyState.features.multiScenario'), color: 'from-blue-500 to-cyan-500' },
                    ].map((item, index) => (
                      <motion.div
                        key={index}
                        initial={{ opacity: 0, scale: 0.9 }}
                        animate={{ opacity: 1, scale: 1 }}
                        transition={{ delay: 0.7 + index * 0.1, duration: 0.3 }}
                        className="flex flex-col items-center p-4 rounded-xl bg-white/50 dark:bg-neutral-800/50 backdrop-blur-sm border border-gray-200/50 dark:border-neutral-700/50"
                      >
                        <div className={`w-12 h-12 rounded-lg bg-gradient-to-br ${item.color} flex items-center justify-center mb-2 shadow-lg`}>
                          <item.icon className="w-6 h-6" />
                        </div>
                        <span className="text-sm font-medium text-gray-700 dark:text-gray-300">{item.text}</span>
                      </motion.div>
                    ))}
                  </motion.div>

                  {/* 创建按钮 */}
                  <motion.div
                    initial={{ opacity: 0, y: 10 }}
                    animate={{ opacity: 1, y: 0 }}
                    transition={{ delay: 0.8, duration: 0.5 }}
                  >
                    <Button
                      onClick={() => setShowAddModal(true)}
                      variant="primary"
                      size="lg"
                      leftIcon={<Plus className="w-5 h-5" />}
                      className="bg-gradient-to-r from-purple-500 to-pink-500 hover:from-purple-600 hover:to-pink-600 shadow-lg hover:shadow-xl transform hover:scale-105 transition-all duration-200"
                    >
                      {t('assistants.emptyState.createButton')}
                    </Button>
                  </motion.div>
                </div>
              </div>
            </motion.div>
          )}
          {(assistants || []).map((assistant, index) => {
            const iconKey = assistant.icon as keyof typeof ICON_MAP;
            const gradient = ICON_GRADIENTS[iconKey] || ICON_GRADIENTS.Circle;
            
            return (
              <motion.div
                key={assistant.id}
                initial={{ opacity: 0, y: 20 }}
                animate={{ opacity: 1, y: 0 }}
                transition={{ delay: index * 0.05, duration: 0.3 }}
                whileHover={{ y: -4, scale: 1.01 }}
                className="group relative bg-white dark:bg-neutral-800 rounded-2xl overflow-hidden shadow-sm hover:shadow-xl transition-all duration-300 border border-gray-200/60 dark:border-neutral-700/60 hover:border-purple-300/50 dark:hover:border-purple-500/30 cursor-pointer"
                onClick={() => navigate(`/voice-assistant/${assistant.id}`)}
              >
                {/* 简化的顶部装饰 */}
                <div className={`h-1 bg-gradient-to-r ${gradient} opacity-60 group-hover:opacity-100 transition-opacity duration-300`} />
                
                {/* 简化的背景装饰 */}
                <div className="absolute top-0 right-0 w-32 h-32 bg-gradient-to-br from-purple-50/30 to-pink-50/30 dark:from-purple-900/10 dark:to-pink-900/10 rounded-full blur-2xl opacity-0 group-hover:opacity-100 transition-opacity duration-500 transform translate-x-12 -translate-y-12" />
                
                <div className="relative p-5">
                  {/* 头部区域 - 图标和操作按钮 */}
                  <div className="flex items-start justify-between mb-4">
                    <motion.div 
                      className="relative"
                      whileHover={{ scale: 1.05 }}
                      transition={{ type: "spring", stiffness: 300 }}
                    >
                      <div className={`w-14 h-14 rounded-2xl bg-gradient-to-br ${gradient} flex items-center justify-center shadow-md group-hover:shadow-lg transition-all duration-300 ring-1 ring-white/20`}>
                        {ICON_MAP[iconKey] ?? ICON_MAP.Circle}
                      </div>
                    </motion.div>
                    
                    {/* 操作按钮组 */}
                    <div className="flex items-center gap-1 opacity-0 group-hover:opacity-100 transition-opacity duration-300">
                      <motion.button
                        onClick={(e) => {
                          e.stopPropagation();
                          navigate(`/assistants/${assistant.id}/graph`);
                        }}
                        whileHover={{ scale: 1.1 }}
                        whileTap={{ scale: 0.9 }}
                        className="p-2 rounded-xl hover:bg-blue-50 dark:hover:bg-blue-900/20 text-gray-400 hover:text-blue-600 dark:hover:text-blue-400 transition-all duration-200"
                        title="知识图谱"
                      >
                        <Network className="w-4 h-4" />
                      </motion.button>
                      <motion.button
                        onClick={(e) => {
                          e.stopPropagation();
                          navigate(`/assistants/${assistant.id}/tools`);
                        }}
                        whileHover={{ scale: 1.1 }}
                        whileTap={{ scale: 0.9 }}
                        className="p-2 rounded-xl hover:bg-purple-50 dark:hover:bg-purple-900/20 text-gray-400 hover:text-purple-600 dark:hover:text-purple-400 transition-all duration-200"
                        title={t('assistants.manageTools')}
                      >
                        <Settings className="w-4 h-4" />
                      </motion.button>
                    </div>
                  </div>

                  {/* 标题和ID */}
                  <div className="mb-3">
                    <div className="flex items-center gap-2 mb-1">
                      <h3 className="font-semibold text-lg text-gray-900 dark:text-gray-100 truncate group-hover:text-purple-600 dark:group-hover:text-purple-400 transition-colors duration-300">
                        {assistant.name}
                      </h3>
                      <span className="px-2 py-0.5 text-xs font-medium bg-gray-100 dark:bg-neutral-700 text-gray-500 dark:text-gray-400 rounded-md flex-shrink-0">
                        #{assistant.id}
                      </span>
                    </div>
                  </div>

                  {/* 描述 */}
                  <p className="text-gray-600 dark:text-gray-400 text-sm leading-relaxed mb-4 line-clamp-2 min-h-[2.5rem]">
                    {assistant.description || t('assistants.noDescription')}
                  </p>

                  {/* 核心标签 - 简化布局 */}
                  <div className="space-y-2 mb-4">
                    {/* 主要标签 */}
                    <div className="flex items-center gap-2 flex-wrap">
                      {assistant.groupId && (
                        <span className="inline-flex items-center gap-1 px-2.5 py-1 rounded-md bg-blue-50 dark:bg-blue-900/20 text-blue-700 dark:text-blue-300 text-xs font-medium">
                          <Building2 className="w-3 h-3" />
                          {t('assistants.groupShared')}
                        </span>
                      )}
                      {assistant.personaTag && (
                        <span className="inline-flex items-center gap-1 px-2.5 py-1 rounded-md bg-indigo-50 dark:bg-indigo-900/20 text-indigo-700 dark:text-indigo-300 text-xs font-medium">
                          <TrendingUp className="w-3 h-3" />
                          {truncate(assistant.personaTag, 12)}
                        </span>
                      )}
                    </div>
                    
                    {/* 技术参数 */}
                    {(typeof assistant.temperature === 'number' || typeof assistant.maxTokens === 'number' || assistant.jsSourceId) && (
                      <div className="flex items-center gap-2 flex-wrap">
                        {assistant.jsSourceId && (
                          <span className="inline-flex items-center px-2 py-1 rounded-md bg-gray-50 dark:bg-neutral-700 text-gray-600 dark:text-gray-400 text-xs font-medium">
                            JS: {truncate(assistant.jsSourceId, 8)}
                          </span>
                        )}
                        {typeof assistant.temperature === 'number' && (
                          <span className="inline-flex items-center px-2 py-1 rounded-md bg-orange-50 dark:bg-orange-900/20 text-orange-600 dark:text-orange-400 text-xs font-medium">
                            T: {assistant.temperature}
                          </span>
                        )}
                        {typeof assistant.maxTokens === 'number' && (
                          <span className="inline-flex items-center px-2 py-1 rounded-md bg-emerald-50 dark:bg-emerald-900/20 text-emerald-600 dark:text-emerald-400 text-xs font-medium">
                            {assistant.maxTokens > 1000 ? `${Math.round(assistant.maxTokens/1000)}k` : assistant.maxTokens}
                          </span>
                        )}
                      </div>
                    )}
                  </div>

                  {/* 底部信息栏 */}
                  <div className="flex items-center justify-between pt-3 border-t border-gray-100 dark:border-neutral-700">
                    <div className="flex items-center gap-2 text-xs text-gray-500 dark:text-gray-400">
                      {assistant.createdAt && (
                        <span className="flex items-center gap-1.5">
                          <div className="w-1 h-1 bg-gray-400 rounded-full" />
                          {fmtDate(assistant.createdAt)}
                        </span>
                      )}
                    </div>
                  </div>
                </div>

                {/* 简化的悬浮效果 */}
                <div className="absolute inset-0 bg-gradient-to-br from-purple-500/3 via-transparent to-pink-500/3 opacity-0 group-hover:opacity-100 transition-opacity duration-300 pointer-events-none" />
              </motion.div>
            );
          })}
        </div>
      </div>
      <AddAssistantModal isOpen={showAddModal} onClose={() => setShowAddModal(false)} onAdd={handleAddAssistant} />
    </div>
  );
};

export default Assistants;
