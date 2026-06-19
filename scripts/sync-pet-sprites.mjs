#!/usr/bin/env node
/** Sync repo sprites/ + root panda_lanlan_*.png → static/pet/examples/sprites/ */
import { cpSync, existsSync, mkdirSync, readdirSync } from 'node:fs'
import { dirname, join } from 'node:path'
import { fileURLToPath } from 'node:url'

const repoRoot = join(dirname(fileURLToPath(import.meta.url)), '..')
const spritesOut = join(repoRoot, 'static', 'pet', 'examples', 'sprites')
mkdirSync(spritesOut, { recursive: true })

const spritesSrc = join(repoRoot, 'sprites')
if (existsSync(spritesSrc)) {
  for (const name of readdirSync(spritesSrc)) {
    if (!/^ghost_.*\.png$/i.test(name)) continue
    cpSync(join(spritesSrc, name), join(spritesOut, name), { force: true })
  }
}

for (const name of readdirSync(repoRoot)) {
  if (!/^panda_lanlan_.*\.png$/i.test(name)) continue
  cpSync(join(repoRoot, name), join(spritesOut, name), { force: true })
}

console.log('[sync-pet-sprites] done → static/pet/examples/sprites')
