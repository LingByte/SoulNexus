// src/pages/KnowledgeBaseDetail.tsx
import React, { useState, useEffect } from 'react';
import { ArrowLeft, Upload, Search, FileText, ChevronDown, ChevronUp } from 'lucide-react';
import { useParams, useNavigate } from 'react-router-dom';
import { showAlert } from '@/utils/notification'
import { useI18nStore } from '@/stores/i18nStore'
import { listKnowledgeBaseContent, uploadKnowledgeBase } from '@/api/knowledge';
import Button from '@/components/UI/Button'
import Input from '@/components/UI/Input'
import Modal, { ModalContent, ModalFooter } from '@/components/UI/Modal'
import FileUpload from '@/components/UI/FileUpload'
import PageContainer from '@/components/Layout/PageContainer'
import FadeIn from '@/components/Animations/FadeIn'

interface Document {
    source: string;
    chunk_count: number;
    chunks: Array<{
        content: string;
        score: number;
        metadata?: Record<string, any>;
    }>;
}

interface ContentData {
    knowledge_key: string;
    document_count: number;
    total_chunks: number;
    documents: Document[];
}

const KnowledgeBaseDetail = () => {
    const { t } = useI18nStore()
    const navigate = useNavigate()
    const { knowledgeKey } = useParams<{ knowledgeKey: string }>()
    
    const [contentData, setContentData] = useState<ContentData | null>(null)
    const [isLoading, setIsLoading] = useState(true)
    const [searchTerm, setSearchTerm] = useState('')
    const [expandedDocs, setExpandedDocs] = useState<Set<string>>(new Set())
    const [expandedChunks, setExpandedChunks] = useState<Set<string>>(new Set())
    const [isUploadModalOpen, setIsUploadModalOpen] = useState(false)
    const [uploadFiles, setUploadFiles] = useState<File[]>([])

    useEffect(() => {
        if (knowledgeKey) {
            fetchContent()
        }
    }, [knowledgeKey])

    const fetchContent = async () => {
        try {
            setIsLoading(true)
            if (!knowledgeKey) return
            
            const response = await listKnowledgeBaseContent(knowledgeKey)
            if (response.code === 200) {
                setContentData(response.data)
            } else {
                showAlert(response.msg || t('knowledgeBase.messages.fetchFailed'), 'error', t('knowledgeBase.messages.fetchFailed'))
            }
        } catch (error) {
            console.error('获取知识库内容失败:', error)
            showAlert(t('knowledgeBase.messages.fetchFailed'), 'error', t('knowledgeBase.messages.fetchFailed'))
        } finally {
            setIsLoading(false)
        }
    }

    const toggleDocExpanded = (source: string) => {
        const newSet = new Set(expandedDocs)
        if (newSet.has(source)) {
            newSet.delete(source)
        } else {
            newSet.add(source)
        }
        setExpandedDocs(newSet)
    }

    const toggleChunkExpanded = (chunkId: string) => {
        const newSet = new Set(expandedChunks)
        if (newSet.has(chunkId)) {
            newSet.delete(chunkId)
        } else {
            newSet.add(chunkId)
        }
        setExpandedChunks(newSet)
    }

    const handleUploadFileChange = (files: File[]) => {
        setUploadFiles(files)
    }

    const handleUploadSubmit = async (e: React.FormEvent) => {
        e.preventDefault()
        if (uploadFiles.length === 0 || !knowledgeKey) return

        try {
            let successCount = 0
            let failCount = 0

            for (const file of uploadFiles) {
                try {
                    const response = await uploadKnowledgeBase({
                        file: file,
                        knowledgeKey: knowledgeKey
                    })

                    if (response.code === 200) {
                        successCount++
                    } else {
                        failCount++
                    }
                } catch (error) {
                    failCount++
                }
            }

            if (successCount > 0) {
                showAlert(
                    `成功上传 ${successCount} 个文件${failCount > 0 ? `，失败 ${failCount} 个` : ''}`,
                    'success',
                    t('knowledgeBase.messages.uploadSuccess')
                )
                setIsUploadModalOpen(false)
                setUploadFiles([])
                await fetchContent()
            } else {
                showAlert('所有文件上传失败', 'error', t('knowledgeBase.messages.uploadFailed'))
            }
        } catch (error) {
            console.error('上传文件失败:', error)
            showAlert(t('knowledgeBase.messages.uploadError'), 'error', t('knowledgeBase.messages.uploadFailed'))
        }
    }

    const filteredDocuments = contentData?.documents.filter(doc =>
        doc.source.toLowerCase().includes(searchTerm.toLowerCase()) ||
        doc.chunks.some(chunk => chunk.content.toLowerCase().includes(searchTerm.toLowerCase()))
    ) || []

    return (
        <PageContainer maxWidth="full" padding="lg">
            {/* 页面头部 */}
            <FadeIn direction="down">
                <div className="mb-8">
                    <div className="flex items-center gap-4 mb-4">
                        <Button
                            variant="ghost"
                            size="sm"
                            leftIcon={<ArrowLeft className="w-4 h-4" />}
                            onClick={() => navigate('/knowledge')}
                        >
                            返回
                        </Button>
                    </div>
                    <h1 className="text-3xl font-bold text-gray-900 dark:text-white mb-2">
                        知识库详情
                    </h1>
                    {contentData && (
                        <p className="text-gray-600 dark:text-gray-400">
                            文档数: {contentData.document_count} | 总段落数: {contentData.total_chunks}
                        </p>
                    )}
                </div>
            </FadeIn>

            {/* 操作栏 */}
            <FadeIn direction="down" delay={0.1}>
                <div className="flex flex-col sm:flex-row justify-between items-start sm:items-center gap-4 mb-6">
                    <div className="relative w-full sm:w-80">
                        <Input
                            type="text"
                            placeholder="搜索文档或内容..."
                            value={searchTerm}
                            onChange={(e) => setSearchTerm(e.target.value)}
                            leftIcon={<Search className="w-4 h-4" />}
                        />
                    </div>

                    <Button
                        variant="primary"
                        leftIcon={<Upload className="w-4 h-4" />}
                        onClick={() => setIsUploadModalOpen(true)}
                    >
                        上传文件
                    </Button>
                </div>
            </FadeIn>

            {/* 内容列表 */}
            {isLoading ? (
                <div className="flex justify-center items-center h-64">
                    <div className="animate-spin rounded-full h-12 w-12 border-t-2 border-b-2 border-purple-500"></div>
                </div>
            ) : filteredDocuments.length === 0 ? (
                <div className="text-center py-12">
                    <FileText className="w-12 h-12 text-gray-400 mx-auto mb-4" />
                    <p className="text-gray-600 dark:text-gray-400">
                        {searchTerm ? '没有找到匹配的内容' : '知识库为空'}
                    </p>
                </div>
            ) : (
                <div className="space-y-4">
                    {filteredDocuments.map((doc, docIndex) => (
                        <FadeIn key={docIndex} direction="up" delay={docIndex * 0.05}>
                            <div className="border border-gray-200 dark:border-gray-700 rounded-lg overflow-hidden">
                                {/* 文档头部 */}
                                <button
                                    onClick={() => toggleDocExpanded(doc.source)}
                                    className="w-full px-4 py-3 bg-gray-50 dark:bg-gray-800 hover:bg-gray-100 dark:hover:bg-gray-700 flex items-center justify-between transition-colors"
                                >
                                    <div className="flex items-center gap-3 flex-1 min-w-0">
                                        <FileText className="w-5 h-5 text-purple-600 dark:text-purple-400 flex-shrink-0" />
                                        <div className="text-left min-w-0">
                                            <p className="font-medium text-gray-900 dark:text-white truncate">
                                                {doc.source}
                                            </p>
                                            <p className="text-sm text-gray-500 dark:text-gray-400">
                                                {doc.chunk_count} 个段落
                                            </p>
                                        </div>
                                    </div>
                                    {expandedDocs.has(doc.source) ? (
                                        <ChevronUp className="w-5 h-5 text-gray-600 dark:text-gray-400 flex-shrink-0" />
                                    ) : (
                                        <ChevronDown className="w-5 h-5 text-gray-600 dark:text-gray-400 flex-shrink-0" />
                                    )}
                                </button>

                                {/* 段落列表 */}
                                {expandedDocs.has(doc.source) && (
                                    <div className="border-t border-gray-200 dark:border-gray-700 divide-y divide-gray-200 dark:divide-gray-700">
                                        {doc.chunks.map((chunk, chunkIndex) => {
                                            const chunkId = `${docIndex}-${chunkIndex}`
                                            const isExpanded = expandedChunks.has(chunkId)
                                            return (
                                                <div key={chunkIndex} className="p-4">
                                                    <button
                                                        onClick={() => toggleChunkExpanded(chunkId)}
                                                        className="w-full text-left flex items-start justify-between gap-3 hover:opacity-75 transition-opacity"
                                                    >
                                                        <div className="flex-1 min-w-0">
                                                            <p className="text-sm font-medium text-gray-900 dark:text-white line-clamp-2">
                                                                {chunk.content}
                                                            </p>
                                                            <div className="flex items-center gap-2 mt-2">
                                                                <span className="text-xs bg-purple-100 dark:bg-purple-900/30 text-purple-700 dark:text-purple-300 px-2 py-1 rounded">
                                                                    相关度: {(chunk.score * 100).toFixed(1)}%
                                                                </span>
                                                            </div>
                                                        </div>
                                                        {isExpanded ? (
                                                            <ChevronUp className="w-4 h-4 text-gray-600 dark:text-gray-400 flex-shrink-0 mt-1" />
                                                        ) : (
                                                            <ChevronDown className="w-4 h-4 text-gray-600 dark:text-gray-400 flex-shrink-0 mt-1" />
                                                        )}
                                                    </button>

                                                    {/* 完整内容 */}
                                                    {isExpanded && (
                                                        <div className="mt-3 p-3 bg-gray-50 dark:bg-gray-800 rounded text-sm text-gray-700 dark:text-gray-300 whitespace-pre-wrap break-words">
                                                            {chunk.content}
                                                        </div>
                                                    )}

                                                    {/* 元数据 */}
                                                    {chunk.metadata && Object.keys(chunk.metadata).length > 0 && (
                                                        <div className="mt-2 text-xs text-gray-500 dark:text-gray-400">
                                                            {Object.entries(chunk.metadata).map(([key, value]) => (
                                                                <div key={key}>
                                                                    {key}: {String(value)}
                                                                </div>
                                                            ))}
                                                        </div>
                                                    )}
                                                </div>
                                            )
                                        })}
                                    </div>
                                )}
                            </div>
                        </FadeIn>
                    ))}
                </div>
            )}

            {/* 上传模态框 */}
            <Modal
                isOpen={isUploadModalOpen}
                onClose={() => {
                    setIsUploadModalOpen(false)
                    setUploadFiles([])
                }}
                title="上传文件到知识库"
                size="md"
            >
                <form onSubmit={handleUploadSubmit}>
                    <ModalContent>
                        <div className="space-y-4">
                            <FileUpload
                                onFileSelect={handleUploadFileChange}
                                accept=".pdf,.doc,.docx,.txt,.md"
                                multiple={true}
                                maxSize={50}
                                maxFiles={10}
                                label="选择文件"
                                className="w-full"
                            />
                            {uploadFiles.length > 0 && (
                                <div className="space-y-2">
                                    <p className="text-sm font-medium text-gray-700 dark:text-gray-300">
                                        已选择 {uploadFiles.length} 个文件
                                    </p>
                                    <div className="max-h-48 overflow-y-auto space-y-2">
                                        {uploadFiles.map((file, index) => (
                                            <div key={index} className="p-2 bg-blue-50 dark:bg-blue-900/20 rounded border border-blue-200 dark:border-blue-800">
                                                <div className="flex items-center gap-2">
                                                    <FileText className="w-4 h-4 text-blue-600 dark:text-blue-400 flex-shrink-0" />
                                                    <div className="flex-1 min-w-0">
                                                        <p className="text-xs font-medium text-blue-900 dark:text-blue-100 truncate">
                                                            {file.name}
                                                        </p>
                                                        <p className="text-xs text-blue-700 dark:text-blue-300">
                                                            {(file.size / 1024 / 1024).toFixed(2)} MB
                                                        </p>
                                                    </div>
                                                </div>
                                            </div>
                                        ))}
                                    </div>
                                </div>
                            )}
                        </div>
                    </ModalContent>
                    <ModalFooter>
                        <Button
                            variant="outline"
                            onClick={() => {
                                setIsUploadModalOpen(false)
                                setUploadFiles([])
                            }}
                        >
                            取消
                        </Button>
                        <Button
                            variant="primary"
                            type="submit"
                            leftIcon={<Upload className="w-4 h-4" />}
                            disabled={uploadFiles.length === 0}
                        >
                            上传 ({uploadFiles.length})
                        </Button>
                    </ModalFooter>
                </form>
            </Modal>
        </PageContainer>
    );
};

export default KnowledgeBaseDetail;
