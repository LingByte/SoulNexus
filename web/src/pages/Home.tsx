import { motion } from 'framer-motion'
import { useState, useEffect, useRef } from 'react'
import { Link } from 'react-router-dom'
import {
    Zap,
    Settings as SettingsIcon,
    BookOpen as BookOpenIcon,
    Users as UsersIcon,
    MessageCircle as MessageCircleIcon,
    Activity as ActivityIcon,
    Target,
    Eye,
    Award,
    CheckCircle,
    Phone,
    Mic,
    Key,
    Code,
    User as UserIcon,
    LogOut,
    Menu,
    X
} from 'lucide-react'
import {Typewriter} from "@/components/UX/MicroInteractions.tsx";
import Card, { CardContent, CardDescription, CardHeader, CardTitle } from "@/components/UI/Card";
import StaggeredList from "@/components/Animations/StaggeredList";
import Button from "@/components/UI/Button";
import AuthModal from "@/components/Auth/AuthModal";
import { useAuthStore } from "@/stores/authStore";
import EnhancedThemeToggle from "@/components/UI/EnhancedThemeToggle";
import LanguageSelector from "@/components/UI/LanguageSelector";
import { useI18nStore } from "@/stores/i18nStore";
import Footer from "@/components/Layout/Footer.tsx";

const iconMap: Record<string, any> = {
    Zap,
    Settings: SettingsIcon,
    BookOpen: BookOpenIcon,
    Users: UsersIcon,
    MessageCircle: MessageCircleIcon,
    Activity: ActivityIcon,
    Phone,
    Mic,
    Key,
    Code,
}

