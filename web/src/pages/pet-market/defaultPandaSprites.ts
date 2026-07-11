import { SPRITE_ASSETS_PREFIX } from './templates/spriteShared'
import { PANDA_SEQUENCE_MARKER } from './pandaSpriteCatalog'
import { getApiBaseURL } from '@/config/apiConfig'

export function defaultPandaSpriteStaticBase(): string {
  const api = getApiBaseURL().replace(/\/$/, '')
  return `${api}/static/pet/examples/sprites/`
}

/**
 * 懒懒熊猫逐帧资源体积过大（~250MB），不嵌入 project base64。
 * 运行时从 /static/pet/examples/sprites/panda/action_* 加载。
 * 本地开发请先运行：node scripts/sync-pet-sprites.mjs
 */
export async function fetchDefaultPandaSpriteAssets(): Promise<Record<string, string>> {
  return {
    [`${SPRITE_ASSETS_PREFIX}README.md`]: `# 懒懒熊猫 · 逐帧动画

动作帧位于 \`${PANDA_SEQUENCE_MARKER}\` 等路径，由静态资源目录提供。

同步命令（仓库根目录）：

\`\`\`bash
node scripts/sync-pet-sprites.mjs
\`\`\`

动作：action_1=idle · action_2=hello · action_3=coy · action_4=angry
`,
  }
}
