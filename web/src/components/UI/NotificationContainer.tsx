// NotificationContainer.tsx
import { AnimatePresence } from 'framer-motion'
import { useUIStore } from '@/stores/uiStore.ts'
import PositionedToast from './PositionedToast'

type ToastPosition =
    | 'top-left'
    | 'top-right'
    | 'bottom-left'
    | 'bottom-right'
    | 'top-center'
    | 'bottom-center'

/** 把通知按位置分组，便于分别堆叠渲染 */
function groupByPosition<T extends { position?: ToastPosition }>(arr: T[]) {
  const map = new Map<ToastPosition, T[]>()
  const allPositions: ToastPosition[] = [
    'top-left',
    'top-right',
    'bottom-left',
    'bottom-right',
    'top-center',
    'bottom-center',
  ]
  for (const p of allPositions) map.set(p, [])
  for (const n of arr) {
    const pos = (n.position ?? 'top-right') as ToastPosition
    map.get(pos)!.push(n)
  }
  return map
}

const stackClassByPosition: Record<ToastPosition, string> = {
  'top-left': 'left-4 top-4 items-start',
  'top-right': 'right-4 top-4 items-end',
  'bottom-left': 'left-4 bottom-4 items-start',
  'bottom-right': 'right-4 bottom-4 items-end',
  'top-center': 'left-1/2 -translate-x-1/2 top-4 items-center',
  'bottom-center': 'left-1/2 -translate-x-1/2 bottom-4 items-center',
}

const NotificationContainer = () => {
  const { notifications, removeNotification } = useUIStore()

  const grouped = groupByPosition(notifications)

  return (
      <>
        {/* 为每个位置渲染一个可交互的堆叠容器 */}
        {Array.from(grouped.entries()).map(([position, list]) => {
          if (!list.length) return null
          return (
              <div
                  key={position}
                  className={`
              fixed flex flex-col gap-3 max-w-[92vw] sm:max-w-md z-[10000]
              ${stackClassByPosition[position]}
              pointer-events-auto
            `}
              >
                <AnimatePresence initial={false}>
                  {list.map((n) => (
                      <PositionedToast
                          key={n.id}
                          id={n.id}
                          type={n.type}
                          title={n.title}
                          message={n.message}
                          duration={n.duration}
                          onClose={removeNotification}
                          // 让单条通知可覆盖自身位置，否则用分组的 position
                          position={(n.position as ToastPosition) ?? position}
                          scrollToPosition={n.scrollToPosition}
                      />
                  ))}
                </AnimatePresence>
              </div>
          )
        })}
      </>
  )
}

export default NotificationContainer
