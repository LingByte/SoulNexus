/** 时域中值(窗口3) + EMA — 对齐 example/SoulMyimage temporal.py */

export class MedianEMASmoother {
  private beta: number

  constructor(beta = 0.6) {
    this.beta = beta
  }

  *smooth(alphaIter: Iterable<Float32Array>): Generator<Float32Array> {
    const it = alphaIter[Symbol.iterator]()
    let cur = it.next()
    if (cur.done) return

    let prev: Float32Array | null = null
    let prevOut: Float32Array | null = null

    let nxt = it.next()
    while (!nxt.done) {
      const med = median3(prev ?? cur.value, cur.value, nxt.value)
      const out = prevOut ? ema(med, prevOut, this.beta) : med
      yield out
      prevOut = out
      prev = cur.value
      cur = nxt
      nxt = it.next()
    }

    const lo = prev ?? cur.value
    const med = median3(lo, cur.value, cur.value)
    yield prevOut ? ema(med, prevOut, this.beta) : med
  }
}

function ema(cur: Float32Array, prev: Float32Array, beta: number): Float32Array {
  const out = new Float32Array(cur.length)
  const inv = 1 - beta
  for (let i = 0; i < cur.length; i++) out[i] = beta * cur[i] + inv * prev[i]
  return out
}

function median3(a: Float32Array, b: Float32Array, c: Float32Array): Float32Array {
  const out = new Float32Array(a.length)
  for (let i = 0; i < a.length; i++) {
    const x = a[i]
    const y = b[i]
    const z = c[i]
    out[i] = x > y ? (y > z ? y : x > z ? z : x) : x > z ? x : y > z ? z : y
  }
  return out
}

export function smoothAlphaSequence(alphas: Float32Array[], beta = 0.6): Float32Array[] {
  const smoother = new MedianEMASmoother(beta)
  return Array.from(smoother.smooth(alphas))
}
