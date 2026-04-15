// 字符串工具函数
export const capitalize = (str: string) => {
  return str.charAt(0).toUpperCase() + str.slice(1).toLowerCase()
}

export const capitalizeWords = (str: string) => {
  return str.replace(/\w\S*/g, (txt) => 
    txt.charAt(0).toUpperCase() + txt.substr(1).toLowerCase()
  )
}

export const camelCase = (str: string) => {
  return str.replace(/-([a-z])/g, (g) => g[1].toUpperCase())
}

export const kebabCase = (str: string) => {
  return str.replace(/([a-z0-9])([A-Z])/g, '$1-$2').toLowerCase()
}

export const snakeCase = (str: string) => {
  return str.replace(/([a-z0-9])([A-Z])/g, '$1_$2').toLowerCase()
}

export const pascalCase = (str: string) => {
  return str.replace(/(?:^|[-_])(\w)/g, (_, c) => c.toUpperCase())
}

export const truncate = (str: string, length: number, suffix: string = '...') => {
  if (str.length <= length) return str
  return str.substring(0, length - suffix.length) + suffix
}

export const truncateWords = (str: string, wordCount: number, suffix: string = '...') => {
  const words = str.split(' ')
  if (words.length <= wordCount) return str
  return words.slice(0, wordCount).join(' ') + suffix
}

export const slugify = (str: string) => {
  return str
    .toLowerCase()
    .replace(/[^\w\s-]/g, '')
    .replace(/[\s_-]+/g, '-')
    .replace(/^-+|-+$/g, '')
}

export const removeAccents = (str: string) => {
  return str.normalize('NFD').replace(/[\u0300-\u036f]/g, '')
}

export const escapeHtml = (str: string) => {
  const div = document.createElement('div')
  div.textContent = str
  return div.innerHTML
}

export const unescapeHtml = (str: string) => {
  const div = document.createElement('div')
  div.innerHTML = str
  return div.textContent || div.innerText || ''
}

export const stripHtml = (str: string) => {
  return str.replace(/<[^>]*>/g, '')
}

export const highlight = (text: string, searchTerm: string, className: string = 'highlight') => {
  if (!searchTerm) return text
  const regex = new RegExp(`(${searchTerm})`, 'gi')
  return text.replace(regex, `<span class="${className}">$1</span>`)
}

export const mask = (str: string, visibleStart: number = 4, visibleEnd: number = 4, maskChar: string = '*') => {
  if (str.length <= visibleStart + visibleEnd) return str
  const start = str.substring(0, visibleStart)
  const end = str.substring(str.length - visibleEnd)
  const middle = maskChar.repeat(str.length - visibleStart - visibleEnd)
  return start + middle + end
}

export const maskEmail = (email: string) => {
  const [local, domain] = email.split('@')
  if (!local || !domain) return email
  const maskedLocal = local.length > 2 
    ? local[0] + '*'.repeat(local.length - 2) + local[local.length - 1]
    : local
  return `${maskedLocal}@${domain}`
}

export const maskPhone = (phone: string) => {
  const cleaned = phone.replace(/\D/g, '')
  if (cleaned.length < 7) return phone
  return cleaned.replace(/(\d{3})\d{4}(\d{4})/, '$1****$2')
}

export const generateId = (length: number = 8) => {
  const chars = 'ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789'
  let result = ''
  for (let i = 0; i < length; i++) {
    result += chars.charAt(Math.floor(Math.random() * chars.length))
  }
  return result
}

export const generateSlug = (str: string, length: number = 8) => {
  const slug = slugify(str)
  const randomId = generateId(length)
  return `${slug}-${randomId}`
}

export const pluralize = (count: number, singular: string, plural?: string) => {
  if (count === 1) return singular
  return plural || `${singular}s`
}

export const ordinalize = (num: number) => {
  const j = num % 10
  const k = num % 100
  if (j === 1 && k !== 11) return `${num}st`
  if (j === 2 && k !== 12) return `${num}nd`
  if (j === 3 && k !== 13) return `${num}rd`
  return `${num}th`
}

export const wrap = (str: string, width: number, breakChar: string = '\n') => {
  if (str.length <= width) return str
  const words = str.split(' ')
  const lines = []
  let currentLine = ''
  
  for (const word of words) {
    if ((currentLine + word).length <= width) {
      currentLine += (currentLine ? ' ' : '') + word
    } else {
      if (currentLine) lines.push(currentLine)
      currentLine = word
    }
  }
  
  if (currentLine) lines.push(currentLine)
  return lines.join(breakChar)
}

export const reverse = (str: string) => {
  return str.split('').reverse().join('')
}

export const isPalindrome = (str: string) => {
  const cleaned = str.toLowerCase().replace(/[^a-z0-9]/g, '')
  return cleaned === reverse(cleaned)
}

export const countWords = (str: string) => {
  return str.trim().split(/\s+/).filter(word => word.length > 0).length
}

export const countCharacters = (str: string, includeSpaces: boolean = true) => {
  return includeSpaces ? str.length : str.replace(/\s/g, '').length
}

export const countLines = (str: string) => {
  return str.split('\n').length
}

export const removeDuplicates = (str: string, separator: string = ' ') => {
  const items = str.split(separator)
  return [...new Set(items)].join(separator)
}

export const shuffle = (str: string) => {
  return str.split('').sort(() => Math.random() - 0.5).join('')
}

export const sort = (str: string, separator: string = '') => {
  return str.split(separator).sort().join(separator)
}
