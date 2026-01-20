/**
 * JS代码验证工具
 * 使用简单的语法检查来验证JavaScript代码
 */

export interface ValidationResult {
  isValid: boolean
  error?: string
  line?: number
  column?: number
}

/**
 * 验证JavaScript代码语法
 * 使用Function构造器进行基本语法检查
 */
export function validateJavaScript(code: string): ValidationResult {
  if (!code || code.trim() === '') {
    return { isValid: true }
  }

  try {
    // 使用Function构造器进行语法检查
    // 注意：这不会执行代码，只是检查语法
    new Function(code)
    return { isValid: true }
  } catch (error: any) {
    // 尝试解析错误信息
    const errorMessage = error.message || String(error)
    const lineMatch = errorMessage.match(/line (\d+)/i)
    const columnMatch = errorMessage.match(/column (\d+)/i)
    
    return {
      isValid: false,
      error: errorMessage,
      line: lineMatch ? parseInt(lineMatch[1]) : undefined,
      column: columnMatch ? parseInt(columnMatch[1]) : undefined,
    }
  }
}

/**
 * 转义HTML特殊字符，防止XSS攻击
 */
export function escapeHtml(unsafe: string): string {
  return unsafe
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;')
    .replace(/'/g, '&#039;')
}

/**
 * 转义JavaScript字符串，用于在HTML中安全嵌入
 */
export function escapeJavaScript(code: string): string {
  // 转义反斜杠
  let escaped = code.replace(/\\/g, '\\\\')
  // 转义单引号
  escaped = escaped.replace(/'/g, "\\'")
  // 转义双引号
  escaped = escaped.replace(/"/g, '\\"')
  // 转义换行符
  escaped = escaped.replace(/\n/g, '\\n')
  // 转义回车符
  escaped = escaped.replace(/\r/g, '\\r')
  // 转义制表符
  escaped = escaped.replace(/\t/g, '\\t')
  return escaped
}

