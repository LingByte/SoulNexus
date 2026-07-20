/// <reference types="vite/client" />

interface ImportMetaEnv {
  readonly MODE: string
  readonly DEV: boolean
  readonly PROD: boolean
  readonly SSR: boolean
  /** Legal/display company name in auth footer (© year …). */
  readonly VITE_COMPANY_NAME?: string
  /** ICP filing number shown in site footer (e.g. 沪ICP备xxxxxxxx号). */
  readonly VITE_ICP_NUMBER?: string
  /** Link for ICP number; defaults to https://beian.miit.gov.cn/ */
  readonly VITE_ICP_LINK?: string
  /** Public security filing (公安备案), optional. */
  readonly VITE_PUBLIC_SECURITY_RECORD?: string
  readonly VITE_PUBLIC_SECURITY_LINK?: string
  readonly VITE_CONTACT_EMAIL?: string
  readonly VITE_GITHUB_URL?: string
  readonly VITE_PRIVACY_URL?: string
  readonly VITE_TERMS_URL?: string
  readonly VITE_API_BASE_URL?: string
  readonly VITE_WS_BASE_URL?: string
  readonly VITE_UPLOADS_BASE_URL?: string
  readonly VITE_LINGECHO_EMBED_SCRIPT_SRC?: string
  readonly VITE_LINGECHO_EMBED_API_BASE?: string
  readonly VITE_LINGECHO_EMBED_API_KEY?: string
  readonly VITE_LINGECHO_EMBED_ASSISTANT_ID?: string
  readonly VITE_LINGECHO_EMBED_TRANSPORT?: 'text' | 'websocket'
}

interface ImportMeta {
  readonly env: ImportMetaEnv
}
