// 与 web/src/utils/passwordEncrypt.ts 与 templates/signin.html 中
// buildEncryptedPassword 保持完全一致：
//   format = passwordHash:encryptedHash:salt:timestamp(ms)
//   passwordHash  = SHA256(原始密码)
//   encryptedHash = SHA256(passwordHash + salt + timestamp)
// 后端 models.VerifyEncryptedPassword 使用相同算法校验。
//
// 实现使用 Web Crypto Subtle，避免引入额外依赖；本目录其他登录入口都应通过它发包。

import { authGet } from '@/api/client'

const sha256Hex = async (text: string): Promise<string> => {
  if (!window.crypto || !crypto.subtle) {
    throw new Error('当前浏览器不支持安全密码加密，请使用现代浏览器或 HTTPS 环境')
  }
  const buf = await crypto.subtle.digest('SHA-256', new TextEncoder().encode(text))
  return Array.from(new Uint8Array(buf))
    .map((b) => b.toString(16).padStart(2, '0'))
    .join('')
}

const fetchSalt = async (): Promise<string> => {
  const res = await authGet<{ salt: string; timestamp: number; expiresIn: number }>('/auth/salt')
  if (!res || res.code !== 200 || !res.data?.salt) {
    throw new Error(res?.msg || '获取登录盐失败')
  }
  return res.data.salt
}

/**
 * 把明文密码包装为后端可接受的加密传输串。
 * 空密码返回空串（让上层报「请输入密码」）。
 */
export const buildEncryptedPassword = async (rawPassword: string): Promise<string> => {
  if (!rawPassword) return ''
  const salt = await fetchSalt()
  const timestamp = Date.now()
  const passwordHash = await sha256Hex(rawPassword)
  const encryptedHash = await sha256Hex(`${passwordHash}${salt}${timestamp}`)
  return `${passwordHash}:${encryptedHash}:${salt}:${timestamp}`
}
