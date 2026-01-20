import { useState, useCallback } from 'react'

interface UseCopyToClipboardResult {
  copy: (text: string) => Promise<boolean>
  copied: boolean
  error: string | null
}

export function useCopyToClipboard(): UseCopyToClipboardResult {
  const [copied, setCopied] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const copy = useCallback(async (text: string): Promise<boolean> => {
    try {
      setError(null)
      
      if (!navigator.clipboard) {
        throw new Error('Clipboard API not supported')
      }

      await navigator.clipboard.writeText(text)
      setCopied(true)
      
      // 重置状态
      setTimeout(() => setCopied(false), 2000)
      
      return true
    } catch (err) {
      const errorMessage = err instanceof Error ? err.message : 'Failed to copy'
      setError(errorMessage)
      setCopied(false)
      return false
    }
  }, [])

  return { copy, copied, error }
}
