import type { SpriteAnimationDef } from './templates/spriteShared'

/** resources/action_* 逐帧 PNG（968×948） */
export const PANDA_FRAME_WIDTH = 968
export const PANDA_FRAME_HEIGHT = 948
export const PANDA_FRAME_COUNT = 121

export const PANDA_LANLAN_IDLE_FPS = 12

function pandaFramePaths(actionIndex: 1 | 2 | 3 | 4): string[] {
  return Array.from(
    { length: PANDA_FRAME_COUNT },
    (_, i) => `panda/action_${actionIndex}/frame_${String(i + 1).padStart(6, '0')}.png`,
  )
}

function sequenceAnim(
  actionIndex: 1 | 2 | 3 | 4,
  fps: number,
  loop: boolean,
): SpriteAnimationDef {
  const files = pandaFramePaths(actionIndex)
  return {
    files,
    frameWidth: PANDA_FRAME_WIDTH,
    frameHeight: PANDA_FRAME_HEIGHT,
    frames: PANDA_FRAME_COUNT,
    fps,
    loop,
  }
}

/**
 * 四个动作映射（resources/action_1 … action_4）：
 * - idle：待机循环
 * - hello：打招呼 / TTS 说话
 * - coy：害羞 / 点击互动
 * - angry：生气
 */
export function buildPandaLanlanAnimations(): Record<string, SpriteAnimationDef> {
  return {
    idle: sequenceAnim(1, PANDA_LANLAN_IDLE_FPS, true),
    hello: sequenceAnim(2, 24, true),
    coy: sequenceAnim(3, 20, false),
    angry: sequenceAnim(4, 20, false),
  }
}

/** 用于检测项目是否使用新版逐帧熊猫资源 */
export const PANDA_SEQUENCE_MARKER = 'panda/action_1/'

export const PANDA_LANLAN_EMOTION_MAP: Record<string, string> = {
  neutral: 'idle',
  speaking: 'hello',
  joy: 'coy',
  sad: 'coy',
  angry: 'angry',
}

export const PANDA_LANLAN_BEHAVIORS: Record<string, unknown> = {
  lipSync: 'volume',
  talkAnimation: 'hello',
  dragEnabled: true,
  bounceOnTap: true,
  clickActions: ['hello', 'coy', 'angry'],
  doubleTapAnimation: 'hello',
  ambientAnimations: ['hello', 'coy'],
  ambientIntervalMs: [12000, 28000],
}
