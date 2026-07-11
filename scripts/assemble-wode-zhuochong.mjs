#!/usr/bin/env node
/** Assemble 我的桌宠.soulpet from resources/action_* frame sequences. */
import { cpSync, existsSync, mkdirSync, readdirSync } from 'node:fs'
import { dirname, join } from 'node:path'
import { fileURLToPath } from 'node:url'

const repoRoot = join(dirname(fileURLToPath(import.meta.url)), '..')
const pkg = join(repoRoot, '我的桌宠.soulpet')
const resources = join(repoRoot, 'resources')

if (!existsSync(pkg)) {
  console.error('[assemble] missing', pkg)
  process.exit(1)
}

for (let action = 1; action <= 4; action += 1) {
  const src = join(resources, `action_${action}`)
  const dst = join(pkg, 'assets', 'sprites', 'panda', `action_${action}`)
  if (!existsSync(src)) {
    console.warn(`[assemble] skip missing ${src}`)
    continue
  }
  mkdirSync(dst, { recursive: true })
  let count = 0
  for (const name of readdirSync(src)) {
    if (!/\.png$/i.test(name)) continue
    cpSync(join(src, name), join(dst, name), { force: true })
    count += 1
  }
  console.log(`[assemble] action_${action}: ${count} frames`)
}

console.log('[assemble] done →', pkg)
