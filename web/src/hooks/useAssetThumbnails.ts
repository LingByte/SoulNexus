import { useEffect, useState } from 'react'
import { assetToBlob, type LocalAsset } from '@/lib/pixelcraft/localAssetStore'

export function useAssetThumbnails(assets: LocalAsset[]) {
  const [thumbMap, setThumbMap] = useState<Record<number, string>>({})

  useEffect(() => {
    const urls: string[] = []
    const map: Record<number, string> = {}
    for (const asset of assets) {
      if (!asset.mimeType?.startsWith('image/') || asset.id == null) continue
      const url = URL.createObjectURL(assetToBlob(asset))
      map[asset.id] = url
      urls.push(url)
    }
    setThumbMap(map)
    return () => {
      urls.forEach((u) => URL.revokeObjectURL(u))
    }
  }, [assets])

  return thumbMap
}
