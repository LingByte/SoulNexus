import { motion } from 'framer-motion'
import { useState, useEffect, useRef } from 'react'
import { Link } from 'react-router-dom'
import {
    Zap,
    Settings as SettingsIcon,
    BookOpen as BookOpenIcon,
    Building2,
    MapPin,
    Mail,
    MessageCircle as MessageCircleIcon,
    Activity as ActivityIcon,
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
import Button from "@/components/UI/Button";
import { useAuthStore } from "@/stores/authStore";
import EnhancedThemeToggle from "@/components/UI/EnhancedThemeToggle";
import LanguageSelector from "@/components/UI/LanguageSelector";
import { useI18nStore } from "@/stores/i18nStore";
import Footer from "@/components/Layout/Footer.tsx";
import PageSEO from "@/components/SEO/PageSEO.tsx";
import ContentCarousel from "@/components/Home/ContentCarousel.tsx";
import { beginSSOLogin } from '@/utils/sso';

const iconMap: Record<string, any> = {
    Zap,
    Settings: SettingsIcon,
    BookOpen: BookOpenIcon,
    Users: UserIcon,
    MessageCircle: MessageCircleIcon,
    Activity: ActivityIcon,
    Phone,
    Mic,
    Key,
    Code,
}

const Home = () => {
    const [showMobileMenu, setShowMobileMenu] = useState(false)
    const [showUserDropdown, setShowUserDropdown] = useState(false)
    const [hoveredContactPanel, setHoveredContactPanel] = useState<'company' | 'contact' | null>(null)
    const { user, isAuthenticated, logout } = useAuthStore()
    const { t } = useI18nStore()
    const userDropdownRef = useRef<HTMLDivElement>(null)

    useEffect(() => {
        document.title = 'SoulMy - 智能AI语音通话平台 | WebRTC实时通信解决方案'
        
        // 更新meta description
        const metaDescription = document.querySelector('meta[name="description"]')
        if (metaDescription) {
            metaDescription.setAttribute('content', 'SoulMy是基于WebRTC的智能AI语音通话平台，提供低延迟实时通信、AI语音助手、声音克隆、知识库管理等企业级解决方案。')
        }
    }, [])

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
    t('tech.frontend');
    t('tech.react');
    t('tech.reactDesc');
    t('tech.typescript');
    t('tech.typescriptDesc');
    t('tech.tailwind');
    t('tech.tailwindDesc');
    t('tech.webrtc');
    t('tech.latest');
    t('tech.webrtcDesc');
    t('tech.backend');
    t('tech.go');
    t('tech.goDesc');
    t('tech.gin');
    t('tech.ginDesc');
    t('tech.websocket');
    t('tech.latest');
    t('tech.websocketDesc');
    t('tech.aiml');
    t('tech.asr');
    t('tech.asrDesc');
    t('tech.tts');
    t('tech.ttsDesc');
    t('tech.voiceClone');
    t('tech.voiceCloneDesc');
    t('tech.llm');
    t('tech.llmDesc');

    return (
        <div className="min-h-screen relative">
            {/* SEO优化组件 */}
            <PageSEO
                title="SoulMy - 智能AI语音通话平台 | WebRTC实时通信解决方案"
                description="SoulMy是基于WebRTC的智能AI语音通话平台，提供低延迟实时通信、AI语音助手、声音克隆、知识库管理等企业级解决方案。支持多模态交互，助力企业数字化转型。"
                keywords="AI语音通话,WebRTC,实时通信,语音助手,声音克隆,AI对话,智能客服,语音识别,TTS,ASR,低延迟通话,企业通信,SoulMy"
                ogImage="https://cetide-1325039295.cos.ap-chengdu.myqcloud.com/folder/icon-192x192.ico"
                canonical="https://SoulMy.com/"
                structuredData={{
                    "@context": "https://schema.org",
                    "@type": "WebPage",
                    "name": "SoulMy - 智能AI语音通话平台",
                    "description": "基于WebRTC的企业级AI语音通话解决方案",
                    "url": "https://SoulMy.com/",
                    "publisher": {
                        "@type": "Organization",
                        "name": "成都解忧造物科技有限责任公司",
                        "logo": {
                            "@type": "ImageObject",
                            "url": "https://cetide-1325039295.cos.ap-chengdu.myqcloud.com/folder/icon-192x192.ico"
                        }
                    },
                    "mainEntity": {
                        "@type": "SoftwareApplication",
                        "name": "SoulMy",
                        "applicationCategory": "BusinessApplication",
                        "offers": {
                            "@type": "Offer",
                            "price": "0",
                            "priceCurrency": "CNY"
                        }
                    }
                }}
            />
            
            {/* 顶部导航栏 */}
            <nav className="fixed top-0 left-0 right-0 z-50 bg-background/80 backdrop-blur-md border-b border-border">
                <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
                    <div className="flex items-center justify-between h-16">
                        {/* Logo */}
                        <Link to="/" className="flex items-center gap-2">
                            <img
                                src="/SoulMy.png"
                                alt="SoulMy Logo"
                                className="w-8 h-8 rounded"
                            />
                            <span className="text-xl font-extrabold tracking-wider">
                                <span className="text-purple-600">{t('brand.name')}</span>
                            </span>
                        </Link>

                        {/* 桌面端导航 */}
                        <div className="hidden md:flex items-center gap-4">
                            <a
                                href="https://docs.lingecho.com/"
                                target="_blank"
                                rel="noreferrer"
                                className="text-muted-foreground hover:text-foreground transition-colors"
                            >
                                {t('nav.docs')}
                            </a>
                            
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
                                    onClick={() => beginSSOLogin('/assistants')}
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
                                <a
                                    href="https://docs.lingecho.com/"
                                    target="_blank"
                                    rel="noreferrer"
                                    className="text-muted-foreground hover:text-foreground transition-colors"
                                    onClick={() => setShowMobileMenu(false)}
                                >
                                    {t('nav.docs')}
                                </a>
                                
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
                                        onClick={() => {
                                            beginSSOLogin('/assistants')
                                            setShowMobileMenu(false)
                                        }}
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

            {/* Floating contact dock + expandable card */}
            <aside
                className="fixed right-4 top-[62%] -translate-y-1/2 z-40 hidden lg:block"
                onMouseLeave={() => setHoveredContactPanel(null)}
            >
                {hoveredContactPanel && (
                    <div
                        className="absolute right-[4.8rem] top-1/2 -translate-y-1/2 w-[21rem] rounded-2xl border border-white/20 bg-white/90 dark:bg-gray-900/85 backdrop-blur-md shadow-2xl p-5"
                        onMouseEnter={() => setHoveredContactPanel(hoveredContactPanel)}
                    >
                        {hoveredContactPanel === 'company' ? (
                            <div className="space-y-4 text-sm text-gray-700 dark:text-gray-200">
                                <div className="flex items-start gap-2">
                                    <Building2 className="w-4 h-4 mt-0.5 text-indigo-500 shrink-0" />
                                    <div>
                                        <p className="text-xs text-muted-foreground">公司</p>
                                        <p>成都解忧造物科技有限责任公司</p>
                                    </div>
                                </div>
                                <div className="flex items-start gap-2">
                                    <MapPin className="w-4 h-4 mt-0.5 text-indigo-500 shrink-0" />
                                    <div>
                                        <p className="text-xs text-muted-foreground">地址</p>
                                        <p>四川省成都市成华区一环路东一段159号1栋1层1号附17号</p>
                                    </div>
                                </div>
                            </div>
                        ) : (
                            <div className="space-y-4 text-sm text-gray-700 dark:text-gray-200">
                                <div className="flex items-start gap-2">
                                    <Mail className="w-4 h-4 mt-0.5 text-indigo-500 shrink-0" />
                                    <div>
                                        <p className="text-xs text-muted-foreground">联系</p>
                                        <a
                                            href="mailto:19511899044@163.com"
                                            className="text-indigo-600 hover:text-indigo-700 dark:text-indigo-300 dark:hover:text-indigo-200"
                                        >
                                            19511899044@163.com
                                        </a>
                                    </div>
                                </div>
                                <div>
                                    <a
                                        href="https://github.com/LingByte/SoulNexus"
                                        target="_blank"
                                        rel="noreferrer"
                                        className="text-indigo-600 hover:text-indigo-700 dark:text-indigo-300 dark:hover:text-indigo-200"
                                    >
                                        社区开源项目: https://github.com/LingByte/SoulNexus
                                    </a>
                                </div>
                            </div>
                        )}
                    </div>
                )}
                <div className="ml-auto w-16 rounded-[32px] bg-white/95 dark:bg-gray-100 shadow-xl py-4 px-1.5 flex flex-col items-center gap-4 border border-black/5">
                    <button
                        type="button"
                        onMouseEnter={() => setHoveredContactPanel('company')}
                        className="w-full flex flex-col items-center text-gray-700 hover:text-indigo-600 transition-colors"
                        aria-label="公司信息"
                        title="公司信息"
                    >
                        <Building2 className="w-6 h-6 mb-1" />
                        <span className="text-[12px] leading-4 tracking-wide text-center">
                            公司
                            <br />
                            信息
                        </span>
                    </button>
                    <div className="w-8 h-px bg-gray-300" />
                    <button
                        type="button"
                        onMouseEnter={() => setHoveredContactPanel('contact')}
                        className="w-full flex flex-col items-center text-gray-700 hover:text-indigo-600 transition-colors"
                        aria-label="联系方式"
                        title="联系方式"
                    >
                        <Mail className="w-6 h-6 mb-1" />
                        <span className="text-[12px] leading-4 tracking-wide text-center">
                            联系
                            <br />
                            方式
                        </span>
                    </button>
                </div>
            </aside>

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
            <section className="relative py-15 text-center overflow-hidden" aria-label="主页横幅">
                {/* 主要渐变背景 - 浅蓝到浅紫 */}
                <div className="absolute inset-0 bg-gradient-to-br from-blue-100 via-indigo-100 to-purple-100 dark:from-gray-800/50 dark:via-blue-800/20 dark:to-purple-800/20" aria-hidden="true"></div>
                
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

            {/* Platform Showcase Section */}
            <section id="platform-showcase" className="relative py-24 overflow-hidden">
                {/* 渐变背景 */}
                <div className="absolute inset-0 bg-gradient-to-br from-indigo-50 via-white to-purple-50 dark:from-gray-900 dark:via-gray-900 dark:to-indigo-950/30" aria-hidden="true"></div>
                
                {/* 装饰性光效 */}
                <div className="absolute top-1/4 left-0 w-96 h-96 bg-indigo-400/10 rounded-full blur-3xl" aria-hidden="true"></div>
                <div className="absolute bottom-1/4 right-0 w-96 h-96 bg-purple-400/10 rounded-full blur-3xl" aria-hidden="true"></div>
                
                <div className="max-w-7xl mx-auto px-4 relative z-10">
                    <ContentCarousel
                        subtitle={t('home.platformShowcase.subtitle')}
                        title={t('home.platformShowcase.title')}
                        description={t('home.platformShowcase.description')}
                        features={[
                            t('home.platformShowcase.feature1'),
                            t('home.platformShowcase.feature2'),
                            t('home.platformShowcase.feature3'),
                            t('home.platformShowcase.feature4')
                        ]}
                        carouselItems={[
                            {
                                image: '/images/workflow.png',
                                alt: 'Workflow Automation'
                            },
                            {
                                image: '/images/voiceclone.png',
                                alt: 'Voice Clone'
                            },
                            {
                                image: '/images/debug-assistant.png',
                                alt: 'Debug Assistant'
                            },
                            {
                                image: '/images/js-template.png',
                                alt: 'JS Template'
                            },
                            {
                                image: '/images/device-log.png',
                                alt: 'Device Log'
                            }
                        ]}
                        ctaText={t('home.platformShowcase.cta')}
                        ctaLink="https://docs.lingecho.com/"
                    />
                </div>
            </section>

            {/* Who We Serve Section */}
            <section id="who-we-serve" className="relative py-20 overflow-hidden">
                {/* 渐变背景 */}
                <div className="absolute inset-0 bg-gradient-to-br from-purple-50 via-indigo-50 to-blue-50 dark:from-gray-900 dark:via-indigo-950/30 dark:to-purple-950/30" aria-hidden="true"></div>
                
                {/* 装饰性网格 */}
                <div className="absolute inset-0 opacity-20 [background-image:linear-gradient(to_right,rgba(99,102,241,0.1)_1px,transparent_1px),linear-gradient(to_bottom,rgba(99,102,241,0.1)_1px,transparent_1px)] [background-size:40px_40px]" aria-hidden="true"></div>
                
                {/* 光效 */}
                <div className="absolute top-0 left-1/4 w-96 h-96 bg-indigo-400/10 rounded-full blur-3xl" aria-hidden="true"></div>
                <div className="absolute bottom-0 right-1/4 w-96 h-96 bg-purple-400/10 rounded-full blur-3xl" aria-hidden="true"></div>
                
                <div className="max-w-6xl mx-auto px-4 relative z-10">
                    {/* 标题 */}
                    <motion.div
                        initial={{ opacity: 0, y: 20 }}
                        whileInView={{ opacity: 1, y: 0 }}
                        viewport={{ once: false, margin: "-100px" }}
                        transition={{ duration: 0.6 }}
                        className="text-center mb-12"
                    >
                        <h2 className="text-4xl md:text-5xl font-bold text-gray-900 dark:text-white mb-4">
                            {t('home.whoWeServe.title') || '我们服务的客户'}
                        </h2>
                        <p className="text-lg text-gray-600 dark:text-gray-300 max-w-3xl mx-auto leading-relaxed">
                            {t('home.whoWeServe.subtitle') || '为企业、开发者和创新团队提供完整的AI语音交互解决方案，助力业务数字化转型'}
                        </p>
                    </motion.div>

                    {/* 客户类型卡片 */}
                    <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
                        {/* 智能客服企业 */}
                        <motion.div
                            initial={{ opacity: 0, y: 30 }}
                            whileInView={{ opacity: 1, y: 0 }}
                            viewport={{ once: false, margin: "-100px" }}
                            transition={{ duration: 0.6, delay: 0.1 }}
                            className="group relative bg-white/80 dark:bg-gray-800/80 backdrop-blur-sm rounded-2xl p-6 border border-indigo-200/50 dark:border-indigo-800/50 hover:border-indigo-400 dark:hover:border-indigo-600 transition-all duration-300 hover:shadow-xl hover:shadow-indigo-500/20"
                        >
                            <div className="flex items-start gap-4 mb-4">
                                <div className="w-12 h-12 rounded-xl bg-gradient-to-br from-indigo-500 to-purple-600 flex items-center justify-center flex-shrink-0 group-hover:scale-110 transition-transform duration-300">
                                    <Phone className="w-6 h-6" />
                                </div>
                                <div>
                                    <h3 className="text-xl font-bold text-gray-900 dark:text-white mb-2">
                                        {t('home.whoWeServe.customerService') || '智能客服企业'}
                                    </h3>
                                    <p className="text-sm text-gray-600 dark:text-gray-400 leading-relaxed">
                                        {t('home.whoWeServe.customerServiceDesc') || '为呼叫中心、在线客服平台提供AI语音助手，实现7x24小时智能应答，降低人力成本，提升服务质量'}
                                    </p>
                                </div>
                            </div>
                        </motion.div>

                        {/* 企业应用开发者 */}
                        <motion.div
                            initial={{ opacity: 0, y: 30 }}
                            whileInView={{ opacity: 1, y: 0 }}
                            viewport={{ once: false, margin: "-100px" }}
                            transition={{ duration: 0.6, delay: 0.2 }}
                            className="group relative bg-white/80 dark:bg-gray-800/80 backdrop-blur-sm rounded-2xl p-6 border border-blue-200/50 dark:border-blue-800/50 hover:border-blue-400 dark:hover:border-blue-600 transition-all duration-300 hover:shadow-xl hover:shadow-blue-500/20"
                        >
                            <div className="flex items-start gap-4 mb-4">
                                <div className="w-12 h-12 rounded-xl bg-gradient-to-br from-blue-500 to-cyan-600 flex items-center justify-center flex-shrink-0 group-hover:scale-110 transition-transform duration-300">
                                    <Code className="w-6 h-6" />
                                </div>
                                <div>
                                    <h3 className="text-xl font-bold text-gray-900 dark:text-white mb-2">
                                        {t('home.whoWeServe.developers') || '企业应用开发者'}
                                    </h3>
                                    <p className="text-sm text-gray-600 dark:text-gray-400 leading-relaxed">
                                        {t('home.whoWeServe.developersDesc') || '通过RESTful API和SDK快速集成AI语音能力，支持自定义工作流、知识库和工具插件，加速产品创新'}
                                    </p>
                                </div>
                            </div>
                        </motion.div>

                        {/* 教育培训机构 */}
                        <motion.div
                            initial={{ opacity: 0, y: 30 }}
                            whileInView={{ opacity: 1, y: 0 }}
                            viewport={{ once: false, margin: "-100px" }}
                            transition={{ duration: 0.6, delay: 0.3 }}
                            className="group relative bg-white/80 dark:bg-gray-800/80 backdrop-blur-sm rounded-2xl p-6 border border-purple-200/50 dark:border-purple-800/50 hover:border-purple-400 dark:hover:border-purple-600 transition-all duration-300 hover:shadow-xl hover:shadow-purple-500/20"
                        >
                            <div className="flex items-start gap-4 mb-4">
                                <div className="w-12 h-12 rounded-xl bg-gradient-to-br from-purple-500 to-pink-600 flex items-center justify-center flex-shrink-0 group-hover:scale-110 transition-transform duration-300">
                                    <BookOpenIcon className="w-6 h-6" />
                                </div>
                                <div>
                                    <h3 className="text-xl font-bold text-gray-900 dark:text-white mb-2">
                                        {t('home.whoWeServe.education') || '教育培训机构'}
                                    </h3>
                                    <p className="text-sm text-gray-600 dark:text-gray-400 leading-relaxed">
                                        {t('home.whoWeServe.educationDesc') || '打造AI语音陪练、口语评测、智能答疑系统，提供个性化学习体验，提升教学效率和学习效果'}
                                    </p>
                                </div>
                            </div>
                        </motion.div>

                        {/* 医疗健康平台 */}
                        <motion.div
                            initial={{ opacity: 0, y: 30 }}
                            whileInView={{ opacity: 1, y: 0 }}
                            viewport={{ once: false, margin: "-100px" }}
                            transition={{ duration: 0.6, delay: 0.4 }}
                            className="group relative bg-white/80 dark:bg-gray-800/80 backdrop-blur-sm rounded-2xl p-6 border border-green-200/50 dark:border-green-800/50 hover:border-green-400 dark:hover:border-green-600 transition-all duration-300 hover:shadow-xl hover:shadow-green-500/20"
                        >
                            <div className="flex items-start gap-4 mb-4">
                                <div className="w-12 h-12 rounded-xl bg-gradient-to-br from-green-500 to-emerald-600 flex items-center justify-center flex-shrink-0 group-hover:scale-110 transition-transform duration-300">
                                    <ActivityIcon className="w-6 h-6" />
                                </div>
                                <div>
                                    <h3 className="text-xl font-bold text-gray-900 dark:text-white mb-2">
                                        {t('home.whoWeServe.healthcare') || '医疗健康平台'}
                                    </h3>
                                    <p className="text-sm text-gray-600 dark:text-gray-400 leading-relaxed">
                                        {t('home.whoWeServe.healthcareDesc') || '构建智能问诊助手、健康咨询机器人、用药提醒系统，提供便捷的医疗健康服务，改善患者体验'}
                                    </p>
                                </div>
                            </div>
                        </motion.div>

                        {/* 智能硬件厂商 */}
                        <motion.div
                            initial={{ opacity: 0, y: 30 }}
                            whileInView={{ opacity: 1, y: 0 }}
                            viewport={{ once: false, margin: "-100px" }}
                            transition={{ duration: 0.6, delay: 0.5 }}
                            className="group relative bg-white/80 dark:bg-gray-800/80 backdrop-blur-sm rounded-2xl p-6 border border-orange-200/50 dark:border-orange-800/50 hover:border-orange-400 dark:hover:border-orange-600 transition-all duration-300 hover:shadow-xl hover:shadow-orange-500/20"
                        >
                            <div className="flex items-start gap-4 mb-4">
                                <div className="w-12 h-12 rounded-xl bg-gradient-to-br from-orange-500 to-red-600 flex items-center justify-center flex-shrink-0 group-hover:scale-110 transition-transform duration-300">
                                    <Mic className="w-6 h-6" />
                                </div>
                                <div>
                                    <h3 className="text-xl font-bold text-gray-900 dark:text-white mb-2">
                                        {t('home.whoWeServe.hardware') || '智能硬件厂商'}
                                    </h3>
                                    <p className="text-sm text-gray-600 dark:text-gray-400 leading-relaxed">
                                        {t('home.whoWeServe.hardwareDesc') || '为智能音箱、机器人、车载设备等硬件产品赋能AI语音交互能力，支持声纹识别、音色克隆等高级功能'}
                                    </p>
                                </div>
                            </div>
                        </motion.div>

                        {/* 内容创作者 */}
                        <motion.div
                            initial={{ opacity: 0, y: 30 }}
                            whileInView={{ opacity: 1, y: 0 }}
                            viewport={{ once: false, margin: "-100px" }}
                            transition={{ duration: 0.6, delay: 0.6 }}
                            className="group relative bg-white/80 dark:bg-gray-800/80 backdrop-blur-sm rounded-2xl p-6 border border-pink-200/50 dark:border-pink-800/50 hover:border-pink-400 dark:hover:border-pink-600 transition-all duration-300 hover:shadow-xl hover:shadow-pink-500/20"
                        >
                            <div className="flex items-start gap-4 mb-4">
                                <div className="w-12 h-12 rounded-xl bg-gradient-to-br from-pink-500 to-rose-600 flex items-center justify-center flex-shrink-0 group-hover:scale-110 transition-transform duration-300">
                                    <MessageCircleIcon className="w-6 h-6" />
                                </div>
                                <div>
                                    <h3 className="text-xl font-bold text-gray-900 dark:text-white mb-2">
                                        {t('home.whoWeServe.creators') || '内容创作者'}
                                    </h3>
                                    <p className="text-sm text-gray-600 dark:text-gray-400 leading-relaxed">
                                        {t('home.whoWeServe.creatorsDesc') || '利用声音克隆技术创建虚拟主播、有声读物、播客节目，降低内容制作成本，提升创作效率'}
                                    </p>
                                </div>
                            </div>
                        </motion.div>
                    </div>

                    {/* 底部说明 */}
                    <motion.div
                        initial={{ opacity: 0, y: 20 }}
                        whileInView={{ opacity: 1, y: 0 }}
                        viewport={{ once: false, margin: "-100px" }}
                        transition={{ duration: 0.6, delay: 0.7 }}
                        className="mt-12 text-center"
                    >
                        <div className="inline-flex items-center gap-2 px-6 py-3 bg-gradient-to-r from-indigo-500/10 to-purple-500/10 dark:from-indigo-500/20 dark:to-purple-500/20 rounded-full border border-indigo-200/50 dark:border-indigo-700/50">
                            <UserIcon className="w-5 h-5 text-indigo-600 dark:text-indigo-400" />
                            <p className="text-sm font-medium text-gray-700 dark:text-gray-300">
                                {t('home.whoWeServe.cta') || '无论您是企业、开发者还是创新团队，SoulMy都能为您提供灵活、可扩展的AI语音解决方案'}
                            </p>
                        </div>
                    </motion.div>
                </div>
            </section>

            {/* Highlights Section */}
            <section id="highlights" className="relative py-24 overflow-hidden">
                {/* 浅色渐变背景 */}
                <div className="absolute inset-0 bg-gradient-to-br from-purple-50 via-white to-indigo-50 dark:from-purple-950/30 dark:via-gray-900 dark:to-indigo-950/30" aria-hidden="true"></div>
                
                {/* 动态光效 */}
                <div className="absolute inset-0 bg-gradient-to-r from-transparent via-purple-200/20 to-transparent animate-pulse" aria-hidden="true"></div>
                
                {/* 装饰性网格 */}
                <div className="absolute inset-0 opacity-10 [background-image:linear-gradient(to_right,rgba(147,51,234,0.3)_1px,transparent_1px),linear-gradient(to_bottom,rgba(147,51,234,0.3)_1px,transparent_1px)] [background-size:50px_50px]" aria-hidden="true"></div>
                
                {/* 光球效果 */}
                <div className="absolute top-20 left-20 w-96 h-96 bg-purple-300/20 rounded-full blur-3xl" aria-hidden="true"></div>
                <div className="absolute bottom-20 right-20 w-96 h-96 bg-indigo-300/20 rounded-full blur-3xl" aria-hidden="true"></div>
                
                <div className="max-w-6xl mx-auto px-4 relative z-10">
                    {/* 标题 */}
                    <motion.div
                        initial={{ opacity: 0, y: 20 }}
                        whileInView={{ opacity: 1, y: 0 }}
                        viewport={{ once: false, margin: "-100px" }}
                        transition={{ duration: 0.6 }}
                        className="text-center mb-16"
                    >
                        <h2 className="text-4xl md:text-5xl font-bold text-gray-900 dark:text-white mb-4">
                            {t('home.highlights.title') || '核心亮点'}
                        </h2>
                        <p className="text-lg text-gray-600 dark:text-gray-300 max-w-3xl mx-auto leading-relaxed">
                            {t('home.highlights.subtitle') || '基于Go语言构建的高性能AI语音平台，提供企业级的实时通信、安全加密和灵活扩展能力'}
                        </p>
                    </motion.div>

                    {/* 亮点卡片 */}
                    <div className="grid grid-cols-1 md:grid-cols-3 gap-8">
                        {/* 高并发性能 */}
                        <motion.div
                            initial={{ opacity: 0, y: 30 }}
                            whileInView={{ opacity: 1, y: 0 }}
                            viewport={{ once: false, margin: "-100px" }}
                            transition={{ duration: 0.6, delay: 0.1 }}
                            className="group relative"
                        >
                            <div className="absolute inset-0 bg-gradient-to-br from-purple-400/30 to-white/30 rounded-2xl blur-xl group-hover:blur-2xl transition-all duration-300" aria-hidden="true"></div>
                            <div className="relative bg-gradient-to-br from-purple-100 via-white to-purple-50 dark:from-purple-900/30 dark:via-gray-800/50 dark:to-indigo-900/30 backdrop-blur-sm rounded-2xl p-8 border border-purple-200/50 dark:border-purple-500/30 hover:border-purple-300 dark:hover:border-purple-400/50 transition-all duration-300 h-full shadow-lg hover:shadow-xl">
                                <div className="w-14 h-14 rounded-xl bg-gradient-to-br from-purple-500 to-indigo-600 flex items-center justify-center mb-6 group-hover:scale-110 transition-transform duration-300 shadow-md">
                                    <Zap className="w-7 h-7" />
                                </div>
                                <h3 className="text-2xl font-bold text-gray-900 dark:text-white mb-4">
                                    {t('home.highlights.performance.title') || '高并发性能'}
                                </h3>
                                <p className="text-gray-700 dark:text-gray-300 leading-relaxed">
                                    {t('home.highlights.performance.desc') || '基于Go语言的异步生态系统构建，提供亚毫秒级调度和稳定性能，即使在高负载下也能确保一致的通话质量'}
                                </p>
                            </div>
                        </motion.div>

                        {/* 安全与现代协议 */}
                        <motion.div
                            initial={{ opacity: 0, y: 30 }}
                            whileInView={{ opacity: 1, y: 0 }}
                            viewport={{ once: false, margin: "-100px" }}
                            transition={{ duration: 0.6, delay: 0.2 }}
                            className="group relative"
                        >
                            <div className="absolute inset-0 bg-gradient-to-br from-purple-400/30 to-white/30 rounded-2xl blur-xl group-hover:blur-2xl transition-all duration-300" aria-hidden="true"></div>
                            <div className="relative bg-gradient-to-br from-purple-100 via-white to-purple-50 dark:from-purple-900/30 dark:via-gray-800/50 dark:to-indigo-900/30 backdrop-blur-sm rounded-2xl p-8 border border-purple-200/50 dark:border-purple-500/30 hover:border-purple-300 dark:hover:border-purple-400/50 transition-all duration-300 h-full shadow-lg hover:shadow-xl">
                                <div className="w-14 h-14 rounded-xl bg-gradient-to-br from-purple-500 to-pink-600 flex items-center justify-center mb-6 group-hover:scale-110 transition-transform duration-300 shadow-md">
                                    <Key className="w-7 h-7" />
                                </div>
                                <h3 className="text-2xl font-bold text-gray-900 dark:text-white mb-4">
                                    {t('home.highlights.security.title') || '安全与现代协议'}
                                </h3>
                                <p className="text-gray-700 dark:text-gray-300 leading-relaxed">
                                    {t('home.highlights.security.desc') || '原生支持 WebRTC 与 SRTP 加密会话，以及广泛的编解码器兼容性，为多样化的终端类型提供安全连接'}
                                </p>
                            </div>
                        </motion.div>

                        {/* 开放可扩展架构 */}
                        <motion.div
                            initial={{ opacity: 0, y: 30 }}
                            whileInView={{ opacity: 1, y: 0 }}
                            viewport={{ once: false, margin: "-100px" }}
                            transition={{ duration: 0.6, delay: 0.3 }}
                            className="group relative"
                        >
                            <div className="absolute inset-0 bg-gradient-to-br from-purple-400/30 to-white/30 rounded-2xl blur-xl group-hover:blur-2xl transition-all duration-300" aria-hidden="true"></div>
                            <div className="relative bg-gradient-to-br from-purple-100 via-white to-purple-50 dark:from-purple-900/30 dark:via-gray-800/50 dark:to-indigo-900/30 backdrop-blur-sm rounded-2xl p-8 border border-purple-200/50 dark:border-purple-500/30 hover:border-purple-300 dark:hover:border-purple-400/50 transition-all duration-300 h-full shadow-lg hover:shadow-xl">
                                <div className="w-14 h-14 rounded-xl bg-gradient-to-br from-indigo-500 to-blue-600 flex items-center justify-center mb-6 group-hover:scale-110 transition-transform duration-300 shadow-md">
                                    <Code className="w-7 h-7" />
                                </div>
                                <h3 className="text-2xl font-bold text-gray-900 dark:text-white mb-4">
                                    {t('home.highlights.architecture.title') || '开放可扩展架构'}
                                </h3>
                                <p className="text-gray-700 dark:text-gray-300 leading-relaxed">
                                    {t('home.highlights.architecture.desc') || '完整的开源生态系统，支持社区驱动的插件和可定制的语音/AI工作流，为不断发展的业务需求提供灵活的集成和定制解决方案'}
                                </p>
                            </div>
                        </motion.div>
                    </div>

                    {/* 技术特性列表 */}
                    <motion.div
                        initial={{ opacity: 0, y: 20 }}
                        whileInView={{ opacity: 1, y: 0 }}
                        viewport={{ once: false, margin: "-100px" }}
                        transition={{ duration: 0.6, delay: 0.4 }}
                        className="mt-16 grid grid-cols-2 md:grid-cols-4 gap-6"
                    >
                        <div className="text-center">
                            <div className="text-3xl font-bold text-purple-600 dark:text-purple-400 mb-2">99.9%</div>
                            <div className="text-sm text-gray-600 dark:text-gray-400">{t('home.highlights.stat1') || '系统可用性'}</div>
                        </div>
                        <div className="text-center">
                            <div className="text-3xl font-bold text-purple-600 dark:text-purple-400 mb-2">&lt;600ms</div>
                            <div className="text-sm text-gray-600 dark:text-gray-400">{t('home.highlights.stat2') || '端到端延迟'}</div>
                        </div>
                        <div className="text-center">
                            <div className="text-3xl font-bold text-purple-600 dark:text-purple-400 mb-2">1K+</div>
                            <div className="text-sm text-gray-600 dark:text-gray-400">{t('home.highlights.stat3') || '并发通话'}</div>
                        </div>
                        <div className="text-center">
                            <div className="text-3xl font-bold text-purple-600 dark:text-purple-400 mb-2">100%</div>
                            <div className="text-sm text-gray-600 dark:text-gray-400">{t('home.highlights.stat4') || '开源免费'}</div>
                        </div>
                    </motion.div>
                </div>
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
                        <motion.div 
                            initial={{ opacity: 0, y: 30 }}
                            whileInView={{ opacity: 1, y: 0 }}
                            viewport={{ once: false, margin: "-100px" }}
                            transition={{ duration: 0.6 }}
                            className="flex items-center gap-3 mb-12"
                        >
                            <div className="w-10 h-10 rounded-lg bg-indigo-500/20 border border-white/10 flex items-center justify-center shadow-sm">
                                <Zap className="w-5 h-5 text-indigo-300" />
                            </div>
                            <div>
                                <h3 className="text-2xl font-bold tracking-tight text-gray-900 dark:text-white">{t('home.coreFeatures')}</h3>
                                <p className="text-gray-600 dark:text-neutral-400">{t('home.coreFeaturesDesc')}</p>
                            </div>
                        </motion.div>
                        <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-7 items-stretch">
                            {coreFeatures.map((f: any, idx: number) => {
                                const Icon = iconMap[f.icon] || Zap
                                return (
                                    <motion.div 
                                        key={idx}
                                        initial={{ opacity: 0, y: 30 }}
                                        whileInView={{ opacity: 1, y: 0 }}
                                        viewport={{ once: false, margin: "-100px" }}
                                        transition={{ duration: 0.6, delay: idx * 0.1 }}
                                        className="relative group rounded-2xl p-[1px] bg-gradient-to-br from-indigo-500/30 via-purple-500/20 to-transparent h-full z-10"
                                    >
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
                                    </motion.div>
                                )
                            })}
                        </div>
                </div>
            </section>
            </div>
            <Footer />
        </div>
    )
}

export default Home