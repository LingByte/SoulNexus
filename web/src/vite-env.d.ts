/// <reference types="vite/client" />

interface ImportMetaEnv {
  readonly MODE: string
  readonly DEV: boolean
  readonly PROD: boolean
  readonly SSR: boolean
  /** cmd/sip SIP_WEBSEAT_HTTP_ADDR — e.g. http://127.0.0.1:9080 */
  readonly VITE_SIP_WEBSEAT_HTTP_BASE?: string
  /** Same as gateway SIP_WEBSEAT_WS_TOKEN; optional */
  readonly VITE_SIP_WEBSEAT_WS_TOKEN?: string
}

interface ImportMeta {
  readonly env: ImportMetaEnv
}
