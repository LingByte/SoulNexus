import React from 'react'
import ReactDOM from 'react-dom/client'
import '@arco-design/web-react/dist/css/arco.css'
import App from '@/App'
import './index.css'
import './styles/arco-dark.css'
import './styles/arco-popup.css'
import { ArcoAppProvider } from '@/providers/ArcoAppProvider'
import I18nSync from '@/i18n/I18nSync'
import ToastShell from '@/components/ui/ToastShell'

ReactDOM.createRoot(document.getElementById('root')!).render(
  <React.StrictMode>
    <ArcoAppProvider>
      <ToastShell>
        <I18nSync />
        <App />
      </ToastShell>
    </ArcoAppProvider>
  </React.StrictMode>
)
