import React from 'react'
import { useNavigate } from 'react-router-dom'
import { useI18nStore } from '@/stores/i18nStore'
import Card, { CardHeader, CardTitle, CardDescription, CardContent } from '@/components/UI/Card'
import Button from '@/components/UI/Button'
import { Sparkles, Zap, Mic, ArrowRight } from 'lucide-react'

const VoiceTrainingIndex: React.FC = () => {
    const { t } = useI18nStore()
    const navigate = useNavigate()

    return (
        <div className="min-h-screen bg-gradient-to-br from-slate-50 via-purple-50/30 to-blue-50/20 dark:from-neutral-900 dark:via-neutral-800 dark:to-neutral-900">
            {/* 背景装饰 */}
            <div className="absolute inset-0 overflow-hidden pointer-events-none">
                <div className="absolute -top-40 -right-40 w-80 h-80 bg-purple-300/20 rounded-full blur-3xl"></div>
                <div className="absolute -bottom-40 -left-40 w-80 h-80 bg-blue-300/20 rounded-full blur-3xl"></div>
            </div>
            
            <div className="container mx-auto px-2 py-8 relative z-10">
                {/* 页面头部 */}
                <div className="text-center mb-8">
                    <div className="inline-flex items-center justify-center w-20 h-20 bg-gradient-to-br from-purple-500 to-blue-600 rounded-2xl mb-6 shadow-lg">
                        <Mic className="w-10 h-10" />
                    </div>
                    <h1 className="text-3xl font-bold mb-2 bg-gradient-to-r bg-clip-text">
                        {t('voiceTraining.index.title')}
                    </h1>
                    <p className="text-xl text-gray-600 dark:text-gray-400 max-w-2xl mx-auto">
                        {t('voiceTraining.index.subtitle')}
                    </p>
                </div>

                <div className="grid grid-cols-1 md:grid-cols-2 gap-8 max-w-4xl mx-auto">
                    {/* 讯飞星火 */}
                    <Card
                        variant="elevated"
                        padding="lg"
                        className="backdrop-blur-sm bg-white/95 dark:bg-neutral-800/95 border-2 border-transparent hover:border-blue-300 dark:hover:border-blue-600 shadow-xl hover:shadow-2xl transition-all duration-300 group cursor-pointer relative overflow-hidden"
                        onClick={() => navigate('/voice-training/xunfei')}
                    >
                        {/* 背景渐变 */}
                        <div className="absolute inset-0 bg-gradient-to-br from-blue-50/50 to-purple-50/50 dark:from-blue-900/10 dark:to-purple-900/10 opacity-0 group-hover:opacity-100 transition-opacity duration-300"></div>
                        
                        <CardHeader className="relative z-10">
                            <div className="flex flex-col items-center text-center mb-6">
                                <div className="w-20 h-20 bg-gradient-to-br from-blue-500 via-purple-500 to-pink-500 rounded-2xl flex items-center justify-center mb-4 shadow-lg group-hover:scale-110 transition-transform duration-300">
                                    <Sparkles className="w-10 h-10" />
                                </div>
                                <CardTitle className="text-3xl font-bold mb-2">{t('voiceTraining.index.xunfei.title')}</CardTitle>
                                <CardDescription className="text-base">
                                    {t('voiceTraining.index.xunfei.subtitle')}
                                </CardDescription>
                            </div>
                        </CardHeader>
                        <CardContent className="relative z-10">
                            <ul className="space-y-3 text-sm text-gray-700 dark:text-gray-300 mb-8">
                                <li className="flex items-start gap-3">
                                    <div className="w-2 h-2 bg-blue-500 rounded-full mt-2 flex-shrink-0"></div>
                                    <span>{t('voiceTraining.index.xunfei.feature1')}</span>
                                </li>
                                <li className="flex items-start gap-3">
                                    <div className="w-2 h-2 bg-purple-500 rounded-full mt-2 flex-shrink-0"></div>
                                    <span>{t('voiceTraining.index.xunfei.feature2')}</span>
                                </li>
                                <li className="flex items-start gap-3">
                                    <div className="w-2 h-2 bg-pink-500 rounded-full mt-2 flex-shrink-0"></div>
                                    <span>{t('voiceTraining.index.xunfei.feature3')}</span>
                                </li>
                            </ul>
                            <Button
                                variant="primary"
                                size="lg"
                                fullWidth
                                className="group/btn"
                                onClick={(e) => {
                                    e.stopPropagation()
                                    navigate('/voice-training/xunfei')
                                }}
                            >
                                <span className="flex items-center justify-center gap-2">
                                    {t('voiceTraining.index.xunfei.button')}
                                    <ArrowRight className="w-4 h-4 group-hover/btn:translate-x-1 transition-transform" />
                                </span>
                            </Button>
                        </CardContent>
                    </Card>

                    {/* 火山引擎 */}
                    <Card
                        variant="elevated"
                        padding="lg"
                        className="backdrop-blur-sm bg-white/95 dark:bg-neutral-800/95 border-2 border-transparent hover:border-orange-300 dark:hover:border-orange-600 shadow-xl hover:shadow-2xl transition-all duration-300 group cursor-pointer relative overflow-hidden"
                        onClick={() => navigate('/voice-training/volcengine')}
                    >
                        {/* 背景渐变 */}
                        <div className="absolute inset-0 bg-gradient-to-br from-orange-50/50 to-red-50/50 dark:from-orange-900/10 dark:to-red-900/10 opacity-0 group-hover:opacity-100 transition-opacity duration-300"></div>
                        
                        <CardHeader className="relative z-10">
                            <div className="flex flex-col items-center text-center mb-6">
                                <div className="w-20 h-20 bg-gradient-to-br from-orange-500 via-red-500 to-yellow-500 rounded-2xl flex items-center justify-center mb-4 shadow-lg group-hover:scale-110 transition-transform duration-300">
                                    <Zap className="w-10 h-10" />
                                </div>
                                <CardTitle className="text-3xl font-bold mb-2">{t('voiceTraining.index.volcengine.title')}</CardTitle>
                                <CardDescription className="text-base">
                                    {t('voiceTraining.index.volcengine.subtitle')}
                                </CardDescription>
                            </div>
                        </CardHeader>
                        <CardContent className="relative z-10">
                            <ul className="space-y-3 text-sm text-gray-700 dark:text-gray-300 mb-8">
                                <li className="flex items-start gap-3">
                                    <div className="w-2 h-2 bg-orange-500 rounded-full mt-2 flex-shrink-0"></div>
                                    <span>{t('voiceTraining.index.volcengine.feature1')}</span>
                                </li>
                                <li className="flex items-start gap-3">
                                    <div className="w-2 h-2 bg-red-500 rounded-full mt-2 flex-shrink-0"></div>
                                    <span>{t('voiceTraining.index.volcengine.feature2')}</span>
                                </li>
                                <li className="flex items-start gap-3">
                                    <div className="w-2 h-2 bg-yellow-500 rounded-full mt-2 flex-shrink-0"></div>
                                    <span>{t('voiceTraining.index.volcengine.feature3')}</span>
                                </li>
                            </ul>
                            <Button
                                variant="primary"
                                size="lg"
                                fullWidth
                                className="group/btn"
                                onClick={(e) => {
                                    e.stopPropagation()
                                    navigate('/voice-training/volcengine')
                                }}
                            >
                                <span className="flex items-center justify-center gap-2">
                                    {t('voiceTraining.index.volcengine.button')}
                                    <ArrowRight className="w-4 h-4 group-hover/btn:translate-x-1 transition-transform" />
                                </span>
                            </Button>
                        </CardContent>
                    </Card>
                </div>
            </div>
        </div>
    )
}

export default VoiceTrainingIndex

