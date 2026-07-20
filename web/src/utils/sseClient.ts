/** Parse one SSE message block (lines between blank lines). */
export function parseSSEBlock(
  block: string,
  onEvent: (event: string, data: string) => void,
) {
  let event = 'message'
  const dataLines: string[] = []
  for (const line of block.split('\n')) {
    if (line.startsWith('event:')) {
      event = line.slice(6).trim()
    } else if (line.startsWith('data:')) {
      dataLines.push(line.slice(5).trimStart())
    }
  }
  if (dataLines.length) {
    onEvent(event, dataLines.join('\n'))
  }
}
