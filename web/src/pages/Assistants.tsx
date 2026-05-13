import React, { useEffect, useMemo, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { getAssistantList, createAssistant, AssistantListItem } from '@/api/assistant';
import AddAssistantModal from '@/components/Voice/AddAssistantModal';
import { showAlert } from '@/utils/notification';
import { useI18nStore } from '@/stores/i18nStore';
import { Bot, Users, Zap, Plus, Sparkles, Rocket, Wand2, Search } from 'lucide-react';
import Button from '@/components/UI/Button';
import { motion } from 'framer-motion';

// 根据 id 稳定生成渐变色，避免每次渲染跳色。
const GRADIENT_POOL = [
  'from-purple-500 to-pink-500',
  'from-blue-500 to-cyan-500',
  'from-emerald-500 to-teal-500',
  'from-amber-500 to-orange-500',
  'from-rose-500 to-red-500',
  'from-indigo-500 to-violet-500',
  'from-sky-500 to-blue-600',
  'from-fuchsia-500 to-purple-600',
];

const pickGradient = (id: number) => GRADIENT_POOL[Math.abs(id) % GRADIENT_POOL.length];
const Assistants: React.FC = () => {
  const { t } = useI18nStore();
  const [assistants, setAssistants] = useState<AssistantListItem[]>([]);
  const [showAddModal, setShowAddModal] = useState(false);
  const [keyword, setKeyword] = useState('');
  const navigate = useNavigate();

  const filtered = useMemo(() => {
    const k = keyword.trim().toLowerCase();
    if (!k) return assistants;
    return assistants.filter(a =>
      (a.name || '').toLowerCase().includes(k) ||
      (a.description || '').toLowerCase().includes(k),
    );
  }, [assistants, keyword]);

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

  const handleAddAssistant = async (assistant: { name: string; description: string; groupId?: number | null }) => {
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
      <div className="max-w-6xl w-full mx-auto px-4 pt-8 pb-10 flex flex-col">
        {/* 页面顶部：标题 + 搜索 + 新建 */}
        <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-3 mb-6">
          <h1 className="text-2xl font-semibold text-gray-900 dark:text-gray-100 tracking-tight">
            {t('assistants.title')}
          </h1>
          <div className="flex items-center gap-2 w-full sm:w-auto">
            <div className="relative flex-1 sm:flex-none sm:w-64">
              <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-gray-400" />
              <input
                value={keyword}
                onChange={(e) => setKeyword(e.target.value)}
                placeholder={t('assistants.searchPlaceholder') || '搜索智能体'}
                className="w-full pl-9 pr-3 py-2 text-sm dark:bg-neutral-800 border border-gray-200 dark:border-neutral-700 rounded-lg focus:outline-none focus:ring-2 focus:ring-primary/30 focus:border-primary/60 transition-colors"
              />
            </div>
            <Button
              onClick={() => setShowAddModal(true)}
              variant="primary"
              size="sm"
              leftIcon={<Plus className="w-4 h-4" />}
            >
              {t('assistants.add')}
            </Button>
          </div>
        </div>
        <div className="w-full grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-4">
          {(filtered?.length === 0) && (assistants.length === 0) && (
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
          {filtered.map((assistant, index) => {
            const gradient = pickGradient(assistant.id);
            return (
              <motion.div
                key={assistant.id}
                initial={{ opacity: 0, y: 12 }}
                animate={{ opacity: 1, y: 0 }}
                transition={{ delay: index * 0.03, duration: 0.2 }}
                whileHover={{ y: -3 }}
                className="group relative bg-white dark:bg-neutral-800/80 rounded-2xl overflow-hidden border border-gray-200/70 dark:border-neutral-700/60 hover:border-primary/40 dark:hover:border-primary/40 shadow-[0_1px_2px_rgba(0,0,0,0.03)] hover:shadow-lg hover:shadow-primary/5 transition-all duration-200 cursor-pointer"
                onClick={() => navigate(`/voice-assistant/${assistant.id}`)}
              >
                <div className="p-5">
                  <div className="flex items-start gap-3">
                    <div
                      className={`w-12 h-12 rounded-xl bg-gradient-to-br ${gradient} flex items-center justify-center shadow-sm overflow-hidden flex-shrink-0`}
                    >
                      <Bot className="h-6 w-6" />
                    </div>
                    <div className="min-w-0 flex-1">
                      <div className="flex items-center gap-2">
                        <h3 className="font-semibold text-[15px] text-gray-900 dark:text-gray-100 truncate group-hover:text-primary transition-colors">
                          {assistant.name}
                        </h3>
                      </div>
                      <div className="mt-0.5 text-[11px] text-gray-400 dark:text-gray-500 font-mono">
                        #{assistant.id}
                      </div>
                    </div>
                  </div>

                  <p className="mt-3 text-[13px] text-gray-600 dark:text-gray-400 leading-relaxed line-clamp-2 min-h-[2.5rem]">
                    {assistant.description || t('assistants.noDescription')}
                  </p>

                  {(assistant.personaTag || typeof assistant.temperature === 'number' || typeof assistant.maxTokens === 'number') && (
                    <div className="mt-3 flex items-center gap-1.5 flex-wrap">
                      {assistant.personaTag && (
                        <span className="px-2 py-0.5 rounded-md bg-indigo-50 dark:bg-indigo-900/20 text-indigo-600 dark:text-indigo-300 text-[11px] font-medium">
                          {assistant.personaTag.length > 14 ? assistant.personaTag.slice(0, 14) + '…' : assistant.personaTag}
                        </span>
                      )}
                      {typeof assistant.temperature === 'number' && (
                        <span className="px-2 py-0.5 rounded-md bg-orange-50 dark:bg-orange-900/20 text-orange-600 dark:text-orange-300 text-[11px] font-medium">
                          T {assistant.temperature}
                        </span>
                      )}
                      {typeof assistant.maxTokens === 'number' && (
                        <span className="px-2 py-0.5 rounded-md bg-emerald-50 dark:bg-emerald-900/20 text-emerald-600 dark:text-emerald-300 text-[11px] font-medium">
                          {assistant.maxTokens > 1000 ? `${Math.round(assistant.maxTokens / 1000)}k` : assistant.maxTokens} tok
                        </span>
                      )}
                    </div>
                  )}

                  {assistant.createdAt && (
                    <div className="mt-4 pt-3 border-t border-gray-100 dark:border-neutral-700/60 text-[11px] text-gray-400 dark:text-gray-500">
                      {fmtDate(assistant.createdAt)}
                    </div>
                  )}
                </div>
              </motion.div>
            );
          })}

          {/* 搜索无结果（但列表非空） */}
          {assistants.length > 0 && filtered.length === 0 && (
            <div className="col-span-full text-center py-16 text-sm text-gray-500 dark:text-gray-400">
              {t('assistants.noMatch') || '没有匹配的智能体'}
            </div>
          )}
        </div>
      </div>
      <AddAssistantModal isOpen={showAddModal} onClose={() => setShowAddModal(false)} onAdd={handleAddAssistant} />
    </div>
  );
};

export default Assistants;
