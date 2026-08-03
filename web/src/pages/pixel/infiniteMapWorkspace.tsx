import { useNavigate } from 'react-router-dom'
import { ArrowLeftOutlined } from '@ant-design/icons'
import { Button, Card, Typography } from 'antd'
import 'antd/dist/reset.css'
import InfiniteMapScene from '@/pages/pixel/frameronin/infiniteMap/InfiniteMapScene'
import './frameronin/infiniteMapScene.css'

/** Full FrameRonin Infinite Map shell (same as InfiniteMapPlaceholder). */
export default function InfiniteMapWorkspace() {
  const navigate = useNavigate()
  return (
    <Card
      style={{ width: '100%', display: 'flex', flexDirection: 'column' }}
      styles={{
        body: {
          display: 'flex',
          flexDirection: 'column',
          padding: 16,
        },
      }}
    >
      <div style={{ marginBottom: 12, flexShrink: 0 }}>
        <Button type="text" icon={<ArrowLeftOutlined />} onClick={() => navigate('/pixel/sheet')}>
          返回
        </Button>
      </div>
      <Typography.Title level={4} style={{ marginTop: 0, marginBottom: 8, flexShrink: 0 }}>
        无限地图
      </Typography.Title>
      <Typography.Paragraph type="secondary" style={{ marginTop: 0, marginBottom: 12, fontSize: 13 }}>
        当前为<strong>本地过程化</strong>地形（噪声 + 小镇 + 野怪），不是按提示词 AI 生成。
        若要按风格扩图生成契合瓦片，请用「地图拼接」的 <strong>API生成 / 区域绘制</strong>。
      </Typography.Paragraph>
      <div className="infinite-map-host">
        <InfiniteMapScene />
      </div>
    </Card>
  )
}
