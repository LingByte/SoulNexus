// src/pages/KnowledgeBase.tsx
import React, { useState, useEffect } from 'react';
import { Plus, Trash2, Search, FileText, BookOpen, Upload, Building2, MessageSquare } from 'lucide-react';
import { showAlert } from '@/utils/notification'
import { useI18nStore } from '@/stores/i18nStore'
import { useAuthStore } from '@/stores/authStore'
import { getGroupList, type Group } from '@/api/group'
import {
    getKnowledgeBaseByUser,
    createKnowledgeBase,
    deleteKnowledgeBase,
    uploadKnowledgeBase,
    askKnowledgeBase
} from '@/api/knowledge';
import { highlightContent } from '@/utils/highlight'
import { useSearchHighlight } from '@/hooks/useSearchHighlight'
import Card from '@/components/UI/Card'
import Button from '@/components/UI/Button'
import Input from '@/components/UI/Input'
import Modal, { ModalContent, ModalFooter } from '@/components/UI/Modal'
import ConfirmDialog from '@/components/UI/ConfirmDialog'
import EmptyState from '@/components/UI/EmptyState'
import Badge from '@/components/UI/Badge'
import FileUpload from '@/components/UI/FileUpload'
import FadeIn from '@/components/Animations/FadeIn'
import PageContainer from '@/components/Layout/PageContainer'

interface KnowledgeBaseItem {
    id: number;
    userid: number;
    user_id?: number;
    group_id?: number | null;
    knowledge_key: string;
    knowledge_name: string;
    created_at: string;
    update_at: string;
    delete_at: string;
}

