const STORAGE_KEY = 'soulnexus_pixel_project_snapshots'

export type ProjectSnapshot = {
  id: number
  name: string
  note: string
  createdAt: string
  assetCount: number
  assets: unknown[]
}

export function loadProjectSnapshots(): ProjectSnapshot[] {
  try {
    const raw = localStorage.getItem(STORAGE_KEY)
    return raw ? (JSON.parse(raw) as ProjectSnapshot[]) : []
  } catch {
    return []
  }
}

export function saveProjectSnapshot({
  name,
  assets,
  note = '',
}: {
  name: string
  assets: unknown[]
  note?: string
}): ProjectSnapshot {
  const list = loadProjectSnapshots()
  const entry: ProjectSnapshot = {
    id: Date.now(),
    name,
    note,
    createdAt: new Date().toISOString(),
    assetCount: assets?.length ?? 0,
    assets,
  }
  list.unshift(entry)
  localStorage.setItem(STORAGE_KEY, JSON.stringify(list.slice(0, 20)))
  return entry
}

export function deleteProjectSnapshot(id: number): ProjectSnapshot[] {
  const list = loadProjectSnapshots().filter((s) => s.id !== id)
  localStorage.setItem(STORAGE_KEY, JSON.stringify(list))
  return list
}

export function getProjectSnapshot(id: number): ProjectSnapshot | null {
  return loadProjectSnapshots().find((s) => s.id === id) ?? null
}
