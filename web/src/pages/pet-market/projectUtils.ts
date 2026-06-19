import { PROJECT_FILES, type PetProjectV1 } from './types'
import {
  createProjectFromTemplate,
  DEFAULT_STARTER_TEMPLATE_ID,
} from './templates/registry'

export {
  createProjectFromTemplate,
  createProjectFromTemplateWithAssets,
  DEFAULT_STARTER_TEMPLATE_ID,
} from './templates/registry'
export { listStarterTemplates, getStarterTemplate } from './templates/registry'
export type { PetStarterTemplateId } from './templates/registry'

/** @deprecated use createProjectFromTemplate(templateId, name) */
export function createDefaultProject(name = '我的桌宠'): PetProjectV1 {
  return createProjectFromTemplate(DEFAULT_STARTER_TEMPLATE_ID, name)
}

export function parseProjectContent(content: string, fallbackName: string): PetProjectV1 {
  const trimmed = (content || '').trim()
  if (!trimmed) return createProjectFromTemplate(DEFAULT_STARTER_TEMPLATE_ID, fallbackName)
  if (trimmed.startsWith('{')) {
    try {
      const parsed = JSON.parse(trimmed) as {
        v?: number
        storage?: string
        entry?: string
        files?: Record<string, string>
      }
      if (parsed.v === 2 && parsed.storage === 'object') {
        return createProjectFromTemplate(DEFAULT_STARTER_TEMPLATE_ID, fallbackName)
      }
      if (parsed.v === 1 && parsed.files && typeof parsed.files === 'object') {
        return {
          v: 1,
          entry: parsed.entry || PROJECT_FILES.entry,
          files: { ...parsed.files },
        }
      }
    } catch {
      /* legacy plain JS */
    }
  }
  return migrateLegacyJs(content, fallbackName)
}

function migrateLegacyJs(content: string, name: string): PetProjectV1 {
  const project = createProjectFromTemplate(DEFAULT_STARTER_TEMPLATE_ID, name)
  project.files[PROJECT_FILES.entry] = content
  return project
}

export function serializeProject(project: PetProjectV1): string {
  return JSON.stringify(project, null, 2)
}

export function languageForFile(filename: string): string {
  if (filename.endsWith('.json')) return 'json'
  if (filename.endsWith('.css')) return 'css'
  if (filename.endsWith('.md')) return 'markdown'
  if (filename.endsWith('.html')) return 'html'
  return 'javascript'
}

export function petNameFromProject(project: PetProjectV1, fallback: string): string {
  try {
    const raw = project.files[PROJECT_FILES.manifest]
    if (raw) {
      const m = JSON.parse(raw) as { name?: string }
      if (m.name?.trim()) return m.name.trim()
    }
  } catch {
    /* ignore */
  }
  return fallback
}

export function descriptionFromReadme(project: PetProjectV1): string {
  const readme = project.files[PROJECT_FILES.readme] || ''
  const line = readme.split('\n').find((l) => l.trim() && !l.startsWith('#'))
  return line?.trim().slice(0, 120) || ''
}

export function previewLabelFromManifest(project: PetProjectV1): string {
  try {
    const raw = project.files[PROJECT_FILES.manifest]
    if (!raw) return 'Pet'
    const m = JSON.parse(raw) as {
      type?: string
      assets?: {
        sprite?: { name?: string }
        character?: string
      }
    }
    if (m.type === 'sprite') {
      const sp = m.assets?.sprite as { name?: string } | undefined
      if (sp && typeof sp === 'object' && 'name' in sp && (sp as { name?: string }).name?.trim()) {
        return (sp as { name?: string }).name!.trim()
      }
      return 'Sprite'
    }
    if (m.type === 'lottie') return 'Lottie'
  } catch {
    /* ignore */
  }
  return 'Pet'
}

/** @deprecated use previewLabelFromManifest */
export function previewEmojiFromManifest(project: PetProjectV1): string {
  return previewLabelFromManifest(project)
}
