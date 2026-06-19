import { BINARY_PREFIX } from './projectAssetUtils'
import { getApiBaseURL } from '@/config/apiConfig'
import { SPRITE_ASSETS_PREFIX } from './templates/spriteShared'
import {
  buildGhostAnimations,
  GHOST_SPRITE_FILENAMES,
} from './ghostSpriteCatalog'

export function defaultGhostSpriteStaticBase(): string {
  const api = getApiBaseURL().replace(/\/$/, '')
  return `${api}/static/pet/examples/sprites/`
}

export { buildGhostAnimations as DEFAULT_GHOST_ANIMATIONS }

async function fetchSpriteAsBase64(filename: string): Promise<string | null> {
  const url = `${defaultGhostSpriteStaticBase()}${encodeURIComponent(filename)}`
  try {
    const res = await fetch(url, { credentials: 'omit' })
    if (!res.ok) return null
    const blob = await res.blob()
    return await new Promise((resolve, reject) => {
      const reader = new FileReader()
      reader.onload = () => {
        const result = reader.result
        if (typeof result !== 'string') {
          reject(new Error('read failed'))
          return
        }
        const comma = result.indexOf(',')
        resolve(BINARY_PREFIX + (comma >= 0 ? result.slice(comma + 1) : result))
      }
      reader.onerror = () => reject(reader.error ?? new Error('read failed'))
      reader.readAsDataURL(blob)
    })
  } catch {
    return null
  }
}

/** Load default ghost PNGs into project files (assets/sprites/*). */
export async function fetchDefaultGhostSpriteAssets(): Promise<Record<string, string>> {
  const out: Record<string, string> = {}
  const results = await Promise.all(
    GHOST_SPRITE_FILENAMES.map(async (name) => {
      const content = await fetchSpriteAsBase64(name)
      return content ? ([`${SPRITE_ASSETS_PREFIX}${name}`, content] as const) : null
    }),
  )
  for (const row of results) {
    if (row) out[row[0]] = row[1]
  }
  return out
}
