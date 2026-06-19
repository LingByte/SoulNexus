/**
 * 密码加密工具
 * 使用加盐哈希 + 时间戳防重放攻击
 */

import CryptoJS from 'crypto-js'
import { get } from '@/utils/request'

// 从后端获取随机盐的接口
export interface SaltResponse {
  salt: string
  timestamp: number
  expiresIn: number // 盐的有效期（秒）
}

// 加密后的密码格式
export interface EncryptedPassword {
  hash: string // 加密后的密码哈希
  salt: string // 使用的盐
  timestamp: number // 时间戳
}

/**
 * 获取随机盐（从后端）
 */
export async function getSalt(): Promise<SaltResponse> {
  const response = await get<SaltResponse>('/auth/salt')
  
  if (response.code !== 200) {
    throw new Error(response.msg || 'Failed to get salt')
  }
  
  return response.data
}

/**
 * 加密密码
 * @param password 明文密码
 * @param salt 盐值（如果未提供，会从后端获取）
 * @returns 加密后的密码对象
 */
export async function encryptPassword(
  password: string,
  salt?: string
): Promise<EncryptedPassword> {
  // 如果没有提供盐，从后端获取
  if (!salt) {
    const saltResponse = await getSalt()
    salt = saltResponse.salt
  }
  
  const timestamp = Date.now()
  
  // 第一步：计算原始密码的 SHA256（用于后端验证密码正确性）
  const passwordHash = CryptoJS.SHA256(password).toString()
  
  // 第二步：计算加密哈希：SHA256(SHA256(原始密码) + salt + timestamp)
  // 这样可以防止重放攻击，因为每次请求的时间戳都不同
  // 使用 passwordHash（即 SHA256(原始密码)）而不是原始密码本身
  const hashInput = `${passwordHash}${salt}${timestamp}`
  const encryptedHash = CryptoJS.SHA256(hashInput).toString()
  
  return {
    hash: `${passwordHash}:${encryptedHash}:${salt}:${timestamp}`, // 组合格式：passwordHash:encryptedHash:salt:timestamp
    salt,
    timestamp,
  }
}

/**
 * 验证加密密码格式
 */
export function validateEncryptedPassword(encrypted: EncryptedPassword): boolean {
  if (!encrypted.hash || !encrypted.salt || !encrypted.timestamp) {
    return false
  }
  
  // 检查时间戳是否在有效期内（5分钟内）
  const now = Date.now()
  const maxAge = 5 * 60 * 1000 // 5分钟
  if (now - encrypted.timestamp > maxAge) {
    return false
  }
  
  return true
}

/**
 * 将加密密码对象转换为字符串（用于传输）
 */
export function encryptPasswordToString(password: string, salt?: string): Promise<string> {
  return encryptPassword(password, salt).then(encrypted => {
    // 格式：passwordHash:encryptedHash:salt:timestamp
    return encrypted.hash
  })
}

/**
 * 从字符串解析加密密码对象
 */
export function parseEncryptedPassword(encryptedStr: string): EncryptedPassword {
  const parts = encryptedStr.split(':')
  if (parts.length !== 4) {
    throw new Error('Invalid encrypted password format')
  }
  
  return {
    hash: `${parts[0]}:${parts[1]}:${parts[2]}:${parts[3]}`, // passwordHash:encryptedHash:salt:timestamp
    salt: parts[2],
    timestamp: parseInt(parts[3], 10),
  }
}

