import BaseLayout from '@/components/Layout/BaseLayout'
import PlatformVoiceprintPanel from '@/components/voice/PlatformVoiceprintPanel'
import { useTranslation } from '@/i18n'

export default function PlatformVoiceprintManagement() {
  const { t } = useTranslation()
  return (
    <BaseLayout title={t('nav.platformVoiceprintManagement')} description="查看全租户声纹注册记录、服务状态与连通性自检">
      <div className="mx-auto max-w-6xl">
        <PlatformVoiceprintPanel />
      </div>
    </BaseLayout>
  )
}
