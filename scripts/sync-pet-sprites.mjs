#!/usr/bin/env node
/** Sync repo sprites/ + resources/action_* → static/pet/examples/sprites/ + web/public/pet-examples/sprites/ */
import { cpSync, existsSync, mkdirSync, readdirSync, rmSync } from 'node:fs'
import { dirname, join } from 'node:path'
import { fileURLToPath } from 'node:url'

const repoRoot = join(dirname(fileURLToPath(import.meta.url)), '..')
const spritesOut = join(repoRoot, 'static', 'pet', 'examples', 'sprites')
const webSpritesOut = join(repoRoot, 'web', 'public', 'pet-examples', 'sprites')
mkdirSync(spritesOut, { recursive: true })
mkdirSync(webSpritesOut, { recursive: true })

function copyGhostPng(name, srcDir, dstDir) {
  const src = join(srcDir, name)
  if (!existsSync(src)) return false
  cpSync(src, join(dstDir, name), { force: true })
  return true
}

const spritesSrc = join(repoRoot, 'sprites')
const staticSpritesSrc = join(spritesOut)
const ghostSources = [spritesSrc, staticSpritesSrc].filter((d) => existsSync(d))

function syncGhostFromDir(srcDir) {
  for (const name of readdirSync(srcDir)) {
    if (!/^ghost_.*\.png$/i.test(name)) continue
    if (srcDir !== spritesOut) {
      copyGhostPng(name, srcDir, spritesOut)
    }
    copyGhostPng(name, srcDir, webSpritesOut)
  }
}

for (const dir of ghostSources) {
  syncGhostFromDir(dir)
}

// Legacy grid sheets (replaced by resources/action_*)
for (const name of readdirSync(spritesOut)) {
  if (/^panda_lanlan_.*\.png$/i.test(name)) {
    rmSync(join(spritesOut, name), { force: true })
  }
}

const resourcesDir = join(repoRoot, 'resources')
const pandaOut = join(spritesOut, 'panda')
if (existsSync(resourcesDir)) {
  mkdirSync(pandaOut, { recursive: true })
  for (let action = 1; action <= 4; action += 1) {
    const srcDir = join(resourcesDir, `action_${action}`)
    if (!existsSync(srcDir)) continue
    const dstDir = join(pandaOut, `action_${action}`)
    mkdirSync(dstDir, { recursive: true })
    let count = 0
    for (const name of readdirSync(srcDir)) {
      if (!/\.png$/i.test(name)) continue
      cpSync(join(srcDir, name), join(dstDir, name), { force: true })
      count += 1
    }
    console.log(`[sync-pet-sprites] panda action_${action}: ${count} frames`)
  }
}

console.log('[sync-pet-sprites] done → static/pet/examples/sprites + web/public/pet-examples/sprites')
