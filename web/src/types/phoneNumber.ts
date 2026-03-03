// 号码管理类型定义

export interface PhoneNumber {
  id: number
  userId: number
  phoneNumber: string
  displayName?: string
  carrier?: string // 运营商：移动/联通/电信
  isPrimary: boolean
  
  // 方案绑定
  schemeId?: number
  schemeName?: string
  
  // 呼叫转移状态
  forwardEnabled: boolean
  forwardNumber?: string
  
  // 统计
  callCount: number
  voicemailCount: number
  lastCallAt?: string
  
  enabled: boolean
  createdAt: string
  updatedAt: string
}

export interface CreatePhoneNumberRequest {
  phoneNumber: string
  displayName?: string
  carrier?: string
}

export interface UpdatePhoneNumberRequest {
  displayName?: string
  carrier?: string
  enabled?: boolean
}

export interface PhoneNumberStats {
  totalCalls: number
  totalVoicemails: number
  todayCalls: number
  avgDuration: number
}

export interface ForwardGuide {
  carrier: string
  enableCode: string
  disableCode: string
  checkCode: string
  description: string
}

export const FORWARD_GUIDES: ForwardGuide[] = [
  {
    carrier: '中国移动',
    enableCode: '**21*{number}#',
    disableCode: '##21#',
    checkCode: '*#21#',
    description: '无条件呼叫转移'
  },
  {
    carrier: '中国联通',
    enableCode: '**21*{number}#',
    disableCode: '##21#',
    checkCode: '*#21#',
    description: '无条件呼叫转移'
  },
  {
    carrier: '中国电信',
    enableCode: '**21*{number}#',
    disableCode: '##21#',
    checkCode: '*#21#',
    description: '无条件呼叫转移'
  }
]
