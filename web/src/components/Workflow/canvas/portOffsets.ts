/** 计算节点连接点垂直偏移（相对节点顶部，单位 px） */
export function getPortOffsets(count: number, anchorY = 44): number[] {
  if (count <= 0) return []
  if (count === 1) return [anchorY]
  const span = Math.min(18, 8 * (count - 1))
  const startY = anchorY - span / 2
  const step = count > 1 ? span / (count - 1) : 0
  return Array.from({ length: count }, (_, i) => startY + step * i)
}
