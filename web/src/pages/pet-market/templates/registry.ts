import type { PetProjectV1 } from '../types'
import { PROJECT_FILES } from '../types'
import { fetchDefaultGhostSpriteAssets } from '../defaultGhostSprites'
import { fetchDefaultPandaSpriteAssets } from '../defaultPandaSprites'
import {
  buildPandaLanlanAnimations,
  PANDA_LANLAN_BEHAVIORS,
  PANDA_LANLAN_EMOTION_MAP,
} from '../pandaSpriteCatalog'
import {
  buildSpriteManifest,
  buildSpritePetJs,
  buildSpriteReadme,
  buildSpriteStyle,
} from './spriteShared'

export type PetStarterTemplateId =
  | 'sprite-classic'
  | 'sprite-bouncy'
  | 'sprite-minimal'
  | 'sprite-panda-lanlan'

export interface PetStarterTemplate {
  id: PetStarterTemplateId
  badge: string
  nameKey: string
  descKey: string
  name: string
  description: string
  previewType: 'sprite'
  create: (petName?: string) => PetProjectV1
}

export const DEFAULT_STARTER_TEMPLATE_ID: PetStarterTemplateId = 'sprite-classic'

function buildSpriteProject(
  petName: string,
  variantLabel: string,
  overrides?: {
    behaviors?: Record<string, unknown>
    animations?: Record<string, unknown>
    emotionMap?: Record<string, string>
    layout?: { scale?: number; anchor?: 'bottom-center' | 'center'; offsetY?: number }
  },
): PetProjectV1 {
  const manifest = buildSpriteManifest({
    name: petName,
    animations: overrides?.animations as never,
    behaviors: overrides?.behaviors,
    emotionMap: overrides?.emotionMap,
    layout: overrides?.layout,
  })
  return {
    v: 1,
    entry: PROJECT_FILES.entry,
    files: {
      [PROJECT_FILES.manifest]: JSON.stringify(manifest, null, 2),
      [PROJECT_FILES.entry]: buildSpritePetJs(petName),
      [PROJECT_FILES.style]: buildSpriteStyle(),
      [PROJECT_FILES.readme]: buildSpriteReadme(petName, variantLabel),
    },
  }
}

const templates: PetStarterTemplate[] = [
  {
    id: 'sprite-classic',
    badge: '推荐',
    nameKey: 'petMarket.template.spriteClassic.name',
    descKey: 'petMarket.template.spriteClassic.desc',
    name: '帧动画 · 经典',
    description: 'idle / talk / tap 三态，PNG 精灵帧图，支持 TTS 对口型',
    previewType: 'sprite',
    create: (petName = '我的桌宠') => buildSpriteProject(petName, '帧动画 · 经典'),
  },
  {
    id: 'sprite-bouncy',
    badge: '互动',
    nameKey: 'petMarket.template.spriteBouncy.name',
    descKey: 'petMarket.template.spriteBouncy.desc',
    name: '帧动画 · 活泼',
    description: '可拖拽 + 点击弹跳，适合 Q 版角色',
    previewType: 'sprite',
    create: (petName = '我的桌宠') =>
      buildSpriteProject(petName, '帧动画 · 活泼', {
        behaviors: { dragEnabled: true, bounceOnTap: true },
      }),
  },
  {
    id: 'sprite-minimal',
    badge: '轻量',
    nameKey: 'petMarket.template.spriteMinimal.name',
    descKey: 'petMarket.template.spriteMinimal.desc',
    name: '帧动画 · 极简',
    description: '仅 idle + talk，占位绘制，适合先跑通逻辑再换美术',
    previewType: 'sprite',
    create: (petName = '我的桌宠') =>
      buildSpriteProject(petName, '帧动画 · 极简', {
        animations: {
          idle: {
            sheet: 'ghost_idle.png',
            frameWidth: 96,
            frameHeight: 96,
            frames: 1,
            fps: 4,
            loop: true,
          },
          talk: {
            files: Array.from({ length: 7 }, (_, i) => `ghost_sing_${i + 1}.png`),
            frameWidth: 96,
            frameHeight: 96,
            frames: 7,
            fps: 10,
            loop: true,
          },
        },
        behaviors: { dragEnabled: false, bounceOnTap: false },
      }),
  },
  {
    id: 'sprite-panda-lanlan',
    badge: '新',
    nameKey: 'petMarket.template.spritePandaLanlan.name',
    descKey: 'petMarket.template.spritePandaLanlan.desc',
    name: '懒懒熊猫 · 逐帧',
    description: 'resources 四动作逐帧 — idle / hello / coy / angry',
    previewType: 'sprite',
    create: (petName = '懒懒熊猫') =>
      buildSpriteProject(petName, '懒懒熊猫 · 逐帧', {
        animations: buildPandaLanlanAnimations(),
        emotionMap: PANDA_LANLAN_EMOTION_MAP,
        behaviors: PANDA_LANLAN_BEHAVIORS,
        layout: { scale: 0.32 },
      }),
  },
]

export function listStarterTemplates(): PetStarterTemplate[] {
  return templates
}

export function getStarterTemplate(id: string | null | undefined): PetStarterTemplate {
  const found = templates.find((t) => t.id === id)
  return found ?? templates[0]
}

export function createProjectFromTemplate(
  templateId: string | null | undefined,
  name = '我的桌宠',
): PetProjectV1 {
  return getStarterTemplate(templateId).create(name)
}

export async function createProjectFromTemplateWithAssets(
  templateId: string | null | undefined,
  name = '我的桌宠',
): Promise<PetProjectV1> {
  const project = createProjectFromTemplate(templateId, name)
  const sprites =
    templateId === 'sprite-panda-lanlan'
      ? await fetchDefaultPandaSpriteAssets()
      : await fetchDefaultGhostSpriteAssets()
  if (Object.keys(sprites).length === 0) return project
  return {
    ...project,
    files: { ...project.files, ...sprites },
  }
}
