import type { SpriteAnimationDef } from './templates/spriteShared'

const FW = 128
const FH = 128

function seq(prefix: string, count: number): string[] {
  return Array.from({ length: count }, (_, i) => `${prefix}_${i + 1}.png`)
}

/** All ghost shimeji actions from sprites/ghost_*.png */
export function buildGhostAnimations(): Record<string, SpriteAnimationDef> {
  return {
    idle: {
      files: ['ghost_idle.png', 'ghost_flow_1.png'],
      frameWidth: FW,
      frameHeight: FH,
      frames: 2,
      fps: 3,
      loop: true,
    },
    talk: {
      files: seq('ghost_sing', 7),
      frameWidth: FW,
      frameHeight: FH,
      frames: 7,
      fps: 10,
      loop: true,
    },
    tap: {
      files: seq('ghost_daze', 8),
      frameWidth: FW,
      frameHeight: FH,
      frames: 8,
      fps: 12,
      loop: false,
    },
    sad: {
      files: seq('ghost_sad', 4),
      frameWidth: FW,
      frameHeight: FH,
      frames: 4,
      fps: 8,
      loop: false,
    },
    cry: {
      files: seq('ghost_cry', 4),
      frameWidth: FW,
      frameHeight: FH,
      frames: 4,
      fps: 10,
      loop: false,
    },
    hide: {
      files: seq('ghost_hide', 9),
      frameWidth: FW,
      frameHeight: FH,
      frames: 9,
      fps: 14,
      loop: false,
    },
    falldown: {
      files: seq('ghost_falldown', 3),
      frameWidth: FW,
      frameHeight: FH,
      frames: 3,
      fps: 10,
      loop: false,
    },
  }
}

export const GHOST_SPRITE_FILENAMES: string[] = [
  'ghost_idle.png',
  'ghost_flow_1.png',
  ...seq('ghost_sing', 7),
  ...seq('ghost_daze', 8),
  ...seq('ghost_sad', 4),
  ...seq('ghost_cry', 4),
  ...seq('ghost_hide', 9),
  ...seq('ghost_falldown', 3),
]

export const GHOST_EMOTION_MAP: Record<string, string> = {
  neutral: 'idle',
  joy: 'tap',
  sad: 'sad',
  surprise: 'cry',
  speaking: 'talk',
}

export const GHOST_BEHAVIORS: Record<string, unknown> = {
  lipSync: 'volume',
  talkAnimation: 'talk',
  dragEnabled: true,
  bounceOnTap: true,
  /** 每次左键单击依次播放（可点出全部动作） */
  clickActions: ['tap', 'sad', 'cry', 'hide', 'falldown'],
  doubleTapAnimation: 'hide',
  ambientAnimations: ['tap', 'sad', 'cry', 'hide', 'falldown'],
  ambientIntervalMs: [15000, 35000],
}
