import { ReactNode } from 'react'
import Sidebar from './Sidebar.tsx'
import { WebSeatProvider } from '@/components/WebSeat/WebSeatProvider'

interface LayoutProps {
    children: ReactNode
}

const Layout = ({ children }: LayoutProps) => {
    return (
        <WebSeatProvider>
            <div className="h-screen flex flex-row bg-background text-foreground">
                <Sidebar />
                <main className="flex-1 min-h-0 overflow-auto bg-background lg:ml-0 pt-[104px] lg:pt-0">
                    {children}
                </main>
            </div>
        </WebSeatProvider>
    )
}

export default Layout
