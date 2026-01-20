import React, { useState, useEffect } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import { useI18nStore } from '@/stores/i18nStore'
import { 
  getAssistantTools, 
  getAssistant,
  createAssistantTool, 
  updateAssistantTool, 
  deleteAssistantTool,
  testAssistantTool,
  type AssistantTool,
  type CreateToolForm,
  type UpdateToolForm,
  type Assistant
} from '@/api/assistant';
import { showAlert } from '@/utils/notification';
import { 
  Plus, Edit2, Trash2, Save, Code, Settings, CheckCircle2, XCircle,
  ArrowLeft, Play, Loader2, TestTube, Wrench, Sparkles
} from 'lucide-react';
import Button from '@/components/UI/Button';
import Card from '@/components/UI/Card';
import Modal, { ModalHeader, ModalTitle, ModalContent, ModalFooter } from '@/components/UI/Modal';
import EmptyState from '@/components/UI/EmptyState';

const AssistantTools: React.FC = () => {
  const { t } = useI18nStore()
  const { id } = useParams<{ id: string }>();
  const assistantId = id ? parseInt(id) : 0;
  const navigate = useNavigate();

  const [assistant, setAssistant] = useState<Assistant | null>(null);
  const [tools, setTools] = useState<AssistantTool[]>([]);
  const [loading, setLoading] = useState(false);
  const [showAddModal, setShowAddModal] = useState(false);
  const [showTestModal, setShowTestModal] = useState(false);
  const [testingTool, setTestingTool] = useState<AssistantTool | null>(null);
  const [testInput, setTestInput] = useState<string>('{}');
  const [testResult, setTestResult] = useState<string>('');
  const [isTesting, setIsTesting] = useState(false);
  const [editingTool, setEditingTool] = useState<AssistantTool | null>(null);
  const [formData, setFormData] = useState<CreateToolForm>({
    name: '',
    description: '',
    parameters: '{\n  "type": "object",\n  "properties": {},\n  "required": []\n}',
    code: '',
    webhookUrl: '',
    enabled: true,
  });

  // JSON Schema 模板
  const jsonTemplates = {
    weather: {
      name: 'get_weather',
      description: '获取指定城市的当前天气信息',
      parameters: JSON.stringify({
        type: 'object',
        properties: {
          location: {
            type: 'string',
            description: '城市名称，例如：北京、上海、New York'
          },
          unit: {
            type: 'string',
            enum: ['celsius', 'fahrenheit'],
            description: '温度单位，celsius表示摄氏度，fahrenheit表示华氏度'
          }
        },
        required: ['location']
      }, null, 2),
      code: 'weather'
    },
    calculator: {
      name: 'calculate',
      description: '执行数学计算',
      parameters: JSON.stringify({
        type: 'object',
        properties: {
          expression: {
            type: 'string',
            description: '要计算的数学表达式，例如：2+2, 10*5, sqrt(16)'
          }
        },
        required: ['expression']
      }, null, 2),
      code: 'calculator'
    },
    empty: {
      name: '',
      description: '',
      parameters: JSON.stringify({
        type: 'object',
        properties: {},
        required: []
      }, null, 2),
      code: ''
    }
  };

  useEffect(() => {
    if (assistantId > 0) {
      fetchAssistant();
      fetchTools();
    }
  }, [assistantId]);

  const fetchAssistant = async () => {
    try {
      const res = await getAssistant(assistantId);
      setAssistant(res.data);
    } catch (err: any) {
      showAlert(err?.msg || t('assistantTools.messages.fetchAssistantFailed'), 'error');
    }
  };

  const fetchTools = async () => {
    try {
      setLoading(true);
      const res = await getAssistantTools(assistantId);
      let toolsData: AssistantTool[] = [];
      if (res && res.data) {
        if (Array.isArray(res.data)) {
          toolsData = res.data;
        }
      }
      setTools(toolsData);
    } catch (err: any) {
      console.error('获取工具列表失败:', err);
      showAlert(err?.msg || err?.message || t('assistantTools.messages.fetchToolsFailed'), 'error');
      setTools([]);
    } finally {
      setLoading(false);
    }
  };

  const applyTemplate = (template: keyof typeof jsonTemplates) => {
    const tpl = jsonTemplates[template];
    setFormData({
      ...formData,
      name: tpl.name,
      description: tpl.description,
      parameters: tpl.parameters,
      code: tpl.code,
    });
  };

  const handleAddTool = async () => {
    try {
      JSON.parse(formData.parameters);
      await createAssistantTool(assistantId, formData);
      await fetchTools();
      setShowAddModal(false);
      resetForm();
      showAlert(t('assistantTools.messages.createSuccess'), 'success');
    } catch (err: any) {
      if (err instanceof SyntaxError) {
        showAlert(t('assistantTools.messages.invalidJson'), 'error');
      } else {
        showAlert(err?.msg || t('assistantTools.messages.createFailed'), 'error');
      }
    }
  };

  const handleUpdateTool = async () => {
    if (!editingTool) return;
    
    try {
      if (formData.parameters) {
        JSON.parse(formData.parameters);
      }
      
      const updateData: UpdateToolForm = {};
      if (formData.name) updateData.name = formData.name;
      if (formData.description) updateData.description = formData.description;
      if (formData.parameters) updateData.parameters = formData.parameters;
      if (formData.code !== undefined) updateData.code = formData.code;
      if (formData.webhookUrl !== undefined) updateData.webhookUrl = formData.webhookUrl;
      if (formData.enabled !== undefined) updateData.enabled = formData.enabled;

      await updateAssistantTool(assistantId, editingTool.id, updateData);
      await fetchTools();
      setEditingTool(null);
      resetForm();
      setShowAddModal(false);
      showAlert(t('assistantTools.messages.updateSuccess'), 'success');
    } catch (err: any) {
      if (err instanceof SyntaxError) {
        showAlert(t('assistantTools.messages.invalidJson'), 'error');
      } else {
        showAlert(err?.msg || t('assistantTools.messages.updateFailed'), 'error');
      }
    }
  };

  const handleDeleteTool = async (toolId: number) => {
    if (!confirm(t('assistantTools.messages.deleteConfirm'))) return;
    
    try {
      await deleteAssistantTool(assistantId, toolId);
      await fetchTools();
      showAlert(t('assistantTools.messages.deleteSuccess'), 'success');
    } catch (err: any) {
      showAlert(err?.msg || t('assistantTools.messages.deleteFailed'), 'error');
    }
  };

  const handleEditTool = (tool: AssistantTool) => {
    setEditingTool(tool);
    setFormData({
      name: tool.name,
      description: tool.description,
      parameters: tool.parameters,
      code: tool.code || '',
      webhookUrl: tool.webhookUrl || '',
      enabled: tool.enabled,
    });
    setShowAddModal(true);
  };

  const handleTestTool = (tool: AssistantTool) => {
    setTestingTool(tool);
    
    // 根据工具的参数定义生成示例输入
    let exampleInput = '{}';
    try {
      const params = JSON.parse(tool.parameters);
      if (params.properties) {
        const example: Record<string, any> = {};
        Object.keys(params.properties).forEach(key => {
          const prop = params.properties[key];
          if (prop.type === 'string') {
            example[key] = prop.description?.includes('城市') ? '北京' : '示例值';
          } else if (prop.type === 'number') {
            example[key] = 0;
          } else if (prop.type === 'boolean') {
            example[key] = true;
          } else if (prop.enum && prop.enum.length > 0) {
            example[key] = prop.enum[0];
          }
        });
        exampleInput = JSON.stringify(example, null, 2);
      }
    } catch {
      // 如果解析失败，使用空对象
    }
    
    setTestInput(exampleInput);
    setTestResult('');
    setShowTestModal(true);
  };

  const executeTest = async () => {
    if (!testingTool) return;

    try {
      setIsTesting(true);
      setTestResult('');

      // 解析测试输入
      const args = JSON.parse(testInput);

      // 调用后端工具测试API
      const res = await testAssistantTool(assistantId, testingTool.id, args);
      
      if (res && res.data && res.data.result) {
        setTestResult(res.data.result);
        showAlert(t('assistantTools.messages.testSuccess'), 'success');
      } else {
        setTestResult(t('assistantTools.messages.testNoResult'));
        showAlert(t('assistantTools.messages.testComplete'), 'success');
      }
    } catch (err: any) {
      if (err instanceof SyntaxError) {
        setTestResult(`${t('assistantTools.messages.invalidInput')}\n${err.message}`);
        showAlert(t('assistantTools.messages.invalidInput'), 'error');
      } else {
        const errorMsg = err?.msg || err?.message || t('assistantTools.messages.testFailed');
        setTestResult(`${t('assistantTools.error')}: ${errorMsg}`);
        showAlert(t('assistantTools.messages.testFailed'), 'error');
      }
    } finally {
      setIsTesting(false);
    }
  };

  const resetForm = () => {
    setFormData({
      name: '',
      description: '',
      parameters: '{\n  "type": "object",\n  "properties": {},\n  "required": []\n}',
      code: '',
      webhookUrl: '',
      enabled: true,
    });
    setEditingTool(null);
  };

  if (!assistantId || assistantId <= 0) {
    return (
      <div className="min-h-screen bg-gradient-to-br from-purple-50 via-white to-blue-50 dark:from-neutral-900 dark:via-neutral-800 dark:to-neutral-900 flex items-center justify-center p-4">
        <Card variant="elevated" padding="lg" className="max-w-md text-center">
          <EmptyState
            icon={XCircle}
            title={t('assistantTools.invalidId')}
            iconClassName="text-red-400"
            action={{
              label: t('assistantTools.backToList'),
              onClick: () => navigate('/assistants')
            }}
          />
        </Card>
      </div>
    );
  }

  return (
    <div className="min-h-screen bg-gradient-to-br from-purple-50 via-white to-blue-50 dark:from-neutral-900 dark:via-neutral-800 dark:to-neutral-900">
      <div className="max-w-7xl mx-auto px-4 py-8">
        {/* 头部 */}
        <div className="mb-8">
          <Button
            variant="ghost"
            size="sm"
            leftIcon={<ArrowLeft className="w-4 h-4" />}
            onClick={() => navigate('/assistants')}
            className="mb-6"
          >
            {t('assistantTools.backToList')}
          </Button>
          <div className="flex items-center justify-between flex-wrap gap-4">
            <div>
              <h1 className="text-4xl font-bold bg-gradient-to-r from-purple-600 to-blue-600 bg-clip-text text-transparent dark:from-purple-400 dark:to-blue-400">
                {t('assistantTools.title')}
              </h1>
              <p className="text-gray-600 dark:text-gray-400 mt-2 text-lg">
                {assistant ? (
                  <span className="flex items-center gap-2">
                    <Sparkles className="w-4 h-4 text-purple-500" />
                    {t('assistantTools.assistant')}: <span className="font-semibold text-gray-900 dark:text-gray-100">{assistant.name}</span>
                  </span>
                ) : (
                  t('assistantTools.loading')
                )}
              </p>
            </div>
            <Button
              variant="primary"
              size="lg"
              leftIcon={<Plus className="w-5 h-5" />}
              onClick={() => {
                resetForm();
                setShowAddModal(true);
              }}
              animation="bounce"
            >
              {t('assistantTools.addTool')}
            </Button>
          </div>
        </div>

        {/* 工具列表 */}
        {loading ? (
          <Card variant="elevated" padding="xl" className="text-center">
            <Loader2 className="w-12 h-12 mx-auto animate-spin text-purple-600" />
            <p className="text-gray-500 dark:text-gray-400 mt-4 text-lg">加载中...</p>
          </Card>
        ) : !Array.isArray(tools) || tools.length === 0 ? (
          <Card variant="outlined" padding="xl" className="border-dashed">
            <EmptyState
              icon={Wrench}
              title={t('assistantTools.empty')}
              description={t('assistantTools.emptyDesc')}
              iconClassName="text-purple-400 dark:text-purple-500"
              action={{
                label: t('assistantTools.addTool'),
                onClick: () => {
                  resetForm();
                  setShowAddModal(true);
                }
              }}
            />
          </Card>
        ) : (
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
            {tools.map((tool, index) => (
              <Card
                key={tool.id}
                variant="elevated"
                hover={true}
                padding="md"
                animation="scale"
                delay={index * 0.1}
                className="flex flex-col"
              >
                <div className="flex-1">
                  <div className="flex items-start justify-between mb-4">
                    <div className="flex-1 min-w-0">
                      <div className="flex items-center gap-2 mb-2">
                        <h3 className="font-semibold text-lg text-gray-900 dark:text-gray-100 truncate">
                          {tool.name}
                        </h3>
                        {tool.enabled ? (
                          <CheckCircle2 className="w-5 h-5 text-green-500 flex-shrink-0" />
                        ) : (
                          <XCircle className="w-5 h-5 text-gray-400 flex-shrink-0" />
                        )}
                      </div>
                      <p className="text-sm text-gray-600 dark:text-gray-400 line-clamp-2 mb-3">
                        {tool.description}
                      </p>
                      <div className="mb-3 flex items-center gap-2 flex-wrap">
                        {tool.code && (
                          <span className="inline-flex items-center gap-1 px-2.5 py-1 rounded-md text-xs font-medium bg-purple-50 text-purple-700 dark:bg-purple-900/20 dark:text-purple-300">
                            <Code className="w-3.5 h-3.5" />
                            {tool.code}
                          </span>
                        )}
                        {tool.webhookUrl && (
                          <span className="inline-flex items-center gap-1 px-2.5 py-1 rounded-md text-xs font-medium bg-blue-50 text-blue-700 dark:bg-blue-900/20 dark:text-blue-300">
                            <Settings className="w-3.5 h-3.5" />
                            Webhook
                          </span>
                        )}
                      </div>
                    </div>
                  </div>
                </div>

                <div className="flex items-center gap-2 pt-4 border-t border-gray-200 dark:border-neutral-700 mt-auto">
                  <Button
                    variant="ghost"
                    size="sm"
                    leftIcon={<TestTube className="w-4 h-4" />}
                    onClick={() => handleTestTool(tool)}
                    className="flex-1 text-purple-700 dark:text-purple-300 hover:bg-purple-50 dark:hover:bg-purple-900/20"
                  >
                    {t('assistantTools.test')}
                  </Button>
                  <Button
                    variant="ghost"
                    size="sm"
                    leftIcon={<Edit2 className="w-4 h-4" />}
                    onClick={() => handleEditTool(tool)}
                    className="flex-1"
                  >
                    {t('assistantTools.edit')}
                  </Button>
                  <Button
                    variant="ghost"
                    size="sm"
                    leftIcon={<Trash2 className="w-4 h-4" />}
                    onClick={() => handleDeleteTool(tool.id)}
                    className="flex-1 text-red-600 dark:text-red-400 hover:bg-red-50 dark:hover:bg-red-900/20"
                  >
                    {t('assistantTools.delete')}
                  </Button>
                </div>
              </Card>
            ))}
          </div>
        )}
      </div>

      {/* 添加/编辑工具模态框 */}
      <Modal
        isOpen={showAddModal}
        onClose={() => {
          setShowAddModal(false);
          resetForm();
        }}
        size="xl"
        className="max-h-[90vh] flex flex-col"
      >
        <ModalHeader
          onClose={() => {
            setShowAddModal(false);
            resetForm();
          }}
        >
          <ModalTitle>
            {editingTool ? t('assistantTools.editTool') : t('assistantTools.addToolModal')}
          </ModalTitle>
        </ModalHeader>

        <ModalContent className="flex-1 overflow-y-auto space-y-5">
              <div>
                <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
                  {t('assistantTools.toolName')} <span className="text-red-500">*</span>
                </label>
                <input
                  type="text"
                  value={formData.name}
                  onChange={(e) => setFormData({ ...formData, name: e.target.value })}
                  className="w-full px-4 py-2.5 border border-gray-300 dark:border-neutral-600 rounded-lg bg-white dark:bg-neutral-700 text-gray-900 dark:text-gray-100 focus:ring-2 focus:ring-purple-500 focus:border-transparent"
                  placeholder="例如: get_weather"
                />
              </div>

              <div>
                <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
                  {t('assistantTools.toolDescription')} <span className="text-red-500">*</span>
                </label>
                <textarea
                  value={formData.description}
                  onChange={(e) => setFormData({ ...formData, description: e.target.value })}
                  className="w-full px-4 py-2.5 border border-gray-300 dark:border-neutral-600 rounded-lg bg-white dark:bg-neutral-700 text-gray-900 dark:text-gray-100 focus:ring-2 focus:ring-purple-500 focus:border-transparent"
                  rows={3}
                  placeholder={t('assistantTools.toolDescription')}
                />
              </div>

              <div>
                <div className="flex items-center justify-between mb-2">
                  <label className="block text-sm font-medium text-gray-700 dark:text-gray-300">
                    {t('assistantTools.parametersLabel')} <span className="text-red-500">*</span>
                  </label>
                  <div className="flex items-center gap-2">
                    <span className="text-xs text-gray-500 dark:text-gray-400">{t('assistantTools.quickTemplate')}:</span>
                    <Button
                      type="button"
                      variant="ghost"
                      size="xs"
                      onClick={() => applyTemplate('weather')}
                      className="text-purple-700 dark:text-purple-300"
                    >
                      {t('assistantTools.weather')}
                    </Button>
                    <Button
                      type="button"
                      variant="ghost"
                      size="xs"
                      onClick={() => applyTemplate('calculator')}
                      className="text-purple-700 dark:text-purple-300"
                    >
                      {t('assistantTools.calculator')}
                    </Button>
                    <Button
                      type="button"
                      variant="ghost"
                      size="xs"
                      onClick={() => applyTemplate('empty')}
                    >
                      {t('assistantTools.clear')}
                    </Button>
                  </div>
                </div>
                <textarea
                  value={formData.parameters}
                  onChange={(e) => setFormData({ ...formData, parameters: e.target.value })}
                  className="w-full px-4 py-2.5 border border-gray-300 dark:border-neutral-600 rounded-lg bg-white dark:bg-neutral-700 text-gray-900 dark:text-gray-100 font-mono text-sm focus:ring-2 focus:ring-purple-500 focus:border-transparent"
                  rows={14}
                  placeholder='{"type":"object","properties":{},"required":[]}'
                />
                <p className="text-xs text-gray-500 dark:text-gray-400 mt-1.5">
                  {t('assistantTools.validJsonSchema')}
                </p>
              </div>

              <div>
                <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
                  {t('assistantTools.code')}
                </label>
                <input
                  type="text"
                  value={formData.code}
                  onChange={(e) => setFormData({ ...formData, code: e.target.value })}
                  className="w-full px-4 py-2.5 border border-gray-300 dark:border-neutral-600 rounded-lg bg-white dark:bg-neutral-700 text-gray-900 dark:text-gray-100 focus:ring-2 focus:ring-purple-500 focus:border-transparent"
                  placeholder="例如: weather, calculator"
                />
                <p className="text-xs text-gray-500 dark:text-gray-400 mt-1.5">
                  {t('assistantTools.codeDesc')}
                </p>
              </div>

              <div>
                <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
                  {t('assistantTools.webhookUrl')}
                </label>
                <input
                  type="url"
                  value={formData.webhookUrl}
                  onChange={(e) => setFormData({ ...formData, webhookUrl: e.target.value })}
                  className="w-full px-4 py-2.5 border border-gray-300 dark:border-neutral-600 rounded-lg bg-white dark:bg-neutral-700 text-gray-900 dark:text-gray-100 focus:ring-2 focus:ring-purple-500 focus:border-transparent"
                  placeholder="https://api.example.com/webhook"
                />
                <p className="text-xs text-gray-500 dark:text-gray-400 mt-1.5">
                  {t('assistantTools.webhookUrlDesc')}
                </p>
              </div>

              <div className="flex items-center gap-2">
                <input
                  type="checkbox"
                  id="enabled"
                  checked={formData.enabled}
                  onChange={(e) => setFormData({ ...formData, enabled: e.target.checked })}
                  className="w-4 h-4 text-purple-600 border-gray-300 rounded focus:ring-purple-500"
                />
                <label htmlFor="enabled" className="text-sm text-gray-700 dark:text-gray-300">
                  {t('assistantTools.enable')}
                </label>
              </div>
        </ModalContent>

        <ModalFooter>
          <Button
            variant="secondary"
            onClick={() => {
              setShowAddModal(false);
              resetForm();
            }}
          >
            {t('assistantTools.cancel')}
          </Button>
          <Button
            variant="primary"
            leftIcon={<Save className="w-4 h-4" />}
            onClick={editingTool ? handleUpdateTool : handleAddTool}
          >
            {editingTool ? t('assistantTools.save') : t('assistantTools.create')}
          </Button>
        </ModalFooter>
      </Modal>

      {/* 工具测试模态框 */}
      <Modal
        isOpen={showTestModal && !!testingTool}
        onClose={() => {
          setShowTestModal(false);
          setTestingTool(null);
          setTestInput('{}');
          setTestResult('');
        }}
        size="lg"
        className="max-h-[90vh] flex flex-col"
      >
        {testingTool && (
          <>
            <ModalHeader
              onClose={() => {
                setShowTestModal(false);
                setTestingTool(null);
                setTestInput('{}');
                setTestResult('');
              }}
            >
              <div>
                <ModalTitle>
                  {t('assistantTools.testModal.title')}: {testingTool.name}
                </ModalTitle>
                <p className="text-sm text-gray-500 dark:text-gray-400 mt-1">
                  {testingTool.description}
                </p>
              </div>
            </ModalHeader>

            <ModalContent className="flex-1 overflow-y-auto space-y-4">
              {/* 工具参数说明 */}
              {testingTool && (
                <div className="bg-purple-50 dark:bg-purple-900/20 border border-purple-200 dark:border-purple-800 rounded-lg p-4">
                  <h4 className="text-sm font-semibold text-purple-900 dark:text-purple-100 mb-2">
                    {t('assistantTools.parametersDesc')}
                  </h4>
                  <div className="text-xs text-purple-700 dark:text-purple-300 font-mono whitespace-pre-wrap">
                    {(() => {
                      try {
                        const params = JSON.parse(testingTool.parameters);
                        if (params.properties) {
                          let desc = '';
                          Object.keys(params.properties).forEach(key => {
                            const prop = params.properties[key];
                            desc += `${key} (${prop.type || 'any'})`;
                            if (prop.description) {
                              desc += `: ${prop.description}`;
                            }
                            if (prop.enum) {
                              desc += ` [可选值: ${prop.enum.join(', ')}]`;
                            }
                            desc += '\n';
                          });
                          return desc || t('assistantTools.noParams');
                        }
                        return t('assistantTools.noParams');
                      } catch {
                        return t('assistantTools.cannotParseParams');
                      }
                    })()}
                  </div>
                </div>
              )}

              <div>
                <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
                  {t('assistantTools.testInputLabel')} <span className="text-red-500">*</span>
                </label>
                <textarea
                  value={testInput}
                  onChange={(e) => setTestInput(e.target.value)}
                  className="w-full px-4 py-3 border border-gray-300 dark:border-neutral-600 rounded-lg bg-white dark:bg-neutral-700 text-gray-900 dark:text-gray-100 font-mono text-sm focus:ring-2 focus:ring-purple-500 focus:border-transparent"
                  rows={10}
                  placeholder='{"location": "北京", "unit": "celsius"}'
                />
                <p className="text-xs text-gray-500 dark:text-gray-400 mt-1.5">
                  {t('assistantTools.testInputDesc')}
                </p>
              </div>

              {testResult && (
                <div>
                  <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
                    {t('assistantTools.testResultLabel')}
                  </label>
                  <div className="w-full px-4 py-3 border border-gray-300 dark:border-neutral-600 rounded-lg bg-gray-50 dark:bg-neutral-900 text-gray-900 dark:text-gray-100 font-mono text-sm whitespace-pre-wrap min-h-[120px] max-h-[300px] overflow-y-auto">
                    {testResult}
                  </div>
                </div>
              )}
            </ModalContent>

            <ModalFooter>
              <Button
                variant="secondary"
                onClick={() => {
                  setShowTestModal(false);
                  setTestingTool(null);
                  setTestInput('{}');
                  setTestResult('');
                }}
              >
                {t('assistantTools.close')}
              </Button>
              <Button
                variant="primary"
                leftIcon={isTesting ? <Loader2 className="w-4 h-4 animate-spin" /> : <Play className="w-4 h-4" />}
                onClick={executeTest}
                loading={isTesting}
                disabled={isTesting}
              >
                {isTesting ? t('assistantTools.testing') : t('assistantTools.runTest')}
              </Button>
            </ModalFooter>
          </>
        )}
      </Modal>
    </div>
  );
};

export default AssistantTools;

