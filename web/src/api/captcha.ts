export type CaptchaType = 'slider' | 'click' | 'image'

export interface CaptchaCharMarker {
  char: string
  x: number
  y: number
}

export interface CaptchaGenerateResult {
  id: string
  type: CaptchaType
  data: {
    trackWidth?: number
    width?: number
    height?: number
    length?: number
    targets?: string[]
    chars?: CaptchaCharMarker[]
    tolerance?: number
    image?: string
    background?: string
  }
  expires: string
}

export interface CaptchaProof {
  captchaId: string
  captchaType: CaptchaType
  captchaValue: number | Array<{ x: number; y: number }> | string
}

export function captchaTypeLabel(type: CaptchaType, t: (key: string) => string): string {
  switch (type) {
    case 'slider':
      return t('captcha.typeSlider')
    case 'click':
      return t('captcha.typeClick')
    case 'image':
      return t('captcha.typeImage')
    default:
      return t('captcha.title')
  }
}
