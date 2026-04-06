/**
 * getUserMedia with fallback for environments where navigator.mediaDevices is missing
 * (non-secure origins except localhost, older WebViews, etc.).
 */
export async function getUserMediaAudioOnly(): Promise<MediaStream> {
  const constraints: MediaStreamConstraints = { audio: true, video: false }

  if (typeof navigator === 'undefined') {
    return Promise.reject(new Error('非浏览器环境，无法访问麦克风'))
  }

  const md = navigator.mediaDevices
  if (md?.getUserMedia) {
    return md.getUserMedia(constraints)
  }

  type LegacyGUM = (
    this: Navigator,
    c: MediaStreamConstraints,
    success: (s: MediaStream) => void,
    err: (e: Error) => void
  ) => void

  const n = navigator as Navigator & {
    getUserMedia?: LegacyGUM
    webkitGetUserMedia?: LegacyGUM
    mozGetUserMedia?: LegacyGUM
    msGetUserMedia?: LegacyGUM
  }

  const legacy: LegacyGUM | undefined =
    n.getUserMedia || n.webkitGetUserMedia || n.mozGetUserMedia || n.msGetUserMedia

  if (legacy) {
    return new Promise((resolve, reject) => {
      legacy.call(n, constraints, resolve, reject)
    })
  }

  return Promise.reject(
    new Error(
      '无法访问麦克风：请通过 HTTPS 或 http://localhost 打开本站并允许麦克风；若使用局域网 IP，请改为 HTTPS 或使用 localhost 访问。'
    )
  )
}
