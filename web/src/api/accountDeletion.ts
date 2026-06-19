import { get, post, del } from '@/utils/request'

export interface AccountDeletionEligibility {
  eligible: boolean
  reasons: string[]
  githubBound: boolean
  wechatBound: boolean
  accountLocked: boolean
  remoteLoginRisk: boolean
  recentSuspiciousLogins: boolean
  warnings: string[]
  cooldownHours: number
  deletionPending: boolean
  accountDeletionEffectiveAt?: string | null
  accountDeletionRequestedAt?: string | null
}

export const getAccountDeletionEligibility = () =>
  get<AccountDeletionEligibility>('/auth/account-deletion/eligibility')

export const sendAccountDeletionEmailCode = () =>
  post<null>('/auth/account-deletion/send-email-code', undefined)

export const requestAccountDeletion = (body: {
  password: string
  emailCode: string
  acknowledgeConsequences: boolean
}) => post<unknown>('/auth/account-deletion/request', body)

export const cancelAccountDeletion = (body: { emailCode: string }) =>
  post<null>('/auth/account-deletion/cancel', body)

export const sendAccountDeletionCancelCode = (email: string) =>
  post<null>('/auth/account-deletion/send-cancel-code', { email })

export const cancelAccountDeletionByEmail = (body: {
  email: string
  password: string
  emailCode: string
}) => post<null>('/auth/account-deletion/cancel-by-email', body)

export const unbindGithubAccount = () =>
  del<null>('/auth/bindings/github')

export const unbindWechatAccount = () =>
  del<null>('/auth/bindings/wechat')
