import type { SpriteAnimationDef } from './templates/spriteShared'

/** 1280px 宽，4 列网格雪碧图 */
const COLS = 4
const FW = 320

/** idle: 1280×2219 → 4×7 格，每格 320×317，实际使用 26 帧 */
const IDLE_FH = 317
export const PANDA_LANLAN_IDLE_FPS = 10

/** 其它动作: 1280×3443 → 4×11 格，每格 320×313，共 44 帧 */
const ACTION_FH = 313
const ACTION_FRAMES = 44

function sheetAnim(
  sheet: string,
  frames: number,
  frameHeight: number,
  fps: number,
  loop: boolean,
): SpriteAnimationDef {
  return {
    sheet,
    frameWidth: FW,
    frameHeight,
    frames,
    columns: COLS,
    fps,
    loop,
  }
}

export function buildPandaLanlanAnimations(): Record<string, SpriteAnimationDef> {
  return {
    idle: sheetAnim('panda_lanlan_idle.png', 26, IDLE_FH, PANDA_LANLAN_IDLE_FPS, true),
    hello: sheetAnim('panda_lanlan_hello.png', ACTION_FRAMES, ACTION_FH, 20, false),
    coy: sheetAnim('panda_lanlan_coy.png', ACTION_FRAMES, ACTION_FH, 20, false),
    cry: sheetAnim('panda_lanlan_cry.png', ACTION_FRAMES, ACTION_FH, 18, false),
    angry: sheetAnim('panda_lanlan_angry.png', ACTION_FRAMES, ACTION_FH, 20, false),
  }
}

export const PANDA_LANLAN_FILENAMES = [
  'panda_lanlan_idle.png',
  'panda_lanlan_hello.png',
  'panda_lanlan_coy.png',
  'panda_lanlan_cry.png',
  'panda_lanlan_angry.png',
] as const

export const PANDA_LANLAN_EMOTION_MAP: Record<string, string> = {
  neutral: 'idle',
  speaking: 'hello',
  joy: 'coy',
  sad: 'cry',
  angry: 'angry',
}

export const PANDA_LANLAN_BEHAVIORS: Record<string, unknown> = {
  lipSync: 'volume',
  talkAnimation: 'hello',
  dragEnabled: true,
  bounceOnTap: true,
  clickActions: ['hello', 'coy', 'cry', 'angry'],
  doubleTapAnimation: 'hello',
  ambientAnimations: ['hello', 'coy'],
  ambientIntervalMs: [12000, 28000],
}
