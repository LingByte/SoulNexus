import type { PetProjectV1 } from './types'
import { PROJECT_FILES } from './types'
import { buildGhostAnimations } from './ghostSpriteCatalog'
import {
  buildPandaLanlanAnimations,
  PANDA_LANLAN_BEHAVIORS,
  PANDA_LANLAN_EMOTION_MAP,
  PANDA_LANLAN_IDLE_FPS,
  PANDA_SEQUENCE_MARKER,
} from './pandaSpriteCatalog'
import { fetchDefaultPandaSpriteAssets } from './defaultPandaSprites'
import { fetchDefaultGhostSpriteAssets } from './defaultGhostSprites'
import { buildSpriteManifest, buildSpritePetJs } from './templates/spriteShared'
import { petNameFromProject } from './projectUtils'

const SPRITE_RUNTIME_MARKER = 'forcedAnim'
const DESKTOP_RUNTIME_MARKER = 'soul-pet-desktop-capable'

export function isLive2dProject(project: PetProjectV1): boolean {
  try {
    const raw = project.files[PROJECT_FILES.manifest]
    if (!raw) return false
    const m = JSON.parse(raw) as { type?: string }
    return m.type === 'live2d'
  } catch {
    return false
  }
}

export function isSpriteProject(project: PetProjectV1): boolean {
  try {
    const raw = project.files[PROJECT_FILES.manifest]
    if (!raw) return true
    const m = JSON.parse(raw) as { type?: string }
    return m.type === 'sprite' || m.type == null
  } catch {
    return true
  }
}

export function hasSpriteAssets(project: PetProjectV1): boolean {
  return Object.keys(project.files).some(
    (p) => p.startsWith('assets/sprites/') && /\.(png|jpe?g|webp|gif)$/i.test(p),
  )
}

export function ensureSpriteAssetsInProject(project: PetProjectV1): PetProjectV1 {
  if (hasSpriteAssets(project)) return project
  const readme =
    project.files['assets/sprites/README.md'] ??
    `# Ghost 精灵帧图

动作 PNG 放在此目录，manifest.json 的 assets.sprite.animations 里用 files 数组按帧序播放。

动作：idle / talk(sing) / tap(daze) / sad / cry / hide / falldown
`
  return {
    ...project,
    files: {
      ...project.files,
      'assets/sprites/README.md': readme,
    },
  }
}

export function isPandaLanlanProject(project: PetProjectV1): boolean {
  const raw = project.files[PROJECT_FILES.manifest] || ''
  return raw.includes(PANDA_SEQUENCE_MARKER) || raw.includes('panda_lanlan')
}

export async function fetchDefaultSpriteAssetsForProject(
  project: PetProjectV1,
  templateId?: string | null,
): Promise<Record<string, string>> {
  if (templateId === 'sprite-panda-lanlan' || isPandaLanlanProject(project)) {
    return fetchDefaultPandaSpriteAssets()
  }
  return fetchDefaultGhostSpriteAssets()
}

export function needsSpriteRuntimeUpgrade(project: PetProjectV1): boolean {
  const entry = project.files[PROJECT_FILES.entry] || ''
  const manifestRaw = project.files[PROJECT_FILES.manifest] || ''
  if (isSpriteProject(project) && !entry.includes(SPRITE_RUNTIME_MARKER)) return true
  if (isSpriteProject(project) && !entry.includes(DESKTOP_RUNTIME_MARKER)) return true
  if (isPandaLanlanProject(project)) {
    if (manifestRaw.includes('panda_lanlan_') && !manifestRaw.includes(PANDA_SEQUENCE_MARKER)) return true
    try {
      const m = JSON.parse(project.files[PROJECT_FILES.manifest] || '{}') as {
        assets?: { sprite?: { animations?: { idle?: { fps?: number } }; baseUrl?: string } }
      }
      const idleFps = m.assets?.sprite?.animations?.idle?.fps
      if (idleFps != null && idleFps < PANDA_LANLAN_IDLE_FPS) return true
      const baseUrl = m.assets?.sprite?.baseUrl || ''
      if (baseUrl.startsWith('http')) return true
    } catch {
      /* ignore */
    }
  }
  return false
}

/** Refresh pet.js + manifest (ghost or panda). */
export function upgradeSpriteRuntime(project: PetProjectV1, displayName?: string): PetProjectV1 {
  const petName = displayName || petNameFromProject(project, '我的桌宠')
  const panda = isPandaLanlanProject(project)
  const manifest = buildSpriteManifest({
    name: petName,
    animations: panda ? buildPandaLanlanAnimations() : buildGhostAnimations(),
    emotionMap: panda ? PANDA_LANLAN_EMOTION_MAP : undefined,
    behaviors: panda ? PANDA_LANLAN_BEHAVIORS : undefined,
    layout: panda ? { scale: 0.32 } : undefined,
    spriteBaseUrl: undefined,
  })
  return {
    ...project,
    files: {
      ...project.files,
      [PROJECT_FILES.manifest]: JSON.stringify(manifest, null, 2),
      [PROJECT_FILES.entry]: buildSpritePetJs(petName),
    },
  }
}
