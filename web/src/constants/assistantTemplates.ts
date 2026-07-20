import type { AgentConfigDraft } from '@/constants/assistantAdvancedConfig'
import { defaultAgentConfig } from '@/constants/assistantAdvancedConfig'

export type AssistantTemplateId =
  | 'blank'
  | 'customer_cleaning'
  | 'waking_up'
  | 'inbound_knowledge'
  | 'outbound_notify'
  | 'outbound_collect'

export interface AssistantTemplate {
  id: AssistantTemplateId
  scene: string
  label: string
  description: string
  iconBg: string
  iconColor: string
  defaultName: string
  welcome?: string
  prompt?: string
  knowledgeNamespace?: string
  agent?: Partial<AgentConfigDraft>
}

const CUSTOMER_CLEANING_PROMPT = `你是一名营销客服，主要任务是筛选对产品有兴趣的客户。

对话流程：
1. 开场：确认对方是否有相关需求；无意向则礼貌结束。
2. 产品介绍：简要说明企业/服务与当前优惠；有意向则进入跟进，无意向则尝试挽留（如加微信）。
3. 后续跟进：有意向时询问是否方便转顾问详细沟通；确认后引导至人工协助。
4. 结束：感谢接听并祝福。

要求：语气自然、简洁；每轮只问一个问题；识别高/中/低意向并记录在对话中。`

const WAKING_UP_PROMPT = `你是一名客服，任务是联系长期未互动或沉睡的老客户，唤醒兴趣并促进转化。

对话流程：
1. 开场：说明回访来意，确认是否曾使用/注册过服务。
2. 使用调查：了解近期未使用的原因（价格、需求变化、操作问题等）。
3. 问题处理：若客户反馈问题，表示重视并安排专人跟进。
4. 方案提供：介绍最新活动/试用/优惠，有意向则转顾问。
5. 结束：感谢反馈，祝生活愉快。

要求：耐心倾听；不强行推销；区分「成功激活 / 潜在需求 / 低意向」。`

const KNOWLEDGE_PROMPT = `你是企业知识库问答助手，专门回答客户常见问题。

要求：
- 优先依据知识库内容回答，不确定时诚实说明并引导联系客服/人工协助。
- 回答简洁口语化，适合语音对话场景。
- 一次只回答一个焦点，必要时确认客户是否听清。`

const NOTIFY_PROMPT = `你是消息通知助手，负责向客户传达一条简短通知。

要求：
- 开场说明来意，清晰念出通知要点。
- 若客户没听清，可重复关键信息（最多 2 次）。
- 确认客户已收到后礼貌结束，不展开无关对话。`

const QUESTIONNAIRE_PROMPT = `你是问卷调查/满意度回访助手。

要求：
- 按顺序逐一提问，每题等待客户回答后再进入下一题。
- 对模糊回答做简短确认。
- 全部问题结束后，口头归纳要点并感谢参与。
- 语气礼貌，控制单题时长。`

export const ASSISTANT_TEMPLATES: AssistantTemplate[] = [
  {
    id: 'blank',
    scene: 'general',
    label: '空白模板',
    description: '建议高级用户使用，没有预设，功能齐全，没有偏向性。',
    iconBg: 'bg-gray-100',
    iconColor: 'text-gray-600',
    defaultName: '空白智能体',
  },
  {
    id: 'customer_cleaning',
    scene: 'outbound_collect',
    label: '客户清洗',
    description: '识别并剔除无效、重复或无潜力客户数据。其中意向客户可分为高、中、低意向。',
    iconBg: 'bg-purple-100',
    iconColor: 'text-purple-600',
    defaultName: '客户清洗',
    welcome: '您好，我是客服代表，想占用您一分钟了解一下是否有相关需求，方便吗？',
    prompt: CUSTOMER_CLEANING_PROMPT,
    agent: { maxSilentAskTimes: 2 },
  },
  {
    id: 'waking_up',
    scene: 'outbound_collect',
    label: '沉睡客户唤醒',
    description: '通过营销手段激活长期未响应的老客户，将客户粗略分为成功激活、有潜在需求和低意向。',
    iconBg: 'bg-blue-100',
    iconColor: 'text-blue-600',
    defaultName: '沉睡客户唤醒',
    welcome: '您好，我是客服代表，看到您之前使用过我们的服务，想做个简短回访，方便吗？',
    prompt: WAKING_UP_PROMPT,
    agent: { maxSilentAskTimes: 3 },
  },
  {
    id: 'inbound_knowledge',
    scene: 'inbound_knowledge',
    label: '知识库问答',
    description: '专为回答客户常见问题而设计，直接引用整理好的知识库，流畅回答客户提出的问题。',
    iconBg: 'bg-orange-100',
    iconColor: 'text-orange-600',
    defaultName: '知识库问答',
    welcome: '您好，我是智能客服，请问有什么可以帮您？',
    prompt: KNOWLEDGE_PROMPT,
    agent: { topK: 5, maxDialogueGapTimes: 2 },
  },
  {
    id: 'outbound_notify',
    scene: 'outbound_notify',
    label: '消息通知',
    description: '简单的一句话通知场景；若客户没听清，可要求重复，比简单更简单。',
    iconBg: 'bg-green-100',
    iconColor: 'text-green-600',
    defaultName: '消息通知',
    welcome: '您好，这边有一条重要通知需要向您传达，请留意听。',
    prompt: NOTIFY_PROMPT,
    agent: { enableHangupTool: true, maxSilentAskTimes: 1 },
  },
  {
    id: 'outbound_collect',
    scene: 'outbound_collect',
    label: '问卷调查',
    description: '专为满意度调查、定期回访等场景设计，会话完成后可对收集问题做归纳总结。',
    iconBg: 'bg-cyan-100',
    iconColor: 'text-cyan-600',
    defaultName: '问卷调查',
    welcome: '您好，我们正在进行一项简短调研，大约占用您两三分钟，可以吗？',
    prompt: QUESTIONNAIRE_PROMPT,
    agent: { maxSilentAskTimes: 2 },
  },
]

export function getAssistantTemplate(id?: string | null): AssistantTemplate | undefined {
  if (!id) return undefined
  return ASSISTANT_TEMPLATES.find((t) => t.id === id)
}

export function applyAssistantTemplate(
  template: AssistantTemplate,
): {
  name: string
  scene: string
  description: string
  welcome: string
  prompt: string
  knowledgeNamespace: string
  agent: AgentConfigDraft
} {
  const suffix = Math.random().toString(36).slice(2, 6)
  return {
    name: `${template.defaultName}_${suffix}`,
    scene: template.scene,
    description: template.description,
    welcome: template.welcome || '',
    prompt: template.prompt || '',
    knowledgeNamespace: template.knowledgeNamespace || '',
    agent: { ...defaultAgentConfig(), ...template.agent },
  }
}
