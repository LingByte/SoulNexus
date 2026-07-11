#!/usr/bin/env node
import { run } from '../lib/cli.mjs'

run(process.argv).catch((err) => {
  console.error('[soul-pet]', err.message || err)
  process.exit(1)
})