const KnowledgeBase = () => {
    const { t } = useI18nStore()
    const [isDeleteConfirmOpen, setIsDeleteConfirmOpen] = useState(false);
    const [pendingDeleteKey, setPendingDeleteKey] = useState<string>('');
    const [knowledgeBases, setKnowledgeBases] = useState<KnowledgeBaseItem[]>([]);
    const [filteredKnowledgeBases, setFilteredKnowledgeBases] = useState<KnowledgeBaseItem[]>([]);
    const [searchTerm, setSearchTerm] = useState('');
    const [isCreateModalOpen, setIsCreateModalOpen] = useState(false);
    const [isUploadModalOpen, setIsUploadModalOpen] = useState(false);
    const [isAskModalOpen, setIsAskModalOpen] = useState(false);
    const [currentItem, setCurrentItem] = useState<KnowledgeBaseItem | null>(null);
    const [isCreating, setIsCreating] = useState(false);
    const [formData, setFormData] = useState({
        knowledgeName: '',
        shareToGroup: false,
        selectedGroupId: null as number | null
    });
    const [groups, setGroups] = useState<Group[]>([]);
    const [file, setFile] = useState<File | null>(null);
    const [uploadFile, setUploadFile] = useState<File | null>(null);
    const [question, setQuestion] = useState('');
    const [answer, setAnswer] = useState('');
    const [isLoading, setIsLoading] = useState(true);
    const { user } = useAuthStore()
    const userId = user?.id ? Number(user.id) : 1;
    
    // 获取搜索高亮信息
    const { searchKeyword, highlightFragments, highlightResultId } = useSearchHighlight()

    // 获取组织列表
    useEffect(() => {
        if (isCreateModalOpen) {
            fetchGroups()
        }
    }, [isCreateModalOpen])

    const fetchGroups = async () => {
        try {
            const res = await getGroupList()
            // 只显示用户是创建者或管理员的组织
            const adminGroups = res.data.filter((g: Group) => {
                const userIdNum = user?.id ? Number(user.id) : null
                return g.creatorId === userIdNum || g.myRole === 'admin'
            })
            setGroups(adminGroups)
        } catch (err) {
            console.error('获取组织列表失败', err)
        }
    }

    // 获取知识库列表
    useEffect(() => {
        fetchKnowledgeBases();
    }, []);
    
    useEffect(() => {
        console.log('知识库数据更新:', knowledgeBases);
        // 确保 filteredKnowledgeBases 也同步更新
        if (!searchTerm) {
            setFilteredKnowledgeBases(knowledgeBases);
        }
    }, [knowledgeBases]);
    
    // 搜索过滤
    useEffect(() => {
        if (!searchTerm) {
            setFilteredKnowledgeBases(knowledgeBases || []);
        } else {
            const filtered = (knowledgeBases || []).filter(kb =>
                kb.knowledge_name.toLowerCase().includes(searchTerm.toLowerCase()) ||
                kb.knowledge_key.toLowerCase().includes(searchTerm.toLowerCase())
            );
            setFilteredKnowledgeBases(filtered);
        }
    }, [searchTerm, knowledgeBases]);

    const fetchKnowledgeBases = async () => {
        try {
            setIsLoading(true);
            const response = await getKnowledgeBaseByUser();
            console.log('获取知识库列表响应:', response);

            if (response.code === 200) {
                // 处理 null 或空数组情况
                const responseData = response.data || [];

                // 后端返回格式: { id, knowledge_key, knowledge_name, provider, config, created_at, updated_at, ... }
                const transformedData: KnowledgeBaseItem[] = responseData.map((item: any) => ({
                    id: item.id || 0,
                    userid: item.user_id || userId,
                    user_id: item.user_id || userId,
                    group_id: item.group_id || null,
                    knowledge_key: item.knowledge_key || item.key || '',
                    knowledge_name: item.knowledge_name || item.name || '未命名知识库',
                    created_at: item.created_at || new Date().toISOString(),
                    update_at: item.updated_at || item.update_at || new Date().toISOString(),
                    delete_at: item.delete_at || ''
                }));

                console.log('转换后的数据:', transformedData);
                setKnowledgeBases(transformedData);
                setFilteredKnowledgeBases(transformedData);
            } else {
                // 处理错误情况，清空数据
                setKnowledgeBases([]);
                setFilteredKnowledgeBases([]);
                console.error('获取知识库列表失败:', response.msg);
            }
        } catch (error) {
            console.error('获取知识库列表失败:', error);
            // 出错时清空数据
            setKnowledgeBases([]);
            setFilteredKnowledgeBases([]);
            showAlert(t('knowledgeBase.messages.fetchFailed'), 'error', t('knowledgeBase.messages.fetchFailed'));
        } finally {
            setIsLoading(false);
        }
    };

    const handleCreate = () => {
        setCurrentItem(null);
        setFormData({
            knowledgeName: '',
            shareToGroup: false,
            selectedGroupId: null
        });
        setFile(null);
        setIsCreateModalOpen(true);
    };

    const handleShowUpload = (item: KnowledgeBaseItem) => {
        setCurrentItem(item);
        setUploadFile(null);
        setIsUploadModalOpen(true);
    };

    const handleShowAsk = (item: KnowledgeBaseItem) => {
        setCurrentItem(item);
        setQuestion('');
        setAnswer('');
        setIsAskModalOpen(true);
    };

    const handleDelete = (knowledgeKey: string) => {
        setPendingDeleteKey(knowledgeKey);
        setIsDeleteConfirmOpen(true);
    };

    const confirmDelete = async () => {
        try {
            const response = await deleteKnowledgeBase(pendingDeleteKey);
            if (response.code === 200) {
                setIsLoading(true);
                await fetchKnowledgeBases();
                showAlert(t('knowledgeBase.messages.deleteSuccess'), 'success', t('knowledgeBase.messages.deleteSuccess'));
            } else {
                console.error('删除失败:', response.msg);
                showAlert(t('knowledgeBase.messages.deleteFailed') + ': ' + response.msg, 'error', t('knowledgeBase.messages.deleteFailed'));
            }
        } catch (error) {
            console.error('删除知识库失败:', error);
            showAlert(t('knowledgeBase.messages.deleteError'), 'error', t('knowledgeBase.messages.deleteFailed'));
        } finally {
            setIsDeleteConfirmOpen(false);
            setPendingDeleteKey('');
            setIsLoading(false);
        }
    };

    const handleCreateSubmit = async (e: React.FormEvent) => {
        e.preventDefault();

        if (!file) {
            showAlert(t('knowledgeBase.messages.selectFile'), 'warning', t('knowledgeBase.messages.selectFile'));
            return;
        }

        // 限制知识库名称在10个字符以内
        if (formData.knowledgeName.length > 10) {
            showAlert(t('knowledgeBase.messages.nameTooLong'), 'warning', t('knowledgeBase.messages.nameTooLong'));
            return;
        }

        try {
            setIsCreating(true);
            const response = await createKnowledgeBase({
                knowledgeName: formData.knowledgeName,
                file: file,
                groupId: formData.shareToGroup && formData.selectedGroupId ? formData.selectedGroupId : null
            });

            if (response.code === 200) {
                await fetchKnowledgeBases();
                setIsCreateModalOpen(false);
                showAlert(t('knowledgeBase.messages.createSuccess'), 'success', t('knowledgeBase.messages.createSuccess'));
            } else {
                console.error('创建失败:', response.msg);
                showAlert(t('knowledgeBase.messages.createFailed') + ': ' + response.msg, 'error', t('knowledgeBase.messages.createFailed'));
            }
        } catch (error) {
            console.error('创建知识库失败:', error);
            showAlert(t('knowledgeBase.messages.createError'), 'error', t('knowledgeBase.messages.createFailed'));
        } finally {
            setIsCreating(false);
        }
    };

    const handleUploadSubmit = async (e: React.FormEvent) => {
        e.preventDefault();

        if (!uploadFile || !currentItem) return;

        try {
            console.log('准备上传文件:', uploadFile);
            console.log('知识库key:', currentItem.knowledge_key);

            const response = await uploadKnowledgeBase({
                file: uploadFile,
                knowledgeKey: currentItem.knowledge_key
            });

            console.log('上传响应:', response);

            if (response.code === 200) {
                showAlert(t('knowledgeBase.messages.uploadSuccess'), 'success', t('knowledgeBase.messages.uploadSuccess'));
                setIsUploadModalOpen(false);
            } else {
                console.error('上传失败:', response.msg);
                showAlert(t('knowledgeBase.messages.uploadFailed') + ': ' + response.msg, 'error', t('knowledgeBase.messages.uploadFailed'));
            }
        } catch (error) {
            console.error('上传文件失败:', error);
            showAlert(t('knowledgeBase.messages.uploadError'), 'error', t('knowledgeBase.messages.uploadFailed'));
        }
    };

    const handleAskSubmit = async (e: React.FormEvent) => {
        e.preventDefault();

        if (!question || !currentItem) return;

        try {
            const response = await askKnowledgeBase({
                knowledgeKey: currentItem.knowledge_key,
                message: question
            });

            if (response.code === 200) {
                setAnswer(response.data);
            } else {
                console.error('提问失败:', response.msg);
                showAlert(response.msg || t('knowledgeBase.messages.askFailed'), 'error', t('knowledgeBase.messages.askFailed'));
            }
        } catch (error) {
            console.error('提问失败:', error);
            showAlert(t('knowledgeBase.messages.askError'), 'error', t('knowledgeBase.messages.askFailed'));
        }
    };

    const handleInputChange = (e: React.ChangeEvent<HTMLInputElement | HTMLTextAreaElement>) => {
        const { name, value, type } = e.target;
        if (type === 'checkbox') {
            const checked = (e.target as HTMLInputElement).checked;
            setFormData(prev => ({
                ...prev,
                [name]: checked,
                ...(name === 'shareToGroup' && !checked ? { selectedGroupId: null } : {})
            }));
        } else {
            setFormData(prev => ({
                ...prev,
                [name]: value
            }));
        }
    };

    const handleFileChange = (files: File[]) => {
        if (files && files.length > 0) {
            setFile(files[0]);
        } else {
            setFile(null);
        }
    };

    const handleUploadFileChange = (files: File[]) => {
        if (files && files.length > 0) {
            setUploadFile(files[0]);
        } else {
            setUploadFile(null);
        }
    };

    return (
        <PageContainer maxWidth="full" padding="lg">
            {/* 页面头部 */}
            <FadeIn direction="down">
                <div className="mb-8">
                    <h1 className="text-3xl font-bold text-gray-900 dark:text-white mb-2">
                        {t('knowledgeBase.title')}
                    </h1>
                    <p className="text-gray-600 dark:text-gray-400">
                        {t('knowledgeBase.desc')}
                    </p>
                </div>
            </FadeIn>

            {/* 操作栏 */}
            <FadeIn direction="down" delay={0.1}>
                <div className="flex flex-col sm:flex-row justify-between items-start sm:items-center gap-4 mb-6">
                    <div className="relative w-full sm:w-80">
                        <Input
                            type="text"
                            placeholder={t('knowledgeBase.searchPlaceholder')}
                            value={searchTerm}
                            onChange={(e) => setSearchTerm(e.target.value)}
                            leftIcon={<Search className="w-4 h-4" />}
                        />
                    </div>

                    <Button
                        variant="primary"
                        leftIcon={<Plus className="w-4 h-4" />}
                        onClick={handleCreate}
                    >
                        {t('knowledgeBase.create')}
                    </Button>
                </div>
            </FadeIn>

            {/* 知识库列表 */}
            {isLoading ? (
                <div className="flex justify-center items-center h-64">
                    <div className="animate-spin rounded-full h-12 w-12 border-t-2 border-b-2 border-purple-500"></div>
                </div>
            ) : filteredKnowledgeBases.length === 0 ? (
                <EmptyState
                    icon={BookOpen}
                    title={searchTerm ? t('knowledgeBase.noMatch') : t('knowledgeBase.empty')}
                    description={searchTerm ? t('knowledgeBase.tryOtherKeywords') : t('knowledgeBase.createFirst')}
                    action={!searchTerm ? {
                        label: t('knowledgeBase.create'),
                        onClick: handleCreate
                    } : undefined}
                />
            ) : (
                <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
                    {filteredKnowledgeBases.map((kb, index) => (
                        <FadeIn key={kb.id} direction="up" delay={index * 0.05}>
                                <Card
                                    hover
                                    variant="default"
                                    className="h-full"
                                >
                                    <div className="flex justify-between items-start mb-4">
                                        <div className="flex items-center gap-3 flex-1 min-w-0">
                                            <div className="p-2 bg-purple-100 dark:bg-purple-900/30 rounded-lg flex-shrink-0">
                                                <BookOpen className="w-5 h-5 text-purple-600 dark:text-purple-400" />
                                            </div>
                                            <div className="flex-1 min-w-0">
                                                <div className="flex items-center gap-2 mb-1">
                                                    <h3 
                                                        className={`text-lg font-semibold text-gray-900 dark:text-white truncate ${highlightResultId === `knowledge_${kb.knowledge_key}` ? 'ring-2 ring-yellow-400 rounded px-1' : ''}`}
                                                        dangerouslySetInnerHTML={{
                                                            __html: highlightContent(
                                                                kb.knowledge_name,
                                                                searchKeyword,
                                                                highlightFragments ?? undefined
                                                            )
                                                        }}
                                                    />
                                                    {kb.group_id && (
                                                        <Badge variant="secondary" className="flex items-center gap-1 flex-shrink-0">
                                                            <Building2 className="w-3 h-3" />
                                                            组织共享
                                                        </Badge>
                                                    )}
                                                </div>
                                            </div>
                                        </div>
                                        <div className="flex gap-2">
                                            <Button
                                                variant="ghost"
                                                size="sm"
                                                onClick={() => handleShowUpload(kb)}
                                                leftIcon={<Upload className="w-4 h-4" />}
                                                title={t('knowledgeBase.upload')}
                                            />
                                            <Button
                                                variant="ghost"
                                                size="sm"
                                                onClick={() => handleShowAsk(kb)}
                                                leftIcon={<MessageSquare className="w-4 h-4" />}
                                                title="提问"
                                            />
                                            <Button
                                                variant="ghost"
                                                size="sm"
                                                onClick={() => handleDelete(kb.knowledge_key)}
                                                leftIcon={<Trash2 className="w-4 h-4" />}
                                                title={t('knowledgeBase.delete')}
                                            />
                                        </div>
                                    </div>

                                    <p className="text-gray-600 dark:text-gray-400 text-sm mb-4 line-clamp-2">
                                        {t('knowledgeBase.identifier')}: {kb.knowledge_key}
                                    </p>

                                    <div className="flex justify-between items-center text-sm pt-4 border-t border-gray-200 dark:border-gray-700">
                                        <div className="flex items-center gap-1 text-gray-500 dark:text-gray-400">
                                            <FileText className="w-4 h-4" />
                                            <span>{t('knowledgeBase.userId')}: {kb.userid}</span>
                                        </div>
                                        <div className="text-gray-500 dark:text-gray-400">
                                            {new Date(kb.created_at).toLocaleDateString()}
                                        </div>
                                    </div>
                                </Card>
                            </FadeIn>
                        ))}
                </div>
            )}

            {/* 删除确认对话框 */}
            <ConfirmDialog
                isOpen={isDeleteConfirmOpen}
                onClose={() => setIsDeleteConfirmOpen(false)}
                onConfirm={confirmDelete}
                title={t('knowledgeBase.deleteConfirm.title')}
                description={t('knowledgeBase.deleteConfirm.desc')}
                confirmText={t('knowledgeBase.deleteConfirm.confirm')}
                cancelText={t('knowledgeBase.deleteConfirm.cancel')}
                variant="danger"
            />

            {/* 创建知识库模态框 */}
            <Modal
                isOpen={isCreateModalOpen}
                onClose={() => {
                    setIsCreateModalOpen(false)
                    setFile(null)
                    setFormData({
                        knowledgeName: '',
                        shareToGroup: false,
                        selectedGroupId: null
                    })
                }}
                title={t('knowledgeBase.createModal.title')}
                size="md"
            >
                <form onSubmit={handleCreateSubmit}>
                    <ModalContent>
                        <div className="space-y-4">
                            <Input
                                label={t('knowledgeBase.createModal.nameLabel')}
                                name="knowledgeName"
                                value={formData.knowledgeName}
                                onChange={handleInputChange}
                                placeholder={t('knowledgeBase.createModal.namePlaceholder')}
                                required
                                maxLength={10}
                            />

                            <FileUpload
                                onFileSelect={handleFileChange}
                                accept=".pdf,.doc,.docx,.txt,.md"
                                multiple={false}
                                maxSize={50}
                                maxFiles={1}
                                label={t('knowledgeBase.createModal.fileLabel')}
                                className="w-full"
                            />
                            {file && (
                                <div className="p-3 bg-blue-50 dark:bg-blue-900/20 rounded-lg border border-blue-200 dark:border-blue-800">
                                    <div className="flex items-center gap-2">
                                        <FileText className="w-5 h-5 text-blue-600 dark:text-blue-400" />
                                        <div className="flex-1">
                                            <p className="text-sm font-medium text-blue-900 dark:text-blue-100">
                                                {file.name}
                                            </p>
                                            <p className="text-xs text-blue-700 dark:text-blue-300">
                                                {(file.size / 1024 / 1024).toFixed(2)} MB
                                            </p>
                                        </div>
                                    </div>
                                </div>
                            )}

                        {groups.length > 0 && (
                            <div>
                                <label className="flex items-center gap-2 text-sm font-medium text-gray-700 dark:text-gray-300 cursor-pointer mb-2">
                                    <input
                                        type="checkbox"
                                        name="shareToGroup"
                                        checked={formData.shareToGroup}
                                        onChange={handleInputChange}
                                        className="w-4 h-4 rounded border-gray-300 dark:border-neutral-600"
                                    />
                                    <span className="flex items-center gap-1">
                                        <Building2 className="w-4 h-4" />
                                        共享到组织（所有组织成员都可以使用）
                                    </span>
                                </label>
                                {formData.shareToGroup && (
                                    <select
                                        value={formData.selectedGroupId || ''}
                                        onChange={(e) => setFormData(prev => ({
                                            ...prev,
                                            selectedGroupId: e.target.value ? Number(e.target.value) : null
                                        }))}
                                        className="w-full px-3 py-2 mt-2 border rounded-lg focus:ring-2 focus:ring-purple-500 dark:bg-neutral-700 dark:border-neutral-600"
                                    >
                                        <option value="">选择组织</option>
                                        {groups.map(group => (
                                            <option key={group.id} value={group.id}>
                                                {group.name}
                                            </option>
                                        ))}
                                    </select>
                                )}
                            </div>
                        )}
                        </div>
                    </ModalContent>
                    <ModalFooter>
                        <Button
                            variant="outline"
                            onClick={() => {
                                setIsCreateModalOpen(false)
                                setFile(null)
                                setFormData({
                                    knowledgeName: '',
                                    shareToGroup: false,
                                    selectedGroupId: null
                                })
                            }}
                        >
                            {t('knowledgeBase.deleteConfirm.cancel')}
                        </Button>
                        <Button
                            variant="primary"
                            type="submit"
                            loading={isCreating}
                            leftIcon={!isCreating ? <Plus className="w-4 h-4" /> : undefined}
                            disabled={!file || !formData.knowledgeName}
                        >
                            {isCreating ? t('knowledgeBase.createModal.creating') : t('knowledgeBase.createModal.create')}
                        </Button>
                    </ModalFooter>
                </form>
            </Modal>

            {/* 上传文件模态框 */}
            <Modal
                isOpen={isUploadModalOpen}
                onClose={() => {
                    setIsUploadModalOpen(false)
                    setUploadFile(null)
                }}
                title={t('knowledgeBase.uploadModal.title').replace('{name}', currentItem?.knowledge_name || '')}
                size="md"
            >
                <form onSubmit={handleUploadSubmit}>
                    <ModalContent>
                        <div className="space-y-4">
                            <div>
                                <FileUpload
                                    onFileSelect={handleUploadFileChange}
                                    accept=".pdf,.doc,.docx,.txt,.md"
                                    multiple={false}
                                    maxSize={50}
                                    maxFiles={1}
                                    label={t('knowledgeBase.uploadModal.fileLabel')}
                                    className="w-full"
                                />
                                {uploadFile && (
                                    <div className="mt-4 p-3 bg-blue-50 dark:bg-blue-900/20 rounded-lg border border-blue-200 dark:border-blue-800">
                                        <div className="flex items-center gap-2">
                                            <FileText className="w-5 h-5 text-blue-600 dark:text-blue-400" />
                                            <div className="flex-1">
                                                <p className="text-sm font-medium text-blue-900 dark:text-blue-100">
                                                    {uploadFile.name}
                                                </p>
                                                <p className="text-xs text-blue-700 dark:text-blue-300">
                                                    {(uploadFile.size / 1024 / 1024).toFixed(2)} MB
                                                </p>
                                            </div>
                                        </div>
                                    </div>
                                )}
                            </div>
                        </div>
                    </ModalContent>
                    <ModalFooter>
                        <Button
                            variant="outline"
                            onClick={() => {
                                setIsUploadModalOpen(false)
                                setUploadFile(null)
                            }}
                        >
                            {t('knowledgeBase.deleteConfirm.cancel')}
                        </Button>
                        <Button
                            variant="primary"
                            type="submit"
                            leftIcon={<Upload className="w-4 h-4" />}
                            disabled={!uploadFile}
                        >
                            {t('knowledgeBase.uploadModal.upload')}
                        </Button>
                    </ModalFooter>
                </form>
            </Modal>

            {/* AI提问模态框 */}
            <Modal
                isOpen={isAskModalOpen}
                onClose={() => {
                    setIsAskModalOpen(false);
                    setQuestion('');
                    setAnswer('');
                }}
                title={t('knowledgeBase.askModal.title').replace('{name}', currentItem?.knowledge_name || '')}
                size="lg"
            >
                <form onSubmit={handleAskSubmit}>
                    <div className="space-y-4">
                        <div>
                            <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
                                {t('knowledgeBase.askModal.questionLabel')}
                            </label>
                            <textarea
                                value={question}
                                onChange={(e) => setQuestion(e.target.value)}
                                rows={4}
                                className="w-full px-3 py-2 border rounded-lg focus:ring-2 focus:ring-purple-500 dark:bg-neutral-700 dark:border-neutral-600"
                                placeholder={t('knowledgeBase.askModal.questionPlaceholder')}
                                required
                            />
                        </div>

                        {answer && (
                            <div className="mt-4">
                                <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
                                    {t('knowledgeBase.askModal.answer')}
                                </label>
                                <div className="p-4 bg-gray-50 dark:bg-neutral-700 rounded-lg">
                                    <p className="text-gray-800 dark:text-gray-200 whitespace-pre-wrap">{answer}</p>
                                </div>
                            </div>
                        )}
                    </div>

                    <div className="flex justify-end gap-3 mt-6">
                        <Button
                            variant="outline"
                            onClick={() => {
                                setIsAskModalOpen(false);
                                setQuestion('');
                                setAnswer('');
                            }}
                        >
                            {t('knowledgeBase.askModal.close')}
                        </Button>
                        <Button
                            variant="primary"
                            type="submit"
                            leftIcon={<MessageSquare className="w-4 h-4" />}
                        >
                            {t('knowledgeBase.askModal.ask')}
                        </Button>
                    </div>
                </form>
            </Modal>
        </PageContainer>
    );
};

export default KnowledgeBase;
