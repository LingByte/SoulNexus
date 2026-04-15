/**
 * 根据用户名生成头像颜色
 */
export function getAvatarColor(name: string): string {
  if (!name) return 'from-blue-400 to-indigo-500'
  
  // 根据名字的字符码计算颜色
  let hash = 0
  for (let i = 0; i < name.length; i++) {
    hash = name.charCodeAt(i) + ((hash << 5) - hash)
  }
  
  // 预定义的颜色组合
  const colorPairs = [
    ['from-blue-400', 'to-indigo-500'],
    ['from-purple-400', 'to-pink-500'],
    ['from-green-400', 'to-emerald-500'],
    ['from-orange-400', 'to-red-500'],
    ['from-cyan-400', 'to-blue-500'],
    ['from-violet-400', 'to-purple-500'],
    ['from-rose-400', 'to-pink-500'],
    ['from-amber-400', 'to-orange-500'],
  ]
  
  const index = Math.abs(hash) % colorPairs.length
  return `${colorPairs[index][0]} ${colorPairs[index][1]}`
}

/**
 * 获取用户名的最后两个字（中文）或最后两个字符（英文）
 */
export function getAvatarText(name?: string, email?: string): string {
  const text = name || email || 'A'
  
  // 如果是中文，取最后两个字
  if (/[\u4e00-\u9fa5]/.test(text)) {
    return text.slice(-2)
  }
  
  // 如果是英文，取前两个字符的大写
  return text.slice(0, 2).toUpperCase()
}

