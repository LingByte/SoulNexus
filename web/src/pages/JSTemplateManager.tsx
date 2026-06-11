import { useState, useEffect, useRef, Suspense, lazy } from 'react'
import { Input as ArcoInput, Select as ArcoSelect, Modal, Tag } from '@arco-design/web-react'
import { useI18nStore } from '@/stores/i18nStore'
import Button from '@/components/UI/Button.tsx'
import { jsTemplateService, JSTemplate, CreateJSTemplateForm } from '@/api/jsTemplate'
import { Plus, Code, Eye, AlertCircle, Maximize2, Minimize2, FileText, Trash2, Edit3, ArrowLeft } from 'lucide-react'
import { showAlert } from '@/utils/notification'
import { useDebounce } from '@/hooks/useDebounce'
import { validateJavaScript } from '@/utils/jsValidator'
import { getApiBaseURL } from '@/config/apiConfig'
import MarkdownPreview from '@/components/UI/MarkdownPreview.tsx'

const MonacoEditor = lazy(() => import('@monaco-editor/react'))

const JSTemplateManager = () => {
    const { t } = useI18nStore()
    const [templates, setTemplates] = useState<JSTemplate[]>([])
    const [isCreating, setIsCreating] = useState(false)
    const [isEditing, setIsEditing] = useState(false)
    const [editingTemplate, setEditingTemplate] = useState<JSTemplate | null>(null)
    const [searchTerm, setSearchTerm] = useState('')
    const [filterType, setFilterType] = useState<'all' | 'default' | 'custom'>('all')
    const [isLoading, setIsLoading] = useState(false)
    const [newTemplate, setNewTemplate] = useState<CreateJSTemplateForm>({
        name: '', type: 'custom', content: '', usage: ''
    })
    const [validationError, setValidationError] = useState<string | null>(null)
    const [isCodeEditorFullscreen, setIsCodeEditorFullscreen] = useState(false)
    const [isMarkdownEditorFullscreen, setIsMarkdownEditorFullscreen] = useState(false)

    const debouncedContent = useDebounce(newTemplate.content, 500)
    const iframeRef = useRef<HTMLIFrameElement>(null)

    useEffect(() => { fetchTemplates() }, [])

    const fetchTemplates = async () => {
        setIsLoading(true)
        try {
            const response = await jsTemplateService.getTemplates({ page: 1, limit: 100 })
            if (response.code === 200) setTemplates(response.data.data)
        } catch (error) { console.error(error) }
        finally { setIsLoading(false) }
    }

    const filteredTemplates = templates.filter(template => {
        const matchesSearch = template.name.toLowerCase().includes(searchTerm.toLowerCase()) ||
            template.content.toLowerCase().includes(searchTerm.toLowerCase())
        const matchesFilter = filterType === 'all' || template.type === filterType
        return matchesSearch && matchesFilter
    })

    useEffect(() => {
        const timer = setTimeout(() => { if (searchTerm.trim()) { handleSearch() } else { fetchTemplates() } }, 500)
        return () => clearTimeout(timer)
    }, [searchTerm])

    const handleSearch = async () => {
        setIsLoading(true)
        try {
            const response = await jsTemplateService.searchTemplates({ keyword: searchTerm, page: 1, limit: 100 })
            if (response.code === 200) setTemplates(response.data.data)
        } catch (error) { console.error(error) }
        finally { setIsLoading(false) }
    }

    useEffect(() => {
        if (!isCreating && !isEditing) return
        if (!debouncedContent) { setValidationError(null); return }
        const result = validateJavaScript(debouncedContent)
        setValidationError(result.error || null)
    }, [debouncedContent, isCreating, isEditing])

    useEffect(() => {
        if (iframeRef.current && (isCreating || isEditing)) updateIframe()
    }, [debouncedContent, isCreating, isEditing])

    const updateIframe = () => {
        if (!iframeRef.current) return
        const html = `<!DOCTYPE html><html><head><meta charset="UTF-8"><meta http-equiv="Content-Security-Policy" content="default-src 'self' 'unsafe-inline' 'unsafe-eval'; script-src 'self' 'unsafe-inline' 'unsafe-eval' https://store.lingecho.com;"><style>body{margin:0;font-family:sans-serif;}*{box-sizing:border-box;}</style></head><body><div id="app"></div><script>window.SERVER_BASE='${getApiBaseURL()}';window.ASSISTANT_NAME='preview';window.AGENT_ID=1;window.LANGUAGE='zh-cn';window.ASSISTANT_ID='preview';</script><script src="https://store.lingecho.com/sdk/browser/soulnexus-browser-sdk-v0.1.1.js"></script><script>document.addEventListener('lingllm-ready',()=>{try{${(debouncedContent || '').replace(/<\/script>/g, '').replace(/<!--/g, '').replace(/<script/g, '')}}catch(e){document.body.innerHTML='<pre style="color:red;padding:16px;">'+e.message+'</pre>'}});</script></body></html>`
        iframeRef.current.srcdoc = html
    }

    const handleCreateTemplate = () => {
        setNewTemplate({
            name: '', type: 'custom',
            content: '// 在此编写您的JavaScript代码\nconst box = document.createElement("div");\nbox.style.cssText="width:200px;height:200px;background:#3b82f6;margin:20px auto;border-radius:12px;display:flex;align-items:center;justify-content:center;color:#fff;font-size:18px;font-weight:bold";\nbox.textContent="Hello!";\ndocument.body.appendChild(box);',
            usage: ''
        })
        setEditingTemplate(null)
        setIsCreating(true)
        setValidationError(null)
    }

    const handleEditTemplate = (template: JSTemplate) => {
        setEditingTemplate(template)
        setNewTemplate({ name: template.name, type: template.type, content: template.content, usage: template.usage || '' })
        setIsEditing(true)
        setValidationError(null)
    }

    const handleSaveNewTemplate = async () => {
        if (!newTemplate.name || !newTemplate.content) { showAlert(t('jsTemplate.messages.fillRequired'), 'warning'); return }
        if (validationError) { showAlert(t('jsTemplate.messages.fixSyntax'), 'warning'); return }
        try {
            if (isEditing && editingTemplate) {
                await jsTemplateService.updateTemplate(editingTemplate.id, newTemplate)
                showAlert(t('jsTemplate.messages.updateSuccess'), 'success')
            } else {
                await jsTemplateService.createTemplate(newTemplate)
                showAlert(t('jsTemplate.messages.createSuccess'), 'success')
            }
            handleCancelCreate()
            fetchTemplates()
        } catch (error) {
            showAlert(isEditing ? t('jsTemplate.messages.updateFailed') : t('jsTemplate.messages.createFailed'), 'error')
        }
    }

    const handleDeleteTemplate = async (id: string) => {
        Modal.confirm({
            title: t('jsTemplate.messages.deleteConfirm'),
            onOk: async () => {
                try { await jsTemplateService.deleteTemplate(id); showAlert(t('jsTemplate.messages.deleteSuccess'), 'success'); fetchTemplates() }
                catch (error) { showAlert(t('jsTemplate.messages.deleteFailed'), 'error') }
            }
        })
    }

    const handleCancelCreate = () => { setIsCreating(false); setIsEditing(false); setEditingTemplate(null); setValidationError(null); setNewTemplate({ name: '', type: 'custom', content: '', usage: '' }) }

    const isEditorOpen = isCreating || isEditing

    // ==================== Editor Page (Full Screen) ====================
    if (isEditorOpen) {
        return (
            <div className="w-full h-full flex flex-col bg-gray-50 dark:bg-gray-950">
                {/* Editor Header */}
                <div className="flex items-center justify-between px-4 py-2.5 bg-white dark:bg-gray-900 border-b border-gray-200 dark:border-gray-800 shrink-0">
                    <div className="flex items-center gap-3">
                        <button onClick={handleCancelCreate} className="p-1.5 hover:bg-gray-100 dark:hover:bg-gray-800 rounded-lg transition-colors">
                            <ArrowLeft className="w-4 h-4" />
                        </button>
                        <div>
                            <h2 className="text-sm font-semibold">{isEditing ? t('jsTemplate.editModal.title') : t('jsTemplate.createModal.title')}</h2>
                            <p className="text-xs text-gray-400">{isEditing ? t('jsTemplate.editModal.desc') : t('jsTemplate.createModal.desc')}</p>
                        </div>
                    </div>
                    <Button size="sm" variant="primary" onClick={handleSaveNewTemplate}
                        disabled={!newTemplate.name || !newTemplate.content || !!validationError}>
                        {isEditing ? t('jsTemplate.update') : t('jsTemplate.saveTemplate')}
                    </Button>
                </div>

                {/* Editor Body: Left Form + Right Preview+Code */}
                <div className="flex-1 flex overflow-hidden">
                    {/* Left: Form */}
                    <div className="w-64 border-r border-gray-200 dark:border-gray-800 bg-white dark:bg-gray-900 p-4 space-y-4 overflow-y-auto shrink-0">
                        <div>
                            <label className="text-xs font-medium text-gray-500 mb-1 block">{t('jsTemplate.templateName')}</label>
                            <ArcoInput placeholder={t('jsTemplate.templateNamePlaceholder')} value={newTemplate.name}
                                onChange={(e) => setNewTemplate({ ...newTemplate, name: e })} />
                        </div>
                        <div>
                            <label className="text-xs font-medium text-gray-500 mb-1 block">{t('jsTemplate.usage')}</label>
                            <ArcoInput placeholder={t('jsTemplate.usage')} value={newTemplate.usage}
                                onChange={(e) => setNewTemplate({ ...newTemplate, usage: e })} />
                        </div>
                        {validationError && (
                            <div className="p-2 bg-red-50 dark:bg-red-900/20 rounded-lg flex items-start gap-2">
                                <AlertCircle className="w-4 h-4 text-red-500 shrink-0 mt-0.5" />
                                <span className="text-xs text-red-600 dark:text-red-400">{validationError}</span>
                            </div>
                        )}
                        <div className="pt-2 border-t border-gray-100 dark:border-gray-800">
                            <button onClick={() => setIsCodeEditorFullscreen(!isCodeEditorFullscreen)}
                                className="flex items-center gap-2 text-xs text-gray-500 hover:text-gray-700 dark:hover:text-gray-300">
                                {isCodeEditorFullscreen ? <Minimize2 className="w-3.5 h-3.5" /> : <Maximize2 className="w-3.5 h-3.5" />}
                                {isCodeEditorFullscreen ? 'Exit Fullscreen' : 'Fullscreen Editor'}
                            </button>
                        </div>
                    </div>

                    {/* Right: Preview + Code Editor */}
                    <div className="flex-1 flex flex-col overflow-hidden">
                        {/* Preview */}
                        <div className="h-1/2 border-b border-gray-200 dark:border-gray-800 flex flex-col">
                            <div className="px-3 py-2 flex items-center gap-2 border-b border-gray-100 dark:border-gray-800 bg-white dark:bg-gray-900 shrink-0">
                                <Eye className="w-3.5 h-3.5 text-gray-400" />
                                <span className="text-xs font-medium">{t('jsTemplate.preview.label')}</span>
                            </div>
                            <div className="flex-1 p-2 overflow-hidden bg-white dark:bg-gray-950">
                                <iframe ref={iframeRef} className="w-full h-full border border-gray-200 dark:border-gray-700 rounded" title="Preview" sandbox="allow-scripts allow-same-origin" />
                            </div>
                        </div>
                        {/* Code Editor */}
                        <div className="flex-1 flex flex-col overflow-hidden relative">
                            <div className="px-3 py-2 flex items-center gap-2 border-b border-gray-100 dark:border-gray-800 bg-white dark:bg-gray-900 shrink-0">
                                <Code className="w-3.5 h-3.5 text-gray-400" />
                                <span className="text-xs font-medium">JavaScript</span>
                                <span className="ml-auto text-xs text-gray-400">{newTemplate.content.length} chars</span>
                            </div>
                            <div className="flex-1 overflow-hidden">
                                <Suspense fallback={<div className="flex items-center justify-center h-full text-gray-400 text-xs">Loading editor...</div>}>
                                    <MonacoEditor height="100%" language="javascript" value={newTemplate.content}
                                        onChange={(v) => setNewTemplate({ ...newTemplate, content: v || '' })}
                                        options={{ minimap: { enabled: false }, scrollBeyondLastLine: false, fontSize: 13, lineNumbers: 'on', wordWrap: 'on', automaticLayout: true, theme: 'vs-dark', tabSize: 2 }} />
                                </Suspense>
                            </div>
                            {validationError && (
                                <div className="px-3 py-1.5 bg-red-50 dark:bg-red-900/20 border-t border-red-200 dark:border-red-800 flex items-center gap-2 shrink-0">
                                    <AlertCircle className="w-3.5 h-3.5 text-red-500" />
                                    <span className="text-xs text-red-600 dark:text-red-400">{validationError}</span>
                                </div>
                            )}
                        </div>
                    </div>
                </div>

                {/* Code Editor Fullscreen */}
                {isCodeEditorFullscreen && (
                    <div className="fixed inset-0 z-50 bg-gray-900 flex flex-col">
                        <div className="h-10 bg-gray-800 border-b border-gray-700 flex items-center justify-between px-4 shrink-0">
                            <span className="text-xs text-gray-300">{newTemplate.name || 'Untitled'} — JavaScript</span>
                            <div className="flex items-center gap-2">
                                {validationError ? (
                                    <span className="text-xs text-red-400 flex items-center gap-1"><AlertCircle className="w-3 h-3" />{validationError}</span>
                                ) : (
                                    <span className="text-xs text-green-400">OK</span>
                                )}
                                <button onClick={() => setIsCodeEditorFullscreen(false)} className="p-1 hover:bg-gray-700 rounded text-gray-400 hover:text-white">
                                    <Minimize2 className="w-4 h-4" />
                                </button>
                            </div>
                        </div>
                        <div className="flex-1">
                            <MonacoEditor height="100%" language="javascript" value={newTemplate.content}
                                onChange={(v) => setNewTemplate({ ...newTemplate, content: v || '' })} theme="vs-dark"
                                options={{ minimap: { enabled: true }, scrollBeyondLastLine: false, fontSize: 13, lineNumbers: 'on', wordWrap: 'on', automaticLayout: true, tabSize: 2, mouseWheelZoom: true, smoothScrolling: true }} />
                        </div>
                    </div>
                )}

                {/* Markdown Editor Fullscreen */}
                {isMarkdownEditorFullscreen && (
                    <div className="fixed inset-0 z-50 bg-white dark:bg-gray-900 flex">
                        <div className="w-1/2 flex flex-col border-r border-gray-200 dark:border-gray-700">
                            <div className="h-10 border-b border-gray-200 dark:border-gray-700 flex items-center justify-between px-4 shrink-0">
                                <span className="text-xs font-medium flex items-center gap-2"><FileText className="w-3.5 h-3.5" />Markdown</span>
                                <button onClick={() => setIsMarkdownEditorFullscreen(false)} className="p-1 hover:bg-gray-100 dark:hover:bg-gray-700 rounded text-gray-400">
                                    <Minimize2 className="w-4 h-4" />
                                </button>
                            </div>
                            <div className="flex-1">
                                <MonacoEditor height="100%" language="markdown" value={newTemplate.usage}
                                    onChange={(v) => setNewTemplate({ ...newTemplate, usage: v || '' })} theme="vs-light"
                                    options={{ minimap: { enabled: false }, scrollBeyondLastLine: false, fontSize: 13, lineNumbers: 'on', wordWrap: 'on', automaticLayout: true, tabSize: 2 }} />
                            </div>
                        </div>
                        <div className="w-1/2 flex flex-col">
                            <div className="h-10 border-b border-gray-200 dark:border-gray-700 flex items-center px-4 shrink-0">
                                <span className="text-xs font-medium flex items-center gap-2"><Eye className="w-3.5 h-3.5" />Preview</span>
                            </div>
                            <div className="flex-1 overflow-y-auto p-6">
                                {newTemplate.usage ? <MarkdownPreview content={newTemplate.usage} className="prose prose-sm max-w-none" />
                                    : <div className="text-gray-400 text-xs text-center py-16">Markdown content</div>}
                            </div>
                        </div>
                    </div>
                )}
            </div>
        )
    }

    // ==================== List Page ====================
    return (
        <div className="w-full flex flex-col h-full">
            {/* Header */}
            <div className="flex items-center justify-between px-6 py-4 border-b border-gray-200 dark:border-gray-800">
                <h1 className="text-lg font-semibold">{t('jsTemplate.title')}</h1>
                <Button size="sm" leftIcon={<Plus className="w-4 h-4" />} onClick={handleCreateTemplate}>{t('jsTemplate.create')}</Button>
            </div>

            {/* Toolbar */}
            <div className="px-6 py-3 flex items-center gap-3 border-b border-gray-100 dark:border-gray-800">
                <ArcoInput.Search placeholder={t('jsTemplate.searchPlaceholder')} value={searchTerm} onChange={setSearchTerm} className="w-64" />
                <ArcoSelect value={filterType} onChange={(v) => setFilterType(v as any)} className="w-32"
                    options={[
                        { label: t('jsTemplate.filter.all'), value: 'all' },
                        { label: t('jsTemplate.filter.default'), value: 'default' },
                        { label: t('jsTemplate.filter.custom'), value: 'custom' },
                    ]}
                />
            </div>

            {/* Content */}
            <div className="flex-1 overflow-auto px-6 py-4">
                {isLoading ? (
                    <div className="text-center py-16 text-gray-400">
                        <div className="animate-spin w-8 h-8 border-2 border-gray-300 border-t-gray-600 rounded-full mx-auto mb-3" />
                        <p className="text-sm">{t('jsTemplate.loading')}</p>
                    </div>
                ) : filteredTemplates.length === 0 ? (
                    <div className="text-center py-16">
                        <Code className="w-12 h-12 text-gray-300 mx-auto mb-3" />
                        <p className="text-gray-500 mb-4">{searchTerm ? t('jsTemplate.noMatch') : t('jsTemplate.empty')}</p>
                        <Button leftIcon={<Plus className="w-4 h-4" />} onClick={handleCreateTemplate}>{t('jsTemplate.createFirst')}</Button>
                    </div>
                ) : (
                    <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-4">
                        {filteredTemplates.map(template => (
                            <div key={template.id} className="group border border-gray-200 dark:border-gray-700 rounded-lg hover:border-blue-300 dark:hover:border-blue-600 transition-colors bg-white dark:bg-gray-900">
                                <div className="p-4">
                                    <div className="flex items-start justify-between mb-2">
                                        <div className="flex items-center gap-2 min-w-0">
                                            <div className="w-8 h-8 rounded bg-blue-50 dark:bg-blue-900/30 flex items-center justify-center shrink-0">
                                                <Code className="w-4 h-4 text-blue-500" />
                                            </div>
                                            <div className="min-w-0">
                                                <h3 className="font-medium text-sm truncate">{template.name}</h3>
                                                <Tag size="small" color={template.type === 'default' ? 'blue' : 'green'} className="mt-1">
                                                    {template.type === 'default' ? t('jsTemplate.type.default') : t('jsTemplate.type.custom')}
                                                </Tag>
                                            </div>
                                        </div>
                                        {template.type === 'custom' && (
                                            <div className="flex gap-0.5 opacity-0 group-hover:opacity-100 transition-opacity shrink-0">
                                                <button onClick={(e) => { e.stopPropagation(); handleEditTemplate(template) }}
                                                    className="p-1 hover:bg-gray-100 rounded text-gray-400 hover:text-blue-500">
                                                    <Edit3 className="w-3.5 h-3.5" />
                                                </button>
                                                <button onClick={(e) => { e.stopPropagation(); handleDeleteTemplate(template.id) }}
                                                    className="p-1 hover:bg-gray-100 rounded text-gray-400 hover:text-red-500">
                                                    <Trash2 className="w-3.5 h-3.5" />
                                                </button>
                                            </div>
                                        )}
                                    </div>
                                </div>
                                <div className="px-4 py-2.5 border-t border-gray-100 dark:border-gray-800 flex justify-between items-center text-xs text-gray-400">
                                    <span>{t('jsTemplate.updated')}: {new Date(template.updated_at).toLocaleDateString()}</span>
                                    <span>#{template.id.slice(-4)}</span>
                                </div>
                            </div>
                        ))}
                    </div>
                )}
            </div>
        </div>
    )
}

export default JSTemplateManager
