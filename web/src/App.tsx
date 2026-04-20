import { BrowserRouter as Router, Route, Routes, Navigate, useLocation } from 'react-router-dom';
import { useState } from 'react';
import Home from '@/pages/Home.tsx';
import NotFound from "@/pages/NotFound.tsx";
import PWAInstaller from "@/components/PWA/PWAInstaller.tsx";
import ErrorBoundary from "@/components/ErrorBoundary/ErrorBoundary.tsx";
import VoiceAssistant from "@/pages/VoiceAssistant.tsx";
import VoiceTrainingIndex from "@/pages/VoiceTraining/VoiceTrainingIndex.tsx";
import VoiceTrainingXunfei from "@/pages/VoiceTraining/VoiceTrainingXunfei.tsx";
import VoiceTrainingVolcengine from "@/pages/VoiceTraining/VoiceTrainingVolcengine.tsx";
import DevErrorHandler from "@/components/Dev/DevErrorHandler.tsx";
import GlobalSearch from "@/components/UI/GlobalSearch.tsx";
import NotificationContainer from "@/components/UI/NotificationContainer.tsx";
import { ToastProvider } from "@/components/UI/ToastContainer.tsx";
import NotificationCenter from "@/pages/NotificationCenter.tsx";
import Profile from "@/pages/Profile.tsx";
import AnimationShowcase from "@/pages/AnimationShowcase.tsx";
import Layout from "@/components/Layout/Layout.tsx";
import ResetPassword from "@/pages/ResetPassword.tsx";
import CredentialManager from "@/pages/CredentialManager.tsx";
import ProtectedRoute from "@/components/Auth/ProtectedRoute.tsx";
import JSTemplateManager from "@/pages/JSTemplateManager.tsx";
import Assistants from '@/pages/Assistants.tsx';
import AssistantGraph from '@/pages/AssistantGraph.tsx';
import Billing from '@/pages/Billing.tsx';
import Groups from '@/pages/Groups.tsx';
import GroupMembers from '@/pages/GroupMembers.tsx';
import GroupSettings from '@/pages/GroupSettings.tsx';
import GroupActivityLogs from '@/pages/GroupActivityLogs.tsx';
import OverviewEditorPage from '@/pages/OverviewEditorPage.tsx';
import Alerts from '@/pages/Alerts.tsx';
import AlertRules from '@/pages/AlertRules.tsx';
import AlertRuleForm from '@/pages/AlertRuleForm.tsx';
import AlertDetail from '@/pages/AlertDetail.tsx';
import UserQuotas from '@/pages/UserQuotas.tsx';
import DeviceManagement from '@/pages/DeviceManagement.tsx';
import DeviceDetail from '@/pages/DeviceDetail.tsx';
import RedirectToDevices from '@/components/RedirectToDevices.tsx';
import WorkflowManager from '@/pages/WorkflowManager.tsx';
import Overview from '@/pages/Overview.tsx';
import CallRecordingAnalytics from '@/pages/CallRecordingAnalytics.tsx';
import NodePluginMarket from '@/pages/NodePluginMarket.tsx';
import VoiceprintManagement from '@/pages/VoiceprintManagement.tsx';
import MCPManagement from '@/pages/MCPManagement.tsx';
import MCPMarketplace from '@/pages/MCPMarketplace.tsx';
import Privacy from '@/pages/Privacy.tsx';
import Terms from '@/pages/Terms.tsx';
import CookieConsent from '@/components/CookieConsent.tsx';
import KnowledgeBaseManager from '@/pages/KnowledgeBaseManager.tsx';
import OIDCCallback from '@/pages/OIDCCallback.tsx';
import AccountDeletionRequest from '@/pages/AccountDeletionRequest.tsx';

