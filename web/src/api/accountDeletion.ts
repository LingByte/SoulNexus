import { post, type ApiResponse } from '@/utils/request'

export { sendRevokeDeletionEmailCode } from '@/api/common'

export async function revokeAccountDeletionPublic(body: {
  email: string
  emailCode: string
}): Promise<ApiResponse<{ pending: boolean; coolingDays: number }>> {
  return post('/account/deletion/revoke', body)
}
