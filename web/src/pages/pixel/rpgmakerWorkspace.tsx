import { Button } from '@/components/ui'
import { ExternalLink } from 'lucide-react'
import {
  GEM_RPGMAKER_URL_V1,
  GEM_RPGMAKER_URL_V1_1,
  GEM_RPGMAKER_URL_V3,
  GEM_PIXEL_RPGMAKER_XP_URL,
  GEM_PIXEL_RPGMAKER_VX_URL,
  GEM_PIXEL_POTPOURRI,
} from '@/lib/pixel/gemPixelUrls'

const RPG_CORE = [
  { title: 'RPGMAKER Gem V1', desc: '经典一键角色管线', url: GEM_RPGMAKER_URL_V1 },
  { title: 'RPGMAKER Gem V1.1', desc: '改进提示与构图', url: GEM_RPGMAKER_URL_V1_1 },
  { title: 'RPGMAKER Gem V3', desc: '最新版本', url: GEM_RPGMAKER_URL_V3 },
  { title: 'RPG Maker XP 风格', desc: 'XP 素材风格生成', url: GEM_PIXEL_RPGMAKER_XP_URL },
  { title: 'RPG Maker VX 风格', desc: 'VX 素材风格生成', url: GEM_PIXEL_RPGMAKER_VX_URL },
]

const LABELS: Record<string, string> = {
  moduleNanobananaRpgmakerGemV1: 'RPGMAKER V1',
  moduleNanobananaRpgmakerGemV1_1: 'RPGMAKER V1.1',
  moduleNanobananaRpgmakerGemV3: 'RPGMAKER V3',
  gemV2Link1: '角色 V2 · 1',
  gemV2Link2: '角色 V2 · 2',
  gemV2Link3: '角色 V3',
  moduleGemMonsterZombieB1: '僵尸怪 B1',
  moduleGemMonsterZombieB2: '僵尸怪 B2',
  moduleCharGenV23OT: '角色 V2.3 OT',
  nanobananaSceneLink1: '场景 1',
  nanobananaSceneLink2: '场景 2',
  nanobananaSceneLink3: '场景 3',
  nanobananaSceneLink4: '街机场景',
  moduleIllust: '插画',
  nanobananaFullCharBtn1: '全身 V4TX3',
  nanobananaFullCharBtn2: '横版角色',
  nanobananaFullCharBtn3: '8 向俯视',
  nanobananaFullCharBtn4: '骑马',
  nanobananaFullCharBtn5: '单图全动作',
  nanobananaFullCharBtn6: '单图全动作 2',
  aiPixelAnimalsGemDog: '像素狗',
  aiPixelAnimalsGemBirdMonster: '鸟怪',
  aiPixelAnimalsGemJikun: '鸡鲲',
}

export default function RpgmakerWorkspace() {
  return (
    <div className="space-y-6">
      <div className="rounded-xl border border-border bg-card p-4">
        <h3 className="text-base font-semibold text-foreground">RPGMAKER 一键管线</h3>
        <p className="mt-1 text-sm text-muted-foreground">
          在 Google Gemini Gem 中完成生成（需登录 Google）。本页仅提供入口与说明，处理结果可回到「像素处理 / 去水印 / Sheet 调整」继续本地精修。
        </p>
      </div>

      <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
        {RPG_CORE.map((item) => (
          <a
            key={item.url}
            href={item.url}
            target="_blank"
            rel="noreferrer"
            className="group rounded-xl border border-border bg-card p-4 transition hover:border-primary/40 hover:bg-primary/5"
          >
            <div className="flex items-start justify-between gap-2">
              <div>
                <p className="font-medium text-foreground">{item.title}</p>
                <p className="mt-1 text-xs text-muted-foreground">{item.desc}</p>
              </div>
              <ExternalLink size={16} className="shrink-0 text-muted-foreground group-hover:text-primary" />
            </div>
            <Button className="mt-3" size="sm" variant="outline">
              打开 Gemini Gem
            </Button>
          </a>
        ))}
      </div>

      <div>
        <h4 className="mb-3 text-sm font-semibold text-foreground">更多像素 Gems</h4>
        <div className="flex flex-wrap gap-2">
          {GEM_PIXEL_POTPOURRI.map((item) => (
            <a key={item.url + item.labelKey} href={item.url} target="_blank" rel="noreferrer">
              <Button size="sm" variant="outline" icon={<ExternalLink size={12} />}>
                {LABELS[item.labelKey] || item.labelKey}
              </Button>
            </a>
          ))}
        </div>
      </div>
    </div>
  )
}
