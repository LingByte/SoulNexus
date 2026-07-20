/**
 * Tidy real files on ~/Desktop by type into dated folders.
 * Moves files (not Finder icon coordinates). Always confirm before calling.
 */
'use strict'

const fs = require('fs')
const path = require('path')
const os = require('os')

const CATEGORIES = [
  {
    name: '图片',
    exts: ['.png', '.jpg', '.jpeg', '.gif', '.webp', '.heic', '.bmp', '.svg', '.tiff', '.tif', '.ico'],
  },
  {
    name: '文档',
    exts: ['.pdf', '.doc', '.docx', '.xls', '.xlsx', '.ppt', '.pptx', '.txt', '.md', '.rtf', '.csv', '.pages', '.numbers', '.key'],
  },
  {
    name: '压缩包',
    exts: ['.zip', '.rar', '.7z', '.tar', '.gz', '.bz2', '.xz', '.dmg', '.pkg'],
  },
  {
    name: '代码',
    exts: ['.js', '.ts', '.tsx', '.jsx', '.py', '.go', '.rs', '.java', '.c', '.cpp', '.h', '.json', '.yml', '.yaml', '.toml', '.sh', '.swift', '.kt'],
  },
  {
    name: '影音',
    exts: ['.mp4', '.mov', '.mkv', '.avi', '.mp3', '.wav', '.flac', '.aac', '.m4a'],
  },
]

const SKIP_NAMES = new Set([
  '.DS_Store',
  '.localized',
  'desktop.ini',
  'Thumbs.db',
])

function desktopDir() {
  return path.join(os.homedir(), 'Desktop')
}

function categoryFor(fileName) {
  const ext = path.extname(fileName).toLowerCase()
  if (!ext) return '其它'
  for (const c of CATEGORIES) {
    if (c.exts.includes(ext)) return c.name
  }
  return '其它'
}

function uniquePath(dir, name) {
  const dest = path.join(dir, name)
  if (!fs.existsSync(dest)) return dest
  const parsed = path.parse(name)
  let i = 1
  while (i < 1000) {
    const candidate = path.join(dir, `${parsed.name} (${i})${parsed.ext}`)
    if (!fs.existsSync(candidate)) return candidate
    i += 1
  }
  return path.join(dir, `${parsed.name}-${Date.now()}${parsed.ext}`)
}

/**
 * Preview what would be moved (no writes).
 * @returns {{desktop:string, groups:Record<string,string[]>, total:number}}
 */
function preview() {
  const desktop = desktopDir()
  const groups = {}
  let total = 0
  if (!fs.existsSync(desktop)) return { desktop, groups, total: 0 }
  const entries = fs.readdirSync(desktop, { withFileTypes: true })
  for (const ent of entries) {
    if (!ent.isFile()) continue
    if (SKIP_NAMES.has(ent.name) || ent.name.startsWith('.')) continue
    const cat = categoryFor(ent.name)
    if (!groups[cat]) groups[cat] = []
    groups[cat].push(ent.name)
    total += 1
  }
  return { desktop, groups, total }
}

/**
 * Move files into Desktop/<Category>/YYYY-MM-DD/.
 * @returns {{ok:boolean, moved:number, folders:string[], errors:string[], skipped:number}}
 */
function tidy() {
  const desktop = desktopDir()
  const day = new Date().toISOString().slice(0, 10)
  const result = { ok: true, moved: 0, folders: [], errors: [], skipped: 0 }
  if (!fs.existsSync(desktop)) {
    result.ok = false
    result.errors.push('Desktop 不存在: ' + desktop)
    return result
  }

  const entries = fs.readdirSync(desktop, { withFileTypes: true })
  const folderSet = new Set()

  for (const ent of entries) {
    if (!ent.isFile()) {
      result.skipped += 1
      continue
    }
    if (SKIP_NAMES.has(ent.name) || ent.name.startsWith('.')) {
      result.skipped += 1
      continue
    }
    const cat = categoryFor(ent.name)
    const folder = path.join(desktop, cat, day)
    try {
      fs.mkdirSync(folder, { recursive: true })
      if (!folderSet.has(folder)) {
        folderSet.add(folder)
        result.folders.push(folder)
      }
      const src = path.join(desktop, ent.name)
      const dest = uniquePath(folder, ent.name)
      fs.renameSync(src, dest)
      result.moved += 1
    } catch (e) {
      result.errors.push(ent.name + ': ' + (e && e.message ? e.message : String(e)))
    }
  }

  if (result.errors.length) result.ok = result.moved > 0
  return result
}

module.exports = {
  desktopDir,
  preview,
  tidy,
  CATEGORIES,
}
