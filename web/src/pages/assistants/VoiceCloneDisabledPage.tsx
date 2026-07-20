import { Typography } from '@arco-design/web-react'
import { IconInfoCircle } from '@arco-design/web-react/icon'
import BaseLayout from '@/components/Layout/BaseLayout'

export default function VoiceCloneDisabledPage() {
  return (
    <BaseLayout>
      <div className="mx-auto flex max-w-lg flex-col items-center justify-center gap-4 px-6 py-24 text-center">
        <div className="flex h-16 w-16 items-center justify-center rounded-full bg-muted">
          <IconInfoCircle style={{ fontSize: 32 }} className="text-muted-foreground" />
        </div>
        <Typography.Title heading={5} style={{ margin: 0 }}>
          未开通音色克隆
        </Typography.Title>
        <Typography.Paragraph type="secondary" style={{ marginBottom: 0, fontSize: 14 }}>
          当前环境尚未开启音色克隆功能。如需使用音色克隆，请联系系统管理员开通。
        </Typography.Paragraph>
      </div>
    </BaseLayout>
  )
}
