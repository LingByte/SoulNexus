import React from 'react'
import ReactDOM from 'react-dom/client'
import { ConfigProvider } from '@arco-design/web-react'
import zhCN from '@arco-design/web-react/es/locale/zh-CN'
import '@arco-design/web-react/dist/css/arco.css'
import App from '@/App.tsx'
import './index.css'

ReactDOM.createRoot(document.getElementById('root')!).render(
    <React.StrictMode>
        <ConfigProvider locale={zhCN}>
            <App />
        </ConfigProvider>
    </React.StrictMode>,
)
