/** Multi-file pet project stored in JSTemplate.content JSON. */
export interface PetProjectV1 {
  v: 1
  entry: string
  files: Record<string, string>
}

export type PetProjectFile = keyof PetProjectV1['files'] | string

export const PROJECT_FILES = {
  manifest: 'manifest.json',
  entry: 'pet.js',
  style: 'style.css',
  readme: 'README.md',
} as const

export interface PetMarketItem {
  id: string
  jsSourceId: string
  name: string
  type: 'default' | 'custom'
  description?: string
  updated_at: string
  previewLabel?: string
}
