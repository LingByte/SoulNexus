import { BrowserRouter as Router, Route, Routes, Navigate } from 'react-router-dom';
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
import Documentation from "@/pages/Documentation.tsx";
import GlobalSearch from "@/components/UI/GlobalSearch.tsx";
import NotificationContainer from "@/components/UI/NotificationContainer.tsx";
import { ToastProvider } from "@/components/UI/ToastContainer.tsx";
import About from "@/pages/About.tsx";
import NotificationCenter from "@/pages/NotificationCenter.tsx";
import Profile from "@/pages/Profile.tsx";
import AnimationShowcase from "@/pages/AnimationShowcase.tsx";
import Layout from "@/components/Layout/Layout.tsx";
import KnowledgeBase from "@/pages/KnowledgeBase.tsx";
import KnowledgeBaseDetail from "@/pages/KnowledgeBaseDetail.tsx";
import ResetPassword from "@/pages/ResetPassword.tsx";
import CredentialManager from "@/pages/CredentialManager.tsx";
import ProtectedRoute from "@/components/Auth/ProtectedRoute.tsx";
import JSTemplateManager from "@/pages/JSTemplateManager.tsx";
import Assistants from '@/pages/Assistants.tsx';
import AssistantTools from '@/pages/AssistantTools.tsx';
import AssistantGraph from '@/pages/AssistantGraph.tsx';
import Billing from '@/pages/Billing.tsx';
import Groups from '@/pages/Groups.tsx';
import GroupMembers from '@/pages/GroupMembers.tsx';
import GroupSettings from '@/pages/GroupSettings.tsx';
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
import CallCenter from '@/pages/CallCenter.tsx';
import NodePluginMarket from '@/pages/NodePluginMarket.tsx';
import SchemeManager from '@/pages/SchemeManager.tsx';
import PhoneNumberManager from '@/pages/PhoneNumberManager.tsx';
import VoicemailBox from '@/pages/VoicemailBox.tsx';
import VoicemailDetail from '@/pages/VoicemailDetail.tsx';
import VoiceprintManagement from '@/pages/VoiceprintManagement.tsx';
import MCPManagement from '@/pages/MCPManagement.tsx';
import MCPMarketplace from '@/pages/MCPMarketplace.tsx';

function App() {
    const [showPerformanceMonitor, setShowPerformanceMonitor] = useState(false);
    
    return (
        <ErrorBoundary>
            <ToastProvider>
                <Router>
                <div className="min-h-screen bg-gray-50 dark:bg-gray-900 text-gray-900 dark:text-gray-100">
                    <Routes>
                        {/* 首页 - 独立布局，不需要 Layout */}
                        <Route path="/" element={<Home />} />
                        
                        {/* 关于页面 - 不需要登录 */}
                        <Route path="/about" element={<About />} />
                        
                        {/* 文档页面 - 不需要登录 */}
                        <Route path="/docs" element={<Documentation />} />
                        
                        {/* 重置密码页面 - 不需要登录 */}
                        <Route path="/reset-password" element={<ResetPassword />} />
                        
                        {/* 需要登录的页面 */}
                        <Route path="/overview" element={
                            <ProtectedRoute>
                                <Layout>
                                    <Overview />
                                </Layout>
                            </ProtectedRoute>
                        } />
                        <Route path="/knowledge" element={
                            <ProtectedRoute>
                                <Layout>
                                    <KnowledgeBase />
                                </Layout>
                            </ProtectedRoute>
                        } />
                        <Route path="/knowledge/:knowledgeKey" element={
                            <ProtectedRoute>
                                <Layout>
                                    <KnowledgeBaseDetail />
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
                        <Route path="/assistants/:id/tools" element={
                            <ProtectedRoute>
                                <Layout>
                                    <AssistantTools />
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
                        <Route path="/call-center" element={
                            <ProtectedRoute>
                                <Layout>
                                    <CallCenter />
                                </Layout>
                            </ProtectedRoute>
                        } />
                        <Route path="/schemes" element={
                            <ProtectedRoute>
                                <Layout>
                                    <SchemeManager />
                                </Layout>
                            </ProtectedRoute>
                        } />
                        <Route path="/phone-numbers" element={
                            <ProtectedRoute>
                                <Layout>
                                    <PhoneNumberManager />
                                </Layout>
                            </ProtectedRoute>
                        } />
                        <Route path="/voicemail" element={
                            <ProtectedRoute>
                                <Layout>
                                    <VoicemailBox />
                                </Layout>
                            </ProtectedRoute>
                        } />
                        <Route path="/voicemail/:id" element={
                            <ProtectedRoute>
                                <Layout>
                                    <VoicemailDetail />
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

                    {/* PWA 安装提示 */}
                    <PWAInstaller
                        showOnLoad={true}
                        delay={5000}
                        position="bottom-right"
                    />

                    {/* 自定义通知系统 */}
                    <NotificationContainer />

                    {/* 开发环境错误处理 */}
                    <DevErrorHandler />

                    {/* 全局搜索 */}
                    <GlobalSearch />

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
            </Router>
            </ToastProvider>
        </ErrorBoundary>
    );
}

export default App;