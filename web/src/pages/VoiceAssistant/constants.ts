import type { OnboardingStep } from './types'

export const ONBOARDING_STEPS: OnboardingStep[] = [
  {
    element: 'voice-ball',
    text: '这是语音交互的核心按钮，点击开始或结束对话',
    position: 'right'
  },
  {
    element: 'assistant-list',
    text: '在这里选择您要对话的AI助手，也可以添加新的助手',
    position: 'right'
  },
  {
    element: 'chat-area',
    text: '这里将显示您与AI助手的对话历史',
    position: 'top'
  },
  {
    element: 'control-panel',
    text: '在这里可以配置AI助手的各种参数和设置',
    position: 'left'
  },
  {
    element: 'text-input',
    text: '您也可以直接输入文本与AI助手交流',
    position: 'top',
    isLast: true
  }
]

