import { motion } from 'framer-motion'
import { FileText, ChevronRight } from 'lucide-react'
import { Link } from 'react-router-dom'

const Terms = () => {
  const currentDate = new Date().toLocaleDateString('zh-CN', { year: 'numeric', month: 'long', day: 'numeric' })

  const sections = [
    { id: 'intro', title: '一、提示条款' },
    { id: 'definitions', title: '二、定义' },
    { id: 'scope', title: '三、协议范围' },
    { id: 'account', title: '四、账户注册与使用' },
    { id: 'services', title: '五、产品及服务及规范' },
    { id: 'payment', title: '六、付款说明' },
    { id: 'privacy', title: '七、用户信息的保护及授权' },
    { id: 'violation', title: '八、用户的违约及处理' },
    { id: 'changes', title: '九、协议的变更' },
    { id: 'notice', title: '十、通知' },
    { id: 'termination', title: '十一、协议的终止' },
    { id: 'law', title: '十二、法律适用、管辖与其他' },
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
            <div className="inline-flex items-center justify-center w-16 h-16 rounded-full bg-purple-100 dark:bg-purple-900/30 mb-4">
              <FileText className="w-8 h-8 text-purple-600 dark:text-purple-400" />
            </div>
            <h1 className="text-4xl font-bold text-gray-900 dark:text-white mb-2">
              成都解忧造物科技有限责任公司
            </h1>
            <h2 className="text-3xl font-bold text-gray-900 dark:text-white mb-4">
              用户注册协议
            </h2>
            <p className="text-gray-600 dark:text-gray-400">
              本协议生效日期：{currentDate}
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
                      className="flex items-center gap-2 text-sm text-gray-600 dark:text-gray-400 hover:text-purple-600 dark:hover:text-purple-400 transition-colors group"
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
                  
                  {/* 一、提示条款 */}
                  <section id="intro" className="scroll-mt-24">
                    <h2 className="text-2xl font-semibold mb-4">一、提示条款</h2>
                    <div className="space-y-4 text-gray-600 dark:text-gray-400">
                      <p>1.1 欢迎您与成都解忧造物科技有限责任公司共同签署本《用户注册协议》（下称"本协议"）并享受公司（定义见下文）提供的相关服务（下称"本服务"）。</p>
                      <p>1.2 各服务条款前所列索引关键词仅为帮助您理解该条款表达的主旨之用，本身并无实际涵义，不影响或限制本协议条款的含义或解释。为维护您自身权益，建议您仔细阅读各条款具体表述。</p>
                      <p>
                        1.3 您在申请注册流程中点击同意本协议之前，应当认真阅读本协议。
                        <span className="font-semibold underline">请您务必审慎阅读、充分理解各条款内容，特别是免除或者限制公司责任的条款、法律适用和争议解决条款。免除或者限制责任的条款将以粗体下划线标识提示您重点注意，您应重点阅读。</span>
                      </p>
                      <p>
                        1.4 除非您已充分阅读、完全理解并接受本协议所有条款，否则您无权使用本服务。当您按照注册页面提示填写信息、点击"同意"、"接受"本协议、完成全部注册程序或您的下载、安装、使用、授权使用、登录等行为时，即视为您已充分阅读、理解并接受本协议的全部内容。
                        <span className="font-semibold underline">阅读本协议的过程中，如果您不同意本协议或其中任何条款约定，您应立即停止注册程序或停止使用本服务。</span>
                      </p>
                    </div>
                  </section>

                  {/* 二、定义 */}
                  <section id="definitions" className="scroll-mt-24">
                    <h2 className="text-2xl font-semibold mb-4">二、定义</h2>
                    <div className="space-y-3 text-gray-600 dark:text-gray-400">
                      <p>2.1 <span className="font-semibold">公司：</span>指成都解忧造物科技有限责任公司，本文简称为"公司"或"我们"，为平台经营者。</p>
                      <p>2.2 <span className="font-semibold">关联方：</span>指成都解忧造物科技有限责任公司及其母公司、子公司、分公司及因合作、投资或控制（或共同控制）关系而与公司产生任何联系的公司、组织或个人。</p>
                      <p>2.3 <span className="font-semibold">平台规则：</span>包括在平台内已经发布或将来可能发布/更新的全部规则、细则、说明、流程、解读、公告、通知等内容。</p>
                      <p>2.4 <span className="font-semibold">用户：</span>指平台的使用人，在本协议中更多地称为"您"。</p>
                      <p>2.5 <span className="font-semibold">账户：</span>指用户在使用服务时可能需要注册的账户。用户可在平台注册并获得的账户，作为登录并使用服务的凭证。</p>
                    </div>
                  </section>

                  {/* 三、协议范围 */}
                  <section id="scope" className="scroll-mt-24">
                    <h2 className="text-2xl font-semibold mb-4">三、协议范围</h2>
                    <div className="space-y-4 text-gray-600 dark:text-gray-400">
                      <p>3.1 本协议由您与公司共同缔结，对您与公司均具有法律效力。</p>
                      <p>
                        3.2 由于互联网高速发展，您与公司签署的本协议列明的条款并不能完整罗列并覆盖您与公司所有权利与义务，现有的约定也不能保证完全符合未来发展的需求。
                        <span className="font-semibold underline">您知晓并同意，公司可能会根据需要更新或调整软件、本服务和/或本协议相关内容。</span>
                      </p>
                      <p>
                        3.3 隐私政策、平台规则（及其不时修改、补充）均为本协议的补充协议，与本协议不可分割且具有同等法律效力。
                        <span className="font-semibold underline">如您使用或继续使用产品或服务，视为您已充分阅读并认可和同意上述补充协议（及其不时修改、补充），您应当同样遵守该等补充协议（及其不时修改、补充）。</span>
                      </p>
                    </div>
                  </section>

                  {/* 四、账户注册与使用 */}
                  <section id="account" className="scroll-mt-24">
                    <h2 className="text-2xl font-semibold mb-4">四、账户注册与使用</h2>
                    <div className="space-y-4 text-gray-600 dark:text-gray-400">
                      <h3 className="text-lg font-semibold text-gray-900 dark:text-white">4.1 用户资格</h3>
                      <p>您确认，您必须年满18岁才能注册账户，或您的年龄须足以使您在您所在的地方签订有约束力的合同。若您不具备前述与您行为相适应的民事行为能力，请在法定监护人的陪同、指导下阅读本协议，并在确保监护人同意本协议内容后使用服务。</p>
                      
                      <h3 className="text-lg font-semibold text-gray-900 dark:text-white mt-6">4.2 账户取得、使用、转让</h3>
                      <ul className="list-disc pl-6 space-y-2">
                        <li>当您按照注册页面提示填写信息、阅读并同意本协议且完成全部注册程序后，您可获得平台账户并成为平台用户。</li>
                        <li>平台只允许每位用户使用一个平台账户。如有证据证明您存在不当注册或不当使用多个平台账户的情形，平台可采取冻结或关闭账户、拒绝提供服务等措施。</li>
                        <li>
                          由于您的平台账户关联您的个人信息及商业信息，您的平台账户仅限您本人使用。未经公司事前同意，您直接或间接授权第三方使用您平台账户或获取您账户项下信息的行为无效，
                          <span className="font-semibold underline">且公司不就因此产生的任何损失、损害承担任何责任</span>。
                        </li>
                        <li>您在平台注册的账户未经公司许可不得对外转让，否则公司有权追究您的违约责任。</li>
                      </ul>

                      <h3 className="text-lg font-semibold text-gray-900 dark:text-white mt-6">4.3 账户安全规范</h3>
                      <ul className="list-disc pl-6 space-y-2">
                        <li>您的账户为您自行设置并由您保管，公司任何时候均不会主动要求您提供您的账户密码。</li>
                        <li>
                          <span className="font-semibold underline">账户因您主动泄露、保管不善或因您遭受他人攻击、诈骗等行为导致的损失及后果，公司并不承担任何责任</span>，您应通过司法、行政等救济途径向侵权行为人追偿。
                        </li>
                        <li>除公司存在过错外，您应对您账户项下的所有活动和行为结果负责。</li>
                      </ul>
                    </div>
                  </section>

                  {/* 五、产品及服务及规范 */}
                  <section id="services" className="scroll-mt-24">
                    <h2 className="text-2xl font-semibold mb-4">五、产品及服务及规范</h2>
                    <div className="space-y-4 text-gray-600 dark:text-gray-400">
                      <h3 className="text-lg font-semibold text-gray-900 dark:text-white">5.1 服务内容</h3>
                      <p>您有权在平台上享受软件产品的浏览、收藏、订购及相关配套服务。</p>
                      
                      <h3 className="text-lg font-semibold text-gray-900 dark:text-white mt-6">5.2 禁止性规定</h3>
                      <p>您应当遵守任何可适用的法律法规的规定，不得利用服务及账户实施包括但不限于以下行为：</p>
                      <ul className="list-disc pl-6 space-y-2">
                        <li>危害国家安全，泄露国家秘密，颠覆国家政权，破坏国家统一的；</li>
                        <li>散布谣言，扰乱社会秩序，破坏社会稳定的；</li>
                        <li>散布淫秽、色情、赌博、暴力、凶杀、恐怖或者教唆犯罪的；</li>
                        <li>侮辱、诽谤、恐吓、涉及他人隐私等侵害他人合法权益的；</li>
                        <li>虚构事实、隐瞒真相以误导、欺骗他人的；</li>
                        <li>其他违背公序良俗、相关协议及规则以及适用的法律法规的。</li>
                      </ul>
                    </div>
                  </section>

                  {/* 六、付款说明 */}
                  <section id="payment" className="scroll-mt-24">
                    <h2 className="text-2xl font-semibold mb-4">六、付款说明</h2>
                    <div className="space-y-4 text-gray-600 dark:text-gray-400">
                      <p>6.1 您使用本服务或其中某服务，即表示您同意支付其中的所有费用。您理解并同意，所有服务的价格可能会发生变化，均以您订购并付款时的价格为准。</p>
                      <p>
                        6.2 请注意，
                        <span className="font-semibold underline">除非因为公司方面的问题导致本服务无法正常提供，您支付的有关本服务的所有费用均不能退款。</span>
                      </p>
                    </div>
                  </section>

                  {/* 七、用户信息的保护及授权 */}
                  <section id="privacy" className="scroll-mt-24">
                    <h2 className="text-2xl font-semibold mb-4">七、用户信息的保护及授权</h2>
                    <div className="space-y-4 text-gray-600 dark:text-gray-400">
                      <p>公司非常重视用户信息的保护，在您使用服务时，您同意公司按照在平台上公布的<Link to="/privacy" className="text-blue-600 dark:text-blue-400 hover:underline">隐私政策</Link>收集、存储、使用、披露和保护您的信息。</p>
                      <p>您声明并保证，您对您在平台所发布的信息拥有相应、合法的权利，相关信息为您所有或您已获得必要的授权。</p>
                    </div>
                  </section>

                  {/* 八、用户的违约及处理 */}
                  <section id="violation" className="scroll-mt-24">
                    <h2 className="text-2xl font-semibold mb-4">八、用户的违约及处理</h2>
                    <div className="space-y-4 text-gray-600 dark:text-gray-400">
                      <p>发生如下情形之一的，视为您违约：</p>
                      <ul className="list-disc pl-6 space-y-2">
                        <li>使用平台服务时违反有关适用法律法规规定的；</li>
                        <li>违反本协议或补充协议（包括不时修改及补充）约定的。</li>
                      </ul>
                      <p className="mt-4">您在平台上发布的信息构成违约的，公司可根据相应规则立即对相应信息进行删除、屏蔽，并对您的账户进行封号处理。</p>
                    </div>
                  </section>

                  {/* 九、协议的变更 */}
                  <section id="changes" className="scroll-mt-24">
                    <h2 className="text-2xl font-semibold mb-4">九、协议的变更</h2>
                    <div className="space-y-4 text-gray-600 dark:text-gray-400">
                      <p>公司可根据适用法律法规变化及维护交易秩序、保护消费者权益等需要，不时修改本协议、补充协议，变更后的协议、补充协议将通过法定程序通知您。</p>
                      <p>如您不同意变更事项，您有权于变更事项确定的生效日前联系我们反馈意见。如您在变更事项生效后仍继续使用平台服务，则视为您已充分阅读并认可和同意遵守已生效的变更事项。</p>
                    </div>
                  </section>

                  {/* 十、通知 */}
                  <section id="notice" className="scroll-mt-24">
                    <h2 className="text-2xl font-semibold mb-4">十、通知</h2>
                    <div className="space-y-4 text-gray-600 dark:text-gray-400">
                      <p>在您注册成为平台用户，并接受平台服务时，您应该向公司提供真实有效的联系方式（包括您的电子邮件地址、联系电话等），对于联系方式发生变更的，您有义务及时更新有关信息，并保持可被联系的状态。</p>
                      <p>公司通过上述联系方式向您发出通知，其中以电子的方式发出的书面通知，在发送之日即视为送达；以纸质载体发出的书面通知，按照提供联系地址交邮后的第五个自然日即视为送达。</p>
                    </div>
                  </section>

                  {/* 十一、协议的终止 */}
                  <section id="termination" className="scroll-mt-24">
                    <h2 className="text-2xl font-semibold mb-4">十一、协议的终止</h2>
                    <div className="space-y-4 text-gray-600 dark:text-gray-400">
                      <p>您有权通过以下任一方式终止本协议：</p>
                      <ul className="list-disc pl-6 space-y-2">
                        <li>您自主选择注销账户的；</li>
                        <li>变更事项生效前您停止使用并明示不愿接受变更事项的；</li>
                        <li>您明示不愿继续使用平台服务，且符合平台终止条件的。</li>
                      </ul>
                      <p className="mt-4">本协议终止后，除适用法律有明确规定外，公司无义务向您或您指定的第三方披露您账户中的任何信息。</p>
                    </div>
                  </section>

                  {/* 十二、法律适用、管辖与其他 */}
                  <section id="law" className="scroll-mt-24">
                    <h2 className="text-2xl font-semibold mb-4">十二、法律适用、管辖与其他</h2>
                    <div className="space-y-4 text-gray-600 dark:text-gray-400">
                      <p>12.1 本协议之订立、生效、解释、修订、补充、终止、执行与争议解决均适用中华人民共和国法律；如适用法律无相关规定的，参照商业惯例及/或行业惯例。</p>
                      <p>12.2 您因使用平台服务所产生及与平台服务有关的争议，由公司与您协商解决。协商不成时，任何一方均可向公司所在地有管辖权的人民法院提起诉讼。</p>
                      <p>12.3 本协议任一条款无论因何种原因被视为废止、无效或不可执行，该条应视为可分的且并不影响本协议其余条款的有效性及可执行性，其余条款对双方仍具有约束力。</p>
                    </div>
                  </section>

                  {/* 联系我们 */}
                  <section className="bg-purple-50 dark:bg-purple-900/20 rounded-lg p-6 mt-8">
                    <h3 className="text-lg font-semibold text-gray-900 dark:text-white mb-3">联系我们</h3>
                    <p className="text-gray-600 dark:text-gray-400 mb-2">
                      如您对本协议有任何疑问、意见或建议，请通过以下方式与我们联系：
                    </p>
                    <p className="text-gray-600 dark:text-gray-400">
                      公司名称：成都解忧造物科技有限责任公司
                    </p>
                    <p className="text-gray-600 dark:text-gray-400">
                      联系方式：通过<Link to="/about" className="text-purple-600 dark:text-purple-400 hover:underline">关于我们</Link>页面获取
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

export default Terms
