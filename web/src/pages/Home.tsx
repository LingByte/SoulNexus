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
    X,
    Building2,
    ArrowRight,
    Sparkles
} from 'lucide-react'
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
                { name: t('tech.react'), version: t('tech.reactVersion'), description: t('tech.reactDesc') },
                { name: t('tech.typescript'), version: t('tech.typescriptVersion'), description: t('tech.typescriptDesc') },
                { name: t('tech.tailwind'), version: t('tech.tailwindVersion'), description: t('tech.tailwindDesc') },
                { name: t('tech.webrtc'), version: t('tech.latest'), description: t('tech.webrtcDesc') }
            ]
        },
        {
            name: t('tech.backend'),
            technologies: [
                { name: t('tech.go'), version: t('tech.goVersion'), description: t('tech.goDesc') },
                { name: t('tech.gin'), version: t('tech.ginVersion'), description: t('tech.ginDesc') },
                { name: t('tech.websocket'), version: t('tech.latest'), description: t('tech.websocketDesc') },
            ]
        },
        {
            name: t('tech.aiml'),
            technologies: [
                { name: t('tech.asr'), version: t('tech.asrVersion'), description: t('tech.asrDesc') },
                { name: t('tech.tts'), version: t('tech.ttsVersion'), description: t('tech.ttsDesc') },
                { name: t('tech.voiceClone'), version: t('tech.voiceCloneVersion'), description: t('tech.voiceCloneDesc') },
                { name: t('tech.llm'), version: t('tech.llmVersion'), description: t('tech.llmDesc') }
            ]
        }
    ]

    // About data
    const aboutTeam = [
        { name: t('team.chenting'), role: t('team.fullStack'), avatar: 'C', description: '' },
        { name: t('team.wangyueran'), role: t('team.fullStack'), avatar: 'W', description: '' },
    ]

    return (
        <div className="min-h-screen relative bg-white dark:bg-gray-950">
            {/* 顶部导航栏 */}
            <nav className="fixed top-0 left-0 right-0 z-50 bg-white/80 dark:bg-gray-950/80 backdrop-blur-lg border-b border-gray-200/50 dark:border-gray-800/50">
                <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
                    <div className="flex items-center justify-between h-16">
                        {/* Logo */}
                        <Link to="/" className="flex items-center gap-3">
                            <div className="w-10 h-10 rounded-lg bg-gradient-to-br from-indigo-600 to-purple-600 flex items-center justify-center shadow-lg">
                                <Sparkles className="w-6 h-6" />
                            </div>
                            <span className="text-xl font-bold tracking-tight text-gray-900 dark:text-white">
                                {t('brand.name')}
                            </span>
                        </Link>

                        {/* 桌面端导航 */}
                        <div className="hidden md:flex items-center gap-6">
                            <Link to="/docs" className="text-sm font-medium text-gray-600 dark:text-gray-300 hover:text-gray-900 dark:hover:text-white transition-colors">
                                {t('nav.docs')}
                            </Link>
                            <Link to="/about" className="text-sm font-medium text-gray-600 dark:text-gray-300 hover:text-gray-900 dark:hover:text-white transition-colors">
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
                                        className="flex items-center gap-2 px-3 py-1.5 rounded-lg hover:bg-gray-100 dark:hover:bg-gray-800 transition-colors"
                                        onClick={() => setShowUserDropdown(!showUserDropdown)}
                                    >
                                        <img
                                            src={user.avatar || `https://ui-avatars.com/api/?name=${user.displayName || 'U'}&background=6366f1&color=fff`}
                                            alt={user.displayName}
                                            className="w-8 h-8 rounded-full"
                                        />
                                        <span className="text-sm font-medium text-gray-900 dark:text-white">{user.displayName}</span>
                                    </button>

                                    {/* 用户下拉菜单 */}
                                    {showUserDropdown && (
                                        <div className="absolute right-0 top-full mt-2 w-48 bg-white dark:bg-gray-900 rounded-lg shadow-xl border border-gray-200 dark:border-gray-800 z-50">
                                            <div className="flex flex-col p-2">
                                                <div className="px-3 py-2 border-b border-gray-200 dark:border-gray-800">
                                                    <p className="text-sm font-medium text-gray-900 dark:text-white">{user.displayName}</p>
                                                    <p className="text-xs text-gray-500 dark:text-gray-400">{user.email}</p>
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
                            className="md:hidden p-2 rounded-lg text-gray-600 dark:text-gray-300 hover:text-gray-900 dark:hover:text-white hover:bg-gray-100 dark:hover:bg-gray-800"
                            onClick={() => setShowMobileMenu(!showMobileMenu)}
                        >
                            {showMobileMenu ? <X className="w-6 h-6" /> : <Menu className="w-6 h-6" />}
                        </button>
                    </div>

                    {/* 移动端菜单 */}
                    {showMobileMenu && (
                        <div className="md:hidden py-4 border-t border-gray-200 dark:border-gray-800">
                            <div className="flex flex-col gap-4">
                                <Link
                                    to="/docs"
                                    className="text-sm font-medium text-gray-600 dark:text-gray-300 hover:text-gray-900 dark:hover:text-white transition-colors"
                                    onClick={() => setShowMobileMenu(false)}
                                >
                                    {t('nav.docs')}
                                </Link>
                                <Link
                                    to="/about"
                                    className="text-sm font-medium text-gray-600 dark:text-gray-300 hover:text-gray-900 dark:hover:text-white transition-colors"
                                    onClick={() => setShowMobileMenu(false)}
                                >
                                    {t('nav.about')}
                                </Link>

                                {/* 移动端主题切换和语言选择器 */}
                                <div className="flex items-center gap-3 pt-2 border-t border-gray-200 dark:border-gray-800">
                                    <div className="flex items-center gap-2">
                                        <span className="text-sm text-gray-600 dark:text-gray-400">{t('lang.select')}:</span>
                                        <LanguageSelector size="sm" />
                                    </div>
                                    <div className="flex items-center gap-2">
                                        <span className="text-sm text-gray-600 dark:text-gray-400">{t('theme.toggle')}:</span>
                                        <EnhancedThemeToggle size="sm" />
                                    </div>
                                </div>
                                {isAuthenticated && user ? (
                                    <>
                                        <div className="flex items-center gap-2 pt-2 border-t border-gray-200 dark:border-gray-800 pb-2">
                                            <img
                                                src={user.avatar || `https://ui-avatars.com/api/?name=${user.displayName || 'U'}&background=6366f1&color=fff`}
                                                alt={user.displayName}
                                                className="w-8 h-8 rounded-full"
                                            />
                                            <div className="flex-1">
                                                <p className="text-sm font-medium text-gray-900">{user.displayName}</p>
                                                <p className="text-xs text-gray-500 dark:text-gray-400">{user.email}</p>
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
            <div className="relative pt-16">
                {/* Hero Section - 专业简洁设计 */}
                <section className="relative py-20 md:py-32 overflow-hidden bg-gradient-to-b from-gray-50 via-white to-white dark:from-gray-950 dark:via-gray-900 dark:to-gray-950">
                    {/* 背景装饰 */}
                    <div className="absolute inset-0 overflow-hidden">
                        {/* 网格背景 */}
                        <div className="absolute inset-0 opacity-30 dark:opacity-20 [background-image:linear-gradient(to_right,rgba(99,102,241,0.1)_1px,transparent_1px),linear-gradient(to_bottom,rgba(99,102,241,0.1)_1px,transparent_1px)] [background-size:32px_32px]" />

                        {/* 渐变光晕 */}
                        <div className="absolute top-0 left-1/2 -translate-x-1/2 w-[800px] h-[600px] bg-gradient-to-b from-indigo-500/10 via-purple-500/5 to-transparent rounded-full blur-3xl" />
                    </div>

                    <div className="relative max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
                        <motion.div
                            initial={{ opacity: 0, y: 20 }}
                            animate={{ opacity: 1, y: 0 }}
                            transition={{ duration: 0.6 }}
                            className="text-center max-w-4xl mx-auto"
                        >
                            {/* 公司标识 */}
                            <motion.div
                                initial={{ opacity: 0, scale: 0.9 }}
                                animate={{ opacity: 1, scale: 1 }}
                                transition={{ delay: 0.1, duration: 0.5 }}
                                className="inline-flex items-center gap-2 px-4 py-2 rounded-full bg-indigo-50 dark:bg-indigo-950/50 border border-indigo-100 dark:border-indigo-900/50 mb-8"
                            >
                                <Building2 className="w-4 h-4 text-indigo-600 dark:text-indigo-400" />
                                <span className="text-sm font-medium text-indigo-600 dark:text-indigo-400">
                                    {t('footer.companyName')}
                                </span>
                            </motion.div>

                            {/* 主标题 */}
                            <motion.h1
                                initial={{ opacity: 0, y: 20 }}
                                animate={{ opacity: 1, y: 0 }}
                                transition={{ delay: 0.2, duration: 0.6 }}
                                className="text-5xl md:text-6xl lg:text-7xl font-bold tracking-tight text-gray-900 mb-6"
                            >
                                <span className="bg-gradient-to-r from-indigo-600 via-purple-600 to-indigo-600 bg-clip-text">
                                    {t('home.title')}
                                </span>
                            </motion.h1>

                            {/* 副标题 */}
                            <motion.p
                                initial={{ opacity: 0, y: 20 }}
                                animate={{ opacity: 1, y: 0 }}
                                transition={{ delay: 0.3, duration: 0.6 }}
                                className="text-xl md:text-2xl text-gray-600 dark:text-gray-300 mb-8 leading-relaxed"
                            >
                                {t('home.subtitle')}
                            </motion.p>

                            {/* CTA按钮 */}
                            <motion.div
                                initial={{ opacity: 0, y: 20 }}
                                animate={{ opacity: 1, y: 0 }}
                                transition={{ delay: 0.4, duration: 0.6 }}
                                className="flex flex-col sm:flex-row gap-4 justify-center items-center"
                            >
                                <Link
                                    to="/voice-assistant"
                                    className="group inline-flex items-center gap-2 px-8 py-4 rounded-xl font-semibold bg-gradient-to-r from-indigo-600 to-purple-600 hover:from-indigo-700 hover:to-purple-700 shadow-lg shadow-indigo-500/50 hover:shadow-xl hover:shadow-indigo-500/50 transition-all duration-300 hover:scale-105"
                                >
                                    {t('home.startNow')}
                                    <ArrowRight className="w-5 h-5 group-hover:translate-x-1 transition-transform" />
                                </Link>
                                <Link
                                    to="/docs"
                                    className="inline-flex items-center gap-2 px-8 py-4 rounded-xl font-semibold text-gray-700 dark:text-gray-300 dark:bg-gray-800 border border-gray-200 dark:border-gray-700 hover:bg-gray-50 dark:hover:bg-gray-700 transition-all duration-300"
                                >
                                    {t('nav.docs')}
                                </Link>
                            </motion.div>

                            {/* 使命描述 */}
                            <motion.div
                                initial={{ opacity: 0, y: 20 }}
                                animate={{ opacity: 1, y: 0 }}
                                transition={{ delay: 0.5, duration: 0.6 }}
                                className="mt-12 max-w-3xl mx-auto"
                            >
                                <p className="text-lg text-gray-600 dark:text-gray-400 leading-relaxed">
                                    {t('home.mission')}
                                </p>
                            </motion.div>
                        </motion.div>
                    </div>
                </section>

                {/* Core Features Section */}
                <section className="relative py-20 bg-white dark:bg-gray-950">
                    <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
                        {/* Section Header */}
                        <div className="text-center mb-16">
                            <motion.div
                                initial={{ opacity: 0, y: 20 }}
                                whileInView={{ opacity: 1, y: 0 }}
                                viewport={{ once: true }}
                                transition={{ duration: 0.6 }}
                                className="inline-flex items-center gap-2 px-4 py-2 rounded-full bg-indigo-50 dark:bg-indigo-950/50 border border-indigo-100 dark:border-indigo-900/50 mb-4"
                            >
                                <Zap className="w-4 h-4 text-indigo-600 dark:text-indigo-400" />
                                <span className="text-sm font-medium text-indigo-600 dark:text-indigo-400">
                                    {t('home.coreFeatures')}
                                </span>
                            </motion.div>
                            <motion.h2
                                initial={{ opacity: 0, y: 20 }}
                                whileInView={{ opacity: 1, y: 0 }}
                                viewport={{ once: true }}
                                transition={{ duration: 0.6, delay: 0.1 }}
                                className="text-4xl md:text-5xl font-bold text-gray-900 dark:text-white mb-4"
                            >
                                {t('home.coreFeatures')}
                            </motion.h2>
                            <motion.p
                                initial={{ opacity: 0, y: 20 }}
                                whileInView={{ opacity: 1, y: 0 }}
                                viewport={{ once: true }}
                                transition={{ duration: 0.6, delay: 0.2 }}
                                className="text-xl text-gray-600 dark:text-gray-400"
                            >
                                {t('home.coreFeaturesDesc')}
                            </motion.p>
                        </div>

                        {/* Features Grid */}
                        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
                            {coreFeatures.map((feature: any, idx: number) => {
                                const Icon = iconMap[feature.icon] || Zap
                                return (
                                    <motion.div
                                        key={idx}
                                        initial={{ opacity: 0, y: 30 }}
                                        whileInView={{ opacity: 1, y: 0 }}
                                        viewport={{ once: true }}
                                        transition={{ duration: 0.5, delay: idx * 0.1 }}
                                        className="group relative"
                                    >
                                        <Card className="h-full hover:shadow-xl transition-all duration-300 border-gray-200 dark:border-gray-800 bg-white dark:bg-gray-900">
                                            <CardHeader>
                                                <div className="w-12 h-12 rounded-xl bg-gradient-to-br from-indigo-500 to-purple-600 flex items-center justify-center mb-4 group-hover:scale-110 transition-transform duration-300">
                                                    <Icon className="w-6 h-6 text-white" />
                                                </div>
                                                <CardTitle className="text-xl mb-2 text-gray-900 dark:text-white">
                                                    {feature.title}
                                                </CardTitle>
                                                <CardDescription className="text-gray-600 dark:text-gray-400 leading-relaxed">
                                                    {feature.description}
                                                </CardDescription>
                                            </CardHeader>
                                            <CardContent>
                                                <div className="flex flex-wrap gap-2">
                                                    {feature.features?.map((tag: string, i: number) => (
                                                        <span
                                                            key={i}
                                                            className="px-3 py-1 text-xs font-medium rounded-full bg-indigo-50 dark:bg-indigo-950/50 text-indigo-700 dark:text-indigo-300 border border-indigo-100 dark:border-indigo-900/50"
                                                        >
                                                            {tag}
                                                        </span>
                                                    ))}
                                                </div>
                                            </CardContent>
                                        </Card>
                                    </motion.div>
                                )
                            })}
                        </div>
                    </div>
                </section>

                {/* Tech Stack Section */}
                <section className="relative py-20 bg-gray-50 dark:bg-gray-900">
                    <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
                        {/* Section Header */}
                        <div className="text-center mb-16">
                            <motion.div
                                initial={{ opacity: 0, y: 20 }}
                                whileInView={{ opacity: 1, y: 0 }}
                                viewport={{ once: true }}
                                transition={{ duration: 0.6 }}
                                className="inline-flex items-center gap-2 px-4 py-2 rounded-full bg-purple-50 dark:bg-purple-950/50 border border-purple-100 dark:border-purple-900/50 mb-4"
                            >
                                <BookOpenIcon className="w-4 h-4 text-purple-600 dark:text-purple-400" />
                                <span className="text-sm font-medium text-purple-600 dark:text-purple-400">
                                    {t('home.techStack')}
                                </span>
                            </motion.div>
                            <motion.h2
                                initial={{ opacity: 0, y: 20 }}
                                whileInView={{ opacity: 1, y: 0 }}
                                viewport={{ once: true }}
                                transition={{ duration: 0.6, delay: 0.1 }}
                                className="text-4xl md:text-5xl font-bold text-gray-900 dark:text-white mb-4"
                            >
                                {t('home.techStack')}
                            </motion.h2>
                            <motion.p
                                initial={{ opacity: 0, y: 20 }}
                                whileInView={{ opacity: 1, y: 0 }}
                                viewport={{ once: true }}
                                transition={{ duration: 0.6, delay: 0.2 }}
                                className="text-xl text-gray-600 dark:text-gray-400"
                            >
                                {t('home.techStackDesc')}
                            </motion.p>
                        </div>

                        {/* Tech Stack Grid */}
                        <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
                            {techStack.map((category: any, idx: number) => (
                                <motion.div
                                    key={idx}
                                    initial={{ opacity: 0, y: 30 }}
                                    whileInView={{ opacity: 1, y: 0 }}
                                    viewport={{ once: true }}
                                    transition={{ duration: 0.5, delay: idx * 0.1 }}
                                >
                                    <Card className="h-full border-gray-200 dark:border-gray-800 bg-white dark:bg-gray-900">
                                        <CardHeader>
                                            <CardTitle className="text-xl mb-6 text-gray-900 dark:text-white">
                                                {category.name}
                                            </CardTitle>
                                        </CardHeader>
                                        <CardContent>
                                            <div className="space-y-4">
                                                {category.technologies?.map((tech: any, i: number) => (
                                                    <div key={i} className="flex items-start gap-3 pb-4 border-b border-gray-100 dark:border-gray-800 last:border-0 last:pb-0">
                                                        <div className="mt-1.5 w-2 h-2 rounded-full bg-gradient-to-r from-indigo-500 to-purple-600" />
                                                        <div className="flex-1 min-w-0">
                                                            <div className="flex items-center gap-2 mb-1">
                                                                <span className="font-semibold text-gray-900 dark:text-white">
                                                                    {tech.name}
                                                                </span>
                                                                {tech.version && (
                                                                    <span className="px-2 py-0.5 text-xs font-medium rounded-md bg-gray-100 dark:bg-gray-800 text-gray-700 dark:text-gray-300 border border-gray-200 dark:border-gray-700">
                                                                        {tech.version}
                                                                    </span>
                                                                )}
                                                            </div>
                                                            <p className="text-sm text-gray-600 dark:text-gray-400">
                                                                {tech.description}
                                                            </p>
                                                        </div>
                                                    </div>
                                                ))}
                                            </div>
                                        </CardContent>
                                    </Card>
                                </motion.div>
                            ))}
                        </div>
                    </div>
                </section>

                {/* User Stories & Values Section */}
                <section className="relative py-20 bg-white dark:bg-gray-950">
                    <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
                        <div className="grid grid-cols-1 lg:grid-cols-2 gap-12 mb-20">
                            {/* User Stories */}
                            <motion.div
                                initial={{ opacity: 0, x: -30 }}
                                whileInView={{ opacity: 1, x: 0 }}
                                viewport={{ once: true }}
                                transition={{ duration: 0.6 }}
                            >
                                <div className="flex items-center gap-3 mb-6">
                                    <Target className="w-6 h-6 text-indigo-600 dark:text-indigo-400" />
                                    <h3 className="text-3xl font-bold text-gray-900 dark:text-white">
                                        {t('home.userStories')}
                                    </h3>
                                </div>
                                <p className="text-lg text-gray-600 dark:text-gray-400 mb-6 leading-relaxed">
                                    {t('story.detail')}
                                </p>
                                <div className="space-y-4">
                                    <div className="flex items-start gap-3">
                                        <CheckCircle className="w-5 h-5 text-indigo-600 dark:text-indigo-400 mt-0.5 flex-shrink-0" />
                                        <p className="text-gray-600 dark:text-gray-400">{t('story.developerPoint')}</p>
                                    </div>
                                    <div className="flex items-start gap-3">
                                        <CheckCircle className="w-5 h-5 text-indigo-600 dark:text-indigo-400 mt-0.5 flex-shrink-0" />
                                        <p className="text-gray-600 dark:text-gray-400">{t('story.animeUserPoint')}</p>
                                    </div>
                                    <div className="flex items-start gap-3">
                                        <CheckCircle className="w-5 h-5 text-indigo-600 dark:text-indigo-400 mt-0.5 flex-shrink-0" />
                                        <p className="text-gray-600 dark:text-gray-400">{t('story.creatorPoint')}</p>
                                    </div>
                                </div>
                            </motion.div>

                            {/* Values */}
                            <motion.div
                                initial={{ opacity: 0, x: 30 }}
                                whileInView={{ opacity: 1, x: 0 }}
                                viewport={{ once: true }}
                                transition={{ duration: 0.6 }}
                            >
                                <div className="mb-6">
                                    <h3 className="text-3xl font-bold text-gray-900 dark:text-white mb-2">
                                        {t('values.title')}
                                    </h3>
                                    <p className="text-lg text-gray-600 dark:text-gray-400">
                                        {t('values.desc')}
                                    </p>
                                </div>
                                <div className="grid grid-cols-1 gap-4">
                                    {[
                                        { icon: Eye, title: t('values.userCentric') },
                                        { icon: UsersIcon, title: t('values.communityDriven') },
                                        { icon: Award, title: t('values.excellence') },
                                    ].map((item, i) => {
                                        const Icon = item.icon as any
                                        return (
                                            <div
                                                key={i}
                                                className="flex items-center gap-4 p-4 rounded-xl bg-gray-50 dark:bg-gray-900 border border-gray-200 dark:border-gray-800 hover:border-indigo-300 dark:hover:border-indigo-700 transition-colors"
                                            >
                                                <div className="w-10 h-10 rounded-lg bg-gradient-to-br from-indigo-500 to-purple-600 flex items-center justify-center">
                                                    <Icon className="w-5 h-5 text-white" />
                                                </div>
                                                <span className="font-semibold text-gray-900 dark:text-white">
                                                    {item.title}
                                                </span>
                                            </div>
                                        )
                                    })}
                                </div>
                            </motion.div>
                        </div>

                        {/* User Stories Cards */}
                        <div className="mb-20">
                            <motion.h3
                                initial={{ opacity: 0, y: 20 }}
                                whileInView={{ opacity: 1, y: 0 }}
                                viewport={{ once: true }}
                                transition={{ duration: 0.6 }}
                                className="text-3xl font-bold text-center text-gray-900 dark:text-white mb-12"
                            >
                                {t('home.userStories')}
                            </motion.h3>

                            <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6">
                                {[
                                    { icon: Code, title: t('story.developer'), description: t('story.developerDesc') },
                                    { icon: MessageCircleIcon, title: t('story.animeUser'), description: t('story.animeUserDesc') },
                                    { icon: UsersIcon, title: t('story.education'), description: t('story.educationDesc') },
                                    { icon: Mic, title: t('story.creator'), description: t('story.creatorDesc') },
                                ].map((item, idx) => {
                                    const Icon = item.icon as any
                                    return (
                                        <motion.div
                                            key={idx}
                                            initial={{ opacity: 0, y: 30 }}
                                            whileInView={{ opacity: 1, y: 0 }}
                                            viewport={{ once: true }}
                                            transition={{ duration: 0.5, delay: idx * 0.08 }}
                                        >
                                            <Card className="h-full border-gray-200 dark:border-gray-800 bg-white dark:bg-gray-900 hover:shadow-xl transition-all duration-300">
                                                <CardHeader>
                                                    <div className="w-12 h-12 rounded-xl bg-gradient-to-br from-indigo-500 to-purple-600 flex items-center justify-center mb-4">
                                                        <Icon className="w-6 h-6 text-white" />
                                                    </div>
                                                    <CardTitle className="text-lg text-gray-900 dark:text-white">
                                                        {item.title}
                                                    </CardTitle>
                                                </CardHeader>
                                                <CardContent>
                                                    <p className="text-sm text-gray-600 dark:text-gray-400 leading-relaxed">
                                                        {item.description}
                                                    </p>
                                                </CardContent>
                                            </Card>
                                        </motion.div>
                                    )
                                })}
                            </div>
                        </div>

                        {/* Team & Company */}
                        <div className="mb-6">
                            <div className="text-center mb-12">
                                <motion.div
                                    initial={{ opacity: 0, y: 20 }}
                                    whileInView={{ opacity: 1, y: 0 }}
                                    viewport={{ once: true }}
                                    transition={{ duration: 0.6 }}
                                    className="inline-flex items-center gap-2 px-4 py-2 rounded-full bg-emerald-50 dark:bg-emerald-950/50 border border-emerald-100 dark:border-emerald-900/50 mb-4"
                                >
                                    <Building2 className="w-4 h-4 text-emerald-600 dark:text-emerald-400" />
                                    <span className="text-sm font-medium text-emerald-700 dark:text-emerald-300">
                                        {t('footer.companyName')}
                                    </span>
                                </motion.div>

                                <motion.h3
                                    initial={{ opacity: 0, y: 20 }}
                                    whileInView={{ opacity: 1, y: 0 }}
                                    viewport={{ once: true }}
                                    transition={{ duration: 0.6, delay: 0.1 }}
                                    className="text-3xl font-bold text-gray-900 dark:text-white"
                                >
                                    {t('team.title')}
                                </motion.h3>
                                <motion.p
                                    initial={{ opacity: 0, y: 20 }}
                                    whileInView={{ opacity: 1, y: 0 }}
                                    viewport={{ once: true }}
                                    transition={{ duration: 0.6, delay: 0.2 }}
                                    className="mt-3 text-lg text-gray-600 dark:text-gray-400"
                                >
                                    {t('team.desc')}
                                </motion.p>
                            </div>

                            <StaggeredList className="grid grid-cols-1 md:grid-cols-2 gap-6">
                                {aboutTeam.map((member) => (
                                    <motion.div key={member.name} whileHover={{ y: -4 }} transition={{ duration: 0.2 }}>
                                        <Card className="text-left border-gray-200 dark:border-gray-800 bg-white dark:bg-gray-900 hover:shadow-xl transition-all duration-300">
                                            <CardHeader className="flex flex-row items-center gap-4">
                                                <div className="w-14 h-14 bg-gradient-to-br from-indigo-500 to-purple-600 rounded-2xl flex items-center justify-center font-bold text-xl shadow-lg">
                                                    {member.avatar}
                                                </div>
                                                <div className="min-w-0">
                                                    <CardTitle className="text-lg text-gray-900 dark:text-white truncate">
                                                        {member.name}
                                                    </CardTitle>
                                                    <CardDescription className="text-sm text-indigo-700 dark:text-indigo-300">
                                                        {member.role}
                                                    </CardDescription>
                                                </div>
                                            </CardHeader>
                                            {member.description ? (
                                                <CardContent>
                                                    <p className="text-sm text-gray-600 dark:text-gray-400 leading-relaxed">
                                                        {member.description}
                                                    </p>
                                                </CardContent>
                                            ) : null}
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
                           