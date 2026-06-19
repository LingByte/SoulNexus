import { useEffect, lazy, Suspense } from 'react'
import { BrowserRouter as Router, Route, Routes, Navigate, useLocation } from 'react-router-dom';
import { consumeAuthTokenFromURL } from '@/utils/authBootstrap'
import { useAuthStore } from '@/stores/authStore'
import Home from '@/pages/Home.tsx';
import NotFound from "@/pages/NotFound.tsx";
import PWAInstaller from "@/components/PWA/PWAInstaller.tsx";
import ErrorBoundary from "@/components/ErrorBoundary/ErrorBoundary.tsx";
import DevErrorHandler from "@/components/Dev/DevErrorHandler.tsx";
import GlobalSearch from "@/components/UI/GlobalSearch.tsx";
import Profile from "@/pages/Profile.tsx";
import ProfileLayout from '@/pages/profile/ProfileLayout.tsx';
import Layout from "@/components/Layout/Layout.tsx";
import ResetPassword from "@/pages/ResetPassword.tsx";
import ProtectedRoute from "@/components/Auth/ProtectedRoute.tsx";
import RedirectToDevices from '@/components/RedirectToDevices.tsx';
import Privacy from '@/pages/Privacy.tsx';
import Terms from '@/pages/Terms.tsx';
import CookieConsent from '@/components/CookieConsent.tsx';
import AuthModal from '@/components/Auth/AuthModal.tsx';
import AccountDeletionRequest from '@/pages/AccountDeletionRequest.tsx';

// Lazy-loaded page components for code splitting (reduces initial bundle size)
const VoiceAssistant = lazy(() => import('@/pages/VoiceAssistant.tsx'));
const VoiceTrainingVolcengine = lazy(() => import('@/pages/VoiceTraining/VoiceTrainingVolcengine.tsx'));
const JSTemplateManager = lazy(() => import('@/pages/JSTemplateManager.tsx'));
const PetStudioPage = lazy(() => import('@/pages/pet-market/PetStudioPage.tsx'));
const Assistants = lazy(() => import('@/pages/Assistants.tsx'));
const GroupMembers = lazy(() => import('@/pages/GroupMembers.tsx'));
const GroupSettings = lazy(() => import('@/pages/GroupSettings.tsx'));
const GroupActivityLogs = lazy(() => import('@/pages/GroupActivityLogs.tsx'));
const DeviceManagement = lazy(() => import('@/pages/DeviceManagement.tsx'));
const DeviceDetail = lazy(() => import('@/pages/DeviceDetail.tsx'));
const WorkflowManager = lazy(() => import('@/pages/WorkflowManager.tsx'));
const KnowledgeListPage = lazy(() => import('@/pages/knowledge/KnowledgeListPage.tsx'));
const KnowledgeSpaceDetailPage = lazy(() => import('@/pages/knowledge/KnowledgeSpaceDetailPage.tsx'));
const KnowledgeDocumentDetailPage = lazy(() => import('@/pages/knowledge/KnowledgeDocumentDetailPage.tsx'));
const CallRecordingAnalytics = lazy(() => import('@/pages/CallRecordingAnalytics.tsx'));
const NodePluginMarket = lazy(() => import('@/pages/NodePluginMarket.tsx'));
const VoiceprintManagement = lazy(() => import('@/pages/VoiceprintManagement.tsx'));
const Playground = lazy(() => import('@/pages/Playground.tsx'));

// Shared loading fallback for lazy routes
const PageLoading = () => (
    <div className="flex items-center justify-center min-h-[60vh]">
        <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-500" />
    </div>
);

