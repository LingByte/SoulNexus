import { downloadTextFile } from '@/utils/downloadTextFile'

function escapeCsvCell(value: string): string {
  const v = value ?? ''
  if (/[",\n\r]/.test(v)) return `"${v.replace(/"/g, '""')}"`
  return v
}

export function rowsToCsv(headers: string[], rows: string[][]): string {
  const lines = [headers, ...rows].map((row) => row.map((cell) => escapeCsvCell(cell)).join(','))
  return `\uFEFF${lines.join('\n')}`
}

export function downloadCsvFile(filename: string, headers: string[], rows: string[][]): void {
  downloadTextFile(filename, rowsToCsv(headers, rows), 'text/csv;charset=utf-8')
}

export async function downloadXlsxFile(filename: string, headers: string[], rows: string[][]): Promise<void> {
  const XLSX = await import('@e965/xlsx')
  const sheet = XLSX.utils.aoa_to_sheet([headers, ...rows])
  const book = XLSX.utils.book_new()
  XLSX.utils.book_append_sheet(book, sheet, 'Sheet1')
  const outName = filename.endsWith('.xlsx') ? filename : `${filename}.xlsx`
  XLSX.writeFile(book, outName)
}

export type SpreadsheetExportFormat = 'csv' | 'xlsx'

export async function downloadSpreadsheetFile(
  format: SpreadsheetExportFormat,
  filenameBase: string,
  headers: string[],
  rows: string[][],
): Promise<void> {
  if (format === 'xlsx') {
    await downloadXlsxFile(`${filenameBase}.xlsx`, headers, rows)
  } else {
    downloadCsvFile(`${filenameBase}.csv`, headers, rows)
  }
}
