import { useEffect } from 'react'
import dayjs from 'dayjs'
import 'dayjs/locale/zh-cn'
import 'dayjs/locale/zh-tw'
import 'dayjs/locale/en'
import 'dayjs/locale/ja'
import { useLocaleStore } from '@/stores/localeStore'
import { syncDocumentLanguage } from './index'

function dayjsLocale(appLocale: string): string {
  if (appLocale === 'en-US') return 'en'
  if (appLocale === 'ja-JP') return 'ja'
  if (appLocale === 'zh-TW') return 'zh-tw'
  return 'zh-cn'
}

/** Keeps document.lang and dayjs locale in sync with the locale store. */
export default function I18nSync() {
  const locale = useLocaleStore((s) => s.locale)

  useEffect(() => {
    syncDocumentLanguage(locale)
    dayjs.locale(dayjsLocale(locale))
  }, [locale])

  return null
}
