/** Normalize phone-segment province labels (四川) to GeoJSON names (四川省). */

const SHORT_TO_FULL: Record<string, string> = {
  北京: '北京市',
  天津: '天津市',
  上海: '上海市',
  重庆: '重庆市',
  河北: '河北省',
  山西: '山西省',
  辽宁: '辽宁省',
  吉林: '吉林省',
  黑龙江: '黑龙江省',
  江苏: '江苏省',
  浙江: '浙江省',
  安徽: '安徽省',
  福建: '福建省',
  江西: '江西省',
  山东: '山东省',
  河南: '河南省',
  湖北: '湖北省',
  湖南: '湖南省',
  广东: '广东省',
  海南: '海南省',
  四川: '四川省',
  贵州: '贵州省',
  云南: '云南省',
  陕西: '陕西省',
  甘肃: '甘肃省',
  青海: '青海省',
  台湾: '台湾省',
  内蒙古: '内蒙古自治区',
  广西: '广西壮族自治区',
  西藏: '西藏自治区',
  宁夏: '宁夏回族自治区',
  新疆: '新疆维吾尔自治区',
  香港: '香港特别行政区',
  澳门: '澳门特别行政区',
}

/** GB/T 2260 province-level adcodes for DataV geoJSON. */
const SHORT_TO_ADCODE: Record<string, string> = {
  北京: '110000',
  天津: '120000',
  河北: '130000',
  山西: '140000',
  内蒙古: '150000',
  辽宁: '210000',
  吉林: '220000',
  黑龙江: '230000',
  上海: '310000',
  江苏: '320000',
  浙江: '330000',
  安徽: '340000',
  福建: '350000',
  江西: '360000',
  山东: '370000',
  河南: '410000',
  湖北: '420000',
  湖南: '430000',
  广东: '440000',
  广西: '450000',
  海南: '460000',
  重庆: '500000',
  四川: '510000',
  贵州: '520000',
  云南: '530000',
  西藏: '540000',
  陕西: '610000',
  甘肃: '620000',
  青海: '630000',
  宁夏: '640000',
  新疆: '650000',
  台湾: '710000',
  香港: '810000',
  澳门: '820000',
}

const FULL_TO_SHORT: Record<string, string> = Object.fromEntries(
  Object.entries(SHORT_TO_FULL).map(([short, full]) => [full, short]),
)

export function toGeoProvinceName(raw: string): string {
  const s = raw.trim()
  if (!s || s === '未知') return s
  if (SHORT_TO_FULL[s]) return SHORT_TO_FULL[s]
  if (FULL_TO_SHORT[s]) return s
  for (const [short, full] of Object.entries(SHORT_TO_FULL)) {
    if (s.startsWith(short)) return full
  }
  return s
}

export function toShortProvinceName(raw: string): string {
  const s = raw.trim()
  if (!s) return s
  if (FULL_TO_SHORT[s]) return FULL_TO_SHORT[s]
  if (SHORT_TO_FULL[s]) return s
  return s.replace(/(省|市|壮族自治区|回族自治区|维吾尔自治区|自治区|特别行政区)$/, '')
}

/** Phone segment city (成都) → GeoJSON city name (成都市). */
export function toGeoCityName(raw: string): string {
  const s = raw.trim()
  if (!s || s === '未知') return s
  if (/[市州盟县区]$/.test(s)) return s
  // 自治州 / 地区 commonly appear without suffix in phone DB
  if (s.endsWith('自治') || s.includes('地区') || s.includes('盟')) return s
  return `${s}市`
}

export function toShortCityName(raw: string): string {
  const s = raw.trim()
  if (!s) return s
  return s.replace(/(市|地区|盟)$/, '').replace(/自治州$/, '')
}

export function provinceAdcode(raw: string): string | null {
  const short = toShortProvinceName(raw)
  return SHORT_TO_ADCODE[short] || null
}

export const MAINLAND_PROVINCE_FULL_NAMES = Object.values(SHORT_TO_FULL).filter(
  (n) => !['台湾省', '香港特别行政区', '澳门特别行政区'].includes(n),
)
