import type { WorkflowNode } from '../types/workflow'

/**
 * 获取节点类型的默认配置
 * @param type 节点类型
 * @returns 默认配置对象
 */
export const getDefaultNodeConfig = (type: WorkflowNode['type']): Record<string, any> => {
  switch (type) {
    case 'gateway':
      return {
        condition: 'context.value > 0',
        trueLabel: '是',
        falseLabel: '否'
      }
    case 'task':
      return {
        action: 'process_data',
        timeout: 30000
      }
    case 'event':
      return {
        eventType: 'user_action',
        trigger: 'click'
      }
    case 'subflow':
      return {
        workflowId: '',
        workflowName: ''
      }
    case 'parallel':
      return {
        branches: 2,
        waitAll: true
      }
    case 'wait':
      return {
        duration: 5000,
        untilEvent: ''
      }
    case 'timer':
      return {
        delay: 1000,
        repeat: false
      }
    case 'script':
      return {
        language: 'go',
        code: `// Go 脚本示例
// 必须定义一个 Run 函数，接收 inputs 并返回结果

func Run(inputs map[string]interface{}) (map[string]interface{}, error) {
	// 从 inputs 中获取输入数据
	// inputs 是一个 map，键是输入参数的别名（如 "input-0"）
	
	// 示例：获取第一个输入
	var input interface{}
	if val, ok := inputs["input-0"]; ok {
		input = val
	}
	
	// 处理逻辑
	result := map[string]interface{}{
		"output": input,
		"processed": true,
	}
	
	return result, nil
}`
      }
    case 'workflow_plugin':
      return {
        pluginId: null,
        parameters: {}
      }
    case 'ai_chat':
      return {
        provider: 'openai',
        apiKey: '',
        baseURL: 'https://api.openai.com/v1',
        model: 'gpt-4',
        systemPrompt: '',
        inputVariable: 'user_input',
        outputVariable: 'ai_response',
        temperature: 0.7,
        maxTokens: 2000,
        enableStream: false,
        persistHistory: false
      }
    default:
      return {}
  }
}
