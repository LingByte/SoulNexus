import { useState, useEffect, useRef, Suspense, lazy } from 'react'
import { motion, AnimatePresence } from 'framer-motion'
import { useNavigate } from 'react-router-dom'
import { useI18nStore } from '@/stores/i18nStore'
import Card, { CardFooter, CardHeader, CardTitle } from '@/components/UI/Card.tsx'
import Button from '@/components/UI/Button.tsx'
import Input from '@/components/UI/Input.tsx'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/UI/Select.tsx'
import { jsTemplateService, JSTemplate, CreateJSTemplateForm } from '@/api/jsTemplate'
import { ArrowLeft, Plus, Code, Eye, AlertCircle, Maximize2, Minimize2, FileText } from 'lucide-react'
import { showAlert } from '@/utils/notification'
import { useDebounce } from '@/hooks/useDebounce'
import { validateJavaScript } from '@/utils/jsValidator'
import { getApiBaseURL } from '@/config/apiConfig'
import MarkdownPreview from '@/components/UI/MarkdownPreview.tsx'

// 懒加载Monaco Editor，优化首次加载性能
const MonacoEditor = lazy(() => import('@monaco-editor/react'))

const JSTemplateManager = () => {
    const { t } = useI18nStore()
    const navigate = useNavigate()
    const [templates, setTemplates] = useState<JSTemplate[]>([])
    const [isCreating, setIsCreating] = useState(false)
    const [isEditing, setIsEditing] = useState(false)
    const [editingTemplate, setEditingTemplate] = useState<JSTemplate | null>(null)
    const [searchTerm, setSearchTerm] = useState('')
    const [filterType, setFilterType] = useState<'all' | 'default' | 'custom'>('all')
    const [isLoading, setIsLoading] = useState(false)
    const [newTemplate, setNewTemplate] = useState({
        name: '',
        type: 'custom' as 'default' | 'custom',
        content: '// 在此编写您的JavaScript代码\n// 代码将实时注入到左侧预览页面\n\n// 示例: 创建一个彩色方块\nconst box = document.createElement("div");\nbox.style.width = "200px";\nbox.style.height = "200px";\nbox.style.backgroundColor = "#3b82f6";\nbox.style.margin = "20px auto";\nbox.style.borderRadius = "12px";\nbox.style.display = "flex";\nbox.style.alignItems = "center";\nbox.style.justifyContent = "center";\nbox.style.color = "white";\nbox.style.fontSize = "18px";\nbox.style.fontWeight = "bold";\nbox.textContent = "Hello, JS Template!";\ndocument.body.appendChild(box);',
        usage: ''
    })
    const [validationError, setValidationError] = useState<string | null>(null)
    const [isCodeEditorFullscreen, setIsCodeEditorFullscreen] = useState(false)
    const [isMarkdownEditorFullscreen, setIsMarkdownEditorFullscreen] = useState(false)
    
    // 使用防抖来优化预览性能
    const debouncedContent = useDebounce(newTemplate.content, 500)

    // 获取模板数据
    useEffect(() => {
        fetchTemplates()
    }, [])

    const fetchTemplates = async () => {
        setIsLoading(true)
        try {
            const response = await jsTemplateService.getTemplates({ page: 1, limit: 100 })
            if (response.code === 200) {
                setTemplates(response.data.data)
            }
        } catch (error) {
            console.error('获取模板失败:', error)
        } finally {
            setIsLoading(false)
        }
    }

    const filteredTemplates = templates.filter(template => {
        const matchesSearch = template.name.toLowerCase().includes(searchTerm.toLowerCase()) ||
            template.content.toLowerCase().includes(searchTerm.toLowerCase())
        const matchesFilter = filterType === 'all' || template.type === filterType
        return matchesSearch && matchesFilter
    })

    // 搜索功能
    const handleSearch = async () => {
        if (!searchTerm.trim()) {
            fetchTemplates()
            return
        }

        setIsLoading(true)
        try {
            const response = await jsTemplateService.searchTemplates({
                keyword: searchTerm,
                page: 1,
                limit: 100
            })
            if (response.code === 200) {
                setTemplates(response.data.data)
            } else {
                console.error('搜索模板失败:', response.msg)
            }
        } catch (error) {
            console.error('搜索模板失败:', error)
        } finally {
            setIsLoading(false)
        }
    }

    // 监听搜索词变化，延迟搜索
    useEffect(() => {
        const timer = setTimeout(() => {
            if (searchTerm.trim()) {
                handleSearch()
            } else {
                fetchTemplates()
            }
        }, 500)

        return () => clearTimeout(timer)
    }, [searchTerm])


    const handleCreateTemplate = () => {
        setIsCreating(true)
        setIsEditing(false)
        setEditingTemplate(null)
    }

    const handleEditTemplate = (template: JSTemplate) => {
        setEditingTemplate(template)
        setNewTemplate({
            name: template.name,
            type: template.type,
            content: template.content,
            usage: template.usage || ''
        })
        setIsEditing(true)
        setIsCreating(false)
    }

    const handleSaveNewTemplate = async () => {
        // 验证JS代码语法
        const validation = validateJavaScript(newTemplate.content)
        if (!validation.isValid) {
            const errorMsg = validation.error || '代码语法错误'
            const lineInfo = validation.line ? ` (第 ${validation.line} 行)` : ''
            setValidationError(errorMsg + lineInfo)
            showAlert(`代码验证失败: ${errorMsg}${lineInfo}`, 'error')
            return
        }
        
        setValidationError(null)
        
        try {
            const templateData: CreateJSTemplateForm = {
                name: newTemplate.name,
                type: newTemplate.type,
                content: newTemplate.content,
                usage: newTemplate.usage
            }

            if (isEditing && editingTemplate) {
                // 编辑模式 - 调用更新API
                const response = await jsTemplateService.updateTemplate(editingTemplate.id, templateData)
                if (response.code === 200) {
                    setIsEditing(false)
                    setEditingTemplate(null)
                    setNewTemplate({
                        name: '',
                        type: 'custom',
                        content: '// 在此编写您的JavaScript代码\n// 代码将实时注入到左侧预览页面\n\n// 示例: 创建一个彩色方块\nconst box = document.createElement("div");\nbox.style.width = "200px";\nbox.style.height = "200px";\nbox.style.backgroundColor = "#3b82f6";\nbox.style.margin = "20px auto";\nbox.style.borderRadius = "12px";\nbox.style.display = "flex";\nbox.style.alignItems = "center";\nbox.style.justifyContent = "center";\nbox.style.color = "white";\nbox.style.fontSize = "18px";\nbox.style.fontWeight = "bold";\nbox.textContent = "Hello, JS Template!";\ndocument.body.appendChild(box);',
                        usage: ''
                    })
                    fetchTemplates()
                } else {
                    console.error('更新模板失败:', response.msg)
                    showAlert(t('jsTemplate.messages.updateFailed') + ': ' + response.msg, 'error')
                }
            } else {
                // 创建模式 - 调用创建API
                const response = await jsTemplateService.createTemplate(templateData)
                if (response.code === 200 || response.code === 201) {
                    setIsCreating(false)
                    setNewTemplate({
                        name: '',
                        type: 'custom',
                        content: '// 在此编写您的JavaScript代码\n// 代码将实时注入到左侧预览页面\n\n// 示例: 创建一个彩色方块\nconst box = document.createElement("div");\nbox.style.width = "200px";\nbox.style.height = "200px";\nbox.style.backgroundColor = "#3b82f6";\nbox.style.margin = "20px auto";\nbox.style.borderRadius = "12px";\nbox.style.display = "flex";\nbox.style.alignItems = "center";\nbox.style.justifyContent = "center";\nbox.style.color = "white";\nbox.style.fontSize = "18px";\nbox.style.fontWeight = "bold";\nbox.textContent = "Hello, JS Template!";\ndocument.body.appendChild(box);',
                        usage: ''
                    })
                    fetchTemplates()
                } else {
                    console.error('创建模板失败:', response.msg)
                    showAlert(t('jsTemplate.messages.createFailed') + ': ' + response.msg, 'error')
                }
            }
        } catch (error) {
            console.error(isEditing ? '更新模板失败:' : '创建模板失败:', error)
            showAlert(isEditing ? t('jsTemplate.messages.updateFailed') : t('jsTemplate.messages.createFailed'), 'error')
        }
    }

    const handleDeleteTemplate = async (templateId: string) => {
        console.log('handleDeleteTemplate called with templateId:', templateId)

        if (!templateId) {
            console.error('Template ID is undefined or empty')
            showAlert(t('jsTemplate.messages.invalidId'), 'error')
            return
        }

        if (!confirm(t('jsTemplate.messages.deleteConfirm'))) {
            return
        }

        try {
            console.log('Calling deleteTemplate with ID:', templateId)
            const response = await jsTemplateService.deleteTemplate(templateId)
            if (response.code === 200) {
                // 重新获取模板列表
                fetchTemplates()
                // 如果删除的是当前编辑的模板，清空编辑状态
                if (editingTemplate?.id === templateId) {
                    setEditingTemplate(null)
                    setIsEditing(false)
                }
            } else {
                console.error('删除模板失败:', response.msg)
                showAlert(t('jsTemplate.messages.deleteFailed') + ': ' + response.msg, 'error')
            }
        } catch (error) {
            console.error('删除模板失败:', error)
            showAlert(t('jsTemplate.messages.deleteFailed'), 'error')
        }
    }

    const handleCancelCreate = () => {
        setIsCreating(false)
        setIsEditing(false)
        setEditingTemplate(null)
        setNewTemplate({
            name: '',
            type: 'custom',
            content: '// 在此编写您的JavaScript代码\n// 代码将实时注入到左侧预览页面\n\n// 示例: 创建一个彩色方块\nconst box = document.createElement("div");\nbox.style.width = "200px";\nbox.style.height = "200px";\nbox.style.backgroundColor = "#3b82f6";\nbox.style.margin = "20px auto";\nbox.style.borderRadius = "12px";\nbox.style.display = "flex";\nbox.style.alignItems = "center";\nbox.style.justifyContent = "center";\nbox.style.color = "white";\nbox.style.fontSize = "18px";\nbox.style.fontWeight = "bold";\nbox.textContent = "Hello, JS Template!";\ndocument.body.appendChild(box);',
            usage: ''
        })
    }

    // 用于更新预览页面
    const iframeRef = useRef<HTMLIFrameElement | null>(null)

    const updateIframe = () => {
        if (iframeRef.current) {
            const iframeDoc = iframeRef.current.contentDocument
            if (iframeDoc) {
                // 验证代码语法
                const validation = validateJavaScript(debouncedContent)
                
                // 转义代码内容，防止XSS攻击（转义特殊字符，特别是</script>标签）
                const escapedCode = debouncedContent
                    .replace(/<\/script>/gi, '<\\/script>')
                    .replace(/<!--/g, '<\\!--')
                    .replace(/<script/gi, '<\\script')
                
                // 获取API基础URL，用于加载SDK
                const apiBaseURL = getApiBaseURL()
                // 从API URL提取基础URL（去掉/api后缀）
                const baseURL = apiBaseURL.replace(/\/api$/, '')
                // 使用固定的CDN地址而不是本地地址
                const sdkPath = `https://store.lingecho.com/uploads/buckets/default/lingecho-sdk.js`
                
                // 模拟模板变量（用于预览环境）
                const mockAssistantID = 1
                const mockAssistantName = '预览助手'
                const mockBaseURL = baseURL
                
                iframeDoc.open()
                
                // 构建安全的HTML内容，包含SDK加载
                const htmlContent = `<!DOCTYPE html>
<html lang="zh-CN">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <meta http-equiv="Content-Security-Policy" content="default-src 'self' 'unsafe-inline' 'unsafe-eval'; script-src 'unsafe-inline' 'unsafe-eval' ${baseURL} https:; style-src 'unsafe-inline'; img-src 'self' data: https:; font-src 'self' data:; connect-src ${baseURL} ws: wss:;">
    <title>JS Code Preview</title>
</head>
<body style="margin: 0; padding: 20px; background: #f9fafb; font-family: system-ui, -apple-system, sans-serif;">
    <div id="preview-content"></div>
    
    <!-- 加载LingEcho SDK -->
    <script>
        // 定义模拟的模板变量（用于预览环境）- 使用var避免重复声明错误
        if (typeof SERVER_BASE === 'undefined') {
            var SERVER_BASE = '${mockBaseURL}';
        }
        if (typeof ASSISTANT_NAME === 'undefined') {
            var ASSISTANT_NAME = '${mockAssistantName}';
        }
        if (typeof BaseURL === 'undefined') {
            var BaseURL = '${mockBaseURL}';
        }
        if (typeof Name === 'undefined') {
            var Name = '${mockAssistantName}';
        }
        if (typeof AssistantID === 'undefined') {
            var AssistantID = ${mockAssistantID};
        }
        
        // 加载SDK
        (function() {
            if (typeof LingEchoSDK === 'undefined') {
                const script = document.createElement('script');
                script.src = '${sdkPath}';
                script.async = false;
                script.onload = function() {
                    console.log('[Preview] LingEcho SDK loaded');
                    // 等待SDK类定义后创建实例
                    (function waitForSDK() {
                        if (typeof LingEchoSDK !== 'undefined') {
                            // 确保实例已创建
                            if (!window.lingEcho) {
                                try {
                                    window.lingEcho = new LingEchoSDK({
                                        baseURL: SERVER_BASE,
                                        assistantName: ASSISTANT_NAME,
                                        assistantId: AssistantID
                                    });
                                    console.log('[Preview] SDK instance created');
                                } catch (e) {
                                    console.error('[Preview] Failed to create SDK instance:', e);
                                }
                            }
                            window.__LINGECHO_SDK_READY__ = true;
                            if (typeof window.dispatchEvent !== 'undefined') {
                                window.dispatchEvent(new Event('lingecho-sdk-ready'));
                            }
                        } else {
                            setTimeout(waitForSDK, 50);
                        }
                    })();
                };
                script.onerror = function() {
                    console.error('[Preview] Failed to load SDK');
                };
                document.head.appendChild(script);
            } else {
                // SDK已加载，确保实例存在
                if (!window.lingEcho) {
                    try {
                        window.lingEcho = new LingEchoSDK({
                            baseURL: SERVER_BASE,
                            assistantName: ASSISTANT_NAME,
                            assistantId: AssistantID
                        });
                    } catch (e) {
                        console.error('[Preview] Failed to create SDK instance:', e);
                    }
                }
                window.__LINGECHO_SDK_READY__ = true;
            }
        })();
    </script>
    
    ${validation.isValid ? `<script id="preview-script">
        (function() {
            // 等待SDK加载完成后再执行用户代码
            function runUserCode() {
                try {
                    ${escapedCode}
                } catch (error) {
                    const errorDiv = document.createElement('div');
                    errorDiv.style.cssText = 'background: #fee2e2; border: 2px solid #ef4444; border-radius: 8px; padding: 20px; margin: 20px 0; color: #991b1b;';
                    const errorText = document.createTextNode('⚠️ Code Execution Error: ' + error.message);
                    errorDiv.appendChild(errorText);
                    document.getElementById('preview-content').appendChild(errorDiv);
                    console.error('JS Template Error:', error);
                }
            }
            
            // 如果SDK已就绪，直接执行
            if (window.__LINGECHO_SDK_READY__ && window.lingEcho) {
                runUserCode();
            } else {
                // 等待SDK加载
                const maxWait = 10000; // 最多等待10秒
                const startTime = Date.now();
                const checkSDK = setInterval(function() {
                    if (window.__LINGECHO_SDK_READY__ && window.lingEcho) {
                        clearInterval(checkSDK);
                        runUserCode();
                    } else if (Date.now() - startTime > maxWait) {
                        clearInterval(checkSDK);
                        // 超时后仍然执行代码（可能用户代码不依赖SDK）
                        console.warn('[Preview] SDK加载超时，继续执行代码');
                        runUserCode();
                    }
                }, 100);
                
                // 监听SDK就绪事件
                if (typeof window.addEventListener !== 'undefined') {
                    window.addEventListener('lingecho-sdk-ready', function() {
                        clearInterval(checkSDK);
                        runUserCode();
                    }, { once: true });
                }
            }
        })();
    </script>` : `
    <div style="background: #fef3c7; border: 2px solid #f59e0b; border-radius: 8px; padding: 20px; margin: 20px 0; color: #92400e;">
        <h3 style="margin: 0 0 10px 0; font-size: 18px;">⚠️ 语法错误</h3>
        <pre style="white-space: pre-wrap; font-family: monospace; font-size: 14px; margin: 0;">${validation.error || '代码存在语法错误'}</pre>
    </div>`}
</body>
</html>`
                
                iframeDoc.write(htmlContent)
                iframeDoc.close()
            }
        }
    }

    // 使用防抖后的内容来更新预览
    useEffect(() => {
        if (isCreating || isEditing) {
            updateIframe()
        }
    }, [debouncedContent, isCreating, isEditing])
    
    // 实时验证代码语法（不防抖，用于即时反馈）
    useEffect(() => {
        if (isCreating || isEditing) {
            const validation = validateJavaScript(newTemplate.content)
            if (!validation.isValid) {
                const errorMsg = validation.error || '代码语法错误'
                const lineInfo = validation.line ? ` (第 ${validation.line} 行)` : ''
                setValidationError(errorMsg + lineInfo)
            } else {
                setValidationError(null)
            }
        }
    }, [newTemplate.content, isCreating, isEditing])

    return (
        <div className="min-h-screen bg-gradient-to-br from-slate-50 via-blue-50 to-indigo-50 dark:from-slate-900 dark:via-slate-800 dark:to-slate-900">
            {/* 页面头部 */}
            <div className="bg-white/80 dark:bg-slate-800/80 backdrop-blur-sm border-b border-gray-200/50 dark:border-slate-700/50 sticky top-0 z-40">
                <div className="max-w-7xl mx-auto px-4 py-3">
                    <div className="flex items-center justify-between">
                        <div className="flex items-center gap-4">
                            <Button
                                onClick={() => navigate(-1)}
                                variant="outline"
                                size="sm"
                                leftIcon={<ArrowLeft className="w-4 h-4" />}
                                className="border-slate-300 dark:border-slate-600 hover:border-slate-400 dark:hover:border-slate-500 transition-all duration-200"
                            >
                                {t('jsTemplate.back')}
                            </Button>
                            <div className="h-8 w-px bg-slate-300 dark:bg-slate-600"></div>
                            <div className="relative pl-4">
                                <motion.div
                                  layoutId="pageTitleIndicator"
                                  className="absolute left-0 top-1/2 -translate-y-1/2 w-1 h-6 bg-primary rounded-r-full"
                                  transition={{ type: 'spring', bounce: 0.2, duration: 0.3 }}
                                />
                                <h1 className="text-2xl font-bold">{t('jsTemplate.title')}</h1>
                                <p className="text-sm mt-1">
                                    {t('jsTemplate.desc')}
                                </p>
                            </div>
                        </div>
                        <div className="flex items-center gap-3">
                            <div className="text-right">
                                <div className="text-sm font-medium">
                                    {t('jsTemplate.templateCount', { count: filteredTemplates.length })}
                                </div>
                                <div className="text-xs">
                                    {t('jsTemplate.totalCount', { count: templates.length })}
                                </div>
                            </div>
                            <Button
                                onClick={handleCreateTemplate}
                                leftIcon={<Plus className="w-4 h-4" />}
                                className="px-4 py-2.5 rounded-lg shadow-lg hover:shadow-xl transition-all duration-200 font-medium"
                            >
                                {t('jsTemplate.create')}
                            </Button>
                        </div>
                    </div>
                </div>
            </div>

            <div className="max-w-7xl mx-auto px-4 py-6">
                {/* 搜索和过滤栏 */}
                <div className="bg-white/60 dark:bg-slate-800/60 backdrop-blur-sm rounded-xl border border-white/20 dark:border-slate-700/30 p-4 mb-6 shadow-lg">
                    <div className="flex flex-col lg:flex-row justify-between items-start lg:items-center gap-4">
                        <div className="flex-1 max-w-md">
                            <div className="relative">
                                <svg className="absolute left-3 top-1/2 transform -translate-y-1/2 w-4 h-4 text-slate-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z" />
                                </svg>
                                <Input
                                    placeholder={t('jsTemplate.searchPlaceholder')}
                                    value={searchTerm}
                                    onChange={(e) => setSearchTerm(e.target.value)}
                                    className="pl-10 h-9 bg-white/80 dark:bg-slate-700/80 border-slate-200 dark:border-slate-600 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent transition-all duration-200"
                                />
                            </div>
                        </div>
                        <div className="flex items-center gap-3">
                            <div className="text-sm">{t('jsTemplate.filterType')}</div>
                            <Select value={filterType} onValueChange={(value: string) => setFilterType(value as 'all' | 'default' | 'custom')}>
                                <SelectTrigger className="w-32 h-9 bg-white/80 dark:bg-slate-700/80 border-slate-200 dark:border-slate-600 rounded-lg">
                                    <SelectValue>
                                        {filterType === 'all' && t('jsTemplate.filter.all')}
                                        {filterType === 'default' && t('jsTemplate.filter.default')}
                                        {filterType === 'custom' && t('jsTemplate.filter.custom')}
                                    </SelectValue>
                                </SelectTrigger>
                                <SelectContent className="z-50 max-h-48 overflow-y-auto scrollbar-thin scrollbar-thumb-gray-300 dark:scrollbar-thumb-gray-600 scrollbar-track-transparent">
                                    <SelectItem value="all">{t('jsTemplate.filter.all')}</SelectItem>
                                    <SelectItem value="default">{t('jsTemplate.filter.default')}</SelectItem>
                                    <SelectItem value="custom">{t('jsTemplate.filter.custom')}</SelectItem>
                                </SelectContent>
                            </Select>
                        </div>
                    </div>
                </div>

                {/* 主体内容区域 */}
                <div className="w-full">
                    {/* 模板列表 */}
                    <div className="w-full">
                        {isLoading ? (
                            <div className="flex flex-col items-center justify-center py-12">
                                <div className="relative">
                                    <div className="animate-spin rounded-full h-16 w-16 border-4 border-slate-200 dark:border-slate-700"></div>
                                    <div className="animate-spin rounded-full h-16 w-16 border-4 border-blue-500 border-t-transparent absolute top-0 left-0"></div>
                                </div>
                                <p className="mt-4 font-medium">{t('jsTemplate.loading')}</p>
                            </div>
                        ) : (
                            <>
                                <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-3 gap-4">
                                    <AnimatePresence>
                                        {filteredTemplates.map((template: JSTemplate) => (
                                            <motion.div
                                                key={template.id}
                                                initial={{ opacity: 0, y: 20, scale: 0.95 }}
                                                animate={{ opacity: 1, y: 0, scale: 1 }}
                                                exit={{ opacity: 0, y: -20, scale: 0.95 }}
                                                transition={{ duration: 0.4, ease: "easeOut" }}
                                                className="group min-w-[220px]"
                                            >
                                                <Card
                                                    className="h-full flex flex-col transition-shadow duration-300 hover:shadow-xl bg-white/80 dark:bg-slate-800/80 backdrop-blur-sm rounded-xl border border-white/20 dark:border-slate-700/30 hover:border-slate-300 dark:hover:border-slate-600"
                                                >
                                                    <CardHeader className="p-4 pb-3 flex-1">
                                                        <div className="flex justify-between items-start">
                                                            <div className="flex-1 min-w-0 basis-0">
                                                                <div className="flex items-center gap-3 mb-2">
                                                                    <div className="w-8 h-8 rounded-md bg-slate-100 dark:bg-slate-700 flex items-center justify-center">
                                                                        <Code className="w-4 h-4 text-slate-600 dark:text-slate-300" />
                                                                    </div>
                                                                    <div className="flex-1 min-w-0">
                                                                        <CardTitle className="text-lg font-bold leading-tight break-words">
                                                                            {template.name}
                                                                        </CardTitle>
                                                                        <p className="text-sm mt-1">
                                                                            {template.type === 'default' ? t('jsTemplate.type.default') : t('jsTemplate.type.custom')}
                                                                        </p>
                                                                    </div>
                                                                </div>
                                                            </div>
                                                            <div className="flex items-center gap-1 flex-shrink-0">
                                                                {template.type === 'custom' && (
                                                                    <div className="flex items-center gap-1">
                                                                        <button
                                                                            onClick={(e) => {
                                                                                e.stopPropagation()
                                                                                handleEditTemplate(template)
                                                                            }}
                                                                            className="opacity-0 group-hover:opacity-100 p-1.5 rounded-lg transition-opacity duration-200 group/edit"
                                                                            title={t('jsTemplate.edit')}
                                                                        >
                                                                            <svg className="w-4 h-4 group-hover/edit:scale-110 transition-transform" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                                                                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M11 5H6a2 2 0 00-2 2v11a2 2 0 002 2h11a2 2 0 002-2v-5m-1.414-9.414a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z" />
                                                                            </svg>
                                                                        </button>
                                                                        <button
                                                                            onClick={(e) => {
                                                                                e.stopPropagation()
                                                                                handleDeleteTemplate(template.id)
                                                                            }}
                                                                            className="opacity-0 group-hover:opacity-100 p-1.5 rounded-lg transition-opacity duration-200 group/delete"
                                                                            title={t('jsTemplate.delete')}
                                                                        >
                                                                            <svg className="w-4 h-4 group-hover/delete:scale-110 transition-transform" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                                                                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
                                                                            </svg>
                                                                        </button>
                                                                    </div>
                                                                )}
                                                            </div>
                                                        </div>
                                                    </CardHeader>
                                                    <CardFooter className="px-4 py-3 bg-slate-50/50 dark:bg-slate-700/30 border-t border-slate-200/50 dark:border-slate-600/30 mt-auto">
                                                        <div className="flex justify-between items-center text-sm w-full">
                                                            <div>
                                                                {t('jsTemplate.updated')}: {new Date(template.updated_at).toLocaleDateString()}
                                                            </div>
                                                            <div className="text-slate-400">
                                                                #{template.id.slice(-4)}
                                                            </div>
                                                        </div>
                                                    </CardFooter>
                                                </Card>
                                            </motion.div>
                                        ))}
                                    </AnimatePresence>
                                </div>
                                {filteredTemplates.length === 0 && !isLoading && (
                                    <div className="flex flex-col items-center justify-center py-12 text-center">
                                        <div className="relative mb-6">
                                            <div className="w-24 h-24 bg-gradient-to-br from-slate-100 to-slate-200 dark:from-slate-700 dark:to-slate-800 rounded-full flex items-center justify-center">
                                                <Code className="w-12 h-12 text-slate-400 dark:text-slate-500" />
                                            </div>
                                            <div className="absolute -top-2 -right-2 w-8 h-8 rounded-full flex items-center justify-center">
                                                <span className="text-sm font-bold">?</span>
                                            </div>
                                        </div>
                                        <h3 className="text-xl font-bold mb-2">
                                            {searchTerm ? t('jsTemplate.noMatch') : t('jsTemplate.empty')}
                                        </h3>
                                        <p className="mb-6 max-w-md">
                                            {searchTerm
                                                ? t('jsTemplate.tryOtherKeywords')
                                                : t('jsTemplate.emptyDesc')}
                                        </p>
                                        <Button
                                            onClick={handleCreateTemplate}
                                            leftIcon={<Plus className="w-4 h-4" />}
                                            className="px-6 py-2.5 rounded-lg shadow-lg hover:shadow-xl transition-all duration-200 font-medium"
                                        >
                                            {t('jsTemplate.createFirst')}
                                        </Button>
                                    </div>
                                )}
                            </>
                        )}
                    </div>
                </div>
            </div>

            {/* 全屏创建/编辑模板抽屉 */}
            <AnimatePresence>
                {(isCreating || isEditing) && (
                    <motion.div
                        initial={{ opacity: 0 }}
                        animate={{ opacity: 1 }}
                        exit={{ opacity: 0 }}
                        className="fixed inset-0 z-50 overflow-hidden"
                    >
                        {/* 遮罩层 */}
                        <motion.div
                            initial={{ opacity: 0 }}
                            animate={{ opacity: 1 }}
                            exit={{ opacity: 0 }}
                            className="absolute inset-0 bg-black/50 backdrop-blur-sm"
                            onClick={handleCancelCreate}
                        />

                        {/* 全屏抽屉内容 */}
                        <motion.div
                            initial={{ x: '100%' }}
                            animate={{ x: 0 }}
                            exit={{ x: '100%' }}
                            transition={{ duration: 0.2, ease: 'easeOut' }}
                            className="absolute inset-0 bg-white/95 dark:bg-slate-900/95 backdrop-blur-sm shadow-2xl flex flex-col"
                        >
                            {/* 抽屉头部 */}
                            <div className="flex-shrink-0 border-b border-slate-200/50 dark:border-slate-700/50 bg-white/80 dark:bg-slate-900/80 backdrop-blur-sm px-6 py-4">
                                <div className="flex items-center justify-between">
                                    <div className="flex items-center gap-4">
                                        <div className="p-2 rounded-lg shadow-lg">
                                            <Code className="w-6 h-6" />
                                        </div>
                                        <div>
                                            <h2 className="text-2xl font-bold">
                                                {isEditing ? t('jsTemplate.editModal.title') : t('jsTemplate.createModal.title')}
                                            </h2>
                                            <p className="text-sm mt-1">
                                                {isEditing ? t('jsTemplate.editModal.desc') : t('jsTemplate.createModal.desc')}{t('jsTemplate.modal.descSuffix')}
                                            </p>
                                        </div>
                                    </div>
                                    <div className="flex items-center gap-3">
                                        <Button
                                            onClick={handleCancelCreate}
                                            variant="outline"
                                            size="sm"
                                            leftIcon={<ArrowLeft className="w-4 h-4" />}
                                            className="border-slate-300 dark:border-slate-600 hover:border-slate-400 dark:hover:border-slate-500 transition-all duration-200"
                                        >
                                            {t('jsTemplate.cancel')}
                                        </Button>
                                        <Button
                                            onClick={handleSaveNewTemplate}
                                            size="sm"
                                            leftIcon={<Plus className="w-4 h-4" />}
                                            className="px-4 py-2 rounded-lg shadow-lg hover:shadow-xl transition-all duration-200 font-medium"
                                            disabled={!newTemplate.name || !newTemplate.content || !!validationError}
                                        >
                                            {isEditing ? t('jsTemplate.update') : t('jsTemplate.saveTemplate')}
                                        </Button>
                                    </div>
                                </div>
                            </div>

                            {/* 左右分栏主体内容 */}
                            <div className="flex-1 flex overflow-hidden">
                                {/* 左侧: 实时预览 */}
                                <div className="w-1/2 border-r border-slate-200/50 dark:border-slate-700/50 bg-slate-50/50 dark:bg-slate-800/50 flex flex-col">
                                    <div className="flex-shrink-0 px-6 py-4 border-b border-slate-200/50 dark:border-slate-700/50 bg-white/80 dark:bg-slate-900/80 backdrop-blur-sm">
                                        <h3 className="text-sm font-semibold flex items-center gap-2">
                                            <Eye className="w-4 h-4" />
                                            {t('jsTemplate.preview.label')}
                                        </h3>
                                        <p className="text-xs mt-1">
                                            {t('jsTemplate.preview.desc')}
                                        </p>
                                    </div>
                                    <div className="flex-1 p-4 overflow-hidden">
                                        <iframe
                                            ref={iframeRef}
                                            className="w-full h-full border border-slate-300 dark:border-slate-600 bg-white rounded-lg shadow-sm"
                                            title={t('jsTemplate.preview.title')}
                                            sandbox="allow-scripts allow-same-origin"
                                            referrerPolicy="no-referrer"
                                        />
                                    </div>
                                </div>

                                {/* 右侧: 编辑区域 */}
                                <div className="w-1/2 flex flex-col overflow-hidden">
                                    <div className="flex-1 overflow-y-auto">
                                        <div className="p-6 space-y-6">
                                            <div>
                                                <label className="block text-sm font-semibold mb-1">
                                                    {t('jsTemplate.templateName')}
                                                </label>
                                                <Input
                                                    placeholder={t('jsTemplate.templateNamePlaceholder')}
                                                    value={newTemplate.name}
                                                    onChange={(e) => setNewTemplate({ ...newTemplate, name: e.target.value })}
                                                    className="h-9 bg-white/80 dark:bg-slate-700/80 border-slate-200 dark:border-slate-600 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent transition-all duration-200"
                                                />
                                            </div>

                                            <div>
                                                <label className="block text-sm font-semibold mb-1">
                                                    {t('jsTemplate.usage')}
                                                </label>
                                                <div className="border border-slate-200 dark:border-slate-600 overflow-hidden rounded-lg shadow-sm">
                                                    <div className="flex items-center justify-between p-2 bg-slate-50 dark:bg-slate-800 border-b border-slate-200 dark:border-slate-600">
                                                        <div className="text-xs text-slate-600 dark:text-slate-400">
                                                            Markdown 使用说明编辑器
                                                        </div>
                                                        <Button
                                                            variant="ghost"
                                                            size="xs"
                                                            onClick={() => setIsMarkdownEditorFullscreen(!isMarkdownEditorFullscreen)}
                                                            className="h-6 px-2 text-xs"
                                                        >
                                                            {isMarkdownEditorFullscreen ? (
                                                                <Minimize2 className="w-3 h-3" />
                                                            ) : (
                                                                <Maximize2 className="w-3 h-3" />
                                                            )}
                                                        </Button>
                                                    </div>
                                                </div>
                                            </div>

                                            <div>
                                                <label className="block text-sm font-semibold mb-1">
                                                    {t('jsTemplate.templateContent')}
                                                </label>
                                                <div className="border border-slate-200 dark:border-slate-600 overflow-hidden rounded-lg shadow-sm">
                                                    <div className="flex items-center justify-between p-2 bg-slate-50 dark:bg-slate-800 border-b border-slate-200 dark:border-slate-600">
                                                        <div className="text-xs text-slate-600 dark:text-slate-400">
                                                            JavaScript 代码编辑器
                                                        </div>
                                                        <Button
                                                            variant="ghost"
                                                            size="xs"
                                                            onClick={() => setIsCodeEditorFullscreen(!isCodeEditorFullscreen)}
                                                            className="h-4 px-2 text-xs"
                                                        >
                                                            {isCodeEditorFullscreen ? (
                                                                <Minimize2 className="w-3 h-3" />
                                                            ) : (
                                                                <Maximize2 className="w-3 h-3" />
                                                            )}
                                                        </Button>
                                                    </div>
                                                    <Suspense fallback={
                                                        <div className="h-[300px] flex items-center justify-center bg-slate-50 dark:bg-slate-800">
                                                            <div className="text-center">
                                                                <div className="relative inline-block">
                                                                    <div className="animate-spin rounded-full h-8 w-8 border-4 border-slate-200 dark:border-slate-700"></div>
                                                                    <div className="animate-spin rounded-full h-8 w-8 border-4 border-blue-500 border-t-transparent absolute top-0 left-0"></div>
                                                                </div>
                                                                <p className="mt-3 text-sm text-slate-600 dark:text-slate-400">{t('jsTemplate.preview.loading')}</p>
                                                            </div>
                                                        </div>
                                                    }>
                                                        <MonacoEditor
                                                            height="350px"
                                                            language="javascript"
                                                            value={newTemplate.content}
                                                            onChange={(value) => setNewTemplate({ ...newTemplate, content: value || '' })}
                                                            options={{
                                                                minimap: { enabled: false },
                                                                scrollBeyondLastLine: false,
                                                                fontSize: 14,
                                                                lineNumbers: 'on',
                                                                wordWrap: 'on',
                                                                automaticLayout: true,
                                                                theme: 'vs-dark',
                                                                tabSize: 2,
                                                                formatOnPaste: true,
                                                                formatOnType: true,
                                                                suggestOnTriggerCharacters: true,
                                                                quickSuggestions: true,
                                                            }}
                                                        />
                                                    </Suspense>
                                                </div>
                                                {validationError && (
                                                    <div className="mt-2 flex items-start gap-2 p-3 bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 rounded-lg">
                                                        <AlertCircle className="w-5 h-5 text-red-600 dark:text-red-400 flex-shrink-0 mt-0.5" />
                                                        <div className="flex-1">
                                                            <p className="text-sm font-medium text-red-800 dark:text-red-200">语法错误</p>
                                                            <p className="text-xs text-red-600 dark:text-red-300 mt-1">{validationError}</p>
                                                        </div>
                                                    </div>
                                                )}
                                            </div>
                                        </div>
                                    </div>
                                </div>
                            </div>
                        </motion.div>
                    </motion.div>
                )}
            </AnimatePresence>

            {/* 代码编辑器全屏模式 */}
            <AnimatePresence>
                {isCodeEditorFullscreen && (isCreating || isEditing) && (
                    <>
                        {/* 背景遮罩 */}
                        <motion.div
                            initial={{ opacity: 0 }}
                            animate={{ opacity: 1 }}
                            exit={{ opacity: 0 }}
                            onClick={() => setIsCodeEditorFullscreen(false)}
                            className="fixed inset-0 bg-black/80 backdrop-blur-sm z-[100]"
                        />
                        {/* 全屏编辑器 */}
                        <motion.div
                            initial={{ opacity: 0, scale: 0.95 }}
                            animate={{ opacity: 1, scale: 1 }}
                            exit={{ opacity: 0, scale: 0.95 }}
                            transition={{ duration: 0.2 }}
                            className="fixed inset-4 z-[110] bg-slate-900 rounded-lg shadow-2xl flex flex-col overflow-hidden"
                            onClick={(e) => e.stopPropagation()}
                        >
                            {/* 标题栏 */}
                            <div className="h-14 bg-slate-800 border-b border-slate-700 flex items-center justify-between px-6 flex-shrink-0">
                                <div className="flex items-center gap-3">
                                    <div className="p-2 rounded-lg bg-blue-500/15">
                                        <Code className="w-5 h-5 text-blue-400" />
                                    </div>
                                    <div>
                                        <h3 className="text-base font-semibold text-slate-200">
                                            {newTemplate.name || '未命名模板'} - JavaScript 代码编辑器
                                        </h3>
                                        <p className="text-xs text-slate-400 mt-0.5">
                                            全屏编辑模式
                                        </p>
                                    </div>
                                </div>
                                <div className="flex items-center gap-2">
                                    {validationError && (
                                        <div className="flex items-center gap-2 px-3 py-1.5 bg-red-500/10 border border-red-500/20 rounded-lg">
                                            <AlertCircle className="w-4 h-4 text-red-400" />
                                            <span className="text-xs text-red-300">语法错误</span>
                                        </div>
                                    )}
                                    {!validationError && (
                                        <div className="flex items-center gap-2 px-3 py-1.5 bg-green-500/10 border border-green-500/20 rounded-lg">
                                            <div className="w-2 h-2 rounded-full bg-green-400"></div>
                                            <span className="text-xs text-green-300">语法正确</span>
                                        </div>
                                    )}
                                    <Button
                                        variant="ghost"
                                        size="sm"
                                        onClick={() => setIsCodeEditorFullscreen(false)}
                                        className="flex items-center gap-2 text-slate-300 hover:text-white"
                                    >
                                        <Minimize2 className="w-4 h-4" />
                                        <span>退出全屏</span>
                                    </Button>
                                </div>
                            </div>
                            
                            {/* 编辑器区域 */}
                            <div className="flex-1 overflow-hidden" style={{ height: 'calc(100vh - 8rem)' }}>
                                <Suspense fallback={
                                    <div className="h-full flex items-center justify-center bg-slate-900">
                                        <div className="text-center">
                                            <div className="animate-spin rounded-full h-8 w-8 border-4 border-slate-700 border-t-blue-500 mx-auto mb-3"></div>
                                            <p className="text-sm text-slate-400">加载代码编辑器...</p>
                                        </div>
                                    </div>
                                }>
                                    <MonacoEditor
                                        height="100%"
                                        language="javascript"
                                        value={newTemplate.content}
                                        onChange={(value) => setNewTemplate({ ...newTemplate, content: value || '' })}
                                        theme="vs-dark"
                                        options={{
                                            minimap: { enabled: true },
                                            scrollBeyondLastLine: false,
                                            fontSize: 14,
                                            lineNumbers: 'on',
                                            wordWrap: 'on',
                                            automaticLayout: true,
                                            tabSize: 2,
                                            formatOnPaste: true,
                                            formatOnType: true,
                                            suggestOnTriggerCharacters: true,
                                            quickSuggestions: true,
                                            contextmenu: true,
                                            mouseWheelZoom: true,
                                            smoothScrolling: true,
                                            cursorBlinking: 'smooth',
                                            cursorSmoothCaretAnimation: 'on',
                                        }}
                                    />
                                </Suspense>
                            </div>
                            
                            {/* 底部错误提示 */}
                            {validationError && (
                                <div className="h-auto bg-red-900/20 border-t border-red-800/50 px-6 py-3 flex items-start gap-3">
                                    <AlertCircle className="w-5 h-5 text-red-400 flex-shrink-0 mt-0.5" />
                                    <div className="flex-1">
                                        <p className="text-sm font-medium text-red-200">语法错误</p>
                                        <p className="text-xs text-red-300 mt-1">{validationError}</p>
                                    </div>
                                </div>
                            )}
                        </motion.div>
                    </>
                )}
            </AnimatePresence>

            {/* Markdown编辑器全屏模式 */}
            <AnimatePresence>
                {isMarkdownEditorFullscreen && (isCreating || isEditing) && (
                    <>
                        {/* 背景遮罩 */}
                        <motion.div
                            initial={{ opacity: 0 }}
                            animate={{ opacity: 1 }}
                            exit={{ opacity: 0 }}
                            onClick={() => setIsMarkdownEditorFullscreen(false)}
                            className="fixed inset-0 bg-black/80 backdrop-blur-sm z-[100]"
                        />
                        {/* 全屏编辑器 */}
                        <motion.div
                            initial={{ opacity: 0, scale: 0.95 }}
                            animate={{ opacity: 1, scale: 1 }}
                            exit={{ opacity: 0, scale: 0.95 }}
                            transition={{ duration: 0.2 }}
                            className="fixed inset-4 z-[110] bg-white dark:bg-slate-900 rounded-lg shadow-2xl flex overflow-hidden"
                            onClick={(e) => e.stopPropagation()}
                        >
                            {/* 左侧：Markdown编辑器 */}
                            <div className="w-1/2 flex flex-col border-r border-slate-200 dark:border-slate-700">
                                {/* 标题栏 */}
                                <div className="h-14 bg-slate-50 dark:bg-slate-800 border-b border-slate-200 dark:border-slate-700 flex items-center justify-between px-6 flex-shrink-0">
                                    <div className="flex items-center gap-3">
                                        <div className="p-2 rounded-lg bg-green-500/15">
                                            <FileText className="w-5 h-5 text-green-600 dark:text-green-400" />
                                        </div>
                                        <div>
                                            <h3 className="text-base font-semibold text-slate-800 dark:text-slate-200">
                                                Markdown 编辑器
                                            </h3>
                                            <p className="text-xs text-slate-500 dark:text-slate-400 mt-0.5">
                                                编写使用说明文档
                                            </p>
                                        </div>
                                    </div>
                                    <Button
                                        variant="ghost"
                                        size="sm"
                                        onClick={() => setIsMarkdownEditorFullscreen(false)}
                                        className="flex items-center gap-2 text-slate-600 dark:text-slate-300 hover:text-slate-800 dark:hover:text-white"
                                    >
                                        <Minimize2 className="w-4 h-4" />
                                        <span>退出全屏</span>
                                    </Button>
                                </div>
                                
                                {/* 编辑器区域 */}
                                <div className="flex-1 overflow-hidden">
                                    <Suspense fallback={
                                        <div className="h-full flex items-center justify-center bg-slate-50 dark:bg-slate-800">
                                            <div className="text-center">
                                                <div className="animate-spin rounded-full h-8 w-8 border-4 border-slate-200 dark:border-slate-700 border-t-green-500 mx-auto mb-3"></div>
                                                <p className="text-sm text-slate-600 dark:text-slate-400">加载Markdown编辑器...</p>
                                            </div>
                                        </div>
                                    }>
                                        <MonacoEditor
                                            height="100%"
                                            language="markdown"
                                            value={newTemplate.usage}
                                            onChange={(value) => setNewTemplate({ ...newTemplate, usage: value || '' })}
                                            theme="vs-light"
                                            options={{
                                                minimap: { enabled: false },
                                                scrollBeyondLastLine: false,
                                                fontSize: 14,
                                                lineNumbers: 'on',
                                                wordWrap: 'on',
                                                automaticLayout: true,
                                                tabSize: 2,
                                                formatOnPaste: true,
                                                formatOnType: true,
                                                suggestOnTriggerCharacters: true,
                                                quickSuggestions: true,
                                                contextmenu: true,
                                                mouseWheelZoom: true,
                                                smoothScrolling: true,
                                                cursorBlinking: 'smooth',
                                                cursorSmoothCaretAnimation: 'on',
                                            }}
                                        />
                                    </Suspense>
                                </div>
                            </div>

                            {/* 右侧：实时预览 */}
                            <div className="w-1/2 flex flex-col bg-slate-50/50 dark:bg-slate-800/50">
                                {/* 预览标题栏 */}
                                <div className="h-14 bg-slate-100 dark:bg-slate-800 border-b border-slate-200 dark:border-slate-700 flex items-center px-6 flex-shrink-0">
                                    <div className="flex items-center gap-3">
                                        <div className="p-2 rounded-lg bg-blue-500/15">
                                            <Eye className="w-5 h-5 text-blue-600 dark:text-blue-400" />
                                        </div>
                                        <div>
                                            <h3 className="text-base font-semibold text-slate-800 dark:text-slate-200">
                                                实时预览
                                            </h3>
                                            <p className="text-xs text-slate-500 dark:text-slate-400 mt-0.5">
                                                Markdown 渲染效果
                                            </p>
                                        </div>
                                    </div>
                                </div>
                                
                                {/* 预览区域 */}
                                <div className="flex-1 overflow-y-auto p-6">
                                    {newTemplate.usage ? (
                                        <MarkdownPreview 
                                            content={newTemplate.usage}
                                            className="prose prose-sm max-w-none"
                                        />
                                    ) : (
                                        <div className="flex items-center justify-center h-full text-slate-500 dark:text-slate-400">
                                            <div className="text-center">
                                                <FileText className="w-12 h-12 mx-auto mb-4 opacity-50" />
                                                <p className="text-sm">在左侧编辑器中输入Markdown内容</p>
                                                <p className="text-xs mt-1 opacity-75">支持标题、列表、代码块、链接等格式</p>
                                            </div>
                                        </div>
                                    )}
                                </div>
                            </div>
                        </motion.div>
                    </>
                )}
            </AnimatePresence>
        </div>
    )
}

export default JSTemplateManager