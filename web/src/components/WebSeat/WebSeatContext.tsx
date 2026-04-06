import { createContext, useContext, type ReactNode } from 'react'

export type WebSeatWsState = 'disabled' | 'idle' | 'connecting' | 'open' | 'closed'

export interface WebSeatContextValue {
  /** True when VITE_SIP_WEBSEAT_HTTP_BASE is set */
  configured: boolean
  wsState: WebSeatWsState
  /** Short label for header/badge */
  wsStatusText: string
  /** Last `presence` push from gateway (only while WS is open). */
  presenceWsClients: number
  presenceOnline: boolean
  signalLog: string
  rxLog: string
  inCall: boolean
  hangupDisabled: boolean
  pendingIncomingCallId: string | null
  hangup: () => void
  /** Manual WS reconnect (e.g. after network flap); does not change ACD work state */
  reconnectWebSocket: () => void
  /** Connect WS and set all ACD `web` targets to available */
  goOnline: () => Promise<void>
  /** Set all ACD `web` targets to offline, then disconnect WS */
  goOffline: () => Promise<void>
}

const defaultValue: WebSeatContextValue = {
  configured: false,
  wsState: 'disabled',
  wsStatusText: '',
  presenceWsClients: 0,
  presenceOnline: false,
  signalLog: '',
  rxLog: '',
  inCall: false,
  hangupDisabled: true,
  pendingIncomingCallId: null,
  hangup: () => {},
  reconnectWebSocket: () => {},
  goOnline: async () => {},
  goOffline: async () => {},
}

export const WebSeatContext = createContext<WebSeatContextValue>(defaultValue)

export function useWebSeat(): WebSeatContextValue {
  return useContext(WebSeatContext)
}

export function WebSeatConsumer({ children }: { children: (v: WebSeatContextValue) => ReactNode }) {
  const v = useWebSeat()
  return <>{children(v)}</>
}