const Home = () => {
    const [showAuthModal, setShowAuthModal] = useState(false)
    const [showMobileMenu, setShowMobileMenu] = useState(false)
    const [showUserDropdown, setShowUserDropdown] = useState(false)
    const { user, isAuthenticated, logout } = useAuthStore()
    const { t } = useI18nStore()
    const userDropdownRef = useRef<HTMLDivElement>(null)

    // 点击外部关闭下拉菜单
    useEffect(() => {
        const handleClickOutside = (event: MouseEvent) => {
            if (userDropdownRef.current && !userDropdownRef.current.contains(event.target as Node)) {
                setShowUserDropdown(false)
            }
        }

        if (showUserDropdown) {
            document.addEventListener('mousedown', handleClickOutside)
        }

        return () => {
            document.removeEventListener('mousedown', handleClickOutside)
        }
    }, [showUserDropdown])

    // Core features based on actual functionality
    const coreFeatures = [
        {
            title: t('feature.aiVoiceCall'),
            icon: "Zap",
            description: t('feature.aiVoiceCallDesc'),
            features: [t('tag.webrtc'), t('tag.lowLatency'), t('tag.multiAudio'), t('tag.asr')]
        },
        {
            title: t('feature.voiceClone'),
            icon: "Mic",
            description: t('feature.voiceCloneDesc'),
            features: [t('tag.voiceTraining'), t('tag.voiceClone'), t('tag.personalVoice'), t('tag.multiVoice')]
        },
        {
            title: t('feature.appIntegration'),
            icon: "Settings",
            description: t('feature.appIntegrationDesc'),
            features: [t('tag.jsInjection'), t('tag.painlessIntegration'), t('tag.quickDeploy'), t('tag.standardApi')]
        },
        {
            title: t('feature.knowledgeBase'),
            icon: "BookOpen",
            description: t('feature.knowledgeBaseDesc'),
            features: [t('tag.docManagement'), t('tag.smartSearch'), t('tag.aiAnalysis'), t('tag.versionControl')]
        },
        {
            title: t('feature.workflow'),
            icon: "Activity",
            description: t('feature.workflowDesc'),
            features: [t('tag.visualDesign'), t('tag.processAutomation'), t('tag.conditionalBranch'), t('tag.realTimeMonitor')]
        },
        {
            title: t('feature.credential'),
            icon: "Key",
            description: t('feature.credentialDesc'),
            features: [t('tag.credentialManagement'), t('tag.apiDoc'), t('tag.devTools'), t('tag.securityAuth')]
        }
    ]

    const techStack = [
        {
            name: t('tech.frontend'),
            technologies: [
                { name: t('tech.react'), version: "18.0", description: t('tech.reactDesc') },
                { name: t('tech.typescript'), version: "5.0", description: t('tech.typescriptDesc') },
                { name: t('tech.tailwind'), version: "3.0", description: t('tech.tailwindDesc') },
                { name: t('tech.webrtc'), version: t('tech.latest'), description: t('tech.webrtcDesc') }
            ]
        },
        {
            name: t('tech.backend'),
            technologies: [
                { name: t('tech.go'), version: "1.21", description: t('tech.goDesc') },
                { name: t('tech.gin'), version: "1.9", description: t('tech.ginDesc') },
                { name: t('tech.websocket'), version: t('tech.latest'), description: t('tech.websocketDesc') },
            ]
        },
        {
            name: t('tech.aiml'),
            technologies: [
                { name: t('tech.asr'), version: "ASR", description: t('tech.asrDesc') },
                { name: t('tech.tts'), version: "TTS", description: t('tech.ttsDesc') },
                { name: t('tech.voiceClone'), version: "Voice Clone", description: t('tech.voiceCloneDesc') },
                { name: t('tech.llm'), version: "LLM", description: t('tech.llmDesc') }
            ]
        }
    ]

    // About data
    const aboutValues = [
        { icon: <Target className="w-8 h-8 text-indigo-300" />, title: t('story.developer'), description: t('story.developerDesc') },
        { icon: <Eye className="w-8 h-8 text-purple-300" />, title: t('story.animeUser'), description: t('story.animeUserDesc') },
        { icon: <UsersIcon className="w-8 h-8 text-fuchsia-300" />, title: t('story.education'), description: t('story.educationDesc') },
        { icon: <Award className="w-8 h-8 text-blue-300" />, title: t('story.creator'), description: t('story.creatorDesc') },
    ]


    const aboutTeam = [
        { name: 'chenting', role: t('team.fullStack'), avatar: 'C', description: '' },
        { name: 'wangyueran', role: t('team.fullStack'), avatar: 'W', description: '' },
        { name: 'quenanlin', role: t('team.fullStack'), avatar: 'Q', description: '' },
        { name: 'jianghaotian', role: t('team.fullStack'), avatar: 'J', description: '' },
    ]

    return (
        <div className="min-h-screen relative">
            {/* 顶部导航栏 */}
            <nav className="fixed top-0 left-0 right-0 z-50 bg-background/80 backdrop-blur-md border-b border-border">
                <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
                    <div className="flex items-center justify-between h-16">
                        {/* Logo */}
                        <Link to="/" className="flex items-center gap-2">
                            <img
                                src="https://cetide-1325039295.cos.ap-chengdu.myqcloud.com/folder/icon-192x192.ico"
                                alt="SoulNexus Logo"
                                className="w-8 h-8 rounded"
                            />
                            <span className="text-xl font-extrabold tracking-wider">
                                <span className="text-purple-600">{t('brand.name')}</span>
                            </span>
                        </Link>

                        {/* 桌面端导航 */}
                        <div className="hidden md:flex items-center gap-4">
                            <Link to="/docs" className="text-muted-foreground hover:text-foreground transition-colors">
                                {t('nav.docs')}
                            </Link>
                            <Link to="/about" className="text-muted-foreground hover:text-foreground transition-colors">
                                {t('nav.about')}
                            </Link>
                            
                            {/* 主题切换和语言选择器 */}
                            <div className="flex items-center gap-2">
                                <LanguageSelector size="sm" />
                                <EnhancedThemeToggle size="sm" />
                            </div>
                            
                            {/* 登录按钮或用户信息 */}
                            {isAuthenticated && user ? (
                                <div className="relative" ref={userDropdownRef}>
                                    <button
                                        className="flex items-center gap-2 p-1 rounded-full hover:bg-accent transition-colors"
                                        onClick={() => setShowUserDropdown(!showUserDropdown)}
                                    >
                                        <img
                                            src={user.avatar || `https://ui-avatars.com/api/?name=${user.displayName || 'U'}&background=0ea5e9&color=fff`}
                                            alt={user.displayName}
                                            className="w-8 h-8 rounded-full"
                                        />
                                        <span className="text-sm font-medium">{user.displayName}</span>
                                    </button>
                                    
                                    {/* 用户下拉菜单 */}
                                    {showUserDropdown && (
                                        <div className="absolute right-0 top-full mt-2 w-48 bg-popover rounded-md shadow-lg border z-50">
                                            <div className="flex flex-col p-2">
                                                <div className="px-3 py-2 border-b border-border">
                                                    <p className="text-sm font-medium">{user.displayName}</p>
                                                    <p className="text-xs text-muted-foreground">{user.email}</p>
                                                </div>
                                                <Button
                                                    variant="ghost"
                                                    size="sm"
                                                    className="flex items-center gap-2 w-full justify-start text-sm px-3 py-2 mt-2"
                                                    onClick={() => { 
                                                        setShowUserDropdown(false)
                                                        window.location.href = '/assistants'
                                                    }}
                                                    leftIcon={<UserIcon className="w-4 h-4" />}
                                                >
                                                    {t('nav.enterSystem')}
                                                </Button>
                                                <Button
                                                    variant="ghost"
                                                    size="sm"
                                                    className="flex items-center gap-2 w-full justify-start text-sm px-3 py-2"
                                                    onClick={async () => { 
                                                        setShowUserDropdown(false)
                                                        await logout()
                                                    }}
                                                    leftIcon={<LogOut className="w-4 h-4" />}
                                                >
                                                    {t('nav.logout')}
                                                </Button>
                                            </div>
                                        </div>
                                    )}
                                </div>
                            ) : (
                                <Button
                                    variant="primary"
                                    onClick={() => setShowAuthModal(true)}
                                    leftIcon={<UserIcon className="w-4 h-4" />}
                                >
                                    {t('nav.login')}
                                </Button>
                            )}
                        </div>

                        {/* 移动端菜单按钮 */}
                        <button
                            className="md:hidden p-2 rounded-md text-muted-foreground hover:text-foreground hover:bg-accent"
                            onClick={() => setShowMobileMenu(!showMobileMenu)}
                        >
                            {showMobileMenu ? <X className="w-6 h-6" /> : <Menu className="w-6 h-6" />}
                        </button>
                    </div>

                    {/* 移动端菜单 */}
                    {showMobileMenu && (
                        <div className="md:hidden py-4 border-t border-border">
                            <div className="flex flex-col gap-4">
                                <Link 
                                    to="/docs" 
                                    className="text-muted-foreground hover:text-foreground transition-colors"
                                    onClick={() => setShowMobileMenu(false)}
                                >
                                    {t('nav.docs')}
                                </Link>
                                <Link 
                                    to="/about" 
                                    className="text-muted-foreground hover:text-foreground transition-colors"
                                    onClick={() => setShowMobileMenu(false)}
                                >
                                    {t('nav.about')}
                                </Link>
                                
                                {/* 移动端主题切换和语言选择器 */}
                                <div className="flex items-center gap-3 pt-2 border-t border-border">
                                    <div className="flex items-center gap-2">
                                        <span className="text-sm text-muted-foreground">{t('lang.select')}:</span>
                                        <LanguageSelector size="sm" />
                                    </div>
                                    <div className="flex items-center gap-2">
                                        <span className="text-sm text-muted-foreground">{t('theme.toggle')}:</span>
                                        <EnhancedThemeToggle size="sm" />
                                    </div>
                                </div>
                                {isAuthenticated && user ? (
                                    <>
                                        <div className="flex items-center gap-2 pt-2 border-t border-border pb-2">
                                            <img
                                                src={user.avatar || `https://ui-avatars.com/api/?name=${user.displayName || 'U'}&background=0ea5e9&color=fff`}
                                                alt={user.displayName}
                                                className="w-8 h-8 rounded-full"
                                            />
                                            <div className="flex-1">
                                                <p className="text-sm font-medium">{user.displayName}</p>
                                                <p className="text-xs text-muted-foreground">{user.email}</p>
                                            </div>
                                        </div>
                                        <Button
                                            variant="ghost"
                                            size="sm"
                                            className="flex items-center gap-2 w-full justify-start text-sm px-3 py-2"
                                            onClick={() => { 
                                                setShowMobileMenu(false)
                                                window.location.href = '/assistants'
                                            }}
                                            leftIcon={<UserIcon className="w-4 h-4" />}
                                        >
                                            {t('nav.enterSystem')}
                                        </Button>
                                        <Button
                                            variant="ghost"
                                            size="sm"
                                            className="flex items-center gap-2 w-full justify-start text-sm px-3 py-2"
                                            onClick={async () => { 
                                                await logout()
                                                setShowMobileMenu(false)
                                            }}
                                            leftIcon={<LogOut className="w-4 h-4" />}
                                        >
                                            {t('nav.logout')}
                                        </Button>
                                    </>
                                ) : (
                                    <Button
                                        variant="primary"
                                        className="w-full"
                                        onClick={() => { setShowAuthModal(true); setShowMobileMenu(false); }}
                                        leftIcon={<UserIcon className="w-4 h-4" />}
                                    >
                                        {t('nav.login')}
                                    </Button>
                                )}
                            </div>
                        </div>
                    )}
                </div>
            </nav>

            {/* 登录弹窗 */}
            <AuthModal isOpen={showAuthModal} onClose={() => setShowAuthModal(false)} />

            {/* 主要内容区域 */}
            <div className="relative space-y-20 overflow-hidden pt-16">
            {/* Full-page tech gradient background to override app gray bg */}
            <div className="pointer-events-none absolute inset-0 -z-20">
                {/* 主要渐变背景 */}
                <div className="absolute inset-0 bg-[radial-gradient(1200px_600px_at_50%_-10%,rgba(59,130,246,0.25),transparent),radial-gradient(1000px_500px_at_100%_20%,rgba(147,51,234,0.22),transparent),linear-gradient(180deg,#0B1020, #0E1224_40%, #0B1020)] dark:bg-[radial-gradient(1200px_600px_at_50%_-10%,rgba(59,130,246,0.15),transparent),radial-gradient(1000px_500px_at_100%_20%,rgba(147,51,234,0.12),transparent),linear-gradient(180deg,#1a1a2e, #2d2d44_40%, #1a1a2e)]" />
                
                {/* 保留原有网格 */}
                <div className="absolute inset-0 opacity-30 [background-image:linear-gradient(to_right,rgba(255,255,255,0.06)_1px,transparent_1px),linear-gradient(to_bottom,rgba(255,255,255,0.06)_1px,transparent_1px)] [background-size:26px_26px]" />
                
                {/* 新增科技感特效层 */}
                {/* 动态扫描线 */}
                <div className="absolute inset-0 opacity-20">
                    <div className="absolute top-0 left-0 w-full h-px bg-gradient-to-r from-transparent via-blue-400 to-transparent animate-pulse"></div>
                    <div className="absolute bottom-0 left-0 w-full h-px bg-gradient-to-r from-transparent via-purple-400 to-transparent animate-pulse" style={{ animationDelay: '1s' }}></div>
                    <div className="absolute left-0 top-0 w-px h-full bg-gradient-to-b from-transparent via-indigo-400 to-transparent animate-pulse" style={{ animationDelay: '0.5s' }}></div>
                    <div className="absolute right-0 top-0 w-px h-full bg-gradient-to-b from-transparent via-pink-400 to-transparent animate-pulse" style={{ animationDelay: '1.5s' }}></div>
                </div>
                
                {/* 数据流效果 */}
                <div className="absolute inset-0 opacity-10">
                    <div className="absolute top-1/4 left-0 w-full h-0.5 bg-gradient-to-r from-transparent via-cyan-400 to-transparent animate-pulse" style={{ animationDelay: '2s' }}></div>
                    <div className="absolute top-3/4 left-0 w-full h-0.5 bg-gradient-to-r from-transparent via-emerald-400 to-transparent animate-pulse" style={{ animationDelay: '2.5s' }}></div>
                    <div className="absolute top-1/2 left-0 w-full h-0.5 bg-gradient-to-r from-transparent via-yellow-400 to-transparent animate-pulse" style={{ animationDelay: '3s' }}></div>
                </div>
                
                {/* 科技感光点 */}
                <div className="absolute inset-0 opacity-30">
                    <div className="absolute top-20 left-20 w-2 h-2 bg-blue-400 rounded-full animate-ping"></div>
                    <div className="absolute top-40 right-32 w-1.5 h-1.5 bg-purple-400 rounded-full animate-ping" style={{ animationDelay: '0.8s' }}></div>
                    <div className="absolute bottom-32 left-40 w-1 h-1 bg-indigo-400 rounded-full animate-ping" style={{ animationDelay: '1.6s' }}></div>
                    <div className="absolute bottom-20 right-20 w-2.5 h-2.5 bg-pink-400 rounded-full animate-ping" style={{ animationDelay: '2.4s' }}></div>
                    <div className="absolute top-60 left-1/2 w-1 h-1 bg-cyan-400 rounded-full animate-ping" style={{ animationDelay: '3.2s' }}></div>
                </div>
                
                {/* 电路板纹理 */}
                <div className="absolute inset-0 opacity-5">
                    <div className="absolute top-10 left-10 w-8 h-8 border border-blue-400 rounded-sm rotate-45 animate-pulse"></div>
                    <div className="absolute top-20 right-20 w-6 h-6 border border-purple-400 rounded-sm rotate-12 animate-pulse" style={{ animationDelay: '1s' }}></div>
                    <div className="absolute bottom-20 left-20 w-4 h-4 border border-indigo-400 rounded-sm rotate-45 animate-pulse" style={{ animationDelay: '2s' }}></div>
                    <div className="absolute bottom-10 right-10 w-10 h-10 border border-pink-400 rounded-sm rotate-12 animate-pulse" style={{ animationDelay: '3s' }}></div>
                </div>
            </div>
            {/* Hero Section */}
            <section className="relative py-15 text-center overflow-hidden">
                {/* 主要渐变背景 - 浅蓝到浅紫 */}
                <div className="absolute inset-0 bg-gradient-to-br from-blue-100 via-indigo-100 to-purple-100 dark:from-gray-800/50 dark:via-blue-800/20 dark:to-purple-800/20"></div>
                
                {/* 动态光晕效果 */}
                <div className="absolute inset-0 bg-gradient-to-r from-blue-400/30 via-purple-400/30 to-pink-400/30 animate-pulse"></div>
                
                {/* 若隐若现的网格背景 */}
                <div className="absolute inset-0 z-0 opacity-40 [background-image:linear-gradient(to_right,rgba(99,102,241,0.15)_1px,transparent_1px),linear-gradient(to_bottom,rgba(99,102,241,0.15)_1px,transparent_1px)] [background-size:40px_40px] pointer-events-none"></div>
                
                {/* 网格阴影效果 */}
                <div className="absolute inset-0 z-0 opacity-20 [background-image:linear-gradient(to_right,rgba(99,102,241,0.08)_1px,transparent_1px),linear-gradient(to_bottom,rgba(99,102,241,0.08)_1px,transparent_1px)] [background-size:40px_40px] [background-position:1px_1px] pointer-events-none"></div>
                
                {/* 边缘模糊遮罩 - 增强上下边缘 */}
                <div className="absolute inset-0 z-0 bg-gradient-to-t from-blue-100/80 via-blue-100/20 to-transparent dark:from-blue-800/40 dark:via-blue-800/10 dark:to-transparent pointer-events-none"></div>
                <div className="absolute inset-0 z-0 bg-gradient-to-b from-blue-100/80 via-blue-100/20 to-transparent dark:from-blue-800/40 dark:via-blue-800/10 dark:to-transparent pointer-events-none"></div>
                <div className="absolute inset-0 z-0 bg-gradient-to-l from-transparent via-transparent to-blue-100/50 dark:to-blue-800/20 pointer-events-none"></div>
                <div className="absolute inset-0 z-0 bg-gradient-to-r from-transparent via-transparent to-blue-100/50 dark:to-blue-800/20 pointer-events-none"></div>
                
                {/* 浮动光球 */}
                <div className="absolute -top-24 left-1/2 h-96 w-96 -translate-x-1/2 rounded-full blur-3xl bg-gradient-to-r from-blue-400/30 via-purple-400/30 to-transparent animate-pulse"></div>
                <div className="absolute -bottom-24 right-10 h-80 w-80 rounded-full blur-3xl bg-gradient-to-r from-pink-400/30 via-purple-400/30 to-transparent animate-pulse"></div>
                <div className="absolute top-1/2 left-10 h-64 w-64 rounded-full blur-3xl bg-gradient-to-r from-indigo-400/20 via-blue-400/20 to-transparent animate-pulse"></div>
                
                <div className="absolute inset-0 -z-10" />

                <motion.div
                    initial={{ opacity: 0, y: 30 }}
                    animate={{ opacity: 1, y: 0 }}
                    transition={{ duration: 0.8 }}
                    className="max-w-5xl mx-auto px-4 z-10"
                >

                    <div className="relative">
                        {/* 标题背景光晕 */}
                        <div className="absolute -inset-4 bg-gradient-to-r from-indigo-500/20 via-purple-500/20 to-blue-500/20 rounded-3xl blur-xl animate-pulse"></div>
                        
                        {/* 标题文字容器 */}
                        <div className="relative z-10">
                            <motion.h1 
                                initial={{ opacity: 0, scale: 0.8 }}
                                animate={{ opacity: 1, scale: 1 }}
                                transition={{ duration: 1, ease: "easeOut" }}
                                className="text-6xl md:text-8xl font-display font-bold tracking-tight relative z-20"
                                style={{ lineHeight: 1.2 }}
                            >
                                <span className="text-5xl md:text-7xl font-display font-bold mb-6 tracking-tight relative z-20 text-indigo-600 dark:text-indigo-300" style={{ lineHeight: 2 }}>
                                    {t('home.title')}
                                </span>
                                {/* 文字发光效果 */}
                                <div className="absolute inset-0 bg-gradient-to-r from-indigo-400 via-purple-400 to-blue-400 bg-clip-text text-transparent blur-sm opacity-50 animate-pulse"></div>
                                {/* 动态粒子效果 */}
                                <div className="absolute -top-4 -left-4 w-3 h-3 bg-indigo-400 rounded-full animate-ping opacity-75"></div>
                                <div className="absolute -bottom-2 -right-2 w-2 h-2 bg-purple-400 rounded-full animate-ping opacity-75" style={{ animationDelay: '0.5s' }}></div>
                                <div className="absolute top-1/2 -right-6 w-1.5 h-1.5 bg-blue-400 rounded-full animate-ping opacity-75" style={{ animationDelay: '1s' }}></div>
                            </motion.h1>
                        </div>
                        
                        {/* 装饰性元素 */}
                        <div className="absolute -top-8 left-1/2 transform -translate-x-1/2 w-32 h-1 bg-gradient-to-r from-transparent via-indigo-400 to-transparent rounded-full opacity-60"></div>
                        <div className="absolute -bottom-4 left-1/2 transform -translate-x-1/2 w-24 h-0.5 bg-gradient-to-r from-transparent via-purple-400 to-transparent rounded-full opacity-40"></div>
                    </div>

                    <motion.p
                        initial={{ opacity: 0, y: 20 }}
                        animate={{ opacity: 1, y: 0 }}
                        transition={{ delay: 0.4, duration: 0.6 }}
                        className="text-gray-700 dark:text-gray-200 mb-8 max-w-2xl mx-auto leading-relaxed relative z-20"
                    >
                        <Typewriter
                            text={t('home.subtitle')}
                            speed={30}
                            className="block"
                        />
                    </motion.p>

                    <motion.div
                        initial={{ opacity: 0, y: 20 }}
                        animate={{ opacity: 1, y: 0 }}
                        transition={{ delay: 0.6, duration: 0.6 }}
                        className="flex flex-col sm:flex-row gap-4 justify-center items-center"
                    >
                        <a
                            href="/voice-assistant"
                            className="w-full sm:w-auto inline-flex items-center justify-center px-6 py-3 rounded-xl font-semibold shadow-lg shadow-indigo-500/20 bg-gradient-to-r from-indigo-500 via-purple-500 to-blue-500 hover:from-indigo-600 hover:via-purple-600 hover:to-blue-600 transition-all duration-300 hover:scale-105 hover:shadow-2xl hover:shadow-indigo-500/40 active:scale-95 active:shadow-lg focus:outline-none focus:ring-4 focus:ring-indigo-500/50 focus:ring-offset-2 focus:ring-offset-transparent relative overflow-hidden group"
                        >
                            {/* 动态光效背景 */}
                            <div className="absolute inset-0 bg-gradient-to-r from-transparent via-white/20 to-transparent -translate-x-full group-hover:translate-x-full transition-transform duration-700 ease-out"></div>
                            
                            {/* 按钮文字 */}
                            <span className="relative">{t('home.startNow')}</span>
                            
                            {/* 动态边框效果 */}
                            <div className="absolute inset-0 rounded-xl border-2 border-transparent bg-gradient-to-r from-indigo-500 via-purple-500 to-blue-500 bg-clip-border opacity-0 group-hover:opacity-100 transition-opacity duration-300"></div>
                        </a>
                    </motion.div>

                    <div className="mt-2 max-w-6xl mx-auto text-left relative z-30" id="solutions">
                        {/* 核心价值总结 */}
                        <motion.div
                            initial={{ opacity: 0, y: 30 }}
                            animate={{ opacity: 1, y: 0 }}
                            transition={{ delay: 1.4, duration: 0.8 }}
                            className="mt-12 text-center relative z-30"
                            style={{ opacity: 1 }}
                        >
                            <div className="bg-gradient-to-r from-indigo-50 to-purple-50 dark:from-indigo-900/30 dark:to-purple-900/30 rounded-2xl p-8 border border-indigo-200/50 dark:border-indigo-800/50 relative z-30 backdrop-blur-sm">
                                <p className="text-lg leading-relaxed text-gray-800 dark:text-gray-100 relative z-30 font-medium">
                                    {t('home.mission')}
                                </p>
                            </div>
                        </motion.div>
                    </div>
                </motion.div>
            </section>

            {/* Learn More / Features Section */}
            <section id="more" className="relative py-24 overflow-hidden">
                {/* 渐变背景 - 浅蓝到浅紫 */}
                <div className="absolute inset-0 bg-gradient-to-br from-blue-100 via-indigo-100 to-purple-100 dark:from-gray-800 dark:via-blue-900/20 dark:to-purple-900/20"></div>
                
                {/* 动态光效 */}
                <div className="absolute inset-0 bg-gradient-to-r from-transparent via-blue-400/20 to-transparent animate-pulse"></div>
                
                {/* 若隐若现的网格背景 */}
                <div className="absolute inset-0 z-0 opacity-35 [background-image:linear-gradient(to_right,rgba(99,102,241,0.12)_1px,transparent_1px),linear-gradient(to_bottom,rgba(99,102,241,0.12)_1px,transparent_1px)] [background-size:32px_32px] pointer-events-none"></div>
                
                {/* 网格阴影效果 */}
                <div className="absolute inset-0 z-0 opacity-15 [background-image:linear-gradient(to_right,rgba(99,102,241,0.06)_1px,transparent_1px),linear-gradient(to_bottom,rgba(99,102,241,0.06)_1px,transparent_1px)] [background-size:32px_32px] [background-position:1px_1px] pointer-events-none"></div>
                
                {/* 边缘模糊遮罩 - 增强上下边缘 */}
                <div className="absolute inset-0 z-0 bg-gradient-to-t from-blue-100/80 via-blue-100/20 to-transparent dark:from-blue-900/60 dark:via-blue-900/10 dark:to-transparent pointer-events-none"></div>
                <div className="absolute inset-0 z-0 bg-gradient-to-b from-blue-100/80 via-blue-100/20 to-transparent dark:from-blue-900/60 dark:via-blue-900/10 dark:to-transparent pointer-events-none"></div>
                <div className="absolute inset-0 z-0 bg-gradient-to-l from-transparent via-transparent to-blue-100/40 dark:to-blue-900/20 pointer-events-none"></div>
                <div className="absolute inset-0 z-0 bg-gradient-to-r from-transparent via-transparent to-blue-100/40 dark:to-blue-900/20 pointer-events-none"></div>
                
                {/* 浮动光球 */}
                <div className="absolute top-20 left-10 h-80 w-80 rounded-full blur-3xl bg-gradient-to-r from-blue-400/20 via-indigo-400/20 to-transparent animate-pulse"></div>
                <div className="absolute bottom-10 right-10 h-72 w-72 rounded-full blur-3xl bg-gradient-to-r from-purple-400/20 via-pink-400/20 to-transparent animate-pulse"></div>
                <div className="absolute top-1/2 right-1/4 h-64 w-64 rounded-full blur-3xl bg-gradient-to-r from-indigo-400/15 via-blue-400/15 to-transparent animate-pulse"></div>
                <div className="max-w-6xl mx-auto px-4">
                    {/* Core Features */}
                    <div className="mb-2">
                        <div className="flex items-center gap-3">
                            <div className="w-10 h-10 rounded-lg bg-indigo-500/20 border border-white/10 flex items-center justify-center shadow-sm">
                                <Zap className="w-5 h-5 text-indigo-300" />
                            </div>
                            <div>
                                <h3 className="text-2xl font-bold tracking-tight text-gray-900 dark:text-white">{t('home.coreFeatures')}</h3>
                                <p className="text-gray-600 dark:text-neutral-400">{t('home.coreFeaturesDesc')}</p>
                            </div>
                        </div>
                        <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-7 items-stretch">
                            {coreFeatures.map((f: any, idx: number) => {
                                const Icon = iconMap[f.icon] || Zap
                                return (
                                    <div key={idx} className="relative group rounded-2xl p-[1px] bg-gradient-to-br from-indigo-500/30 via-purple-500/20 to-transparent h-full z-10">
                                        <div className="relative overflow-hidden rounded-2xl border border-white/10 bg-white/5 backdrop-blur-md p-6 shadow-[0_1px_0_rgba(255,255,255,0.25)_inset,0_10px_30px_-10px_rgba(79,70,229,0.25)] h-[300px] flex flex-col z-10">
                                            <div className="absolute -inset-24 opacity-70 bg-[radial-gradient(circle_at_20%_20%,rgba(99,102,241,0.12),transparent_40%),radial-gradient(circle_at_80%_120%,rgba(147,51,234,0.12),transparent_40%)]" />
                                            <div className="relative flex-1">
                                                <div className="inline-flex items-center justify-center w-11 h-11 rounded-xl bg-gradient-to-br from-indigo-500/30 to-purple-500/30 mb-4 border border-white/20">
                                                    <Icon className="w-5 h-5 text-indigo-200" />
                                                </div>
                                                <h4 className="text-lg font-semibold mb-2 tracking-tight text-gray-900 dark:text-white">{f.title}</h4>
                                                <p className="text-sm text-gray-600 dark:text-neutral-300 leading-relaxed">{f.description}</p>
                                            </div>
                                            <div className="relative mt-4 flex flex-wrap gap-2">
                                                {f.features?.map((tag: string, i: number) => (
                                                    <span
                                                        key={i}
                                                        className={`px-2.5 py-1 rounded-full text-xs border bg-gradient-to-r ${[
                                                            'from-indigo-500/20 via-indigo-500/10 to-transparent text-indigo-800 dark:text-indigo-100 border-indigo-400/50 dark:border-indigo-400/30',
                                                            'from-purple-500/20 via-purple-500/10 to-transparent text-purple-800 dark:text-purple-100 border-purple-400/50 dark:border-purple-400/30',
                                                            'from-fuchsia-500/20 via-fuchsia-500/10 to-transparent text-fuchsia-800 dark:text-fuchsia-100 border-fuchsia-400/50 dark:border-fuchsia-400/30',
                                                            'from-cyan-500/20 via-cyan-500/10 to-transparent text-cyan-800 dark:text-cyan-100 border-cyan-400/50 dark:border-cyan-400/30',
                                                            'from-emerald-500/20 via-emerald-500/10 to-transparent text-emerald-800 dark:text-emerald-100 border-emerald-400/50 dark:border-emerald-400/30'
                                                        ][i % 5]}`}
                                                    >
                                                        {tag}
                                                    </span>
                                                ))}
                                            </div>
                                            <div className="pointer-events-none absolute inset-0 opacity-0 group-hover:opacity-100 transition-opacity duration-300 bg-gradient-to-t from-indigo-500/0 via-purple-500/0 to-purple-500/15" />
                                        </div>
                                    </div>
                                )
                            })}
                        </div>
                    </div>

                    {/* Tech Stack */}
                    <div>
                        <div className="flex items-center gap-3">
                            <div className="w-10 h-10 rounded-lg bg-purple-500/20 border border-white/10 flex items-center justify-center shadow-sm">
                                <BookOpenIcon className="w-5 h-5 text-purple-300" />
                            </div>
                            <div>
                                <h3 className="text-2xl font-bold tracking-tight text-gray-900 dark:text-white">{t('home.techStack')}</h3>
                                <p className="text-gray-600 dark:text-neutral-400">{t('home.techStackDesc')}</p>
                            </div>
                        </div>
                        <div className="grid grid-cols-1 lg:grid-cols-3 gap-7 items-stretch">
                            {techStack.map((cat: any, idx: number) => (
                                <div key={idx} className="relative rounded-2xl p-[1px] bg-gradient-to-br from-purple-500/30 via-indigo-500/20 to-transparent h-full z-10">
                                    <div className="relative overflow-hidden rounded-2xl border border-white/10 bg-gradient-to-br from-purple-500/20 via-fuchsia-500/15 to-indigo-500/15 backdrop-blur-md p-6 shadow-[0_1px_0_rgba(255,255,255,0.25)_inset,0_10px_30px_-10px_rgba(147,51,234,0.25)] h-[360px] flex flex-col">
                                        <div className="absolute -inset-24 opacity-70 bg-[radial-gradient(circle_at_30%_0%,rgba(168,85,247,0.12),transparent_40%),radial-gradient(circle_at_80%_120%,rgba(59,130,246,0.12),transparent_40%)]" />
                                        <div className="relative">
                                            <h4 className="text-lg font-semibold mb-4 tracking-tight text-gray-900 dark:text-white">{cat.name}</h4>
                                            <div className="space-y-3">
                                                {cat.technologies?.map((t: any, i: number) => (
                                                    <div key={i} className="flex items-start gap-3">
                                                        <div className="mt-1 w-2.5 h-2.5 rounded-full bg-gradient-to-r from-indigo-400 to-purple-400 ring-2 ring-white/20" />
                                                        <div className="flex-1">
                                                            <div className="flex items-center gap-2">
                                                                <span className="font-medium tracking-tight text-gray-800 dark:text-neutral-100">{t.name}</span>
                                                                {t.version && (
                                                                    <span className="text-xs px-2 py-0.5 rounded-full bg-gray-800 dark:bg-neutral-900/80 text-white border border-gray-300 dark:border-white/10">{t.version}</span>
                                                                )}
                                                            </div>
                                                            <p className="text-sm text-gray-600 dark:text-neutral-300">{t.description}</p>
                                                        </div>
                                                    </div>
                                                ))}
                                            </div>
                                        </div>
                                    </div>
                                </div>
                            ))}
                        </div>
                    </div>
                </div>
            </section>

            {/* About (full) */}
            <section id="about" className="relative py-24 overflow-hidden">
                {/* 渐变背景 - 浅蓝到浅紫 */}
                <div className="absolute inset-0 bg-gradient-to-br from-blue-100 via-indigo-100 to-purple-100 dark:from-gray-900 dark:via-purple-900/30 dark:to-blue-900/30"></div>
                
                {/* 动态光效 */}
                <div className="absolute inset-0 bg-gradient-to-l from-transparent via-purple-400/20 to-transparent animate-pulse"></div>
                
                {/* 若隐若现的网格背景 */}
                <div className="absolute inset-0 z-0 opacity-30 [background-image:linear-gradient(to_right,rgba(168,85,247,0.1)_1px,transparent_1px),linear-gradient(to_bottom,rgba(168,85,247,0.1)_1px,transparent_1px)] [background-size:36px_36px] pointer-events-none"></div>
                
                {/* 网格阴影效果 */}
                <div className="absolute inset-0 z-0 opacity-12 [background-image:linear-gradient(to_right,rgba(168,85,247,0.05)_1px,transparent_1px),linear-gradient(to_bottom,rgba(168,85,247,0.05)_1px,transparent_1px)] [background-size:36px_36px] [background-position:1px_1px] pointer-events-none"></div>
                
                {/* 边缘模糊遮罩 - 增强上下边缘 */}
                <div className="absolute inset-0 z-0 bg-gradient-to-t from-purple-100/80 via-purple-100/20 to-transparent dark:from-purple-900/60 dark:via-purple-900/10 dark:to-transparent pointer-events-none"></div>
                <div className="absolute inset-0 z-0 bg-gradient-to-b from-purple-100/80 via-purple-100/20 to-transparent dark:from-purple-900/60 dark:via-purple-900/10 dark:to-transparent pointer-events-none"></div>
                <div className="absolute inset-0 z-0 bg-gradient-to-l from-transparent via-transparent to-purple-100/40 dark:to-purple-900/20 pointer-events-none"></div>
                <div className="absolute inset-0 z-0 bg-gradient-to-r from-transparent via-transparent to-purple-100/40 dark:to-purple-900/20 pointer-events-none"></div>
                
                {/* 浮动光球 */}
                <div className="absolute top-10 left-10 h-96 w-96 rounded-full blur-3xl bg-gradient-to-r from-purple-400/25 via-pink-400/25 to-transparent animate-pulse"></div>
                <div className="absolute bottom-10 right-10 h-80 w-80 rounded-full blur-3xl bg-gradient-to-r from-blue-400/20 via-indigo-400/20 to-transparent animate-pulse"></div>
                <div className="absolute top-1/3 right-1/3 h-72 w-72 rounded-full blur-3xl bg-gradient-to-r from-pink-400/15 via-purple-400/15 to-transparent animate-pulse"></div>
                <div className="max-w-6xl mx-auto px-2 space-y-20">
                    {/* Mission */}
                    <div className="grid grid-cols-1 lg:grid-cols-2 gap-12 items-start">
                        <div className="relative overflow-hidden rounded-2xl border border-white/10 bg-white/5 backdrop-blur p-8 z-10">
                            <div className="flex items-center gap-3 mb-4">
                                <Target className="w-6 h-6 text-indigo-300" />
                                <h3 className="text-2xl font-semibold tracking-tight text-gray-900 dark:text-white">{t('home.userStories')}</h3>
                            </div>
                            <p className="text-gray-600 dark:text-neutral-300 mb-6 leading-relaxed">
                                {t('story.detail')}
                            </p>
                            <div className="space-y-4">
                                <div className="flex items-start gap-3">
                                    <CheckCircle className="w-5 h-5 text-indigo-300 mt-0.5" />
                                    <p className="text-gray-600 dark:text-neutral-300">{t('story.developerPoint')}</p>
                                </div>
                                <div className="flex items-start gap-3">
                                    <CheckCircle className="w-5 h-5 text-indigo-300 mt-0.5" />
                                    <p className="text-gray-600 dark:text-neutral-300">{t('story.animeUserPoint')}</p>
                                </div>
                                <div className="flex items-start gap-3">
                                    <CheckCircle className="w-5 h-5 text-indigo-300 mt-0.5" />
                                    <p className="text-gray-600 dark:text-neutral-300">{t('story.creatorPoint')}</p>
                                </div>
                            </div>
                        </div>
                        <div className="grid grid-cols-1 sm:grid-cols-3 gap-4">
                            {[
                                { icon: Eye, title: t('values.userCentric') },
                                { icon: UsersIcon, title: t('values.communityDriven') },
                                { icon: Award, title: t('values.excellence') },
                            ].map((item, i) => {
                                const Icon = item.icon as any
                                return (
                                    <div key={i} className="relative rounded-2xl p-[1px] bg-gradient-to-br from-indigo-500/30 via-purple-500/20 to-transparent z-10">
                                        <div className="relative overflow-hidden rounded-2xl border border-gray-200 dark:border-white/10 bg-white/80 dark:bg-white/5 backdrop-blur p-6 h-full shadow-lg z-10">
                                            <div className="absolute -inset-24 opacity-60 bg-[radial-gradient(circle_at_20%_0%,rgba(99,102,241,0.12),transparent_40%),radial-gradient(circle_at_80%_120%,rgba(147,51,234,0.12),transparent_40%)]" />
                                            <div className="relative">
                                                <Icon className="w-6 h-6 text-indigo-600 dark:text-indigo-200 mb-3" />
                                                <div className="text-gray-800 dark:text-neutral-100 font-medium tracking-tight">{item.title}</div>
                                            </div>
                                        </div>
                                    </div>
                                )
                            })}
                        </div>
                    </div>

                    {/* Values */}
                    <div>
                        <div className="text-center mb-6">
                            <h3 className="text-2xl font-bold tracking-tight text-gray-900 dark:text-white">{t('values.title')}</h3>
                            <p className="text-gray-600 dark:text-neutral-400">{t('values.desc')}</p>
                        </div>
                        <StaggeredList className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6">
                            {aboutValues.map((value) => (
                                <motion.div key={value.title} whileHover={{ y: -4 }} transition={{ duration: 0.2 }}>
                                    <Card hover className="h-full text-center bg-white/80 dark:bg-white/5 border-gray-200 dark:border-white/10 shadow-lg z-10">
                                        <CardHeader>
                                            <div className="w-16 h-16 rounded-2xl flex items-center justify-center mx-auto mb-4 bg-gray-100 dark:bg-white/10 border border-gray-200 dark:border-white/10">
                                                {value.icon}
                                            </div>
                                            <CardTitle className="text-lg text-gray-900 dark:text-white">{value.title}</CardTitle>
                                        </CardHeader>
                                        <CardContent>
                                            <CardDescription className="text-sm leading-relaxed text-gray-600 dark:text-neutral-300">
                                                {value.description}
                                            </CardDescription>
                                        </CardContent>
                                    </Card>
                                </motion.div>
                            ))}
                        </StaggeredList>
                    </div>


                    {/* Team */}
                    <div>
                        <div className="text-center mb-12">
                            <h3 className="text-2xl font-bold tracking-tight text-gray-900 dark:text-white">{t('team.title')}</h3>
                            <p className="text-gray-600 dark:text-neutral-400">{t('team.desc')}</p>
                        </div>
                        <StaggeredList className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6">
                            {aboutTeam.map((member) => (
                                <motion.div key={member.name} whileHover={{ y: -4 }} transition={{ duration: 0.2 }}>
                                    <Card hover className="text-center bg-white/80 dark:bg-white/5 border-gray-200 dark:border-white/10 shadow-lg z-10">
                                        <CardHeader>
                                            <div className="w-20 h-20 bg-gradient-to-br from-indigo-500 to-purple-600 rounded-full flex items-center justify-center mx-auto mb-4 font-bold text-2xl shadow-lg">
                                                {member.avatar}
                                            </div>
                                            <CardTitle className="text-lg text-gray-900 dark:text-white">{member.name}</CardTitle>
                                            <CardDescription className="text-indigo-600 dark:text-indigo-200">{member.role}</CardDescription>
                                        </CardHeader>
                                        <CardContent>
                                            <p className="text-gray-600 dark:text-neutral-300 text-sm">{member.description}</p>
                                        </CardContent>
                                    </Card>
                                </motion.div>
                            ))}
                        </StaggeredList>
                    </div>
                </div>
            </section>
            </div>
            <Footer />
        </div>
    )
}

export default Home