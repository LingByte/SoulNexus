import { ReactNode } from 'react'
import Sidebar from './Sidebar.tsx'

interface LayoutProps {
    children: ReactNode
}

const Layout = ({ children }: LayoutProps) => {
    return (
        <div className="h-screen flex flex-row bg-background text-foreground">
            <Sidebar />
            <main className="flex-1 min-h-0 overflow-auto bg-background lg:ml-0 pt-[104px] lg:pt-0">
                <div className="h-full flex flex-col">
                    <div className="flex-1 overflow-auto">
                        <div className="mx-auto w-full max-w-7xl px-4 py-6 sm:px-6 lg:px-8">
                            {children}
                        </div>
                    </div>
                </div>
            </main>
        </div>
    )
}

export default Layout
