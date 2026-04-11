import { motion } from 'framer-motion'
import { Shield, ChevronRight } from 'lucide-react'
import { Link } from 'react-router-dom'

const Privacy = () => {
  const currentDate = new Date().toLocaleDateString('zh-CN', { year: 'numeric', month: 'long', day: 'numeric' })

  const sections = [
    { id: 'scope', title: '一、隐私政策的适用范围' },
    { id: 'collection', title: '二、用户信息收集的范围与方式' },
    { id: 'cookie', title: '三、对Cookie和同类技术的使用' },
    { id: 'storage', title: '四、用户信息的存储' },
    { id: 'usage', title: '五、用户信息的使用' },
    { id: 'sharing', title: '六、用户信息的共享、转移与披露' },
    { id: 'security', title: '七、用户信息的安全保护' },
    { id: 'management', title: '八、用户信息的管理' },
    { id: 'minors', title: '九、未成年人使用条款' },
    { id: 'changes', title: '十、隐私政策的变更' },
    { id: 'law', title: '十一、法律适用与争议解决' },
    { id: 'other', title: '十二、其他' },
  ]

  return (
    <div className="min-h-screen bg-gradient-to-b from-gray-50 to-white dark:from-gray-900 dark:to-gray-950">
      <div className="max-w-7xl mx-auto px-4 py-16 sm:px-6 lg:px-8">
        <motion.div
          initial={{ opacity: 0, y: 20 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ duration: 0.5 }}
        >
          {/* 标题 */}
          <div className="text-center mb-12">
            <div className="inline-flex items-center justify-center w-16 h-16 rounded-full bg-blue-100 dark:bg-blue-900/30 mb-4">
              <Shield className="w-8 h-8 text-blue-600 dark:text-blue-400" />
            </div>
            <h1 className="text-4xl font-bold text-gray-900 dark:text-white mb-4">
              隐私政策
            </h1>
            <p className="text-gray-600 dark:text-gray-400">
              发布日期：{currentDate}
            </p>
          </div>

          <div className="grid grid-cols-1 lg:grid-cols-4 gap-8">
            {/* 侧边栏目录 */}
            <div className="lg:col-span-1">
              <div className="sticky top-24 bg-white dark:bg-gray-800 rounded-xl shadow-sm border border-gray-200 dark:border-gray-700 p-6">
                <h3 className="text-sm font-semibold text-gray-900 dark:text-white mb-4 uppercase tracking-wider">
                  目录
                </h3>
                <nav className="space-y-2">
                  {sections.map((section) => (
                    <a
                      key={section.id}
                      href={`#${section.id}`}
                      className="flex items-center gap-2 text-sm text-gray-600 dark:text-gray-400 hover:text-blue-600 dark:hover:text-blue-400 transition-colors group"
                    >
                      <ChevronRight className="w-4 h-4 opacity-0 group-hover:opacity-100 transition-opacity" />
                      <span className="group-hover:translate-x-1 transition-transform">{section.title}</span>
                    </a>
                  ))}
                </nav>
              </div>
            </div>

            {/* 主要内容 */}
            <div className="lg:col-span-3">
              <div className="prose prose-gray dark:prose-invert max-w-none">
                <div className="bg-white dark:bg-gray-800 rounded-xl shadow-sm border border-gray-200 dark:border-gray-700 p-8 space-y-8">
                  
                  {/* 导言 */}
                  <section>
                    <h2 className="text-2xl font-semibold mb-4 text-center">导 言</h2>
                    <p className="text-gray-600 dark:text-gray-400 leading-relaxed indent-8">
                      成都解忧造物科技有限责任公司（以下简称"公司"或"我们"）深知用户信息保护对用户的重要性，并会尽我们最大努力保护用户信息安全。我们将依法采取安全保护措施，保护用户的信息及隐私安全，并将恪守以下原则：权责一致原则、目的明确原则、自主选择原则、合理必要原则、确保安全原则、主体参与原则、公开透明原则等。
                    </p>
                    <p className="text-gray-600 dark:text-gray-400 leading-relaxed indent-8 mt-4">
                      在用户使用我们的相关产品或服务时，我们将按本《隐私政策》收集、保存、使用、共享、转移、披露及保护用户的信息。我们希望通过本《隐私政策》向用户介绍我们对用户信息的处理方式。因此，我们提请用户：在浏览本网站相关信息或享受相关服务时，仔细阅读并充分理解本《隐私政策》，在确认充分理解并同意后方可使用相关产品和服务。
                      <span className="font-semibold underline">本《隐私政策》中与您权益（可能）存在重大关系的条款，我们已使用粗体下划线标识予以区别，请您重点查阅。</span>
                    </p>
                  </section>

                  {/* 定义 */}
                  <section>
                    <h2 className="text-2xl font-semibold mb-4 text-center">定 义</h2>
                    <div className="space-y-3">
                      <p className="text-gray-600 dark:text-gray-400">
                        <span className="font-semibold">1.1 产品或服务：</span>是指，用户向公司订购软件产品及配套服务，以及后续通过网页、电子邮件、公众号、短信、电话等其他可行的方式一次或多次向用户提供产品、推广及宣传服务。
                      </p>
                      <p className="text-gray-600 dark:text-gray-400">
                        <span className="font-semibold">1.2 关联方：</span>是指，因合作、投资或控制（或共同控制）关系而与公司产生任何联系的公司、组织或个人。
                      </p>
                      <p className="text-gray-600 dark:text-gray-400">
                        <span className="font-semibold">1.3 第三方：</span>是指，本《隐私政策》约定的关联方范围以外的其他公司、组织或个人。
                      </p>
                    </div>
                  </section>

                  {/* 一、隐私政策的适用范围 */}
                  <section id="scope" className="scroll-mt-24">
                    <h2 className="text-2xl font-semibold mb-4">一、隐私政策的适用范围</h2>
                    <p className="text-gray-600 dark:text-gray-400 leading-relaxed">
                      《隐私政策》适用于由公司及其关联方销售产品及提供服务过程中对用户的信息的收集、存储、分析、传输、使用、处理、转移、备份、删除及保护等活动。如信息无法单独或结合其他信息识别到用户的真实身份或反映特定用户活动情况的，其不属于法律意义上的个人信息；当用户的信息可以单独或结合其他信息识别到用户身份时或我们将无法与任何特定信息建立联系的数据与其他用户的信息结合使用时，这些信息在结合使用期间，将作为用户的信息按照本《隐私政策》处理与保护。
                    </p>
                  </section>

                  {/* 二、用户信息收集的范围与方式 */}
                  <section id="collection" className="scroll-mt-24">
                    <h2 className="text-2xl font-semibold mb-4">二、用户信息收集的范围与方式</h2>
                    <p className="text-gray-600 dark:text-gray-400 leading-relaxed mb-4">
                      当您在我们的平台注册并享受相关服务时，您理解并同意，我们将可能收集、储存和使用下列与用户信息有关的数据：
                    </p>
                    <ul className="list-disc pl-8 space-y-2 text-gray-600 dark:text-gray-400">
                      <li>用户填写和/或提供的信息，包括：姓名/名称、手机号码、电子邮箱、地址等能够单独或者与其他信息结合识别用户身份的信息。</li>
                      <li>设备信息：设备名称、操作系统、浏览器及版本、设备类型、唯一设备标识符等软硬件特征信息。</li>
                      <li>日志信息：IP地址、浏览信息、访问记录、点击信息、使用语言、访问服务的日期和时间、Cookie、Web Beacon等。</li>
                      <li>交易及支付信息：交易商品/服务信息、订单号、下单时间、交易金额、支付方式、第三方支付账号等信息。</li>
                    </ul>
                  </section>

                  {/* 三、对Cookie和同类技术的使用 */}
                  <section id="cookie" className="scroll-mt-24">
                    <h2 className="text-2xl font-semibold mb-4">三、对Cookie和同类技术的使用</h2>
                    <p className="text-gray-600 dark:text-gray-400 leading-relaxed mb-4">
                      为使我们更好地向用户提供服务，我们可能会使用相关技术来收集和存储用户信息（包括透过Cookie不时收集的用户信息），在此过程中可能会向用户的计算机或移动设备上发送一个或多个Cookie或匿名标识符。
                    </p>
                    <p className="text-gray-600 dark:text-gray-400 leading-relaxed">
                      您可以根据自己的偏好留存或删除Cookie，也可以清除您网页中保存的所有Cookie。如果您的浏览器启用了 Do Not Track，或选择不使用Cookie，那么我们会尊重您的选择，但您可能无法登录或使用依赖于Cookie的服务或功能。
                    </p>
                  </section>

                  {/* 四、用户信息的存储 */}
                  <section id="storage" className="scroll-mt-24">
                    <h2 className="text-2xl font-semibold mb-4">四、用户信息的存储</h2>
                    <p className="text-gray-600 dark:text-gray-400 leading-relaxed mb-4">
                      在用户未提出修改、删除请求的情况下，我们会采取符合业界标准、合理可行的安全防护措施有效地保存用户信息。我们会在达成本政策所述目的所需的期限内保留用户信息，但法律法规另有规定的除外。
                    </p>
                    <p className="text-gray-600 dark:text-gray-400 leading-relaxed">
                      在超出保留期间后，我们会根据适用法律的要求删除或匿名化处理您的个人用户信息。
                    </p>
                  </section>

                  {/* 五、用户信息的使用 */}
                  <section id="usage" className="scroll-mt-24">
                    <h2 className="text-2xl font-semibold mb-4">五、用户信息的使用</h2>
                    <p className="text-gray-600 dark:text-gray-400 leading-relaxed mb-4">
                      为实现提供相关服务之目的，我们会根据《隐私政策》及授权协议的约定对所收集的用户信息进行使用。具体而言，我们将为以下目的使用所收集的用户信息：
                    </p>
                    <ul className="list-disc pl-8 space-y-2 text-gray-600 dark:text-gray-400">
                      <li>为了向您提供服务，我们会向您发送信息、通知或与您进行业务沟通。</li>
                      <li>我们可能以用户信息统计数据为基础，设计、开发、推广全新的产品及服务。</li>
                      <li>为提高您使用服务的安全性，确保操作环境安全与识别账号异常状态，保护您或其他用户或公众的人身财产安全免遭侵害。</li>
                    </ul>
                  </section>

                  {/* 六、用户信息的共享、转移与披露 */}
                  <section id="sharing" className="scroll-mt-24">
                    <h2 className="text-2xl font-semibold mb-4">六、用户信息的共享、转移与披露</h2>
                    <p className="text-gray-600 dark:text-gray-400 leading-relaxed mb-4">
                      我们不会向除公司的关联方外的任何第三方，提供、共享或转移用户信息，但下列情形除外：
                    </p>
                    <ul className="list-disc pl-8 space-y-2 text-gray-600 dark:text-gray-400">
                      <li>事先获得用户的明确授权或同意；</li>
                      <li>用户自行向第三方共享的；</li>
                      <li>基于与国家安全、国防安全、公共安全、公共卫生、公共利益直接相关的；</li>
                      <li>根据相关法律法规强制性要求所必需的情况下进行披露或提供；</li>
                      <li>在法律法规允许的范围内，为维护其他用户、公司及其关联方的生命、财产等合法权益所必需的。</li>
                    </ul>
                  </section>

                  {/* 七、用户信息的安全保护 */}
                  <section id="security" className="scroll-mt-24">
                    <h2 className="text-2xl font-semibold mb-4">七、用户信息的安全保护</h2>
                    <p className="text-gray-600 dark:text-gray-400 leading-relaxed mb-4">
                      我们非常重视用户信息的安全，将努力采取各种符合业界标准的合理安全措施（包括技术方面和管理方面）来保护用户信息，防止用户提供的用户信息被不当使用或被未经授权的访问、公开披露、使用、修改、损坏、丢失或泄漏。
                    </p>
                    <ul className="list-disc pl-8 space-y-2 text-gray-600 dark:text-gray-400">
                      <li>我们会使用加密技术、匿名化处理等合理可行的手段保护用户信息。</li>
                      <li>我们会建立专门的信息安全工作小组、安全管理制度、数据安全流程保障用户信息安全。</li>
                      <li>我们会制定应急处理预案，并在发生用户信息安全事件时立即启动应急预案。</li>
                    </ul>
                  </section>

                  {/* 八、用户信息的管理 */}
                  <section id="management" className="scroll-mt-24">
                    <h2 className="text-2xl font-semibold mb-4">八、用户信息的管理</h2>
                    <p className="text-gray-600 dark:text-gray-400 leading-relaxed mb-4">
                      我们非常重视用户对信息的管理，并尽全力保护用户对于用户信息的查阅、修改、补充、删除、改变授权范围以及撤回授权的权利。
                    </p>
                    <p className="text-gray-600 dark:text-gray-400 leading-relaxed">
                      用户如需对用户信息进行查阅、修改、补充、删除、改变授权范围以及撤回授权的，可通过我们提供的联系方式与我们取得联系，向我们提出有关申请，我们将在核验用户身份后根据用户的需求为用户提供相应服务。
                    </p>
                  </section>

                  {/* 九、未成年人使用条款 */}
                  <section id="minors" className="scroll-mt-24">
                    <h2 className="text-2xl font-semibold mb-4">九、未成年人使用条款</h2>
                    <p className="text-gray-600 dark:text-gray-400 leading-relaxed">
                      <span className="font-semibold underline">
                        若用户是未满18周岁的未成年人，在使用相关产品或服务前，应在法定监护人监护、指导下共同阅读本《隐私政策》，并在征得监护人同意的前提下使用相关产品或服务。我们根据国家相关或适用法律法规的规定保护未成年人的个人信息，只会在法律允许、父母或其他监护人明确同意或保护儿童所必要的情况下收集、使用、存储、共享、转移或披露未成年人的个人信息。
                      </span>
                    </p>
                  </section>

                  {/* 十、隐私政策的变更 */}
                  <section id="changes" className="scroll-mt-24">
                    <h2 className="text-2xl font-semibold mb-4">十、隐私政策的变更</h2>
                    <p className="text-gray-600 dark:text-gray-400 leading-relaxed mb-4">
                      为了给用户提供更好的服务，我们的相关服务将不时更新与变化，我们会适时对《隐私政策》进行修订，该等修订构成《隐私政策》的一部分并具有等同于《隐私政策》的效力。
                    </p>
                    <p className="text-gray-600 dark:text-gray-400 leading-relaxed">
                      本《隐私政策》更新后，我们会通过公告或其他适当的方式，说明隐私政策的具体变更内容，以便用户及时了解本《隐私政策》的最新版本。
                      <span className="font-semibold underline">若您不同意该修改部分，您应立即与我们联系并停止使用本服务。若您继续使用公司提供的服务，视为您同意并接受该修改部分。</span>
                    </p>
                  </section>

                  {/* 十一、法律适用与争议解决 */}
                  <section id="law" className="scroll-mt-24">
                    <h2 className="text-2xl font-semibold mb-4">十一、法律适用与争议解决</h2>
                    <p className="text-gray-600 dark:text-gray-400 leading-relaxed mb-4">
                      本协议的成立、生效、履行和解释，均适用中华人民共和国法律；法律无明文规定的，可适用通行的行业惯例。
                    </p>
                    <p className="text-gray-600 dark:text-gray-400 leading-relaxed">
                      双方在履行本协议的过程中，如发生争议，应协商解决，包括向数据保护机构投诉。协商不成的，双方均可向中华人民共和国法院起诉。
                    </p>
                  </section>

                  {/* 十二、其他 */}
                  <section id="other" className="scroll-mt-24">
                    <h2 className="text-2xl font-semibold mb-4">十二、其他</h2>
                    <p className="text-gray-600 dark:text-gray-400 leading-relaxed mb-4">
                      如果用户对本《隐私政策》或其中有关用户信息的收集、保存、使用、共享、披露、保护等功能存在意见或建议时，用户可以通过我们的反馈渠道反馈意见或投诉。我们会在收到用户的意见及建议后尽快向用户反馈。
                    </p>
                    <p className="text-gray-600 dark:text-gray-400 leading-relaxed">
                      本《隐私政策》的版权为公司所有，在法律允许的范围内，公司保留最终解释和修改的权利。
                    </p>
                  </section>

                  {/* 联系我们 */}
                  <section className="bg-blue-50 dark:bg-blue-900/20 rounded-lg p-6 mt-8">
                    <h3 className="text-lg font-semibold text-gray-900 dark:text-white mb-3">联系我们</h3>
                    <p className="text-gray-600 dark:text-gray-400 mb-2">
                      如您对本隐私政策有任何疑问、意见或建议，请通过以下方式与我们联系：
                    </p>
                    <p className="text-gray-600 dark:text-gray-400">
                      公司名称：成都解忧造物科技有限责任公司
                    </p>
                    <p className="text-gray-600 dark:text-gray-400">
                      联系方式：通过<a href="https://docs.lingecho.com/" target="_blank" rel="noreferrer" className="text-blue-600 dark:text-blue-400 hover:underline">文档站</a>获取
                    </p>
                  </section>
                </div>
              </div>
            </div>
          </div>
        </motion.div>
      </div>
    </div>
  )
}

export default Privacy
