import BaseLayout from '@/components/Layout/BaseLayout'
import AssistantPageHeader from '@/pages/assistants/AssistantPageHeader'
import AssistantTemplatePicker from '@/pages/assistants/AssistantTemplatePicker'
import { useTranslation } from '@/i18n'

export default function AssistantManagerNew() {
  const { t } = useTranslation()
  const params = new URLSearchParams(window.location.search)
  return (
    <BaseLayout hideHeader>
      <AssistantPageHeader currentLabel={t('assistant.createLabel')} />
      <div className="w-full px-6 py-6">
        <AssistantTemplatePicker scopeTenantId={params.get('tenantId') || undefined} />
      </div>
    </BaseLayout>
  )
}
