import React from 'react'
import ReactDOM from 'react-dom/client'
import App from '@/App.tsx'
import './index.css'
import '@arco-design/web-react/dist/css/arco.css'
import { applyStoredThemeBeforeReact } from '@/stores/themeStore'

applyStoredThemeBeforeReact()

// 生产环境移除 StrictMode 以避免双重渲染导致的闪烁
const isDevelopment = import.meta.env.DEV

ReactDOM.createRoot(document.getElementById('root')!).render(
    isDevelopment ? (
        <React.StrictMode>
            <App />
        </React.StrictMode>
    ) : (
        <App />
    )
)
