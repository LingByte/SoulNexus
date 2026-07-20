import { Loading } from '@/components/ui/loading'
import { useSiteConfig } from '@/contexts/siteConfig'
import VoiceprintDisabledPage from '@/pages/assistants/VoiceprintDisabledPage'
import VoiceprintWorkbenchPage from '@/pages/assistants/VoiceprintWorkbenchPage'

export default function VoiceprintManagerPage() {
  const { config, ready } = useSiteConfig()
  const provider = (config.VOICEPRINT_PROVIDER || '').trim().toLowerCase()

  if (!ready) {
    return (
      <div className="flex min-h-[40vh] items-center justify-center">
        <Loading tip="加载中…" />
      </div>
    )
  }

  if (!provider) return <VoiceprintDisabledPage />
  return <VoiceprintWorkbenchPage />
}
