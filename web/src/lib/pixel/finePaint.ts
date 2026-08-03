/** Flood-fill erase by color distance (4-connected). Mutates ImageData in place. */
export function superEraseAt(
  imageData: ImageData,
  x: number,
  y: number,
  tolerance: number,
): void {
  const { width: w, height: h, data } = imageData
  const sx = Math.floor(x)
  const sy = Math.floor(y)
  if (sx < 0 || sy < 0 || sx >= w || sy >= h) return

  const start = (sy * w + sx) * 4
  const tr = data[start]!
  const tg = data[start + 1]!
  const tb = data[start + 2]!
  const ta = data[start + 3]!
  if (ta === 0) return

  const match = (i: number) => {
    const dr = data[i]! - tr
    const dg = data[i + 1]! - tg
    const db = data[i + 2]! - tb
    const da = data[i + 3]! - ta
    return Math.sqrt(dr * dr + dg * dg + db * db + da * da * 0.25) <= tolerance
  }

  const visited = new Uint8Array(w * h)
  const stack: number[] = [sy * w + sx]
  visited[sy * w + sx] = 1
  const dx = [0, 1, 0, -1]
  const dy = [-1, 0, 1, 0]

  while (stack.length) {
    const idx = stack.pop()!
    const px = idx % w
    const py = (idx / w) | 0
    const o = idx * 4
    if (!match(o)) continue
    data[o + 3] = 0
    for (let k = 0; k < 4; k++) {
      const nx = px + dx[k]!
      const ny = py + dy[k]!
      if (nx < 0 || nx >= w || ny < 0 || ny >= h) continue
      const ni = ny * w + nx
      if (visited[ni]) continue
      visited[ni] = 1
      stack.push(ni)
    }
  }
}

export function paintDisk(
  imageData: ImageData,
  x: number,
  y: number,
  radius: number,
  color: { r: number; g: number; b: number; a: number },
  erase = false,
): void {
  const { width: w, height: h, data } = imageData
  const r = Math.max(1, Math.round(radius))
  const cx = Math.floor(x)
  const cy = Math.floor(y)
  for (let dy = -r; dy <= r; dy++) {
    for (let dx = -r; dx <= r; dx++) {
      if (dx * dx + dy * dy > r * r) continue
      const px = cx + dx
      const py = cy + dy
      if (px < 0 || py < 0 || px >= w || py >= h) continue
      const o = (py * w + px) * 4
      if (erase) {
        data[o + 3] = 0
      } else {
        data[o] = color.r
        data[o + 1] = color.g
        data[o + 2] = color.b
        data[o + 3] = color.a
      }
    }
  }
}