function AppRoutes() {
    const location = useLocation();
    const isHomePage = location.pathname === '/';
    const login = useAuthStore((s) => s.login)

    useEffect(() => {
        const token = consumeAuthTokenFromURL()
        if (token) {
            void login(token)
        }
    }, [login])
    
    return (
        <div className="min-h-screen bg-gray-50 dark:bg-gray-900 text-gray-900 dark:text-gray-100">
            <Suspense fallback={<PageLoading />}>
                    <Routes>
                        {/* 首页 - 独立布局，不需要 Layout */}
                        <Route path="/" element={<Home />} />
                        
                        {/* 隐私政策和服务条款 - 不需要登录 */}
                        <Route path="/privacy" element={<Privacy />} />
                        <Route path="/terms" element={<Terms />} />
                        
                        {/* 重置密码页面 - 不需要登录 */}
                        <Route path="/reset-password" element={<ResetPassword />} />

                        <Route path="/account-deletion/request" element={
                            <ProtectedRoute>
                                <AccountDeletionRequest />
                            </ProtectedRoute>
                        } />
                        
                        {/* 需要登录的页面 */}
                        {/* 个人中心：独立全屏布局，不含主导航 Sidebar（自有侧栏见 ProfileLayout） */}
                        <Route path="/profile" element={
                            <ProtectedRoute>
                                <ProfileLayout />
                            </ProtectedRoute>
                        }>
                            <Route index element={<Navigate to="personal" replace />} />
                            <Route path=":section" element={<Profile />} />
                        </Route>
                        <Route path="/devices" element={
                            <ProtectedRoute>
                                <Layout>
                                    <DeviceManagement />
                                </Layout>
                            </ProtectedRoute>
                        } />
                        <Route path="/devices/:deviceId" element={
                            <ProtectedRoute>
                                <Layout>
                                    <DeviceDetail />
                                </Layout>
                            </ProtectedRoute>
                        } />
                        {/* Redirect old device-management URLs to new devices URLs */}
                        <Route path="/device-management" element={<Navigate to="/devices" replace />} />
                        <Route path="/device-management/:deviceId" element={<RedirectToDevices />} />
                        <Route path="/playground" element={
                            <ProtectedRoute>
                                <Layout>
                                    <Playground />
                                </Layout>
                            </ProtectedRoute>
                        } />
                        <Route path="/assistants" element={
                            <ProtectedRoute>
                                <Layout>
                                    <Assistants />
                                </Layout>
                            </ProtectedRoute>
                        } />
                        <Route path="/voice-assistant/:id" element={
                            <ProtectedRoute>
                                <Layout>
                                    <VoiceAssistant />
                                </Layout>
                            </ProtectedRoute>
                        }/>
                        <Route path="/voice-assistant" element={<Navigate to="/assistants" replace />} />
                        <Route path="/voice-training" element={
                            <ProtectedRoute>
                                <Layout>
                                    <VoiceTrainingVolcengine />
                                </Layout>
                            </ProtectedRoute>
                        } />
                        <Route path="/notification" element={
                            <ProtectedRoute>
                                <Navigate to="/profile/notifications" replace />
                            </ProtectedRoute>
                        } />
                        <Route path="/credential" element={<Navigate to="/profile/credential" replace />} />
                        <Route path="/js-template" element={<Navigate to="/js-templates" replace />} />
                        <Route path="/js-templates" element={
                            <ProtectedRoute>
                                <Layout>
                                    <JSTemplateManager />
                                </Layout>
                            </ProtectedRoute>
                        } />
                        <Route path="/js-templates/new/edit" element={
                            <ProtectedRoute>
                                <PetStudioPage />
                            </ProtectedRoute>
                        } />
                        <Route path="/js-templates/:id/edit" element={
                            <ProtectedRoute>
                                <PetStudioPage />
                            </ProtectedRoute>
                        } />
                        <Route path="/billing" element={
                            <ProtectedRoute>
                                <Navigate to="/profile/billing" replace />
                            </ProtectedRoute>
                        } />
                        <Route path="/groups" element={
                            <ProtectedRoute>
                                <Navigate to="/profile/teams" replace />
                            </ProtectedRoute>
                        } />
                        <Route path="/groups/:id/members" element={
                            <ProtectedRoute>
                                <Layout>
                                    <GroupMembers />
                                </Layout>
                            </ProtectedRoute>
                        } />
                        <Route path="/groups/:id/settings" element={
                            <ProtectedRoute>
                                <Layout>
                                    <GroupSettings />
                                </Layout>
                            </ProtectedRoute>
                        } />
                        <Route path="/groups/:id/activity-logs" element={
                            <ProtectedRoute>
                                <Layout>
                                    <GroupActivityLogs />
                                </Layout>
                            </ProtectedRoute>
                        } />
                        <Route path="/knowledge" element={
                            <ProtectedRoute>
                                <Layout>
                                    <KnowledgeListPage />
                                </Layout>
                            </ProtectedRoute>
                        } />
                        <Route path="/knowledge/ns/:id" element={
                            <ProtectedRoute>
                                <Layout>
                                    <KnowledgeSpaceDetailPage />
                                </Layout>
                            </ProtectedRoute>
                        } />
                        <Route path="/knowledge/documents/:docId" element={
                            <ProtectedRoute>
                                <Layout>
                                    <KnowledgeDocumentDetailPage />
                                </Layout>
                            </ProtectedRoute>
                        } />
                        <Route path="/workflows" element={
                            <ProtectedRoute>
                                <Layout>
                                    <WorkflowManager />
                                </Layout>
                            </ProtectedRoute>
                        } />
                        <Route path="/node-plugins" element={
                            <ProtectedRoute>
                                <Layout>
                                    <NodePluginMarket />
                                </Layout>
                            </ProtectedRoute>
                        } />
                        <Route path="/voiceprint-management" element={
                            <ProtectedRoute>
                                <Layout>
                                    <VoiceprintManagement />
                                </Layout>
                            </ProtectedRoute>
                        } />
                        <Route path="/call-recording-analytics/:deviceId" element={
                            <ProtectedRoute>
                                <Layout>
                                    <CallRecordingAnalytics />
                                </Layout>
                            </ProtectedRoute>
                        } />
                        {/* 404页面 */}
                        <Route path="*" element={<NotFound />}/>
                    </Routes>

                    {/* PWA 安装提示：首页不展示，其它页面展示 */}
                    {!isHomePage && (
                        <PWAInstaller
                            showOnLoad={true}
                            delay={5000}
                            position="bottom-right"
                        />
                    )}

                    {/* 开发环境错误处理 */}
                    <DevErrorHandler />

                    {/* 全局搜索 */}
                    <GlobalSearch />

                    {/* Cookie 同意弹窗 */}
                    <CookieConsent />

                    {/* 全局登录弹窗 */}
                    <AuthModal />
            </Suspense>
        </div>
    );
}

function App() {
    return (
        <ErrorBoundary>
            <Router>
                <AppRoutes />
            </Router>
        </ErrorBoundary>
    );
}

export default App;