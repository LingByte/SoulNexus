import { BrowserRouter as Router, Route, Routes, Navigate, useLocation } from 'react-router-dom';
import Home from '@/pages/Home.tsx';
import NotFound from "@/pages/NotFound.tsx";
import PWAInstaller from "@/components/PWA/PWAInstaller.tsx";
import ErrorBoundary from "@/components/ErrorBoundary/ErrorBoundary.tsx";
import VoiceAssistant from "@/pages/VoiceAssistant.tsx";
import VoiceTrainingVolcengine from "@/pages/VoiceTraining/VoiceTrainingVolcengine.tsx";
import DevErrorHandler from "@/components/Dev/DevErrorHandler.tsx";
import GlobalSearch from "@/components/UI/GlobalSearch.tsx";
import Profile from "@/pages/Profile.tsx";
import ProfileLayout from '@/pages/profile/ProfileLayout.tsx';
import Layout from "@/components/Layout/Layout.tsx";
import ResetPassword from "@/pages/ResetPassword.tsx";
import ProtectedRoute from "@/components/Auth/ProtectedRoute.tsx";
import JSTemplateManager from "@/pages/JSTemplateManager.tsx";
import Assistants from '@/pages/Assistants.tsx';
import GroupMembers from '@/pages/GroupMembers.tsx';
import GroupSettings from '@/pages/GroupSettings.tsx';
import GroupActivityLogs from '@/pages/GroupActivityLogs.tsx';
import DeviceManagement from '@/pages/DeviceManagement.tsx';
import DeviceDetail from '@/pages/DeviceDetail.tsx';
import RedirectToDevices from '@/components/RedirectToDevices.tsx';
import WorkflowManager from '@/pages/WorkflowManager.tsx';
import KnowledgeListPage from '@/pages/knowledge/KnowledgeListPage.tsx';
import KnowledgeSpaceDetailPage from '@/pages/knowledge/KnowledgeSpaceDetailPage.tsx';
import KnowledgeDocumentDetailPage from '@/pages/knowledge/KnowledgeDocumentDetailPage.tsx';
import CallRecordingAnalytics from '@/pages/CallRecordingAnalytics.tsx';
import NodePluginMarket from '@/pages/NodePluginMarket.tsx';
import VoiceprintManagement from '@/pages/VoiceprintManagement.tsx';
import Privacy from '@/pages/Privacy.tsx';
import Terms from '@/pages/Terms.tsx';
import CookieConsent from '@/components/CookieConsent.tsx';
import OIDCCallback from '@/pages/OIDCCallback.tsx';
import AccountDeletionRequest from '@/pages/AccountDeletionRequest.tsx';
import Playground from '@/pages/Playground.tsx';

function AppRoutes() {
    const location = useLocation();
    const isHomePage = location.pathname === '/';
    
    return (
        <div className="min-h-screen bg-gray-50 dark:bg-gray-900 text-gray-900 dark:text-gray-100">
                    <Routes>
                        {/* 首页 - 独立布局，不需要 Layout */}
                        <Route path="/" element={<Home />} />
                        
                        {/* 隐私政策和服务条款 - 不需要登录 */}
                        <Route path="/privacy" element={<Privacy />} />
                        <Route path="/terms" element={<Terms />} />
                        
                        {/* 重置密码页面 - 不需要登录 */}
                        <Route path="/reset-password" element={<ResetPassword />} />
                        <Route path="/auth/callback" element={<OIDCCallback />} />

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
                        <Route path="/js-templates" element={
                            <ProtectedRoute>
                                <Layout>
                                    <JSTemplateManager />
                                </Layout>
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