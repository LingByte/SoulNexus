import fs from 'node:fs'
import path from 'node:path'
import zlib from 'node:zlib'

function crc32(buf) {
  let c = ~0
  for (let i = 0; i < buf.length; i++) {
    c ^= buf[i]
    for (let k = 0; k < 8; k++) c = (c >>> 1) ^ (0xedb88320 & -(c & 1))
  }
  return ~c >>> 0
}

function u16(n) {
  const b = Buffer.alloc(2)
  b.writeUInt16LE(n)
  return b
}

function u32(n) {
  const b = Buffer.alloc(4)
  b.writeUInt32LE(n >>> 0)
  return b
}

export function writeZip(outPath, files) {
  const parts = []
  const central = []
  let offset = 0
  for (const name of Object.keys(files).sort()) {
    const body = files[name].startsWith('base64:')
      ? Buffer.from(files[name].slice(7), 'base64')
      : Buffer.from(files[name], 'utf8')
    const compressed = zlib.deflateRawSync(body)
    const nameBuf = Buffer.from(name, 'utf8')
    const local = Buffer.concat([
      u32(0x04034b50),
      u16(20),
      u16(0),
      u16(8),
      u16(0),
      u16(0),
      u32(crc32(body)),
      u32(compressed.length),
      u32(body.length),
      u16(nameBuf.length),
      u16(0),
      nameBuf,
      compressed,
    ])
    parts.push(local)
    const cent = Buffer.concat([
      u32(0x02014b50),
      u16(20),
      u16(20),
      u16(0),
      u16(8),
      u16(0),
      u16(0),
      u32(crc32(body)),
      u32(compressed.length),
      u32(body.length),
      u16(nameBuf.length),
      u16(0),
      u16(0),
      u16(0),
      u16(0),
      u32(0),
      u32(offset),
      nameBuf,
    ])
    central.push(cent)
    offset += local.length
  }
  const centralBuf = Buffer.concat(central)
  const end = Buffer.concat([
    u32(0x06054b50),
    u16(0),
    u16(0),
    u16(Object.keys(files).length),
    u16(Object.keys(files).length),
    u32(centralBuf.length),
    u32(offset),
    u16(0),
  ])
  fs.writeFileSync(outPath, Buffer.concat([...parts, centralBuf, end]))
}