function AppRoutes() {
    const [showPerformanceMonitor, setShowPerformanceMonitor] = useState(false);
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
                        <Route path="/overview" element={
                            <ProtectedRoute>
                                <Layout>
                                    <Overview />
                                </Layout>
                            </ProtectedRoute>
                        } />
                        <Route path="/profile" element={
                            <ProtectedRoute>
                                <Layout>
                                    <Profile />
                                </Layout>
                            </ProtectedRoute>
                        } />
                        <Route path="/quotas" element={
                            <ProtectedRoute>
                                <Layout>
                                    <UserQuotas />
                                </Layout>
                            </ProtectedRoute>
                        } />
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
                        <Route path="/assistants" element={
                            <ProtectedRoute>
                                <Layout>
                                    <Assistants />
                                </Layout>
                            </ProtectedRoute>
                        } />
                                                <Route path="/assistants/:id/graph" element={
                            <ProtectedRoute>
                                <Layout>
                                    <AssistantGraph />
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
                                    <VoiceTrainingIndex />
                                </Layout>
                            </ProtectedRoute>
                        } />
                        <Route path="/voice-training/xunfei" element={
                            <ProtectedRoute>
                                <Layout>
                                    <VoiceTrainingXunfei />
                                </Layout>
                            </ProtectedRoute>
                        } />
                        <Route path="/voice-training/volcengine" element={
                            <ProtectedRoute>
                                <Layout>
                                    <VoiceTrainingVolcengine />
                                </Layout>
                            </ProtectedRoute>
                        } />
                        <Route path="/animate" element={
                            <ProtectedRoute>
                                <Layout>
                                    <AnimationShowcase />
                                </Layout>
                            </ProtectedRoute>
                        } />
                        <Route path="/notification" element={
                            <ProtectedRoute>
                                <Layout>
                                    <NotificationCenter />
                                </Layout>
                            </ProtectedRoute>
                        } />
                        <Route path="/credential" element={
                            <ProtectedRoute>
                                <Layout>
                                    <CredentialManager />
                                </Layout>
                            </ProtectedRoute>
                        } />
                        <Route path="/js-templates" element={
                            <ProtectedRoute>
                                <Layout>
                                    <JSTemplateManager />
                                </Layout>
                            </ProtectedRoute>
                        } />
                        <Route path="/knowledge-base" element={
                            <ProtectedRoute>
                                <Layout>
                                    <KnowledgeBaseManager />
                                </Layout>
                            </ProtectedRoute>
                        } />
                        <Route path="/billing" element={
                            <ProtectedRoute>
                                <Layout>
                                    <Billing />
                                </Layout>
                            </ProtectedRoute>
                        } />
                        <Route path="/groups" element={
                            <ProtectedRoute>
                                <Layout>
                                    <Groups />
                                </Layout>
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
                        <Route path="/groups/:id/overview/edit" element={
                            <ProtectedRoute>
                                <Layout>
                                    <OverviewEditorPage />
                                </Layout>
                            </ProtectedRoute>
                        } />
                        <Route path="/alerts" element={
                            <ProtectedRoute>
                                <Layout>
                                    <Alerts />
                                </Layout>
                            </ProtectedRoute>
                        } />
                        <Route path="/alerts/rules" element={
                            <ProtectedRoute>
                                <Layout>
                                    <AlertRules />
                                </Layout>
                            </ProtectedRoute>
                        } />
                        <Route path="/alerts/rules/new" element={
                            <ProtectedRoute>
                                <Layout>
                                    <AlertRuleForm />
                                </Layout>
                            </ProtectedRoute>
                        } />
                        <Route path="/alerts/rules/:id/edit" element={
                            <ProtectedRoute>
                                <Layout>
                                    <AlertRuleForm />
                                </Layout>
                            </ProtectedRoute>
                        } />
                        <Route path="/alerts/:id" element={
                            <ProtectedRoute>
                                <Layout>
                                    <AlertDetail />
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
                        <Route path="/mcp-management" element={
                            <ProtectedRoute>
                                <Layout>
                                    <MCPManagement />
                                </Layout>
                            </ProtectedRoute>
                        } />
                        <Route path="/mcp-marketplace" element={
                            <ProtectedRoute>
                                <Layout>
                                    <MCPMarketplace />
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

                    {/* 自定义通知系统 */}
                    <NotificationContainer />

                    {/* 开发环境错误处理 */}
                    <DevErrorHandler />

                    {/* 全局搜索 */}
                    <GlobalSearch />

                    {/* Cookie 同意弹窗 */}
                    <CookieConsent />

                    {/* 性能监控 */}
                    <div className="fixed -left-4 top-1/2 transform -translate-y-1/2 z-50">
                        <div className="relative">
                            {/* 小触发按钮 */}
                            <button 
                                className="w-8 h-8 bg-black/80 hover:bg-black text-white rounded-full flex items-center justify-center text-xs font-bold border border-gray-600 hover:scale-110 transition-all duration-200"
                                onClick={() => setShowPerformanceMonitor(!showPerformanceMonitor)}
                            >
                                P
                            </button>
                            
                            {/* 展开的性能监控面板 */}
                            {showPerformanceMonitor && (
                                <div className="absolute left-10 top-0 w-80 h-48 bg-black/95 rounded-lg p-4 text-white text-xs border border-gray-600 shadow-2xl z-50">
                                    <div className="flex justify-between items-center mb-3">
                                        <div className="font-bold text-sm">性能监控</div>
                                        <button 
                                            className="text-gray-400 hover:text-white text-lg"
                                            onClick={() => setShowPerformanceMonitor(false)}
                                        >
                                            ×
                                        </button>
                                    </div>
                                    <div className="space-y-2">
                                        <div className="flex justify-between">
                                            <span>FPS:</span>
                                            <span className="text-green-400">60</span>
                                        </div>
                                        <div className="flex justify-between">
                                            <span>内存使用:</span>
                                            <span className="text-blue-400">45MB</span>
                                        </div>
                                        <div className="flex justify-between">
                                            <span>网络状态:</span>
                                            <span className="text-green-400">正常</span>
                                        </div>
                                        <div className="flex justify-between">
                                            <span>CPU使用率:</span>
                                            <span className="text-yellow-400">15%</span>
                                        </div>
                                    </div>
                                </div>
                            )}
                        </div>
                    </div>
        </div>
    );
}

function App() {
    return (
        <ErrorBoundary>
            <ToastProvider>
                <Router>
                    <AppRoutes />
                </Router>
            </ToastProvider>
        </ErrorBoundary>
    );
}

export default App;