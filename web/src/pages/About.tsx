import { motion } from 'framer-motion'
import { 
  Target, 
  Eye, 
  Users, 
  Award, 
  CheckCircle, 
  Heart,
  Rocket,
  Building2,
  Zap,
  Building,
  Link,
  FlaskConical,
  PartyPopper
} from 'lucide-react'
import Card, { CardContent, CardDescription, CardHeader, CardTitle } from '../components/UI/Card'
import FadeIn from '../components/Animations/FadeIn'
import StaggeredList from '../components/Animations/StaggeredList'
import { useI18nStore } from '../stores/i18nStore'

const About = () => {
  const { t } = useI18nStore()
  const values = [
    {
      icon: <Target className="w-8 h-8 text-indigo-500" />,
      title: t('about.values.tech.title'),
      description: t('about.values.tech.desc'),
    },
    {
      icon: <Eye className="w-8 h-8 text-purple-500" />,
      title: t('about.values.ux.title'),
      description: t('about.values.ux.desc'),
    },
    {
      icon: <Users className="w-8 h-8 text-pink-500" />,
      title: t('about.values.features.title'),
      description: t('about.values.features.desc'),
    },
    {
      icon: <Award className="w-8 h-8 text-blue-500" />,
      title: t('about.values.openSource.title'),
      description: t('about.values.openSource.desc'),
    },
  ]

  const milestones = [
    {
      day: 'Day 1',
      title: t('about.timeline.day1.title'),
      description: t('about.timeline.day1.desc'),
      icon: <Rocket className="w-6 h-6" />,
      color: 'from-blue-500 to-cyan-500'
    },
    {
      day: 'Day 2',
      title: t('about.timeline.day2.title'),
      description: t('about.timeline.day2.desc'),
      icon: <Building2 className="w-6 h-6" />,
      color: 'from-purple-500 to-pink-500'
    },
    {
      day: 'Day 3',
      title: t('about.timeline.day3.title'),
      description: t('about.timeline.day3.desc'),
      icon: <Zap className="w-6 h-6" />,
      color: 'from-green-500 to-emerald-500'
    },
    {
      day: 'Day 4',
      title: t('about.timeline.day4.title'),
      description: t('about.timeline.day4.desc'),
      icon: <Building className="w-6 h-6" />,
      color: 'from-orange-500 to-red-500'
    },
    {
      day: 'Day 5',
      title: t('about.timeline.day5.title'),
      description: t('about.timeline.day5.desc'),
      icon: <Link className="w-6 h-6" />,
      color: 'from-indigo-500 to-purple-500'
    },
    {
      day: 'Day 6',
      title: t('about.timeline.day6.title'),
      description: t('about.timeline.day6.desc'),
      icon: <FlaskConical className="w-6 h-6" />,
      color: 'from-pink-500 to-rose-500'
    },
    {
      day: 'Day 7',
      title: t('about.timeline.day7.title'),
      description: t('about.timeline.day7.desc'),
      icon: <PartyPopper className="w-6 h-6" />,
      color: 'from-yellow-500 to-orange-500'
    },
  ]

  return (
      <div className="space-y-20">
        {/* Hero Section */}
        <section className="py-20 text-center">
          <FadeIn direction="up">
            <h1 className="text-5xl md:text-6xl font-display font-bold mb-6 text-foreground">
              {t('about.title')}
            </h1>
            <p className="text-xl text-muted-foreground max-w-3xl mx-auto leading-relaxed">
              {t('about.subtitle')}
            </p>
          </FadeIn>
        </section>

        {/* Mission Section */}
        <section className="py-20 bg-muted">
          <div className="max-w-6xl mx-auto px-4">
            <div className="grid grid-cols-1 lg:grid-cols-2 gap-12 items-center">
              <FadeIn direction="left">
                <div>
                  <h2 className="text-4xl font-display font-bold mb-6 text-foreground">
                    {t('about.mission.title')}
                  </h2>
                  <p className="text-lg text-muted-foreground mb-6 leading-relaxed">
                    {t('about.mission.desc')}
                  </p>
                  <div className="space-y-4">
                    <div className="flex items-start space-x-3">
                      <CheckCircle className="w-6 h-6 text-primary mt-1 flex-shrink-0" />
                      <p className="text-foreground">
                        {t('about.mission.item1')}
                      </p>
                    </div>
                    <div className="flex items-start space-x-3">
                      <CheckCircle className="w-6 h-6 text-primary mt-1 flex-shrink-0" />
                      <p className="text-foreground">
                        {t('about.mission.item2')}
                      </p>
                    </div>
                    <div className="flex items-start space-x-3">
                      <CheckCircle className="w-6 h-6 text-primary mt-1 flex-shrink-0" />
                      <p className="text-foreground">
                        {t('about.mission.item3')}
                      </p>
                    </div>
                  </div>
                </div>
              </FadeIn>

              <FadeIn direction="right">
                <div className="relative">
                  <div className="absolute inset-0 bg-gradient-to-r from-primary/80 via-secondary/80 to-primary/80 rounded-3xl transform rotate-3"></div>
                  <div className="relative bg-background rounded-3xl p-8 shadow-2xl border">
                    <div className="text-center">
                      <div className="w-20 h-20 bg-accent from-primary via-secondary to-primary rounded-full flex items-center justify-center mx-auto mb-6">
                        <Heart className="w-10 h-10 text-foreground " />
                      </div>
                      <h3 className="text-2xl font-bold mb-4 text-foreground">{t('about.madeWithHeart')}</h3>
                      <p className="text-muted-foreground">
                        {t('about.madeWithHeartDesc')}
                      </p>
                    </div>
                  </div>
                </div>
              </FadeIn>
            </div>
          </div>
        </section>

        {/* Values Section */}
        <section className="py-20">
          <div className="max-w-6xl mx-auto px-4">
            <FadeIn direction="up" className="text-center mb-16">
              <h2 className="text-4xl font-display font-bold mb-6 text-foreground">
                {t('about.values.title')}
              </h2>
              <p className="text-xl text-muted-foreground max-w-3xl mx-auto">
                {t('about.values.desc')}
              </p>
            </FadeIn>

            <StaggeredList className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-8">
              {values.map((value) => (
                  <motion.div
                      key={value.title}
                      whileHover={{ y: -5 }}
                      transition={{ duration: 0.2 }}
                  >
                    <Card hover className="h-full text-center border shadow-lg hover:shadow-xl transition-all duration-300">
                      <CardHeader>
                        <div className="w-16 h-16 bg-muted rounded-2xl flex items-center justify-center mx-auto mb-4">
                          {value.icon}
                        </div>
                        <CardTitle className="text-xl text-foreground">{value.title}</CardTitle>
                      </CardHeader>
                      <CardContent>
                        <CardDescription className="text-base leading-relaxed text-muted-foreground">
                          {value.description}
                        </CardDescription>
                      </CardContent>
                    </Card>
                  </motion.div>
              ))}
            </StaggeredList>
          </div>
        </section>

        {/* Timeline Section */}
        <section className="py-20 bg-muted">
          <div className="max-w-6xl mx-auto px-4">
            <FadeIn direction="up" className="text-center mb-16">
              <h2 className="text-4xl font-display font-bold mb-6 text-foreground">
                {t('about.timeline.title')}
              </h2>
              <p className="text-xl text-muted-foreground">
                {t('about.timeline.desc')}
              </p>
            </FadeIn>

            <div className="relative">
              {/* 时间线 */}
              <div className="absolute left-8 top-0 bottom-0 w-1 bg-accent rounded-full"></div>

              <StaggeredList className="space-y-8">
                {milestones.map((milestone, index) => (
                    <motion.div
                        key={milestone.day}
                        initial={{ opacity: 0, x: -50 }}
                        whileInView={{ opacity: 1, x: 0 }}
                        whileHover={{ x: 15, scale: 1.02 }}
                        transition={{ duration: 0.6, delay: index * 0.1 }}
                        className="relative flex items-start space-x-6 group"
                    >
                      {/* 时间节点 */}
                      <div className="flex-shrink-0 relative z-20">
                        <motion.div
                            whileHover={{ scale: 1.2, rotate: 360 }}
                            transition={{ duration: 0.6 }}
                            className={`w-16 h-16 bg-accent ${milestone.color} rounded-full flex items-center justify-center font-bold text-lg shadow-2xl relative overflow-hidden`}
                        >
                          <div className="absolute inset-0 bg-accent from-white/20 to-transparent rounded-full"></div>
                          <span className="relative z-10">{milestone.icon}</span>
                        </motion.div>
                      </div>

                      {/* 内容卡片 */}
                      <motion.div
                          whileHover={{ y: -5 }}
                          transition={{ duration: 0.3 }}
                          className="flex-1 relative"
                      >
                        <Card className="relative border shadow-xl hover:shadow-2xl transition-all duration-300">
                          <CardHeader>
                            <div className="flex items-center justify-between">
                              <CardTitle className="text-xl text-foreground">{milestone.title}</CardTitle>
                              <span className={`px-3 py-1 rounded-full text-sm font-bold bg-accent ${milestone.color} text-foreground `}>
                            {milestone.day}
                          </span>
                            </div>
                          </CardHeader>
                          <CardContent>
                            <p className="text-muted-foreground leading-relaxed">{milestone.description}</p>
                          </CardContent>
                        </Card>
                      </motion.div>
                    </motion.div>
                ))}
              </StaggeredList>
            </div>
          </div>
        </section>

        {/* Features Section */}
        <section className="py-20">
          <div className="max-w-6xl mx-auto px-4">
            <FadeIn direction="up" className="text-center mb-16">
              <h2 className="text-4xl font-display font-bold mb-6 text-foreground">
                {t('about.features.title')}
              </h2>
              <p className="text-xl text-muted-foreground max-w-3xl mx-auto">
                {t('about.features.desc')}
              </p>
            </FadeIn>

            <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-8">
              <motion.div
                  initial={{ opacity: 0, y: 50 }}
                  whileInView={{ opacity: 1, y: 0 }}
                  whileHover={{ y: -12, scale: 1.05 }}
                  transition={{ duration: 0.4 }}
                  className="relative group"
              >
                <Card className="h-full border shadow-lg hover:shadow-xl transition-all duration-300">
                  <CardHeader>
                    <div className="w-16 h-16 bg-accent rounded-2xl flex items-center justify-center mb-6">
                      <Target className="w-8 h-8 text-foreground"/>
                    </div>
                    <CardTitle className="text-xl text-foreground">{t('about.features.tech.title')}</CardTitle>
                  </CardHeader>
                  <CardContent>
                    <CardDescription className="text-muted-foreground leading-relaxed">
                      {t('about.features.tech.desc')}
                    </CardDescription>
                  </CardContent>
                </Card>
              </motion.div>

              <motion.div
                  initial={{ opacity: 0, y: 50 }}
                  whileInView={{ opacity: 1, y: 0 }}
                  whileHover={{ y: -12, scale: 1.05 }}
                  transition={{ duration: 0.4, delay: 0.1 }}
                  className="relative group"
              >
                <Card className="h-full border shadow-lg hover:shadow-xl transition-all duration-300">
                  <CardHeader>
                    <div className="w-16 h-16 bg-secondary rounded-2xl flex items-center justify-center mb-6">
                      <Eye className="w-8 h-8 text-foreground " />
                    </div>
                    <CardTitle className="text-xl text-foreground">{t('about.features.ux.title')}</CardTitle>
                  </CardHeader>
                  <CardContent>
                    <CardDescription className="text-muted-foreground leading-relaxed">
                      {t('about.features.ux.desc')}
                    </CardDescription>
                  </CardContent>
                </Card>
              </motion.div>

              <motion.div
                  initial={{ opacity: 0, y: 50 }}
                  whileInView={{ opacity: 1, y: 0 }}
                  whileHover={{ y: -12, scale: 1.05 }}
                  transition={{ duration: 0.4, delay: 0.2 }}
                  className="relative group"
              >
                <Card className="h-full border shadow-lg hover:shadow-xl transition-all duration-300">
                  <CardHeader>
                    <div className="w-16 h-16 bg-accent rounded-2xl flex items-center justify-center mb-6">
                      <Users className="w-8 h-8 text-foreground " />
                    </div>
                    <CardTitle className="text-xl text-foreground">{t('about.features.complete.title')}</CardTitle>
                  </CardHeader>
                  <CardContent>
                    <CardDescription className="text-muted-foreground leading-relaxed">
                      {t('about.features.complete.desc')}
                    </CardDescription>
                  </CardContent>
                </Card>
              </motion.div>
            </div>
          </div>
        </section>

        {/* Stats Section */}
        <section className="py-20 bg-muted">
          <div className="max-w-6xl mx-auto px-4">
            <FadeIn direction="up" className="text-center mb-16">
              <h2 className="text-4xl font-display font-bold mb-6 text-foreground">
                {t('about.stats.title')}
              </h2>
              <p className="text-xl text-muted-foreground max-w-3xl mx-auto">
                {t('about.stats.desc')}
              </p>
            </FadeIn>

            <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-8">
              <motion.div
                  initial={{ opacity: 0, y: 30, scale: 0.8 }}
                  whileInView={{ opacity: 1, y: 0, scale: 1 }}
                  whileHover={{ scale: 1.1 }}
                  transition={{ duration: 0.6 }}
                  className="text-center group"
              >
                <Card className="border shadow-lg hover:shadow-xl transition-all duration-300">
                  <CardContent className="pt-6">
                    <div className="w-20 h-20 bg-secondary rounded-2xl flex items-center justify-center mx-auto mb-4">
                      <span className="text-2xl font-bold text-foreground">50+</span>
                    </div>
                    <h3 className="text-2xl font-bold mb-2 text-foreground">{t('about.stats.apis')}</h3>
                    <p className="text-muted-foreground">{t('about.stats.apisDesc')}</p>
                  </CardContent>
                </Card>
              </motion.div>

              <motion.div
                  initial={{ opacity: 0, y: 30, scale: 0.8 }}
                  whileInView={{ opacity: 1, y: 0, scale: 1 }}
                  whileHover={{ scale: 1.1 }}
                  transition={{ duration: 0.6, delay: 0.1 }}
                  className="text-center group"
              >
                <Card className="border shadow-lg hover:shadow-xl transition-all duration-300">
                  <CardContent className="pt-6">
                    <div className="w-20 h-20 bg-secondary rounded-2xl flex items-center justify-center mx-auto mb-4">
                      <span className="text-2xl font-bold text-foreground">24/7</span>
                    </div>
                    <h3 className="text-2xl font-bold mb-2 text-foreground">{t('about.stats.running')}</h3>
                    <p className="text-muted-foreground">{t('about.stats.runningDesc')}</p>
                  </CardContent>
                </Card>
              </motion.div>

              <motion.div
                  initial={{ opacity: 0, y: 30, scale: 0.8 }}
                  whileInView={{ opacity: 1, y: 0, scale: 1 }}
                  whileHover={{ scale: 1.1 }}
                  transition={{ duration: 0.6, delay: 0.2 }}
                  className="text-center group"
              >
                <Card className="border shadow-lg hover:shadow-xl transition-all duration-300">
                  <CardContent className="pt-6">
                    <div className="w-20 h-20 bg-accent rounded-2xl flex items-center justify-center mx-auto mb-4">
                      <span className="text-2xl font-bold text-foreground">100%</span>
                    </div>
                    <h3 className="text-2xl font-bold mb-2 text-foreground">{t('about.stats.openSource')}</h3>
                    <p className="text-muted-foreground">{t('about.stats.openSourceDesc')}</p>
                  </CardContent>
                </Card>
              </motion.div>

              <motion.div
                  initial={{opacity: 0, y: 30, scale: 0.8}}
                  whileInView={{ opacity: 1, y: 0, scale: 1 }}
                  whileHover={{ scale: 1.1 }}
                  transition={{ duration: 0.6, delay: 0.3 }}
                  className="text-center group"
              >
                <Card className="border shadow-lg hover:shadow-xl transition-all duration-300">
                  <CardContent className="pt-6">
                    <div className="w-20 h-20 bg-accent rounded-2xl flex items-center justify-center mx-auto mb-4">
                      <span className="text-2xl font-bold text-foreground">∞</span>
                    </div>
                    <h3 className="text-2xl font-bold mb-2 text-foreground">{t('about.stats.infinite')}</h3>
                    <p className="text-muted-foreground">{t('about.stats.infiniteDesc')}</p>
                  </CardContent>
                </Card>
              </motion.div>
            </div>
          </div>
        </section>
      </div>
  )
}

export default About
