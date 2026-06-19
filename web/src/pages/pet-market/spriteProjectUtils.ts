import type { PetProjectV1 } from './types'
import { PROJECT_FILES } from './types'
import { buildGhostAnimations } from './ghostSpriteCatalog'
import {
  buildPandaLanlanAnimations,
  PANDA_LANLAN_BEHAVIORS,
  PANDA_LANLAN_EMOTION_MAP,
  PANDA_LANLAN_IDLE_FPS,
} from './pandaSpriteCatalog'
import { fetchDefaultPandaSpriteAssets } from './defaultPandaSprites'
import { fetchDefaultGhostSpriteAssets } from './defaultGhostSprites'
import { buildSpriteManifest, buildSpritePetJs } from './templates/spriteShared'
import { petNameFromProject } from './projectUtils'

const SPRITE_RUNTIME_MARKER = 'forcedAnim'

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
  return Object.keys(project.files).some((p) => p.startsWith('assets/sprites/'))
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
  return raw.includes('panda_lanlan')
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
  if (isSpriteProject(project) && !entry.includes(SPRITE_RUNTIME_MARKER)) return true
  if (isPandaLanlanProject(project)) {
    try {
      const m = JSON.parse(project.files[PROJECT_FILES.manifest] || '{}') as {
        assets?: { sprite?: { animations?: { idle?: { fps?: number } } } }
      }
      const idleFps = m.assets?.sprite?.animations?.idle?.fps
      if (idleFps != null && idleFps < PANDA_LANLAN_IDLE_FPS) return true
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
    layout: panda ? { scale: 0.55 } : undefined,
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
