import type { PetProjectV1 } from './types'

const draftKey = (id: string) => `pet-studio-draft:${id}`

/** Browser-only draft while editing; cleared after successful save to object storage. */
export function loadPetDraft(id: string): PetProjectV1 | null {
  try {
    const raw = sessionStorage.getItem(draftKey(id))
    if (!raw) return null
    const parsed = JSON.parse(raw) as PetProjectV1
    if (parsed?.v === 1 && parsed.files) return parsed
  } catch {
    /* ignore corrupt draft */
  }
  return null
}

export function savePetDraft(id: string, project: PetProjectV1): void {
  try {
    sessionStorage.setItem(draftKey(id), JSON.stringify(project))
  } catch {
    /* quota exceeded — editing still works in memory */
  }
}

export function clearPetDraft(id: string): void {
  sessionStorage.removeItem(draftKey(id))
}
