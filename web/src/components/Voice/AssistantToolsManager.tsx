import React, { useState, useEffect } from 'react';
import { 
  getAssistantTools, 
  createAssistantTool, 
  updateAssistantTool, 
  deleteAssistantTool,
  type AssistantTool,
  type CreateToolForm,
  type UpdateToolForm
} from '@/api/assistant';
import { showAlert } from '@/utils/notification';
import { Plus, Edit2, Trash2, X, Save, Code, Settings, CheckCircle2, XCircle } from 'lucide-react';

interface AssistantToolsManagerProps {
  assistantId: number;
  assistantName?: string;
}

const AssistantToolsManager: React.FC<AssistantToolsManagerProps> = ({ assistantId, assistantName }) => {
  const [tools, setTools] = useState<AssistantTool[]>([]);
  const [loading, setLoading] = useState(false);
  const [showAddModal, setShowAddModal] = useState(false);
  const [editingTool, setEditingTool] = useState<AssistantTool | null>(null);
  const [formData, setFormData] = useState<CreateToolForm>({
    name: '',
    description: '',
    parameters: '{\n  "type": "object",\n  "properties": {},\n  "required": []\n}',
    code: '',
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

  useEffect(() => {
    fetchTools();
  }, [assistantId]);

  const fetchTools = async () => {
    try {
      setLoading(true);
      const res = await getAssistantTools(assistantId);
      // 确保 data 是数组，处理 null/undefined 的情况
      let toolsData: AssistantTool[] = [];
      if (res && res.data) {
        if (Array.isArray(res.data)) {
          toolsData = res.data;
        } else if (res.data && typeof res.data === 'object') {
          // 如果 data 是对象但不是数组，尝试转换为数组
          toolsData = [];
        }
      }
      setTools(toolsData);
    } catch (err: any) {
      console.error('获取工具列表失败:', err);
      showAlert(err?.msg || err?.message || '获取工具列表失败', 'error');
      // 确保即使出错也设置为空数组
      setTools([]);
    } finally {
      setLoading(false);
    }
  };

  const handleAddTool = async () => {
    try {
      // 验证JSON格式
      JSON.parse(formData.parameters);
      
      await createAssistantTool(assistantId, formData);
      await fetchTools();
      setShowAddModal(false);
      resetForm();
      showAlert('工具创建成功', 'success');
    } catch (err: any) {
      if (err instanceof SyntaxError) {
        showAlert('Parameters必须是有效的JSON格式', 'error');
      } else {
        showAlert(err?.msg || '创建工具失败', 'error');
      }
    }
  };

  const handleUpdateTool = async () => {
    if (!editingTool) return;
    
    try {
      // 验证JSON格式
      if (formData.parameters) {
        JSON.parse(formData.parameters);
      }
      
      const updateData: UpdateToolForm = {};
      if (formData.name) updateData.name = formData.name;
      if (formData.description) updateData.description = formData.description;
      if (formData.parameters) updateData.parameters = formData.parameters;
      if (formData.code !== undefined) updateData.code = formData.code;
      if (formData.enabled !== undefined) updateData.enabled = formData.enabled;

      await updateAssistantTool(assistantId, editingTool.id, updateData);
      await fetchTools();
      setEditingTool(null);
      resetForm();
      showAlert('工具更新成功', 'success');
    } catch (err: any) {
      if (err instanceof SyntaxError) {
        showAlert('Parameters必须是有效的JSON格式', 'error');
      } else {
        showAlert(err?.msg || '更新工具失败', 'error');
      }
    }
  };

  const handleDeleteTool = async (toolId: number) => {
    if (!confirm('确定要删除这个工具吗？')) return;
    
    try {
      await deleteAssistantTool(assistantId, toolId);
      await fetchTools();
      showAlert('工具删除成功', 'success');
    } catch (err: any) {
      showAlert(err?.msg || '删除工具失败', 'error');
    }
  };

  const handleEditTool = (tool: AssistantTool) => {
    setEditingTool(tool);
    setFormData({
      name: tool.name,
      description: tool.description,
      parameters: tool.parameters,
      code: tool.code || '',
      enabled: tool.enabled,
    });
    setShowAddModal(true);
  };

  const resetForm = () => {
    setFormData({
      name: '',
      description: '',
      parameters: '{\n  "type": "object",\n  "properties": {},\n  "required": []\n}',
      code: '',
      enabled: true,
    });
    setEditingTool(null);
  };

  const formatJSON = (json: string) => {
    try {
      const parsed = JSON.parse(json);
      return JSON.stringify(parsed, null, 2);
    } catch {
      return json;
    }
  };

  return (
    <div className="w-full">
      <div className="flex items-center justify-between mb-4">
        <div>
          <h2 className="text-xl font-semibold text-gray-900 dark:text-gray-100">
            自定义工具管理
          </h2>
          <p className="text-sm text-gray-500 dark:text-gray-400 mt-1">
            {assistantName ? `为 "${assistantName}" 管理工具` : '为助手添加和管理自定义工具'}
          </p>
        </div>
        <button
          onClick={() => {
            resetForm();
            setShowAddModal(true);
          }}
          className="flex items-center gap-2 px-4 py-2 bg-purple-600 text-white rounded-lg hover:bg-purple-700 transition-colors"
        >
          <Plus className="w-4 h-4" />
          添加工具
        </button>
      </div>

      {loading ? (
        <div className="text-center py-8 text-gray-500">加载中...</div>
      ) : !Array.isArray(tools) || tools.length === 0 ? (
        <div className="text-center py-12 border border-dashed border-gray-300 dark:border-neutral-700 rounded-lg">
          <Code className="w-12 h-12 mx-auto text-gray-400 mb-3" />
          <p className="text-gray-500 dark:text-gray-400">暂无工具</p>
          <p className="text-sm text-gray-400 dark:text-gray-500 mt-1">
            点击"添加工具"按钮创建第一个工具
          </p>
        </div>
      ) : (
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
          {Array.isArray(tools) && tools.map((tool) => (
            <div
              key={tool.id}
              className="border border-gray-200 dark:border-neutral-700 rounded-lg p-4 bg-white dark:bg-neutral-800 hover:border-purple-400 transition-colors"
            >
              <div className="flex items-start justify-between mb-3">
                <div className="flex-1">
                  <div className="flex items-center gap-2 mb-1">
                    <h3 className="font-semibold text-gray-900 dark:text-gray-100">
                      {tool.name}
                    </h3>
                    {tool.enabled ? (
                      <CheckCircle2 className="w-4 h-4 text-green-500" />
                    ) : (
                      <XCircle className="w-4 h-4 text-gray-400" />
                    )}
                  </div>
                  <p className="text-sm text-gray-600 dark:text-gray-400 line-clamp-2">
                    {tool.description}
                  </p>
                </div>
              </div>

              {tool.code && (
                <div className="mb-2">
                  <span className="inline-flex items-center gap-1 px-2 py-0.5 rounded text-xs bg-purple-50 text-purple-700 dark:bg-purple-900/20 dark:text-purple-300">
                    <Code className="w-3 h-3" />
                    {tool.code}
                  </span>
                </div>
              )}

              <div className="flex items-center gap-2 mt-3 pt-3 border-t border-gray-200 dark:border-neutral-700">
                <button
                  onClick={() => handleEditTool(tool)}
                  className="flex-1 flex items-center justify-center gap-1 px-3 py-1.5 text-sm text-gray-700 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-neutral-700 rounded transition-colors"
                >
                  <Edit2 className="w-3.5 h-3.5" />
                  编辑
                </button>
                <button
                  onClick={() => handleDeleteTool(tool.id)}
                  className="flex-1 flex items-center justify-center gap-1 px-3 py-1.5 text-sm text-red-600 dark:text-red-400 hover:bg-red-50 dark:hover:bg-red-900/20 rounded transition-colors"
                >
                  <Trash2 className="w-3.5 h-3.5" />
                  删除
                </button>
              </div>
            </div>
          ))}
        </div>
      )}

      {/* 添加/编辑工具模态框 */}
      {showAddModal && (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50 p-4">
          <div className="bg-white dark:bg-neutral-800 rounded-lg shadow-xl max-w-2xl w-full max-h-[90vh] overflow-y-auto">
            <div className="sticky top-0 bg-white dark:bg-neutral-800 border-b border-gray-200 dark:border-neutral-700 px-6 py-4 flex items-center justify-between">
              <h3 className="text-lg font-semibold text-gray-900 dark:text-gray-100">
                {editingTool ? '编辑工具' : '添加工具'}
              </h3>
              <button
                onClick={() => {
                  setShowAddModal(false);
                  resetForm();
                }}
                className="text-gray-400 hover:text-gray-600 dark:hover:text-gray-300"
              >
                <X className="w-5 h-5" />
              </button>
            </div>

            <div className="p-6 space-y-4">
              <div>
                <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                  工具名称 <span className="text-red-500">*</span>
                </label>
                <input
                  type="text"
                  value={formData.name}
                  onChange={(e) => setFormData({ ...formData, name: e.target.value })}
                  className="w-full px-3 py-2 border border-gray-300 dark:border-neutral-600 rounded-lg bg-white dark:bg-neutral-700 text-gray-900 dark:text-gray-100"
                  placeholder="例如: get_weather"
                />
              </div>

              <div>
                <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                  工具描述 <span className="text-red-500">*</span>
                </label>
                <textarea
                  value={formData.description}
                  onChange={(e) => setFormData({ ...formData, description: e.target.value })}
                  className="w-full px-3 py-2 border border-gray-300 dark:border-neutral-600 rounded-lg bg-white dark:bg-neutral-700 text-gray-900 dark:text-gray-100"
                  rows={3}
                  placeholder="描述这个工具的功能"
                />
              </div>

              <div>
                <div className="flex items-center justify-between mb-1">
                  <label className="block text-sm font-medium text-gray-700 dark:text-gray-300">
                    Parameters (JSON Schema) <span className="text-red-500">*</span>
                  </label>
                  <div className="flex items-center gap-2">
                    <span className="text-xs text-gray-500 dark:text-gray-400">快速模板:</span>
                    <button
                      type="button"
                      onClick={() => applyTemplate('weather')}
                      className="px-2 py-1 text-xs bg-purple-50 text-purple-700 dark:bg-purple-900/20 dark:text-purple-300 rounded hover:bg-purple-100 dark:hover:bg-purple-900/30"
                    >
                      天气
                    </button>
                    <button
                      type="button"
                      onClick={() => applyTemplate('calculator')}
                      className="px-2 py-1 text-xs bg-purple-50 text-purple-700 dark:bg-purple-900/20 dark:text-purple-300 rounded hover:bg-purple-100 dark:hover:bg-purple-900/30"
                    >
                      计算器
                    </button>
                    <button
                      type="button"
                      onClick={() => applyTemplate('empty')}
                      className="px-2 py-1 text-xs bg-gray-50 text-gray-700 dark:bg-neutral-700 dark:text-gray-300 rounded hover:bg-gray-100 dark:hover:bg-neutral-600"
                    >
                      清空
                    </button>
                  </div>
                </div>
                <textarea
                  value={formData.parameters}
                  onChange={(e) => setFormData({ ...formData, parameters: e.target.value })}
                  className="w-full px-3 py-2 border border-gray-300 dark:border-neutral-600 rounded-lg bg-white dark:bg-neutral-700 text-gray-900 dark:text-gray-100 font-mono text-sm"
                  rows={12}
                  placeholder='{"type":"object","properties":{},"required":[]}'
                />
                <p className="text-xs text-gray-500 dark:text-gray-400 mt-1">
                  必须是有效的JSON Schema格式。可以使用快速模板快速开始。
                </p>
              </div>

              <div>
                <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                  代码标识 (可选)
                </label>
                <input
                  type="text"
                  value={formData.code}
                  onChange={(e) => setFormData({ ...formData, code: e.target.value })}
                  className="w-full px-3 py-2 border border-gray-300 dark:border-neutral-600 rounded-lg bg-white dark:bg-neutral-700 text-gray-900 dark:text-gray-100"
                  placeholder="例如: weather, calculator"
                />
                <p className="text-xs text-gray-500 dark:text-gray-400 mt-1">
                  用于标识工具类型，如: weather, calculator 等
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
                  启用此工具
                </label>
              </div>
            </div>

            <div className="sticky bottom-0 bg-gray-50 dark:bg-neutral-900 border-t border-gray-200 dark:border-neutral-700 px-6 py-4 flex items-center justify-end gap-3">
              <button
                onClick={() => {
                  setShowAddModal(false);
                  resetForm();
                }}
                className="px-4 py-2 text-gray-700 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-neutral-700 rounded-lg transition-colors"
              >
                取消
              </button>
              <button
                onClick={editingTool ? handleUpdateTool : handleAddTool}
                className="flex items-center gap-2 px-4 py-2 bg-purple-600 text-white rounded-lg hover:bg-purple-700 transition-colors"
              >
                <Save className="w-4 h-4" />
                {editingTool ? '保存' : '创建'}
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
};

export default AssistantToolsManager;

