import CaptchaChallenge from '@/components/Captcha/CaptchaChallenge'
import type { CaptchaProof } from '@/api/captcha'

type Props = {
  onChange: (proof: CaptchaProof | null) => void
}

/** Inline captcha wrapper — server picks type at random on each load. */
export default function CaptchaField({ onChange }: Props) {
  return <CaptchaChallenge active onChange={onChange} />
}

export type { CaptchaProof }
