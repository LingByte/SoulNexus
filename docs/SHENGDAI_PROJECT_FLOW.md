# 声代 - 智能AI代接系统详细项目流程文档

## 目录

1. [项目概述](#项目概述)
2. [系统架构设计](#系统架构设计)
3. [核心业务流程](#核心业务流程)
4. [功能模块详细设计](#功能模块详细设计)
5. [技术实现方案](#技术实现方案)
6. [数据库设计](#数据库设计)
7. [接口设计](#接口设计)
8. [部署与运维](#部署与运维)
9. [项目开发计划](#项目开发计划)

---

## 1. 项目概述

### 1.1 产品定位

**声代**是一款智能AI电话代接产品，当用户无法接听电话时，AI助手会根据用户预设的方案自动接听并智能应答，记录通话内容，生成摘要，让用户随时了解错过的重要来电。

### 1.2 核心价值

- **智能代接**：AI根据用户配置自动接听电话，解放双手
- **个性化配置**：支持多种代接方案，适应不同场景
- **语音克隆**：使用用户自己的声音进行应答，更加真实
- **智能摘要**：自动生成通话摘要，快速了解来电意图
- **留言录音**：支持对方留言，不错过任何重要信息

### 1.3 应用场景

- **会议中**：开会时自动代接，会后查看重要来电
- **驾驶中**：开车时无法接听，AI代为应答
- **休息时**：睡觉或休息时避免打扰，AI筛选重要来电
- **防骚扰**：自动识别和应对骚扰电话、推销电话
- **工作繁忙**：忙碌时AI秘书代为接听，记录重要信息

---

## 2. 系统架构设计

### 2.1 整体架构

```
┌─────────────────────────────────────────────────────────────┐
│                      用户端 (Client)                          │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐                   │
│  │ 移动应用  │  │ Web应用   │  │ 微信小程序 │                   │
│  └──────────┘  └──────────┘  └──────────┘                   │
└─────────────────────────────────────────────────────────────┘
                            ↓
┌─────────────────────────────────────────────────────────────┐
│                    API网关层 (Gateway)                        │
│  ┌──────────────────────────────────────────────────────┐   │
│  │  认证鉴权 | 限流控制 | 日志记录 | 负载均衡            │   │
│  └──────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────┘
                            ↓
┌─────────────────────────────────────────────────────────────┐
│                   业务服务层 (Services)                       │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐    │
│  │ 用户服务  │  │ 方案服务  │  │ 通话服务  │  │ 号码服务  │    │
│  └──────────┘  └──────────┘  └──────────┘  └──────────┘    │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐    │
│  │ 音色服务  │  │ 录音服务  │  │ 通知服务  │  │ 会员服务  │    │
│  └──────────┘  └──────────┘  └──────────┘  └──────────┘    │
└─────────────────────────────────────────────────────────────┘
                            ↓
┌─────────────────────────────────────────────────────────────┐
│                   电话接入层 (Telephony)                      │
│  ┌──────────────────────────────────────────────────────┐   │
│  │  SIP服务器 | 呼叫路由 | 媒体网关 | 号码池管理         │   │
│  └──────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────┘
                            ↓
┌─────────────────────────────────────────────────────────────┐
│                   AI能力层 (AI Services)                      │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐    │
│  │   ASR    │  │   TTS    │  │   LLM    │  │   VAD    │    │
│  │ 语音识别  │  │ 语音合成  │  │ 对话生成  │  │ 语音检测  │    │
│  └──────────┘  └──────────┘  └──────────┘  └──────────┘    │
│  ┌──────────┐  ┌──────────┐                                │
│  │ 声音克隆  │  │ 文本摘要  │                                │
│  └──────────┘  └──────────┘                                │
└─────────────────────────────────────────────────────────────┘
                            ↓
┌─────────────────────────────────────────────────────────────┐
│                   数据存储层 (Storage)                        │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐    │
│  │  MySQL   │  │  Redis   │  │ 对象存储  │  │ 消息队列  │    │
│  │ 业务数据  │  │  缓存    │  │ 录音文件  │  │  任务    │    │
│  └──────────┘  └──────────┘  └──────────┘  └──────────┘    │
└─────────────────────────────────────────────────────────────┘
```

### 2.2 技术栈选型

#### 后端技术
- **开发语言**：Go 1.21+
- **Web框架**：Gin
- **数据库**：MySQL 8.0
- **缓存**：Redis 7.0
- **消息队列**：RabbitMQ / Kafka
- **对象存储**：阿里云OSS / 七牛云
- **SIP协议栈**：自研SIP实现

#### 前端技术
- **移动端**：React Native / Flutter
- **Web端**：React 18 + TypeScript
- **小程序**：微信小程序原生开发
- **状态管理**：Zustand / Redux
- **UI组件**：Ant Design Mobile / Material-UI

#### AI服务
- **ASR**：火山引擎、腾讯云、讯飞
- **TTS**：火山引擎、腾讯云、讯飞
- **LLM**：GPT-4、Claude、DeepSeek
- **声音克隆**：火山引擎、讯飞

#### 基础设施
- **容器化**：Docker + Docker Compose
- **编排**：Kubernetes（可选）
- **监控**：Prometheus + Grafana
- **日志**：ELK Stack
- **CI/CD**：GitLab CI / GitHub Actions

---

## 3. 核心业务流程

### 3.1 用户注册与登录流程

```
用户 → 选择登录方式 → 手机号验证码 / 微信授权
                          ↓
                    验证身份信息
                          ↓
                    创建/更新用户账户
                          ↓
                    生成JWT Token
                          ↓
                    返回用户信息
                          ↓
                    进入应用首页
```

**详细步骤**：
1. 用户打开应用，选择登录方式
2. **手机号登录**：
   - 输入手机号
   - 点击获取验证码
   - 后端发送短信验证码（有效期5分钟）
   - 用户输入验证码
   - 后端验证验证码
   - 验证通过，创建/更新用户信息
   - 生成JWT Token返回
3. **微信登录**：
   - 调用微信授权接口
   - 获取微信用户信息（openid、昵称、头像）
   - 后端绑定微信账号
   - 生成JWT Token返回
4. 客户端保存Token，后续请求携带Token

### 3.2 号码绑定与呼叫转移设置流程

```
用户 → 添加手机号 → 验证手机号归属
                      ↓
                分配转接号码
                      ↓
                显示呼叫转移设置指引
                      ↓
                用户在手机上设置呼叫转移
                      ↓
                系统检测转移状态（可选）
                      ↓
                绑定成功，号码可用
```

**详细步骤**：
1. 用户进入"号码管理"页面
2. 点击"添加号码"
3. 输入要绑定的手机号
4. 发送验证码验证手机号归属
5. 验证通过后，系统为该号码分配一个转接号码
6. 显示呼叫转移设置指引：
   - **中国移动**：拨打 `**21*转接号码#`
   - **中国联通**：拨打 `**21*转接号码#`
   - **中国电信**：拨打 `**21*转接号码#`
   - 取消转移：拨打 `##21#`
7. 用户按照指引在手机上设置呼叫转移
8. 系统定期检测转移状态（通过测试呼叫）
9. 绑定成功，该号码可用于代接方案

### 3.3 代接方案创建与配置流程

```
用户 → 创建方案 → 填写方案信息
                    ↓
              配置AI设定
                    ↓
              选择音色
                    ↓
              配置录音选项
                    ↓
              选择生效号码
                    ↓
              保存方案
                    ↓
              启用方案（可选）
```

**详细步骤**：
1. 用户进入"方案管理"页面
2. 点击"创建新方案"
3. **基本信息**：
   - 输入方案名称（如"工作模式"）
   - 选择方案图标/颜色（可选）
4. **AI设定**：
   - 输入AI提示词（人设、语气、风格）
   - 配置开场白（接通后第一句话）
   - 配置关键词回复（可选）：
     - 添加关键词和对应回复
     - 例如：关键词"快递" → 回复"请放在门口"
   - 配置兜底回复策略：
     - 使用固定回复
     - AI自由发挥
5. **音色设置**：
   - 选择系统音色（男声、女声、童声等）
   - 或选择已克隆的个人音色
   - 试听音色效果
6. **录音配置**：
   - 是否开启录音
   - 录音节点：全程录音 / 仅留言阶段
   - 留言时长：默认20秒
7. **生效号码**：
   - 选择该方案生效的手机号
   - 支持多选（一个方案可用于多个号码）
8. 保存方案
9. 可选择立即启用该方案

### 3.4 AI代接通话完整流程

```
来电 → 呼叫转移至平台 → 查询用户方案
                          ↓
                    建立SIP连接
                          ↓
                    播放开场白（TTS）
                          ↓
                    开始录音（如配置）
                          ↓
        ┌───────────────────────────────────┐
        │         对话交互循环               │
        │  接收语音 → ASR识别 → 关键词匹配  │
        │      ↓           ↓          ↓     │
        │  VAD检测   LLM生成   固定回复     │
        │      ↓           ↓          ↓     │
        │  TTS合成 → 播放语音 → 继续监听    │
        └───────────────────────────────────┘
                          ↓
                    检测对话结束
                          ↓
                    播放留言提示
                          ↓
                    等待留言（20秒）
                          ↓
                    通话结束，挂断
                          ↓
                    保存录音文件
                          ↓
                    生成通话记录
                          ↓
                    ASR转文字
                          ↓
                    LLM生成摘要
                          ↓
                    推送通知用户
```

**详细步骤**：

#### 阶段1：来电接入（0-2秒）
1. 用户手机收到来电
2. 运营商将呼叫转移至平台转接号码
3. 平台SIP服务器接收到INVITE请求
4. 解析来电号码和被叫号码
5. 查询被叫号码对应的用户
6. 查询用户当前启用的代接方案
7. 建立SIP会话，返回200 OK
8. 建立RTP媒体流通道

#### 阶段2：开场白播放（2-5秒）
1. 获取方案配置的开场白文本
2. 调用TTS服务合成语音
3. 使用配置的音色进行合成
4. 通过RTP发送音频流给来电方
5. 如配置全程录音，开始录音

#### 阶段3：对话交互（5秒-3分钟）
1. **语音接收**：
   - 通过RTP接收来电方的语音流
   - 实时进行VAD检测
   - 检测到语音活动时开始缓存音频

2. **语音识别**：
   - VAD检测到静音，判断说话结束
   - 将缓存的音频发送给ASR服务
   - ASR识别为文字
   - 识别延迟：< 500ms

3. **意图理解与回复生成**：
   - 检查是否匹配关键词回复
   - 如匹配，使用固定回复内容
   - 如不匹配，将对话历史和用户输入发送给LLM
   - LLM根据提示词生成回复
   - 生成延迟：< 2秒

4. **语音合成与播放**：
   - 将回复文本发送给TTS服务
   - 使用配置的音色合成语音
   - 通过RTP发送音频流
   - 合成延迟：< 300ms

5. **循环交互**：
   - 继续监听来电方语音
   - 重复上述流程
   - 直到检测到对话结束信号

#### 阶段4：对话结束判断
触发条件（满足任一）：
- 来电方说"再见"、"挂了"等结束语
- 连续30秒无语音输入
- 通话时长超过3分钟
- 来电方主动挂断

#### 阶段5：留言阶段（可选，20秒）
1. 播放留言提示语：
   - "如需留言，请在嘀声后开始，我会转告TA"
2. 播放提示音（嘀）
3. 开始录音（如之前未开启）
4. 等待20秒或检测到挂断
5. 停止录音

#### 阶段6：通话结束与后处理
1. 发送SIP BYE消息
2. 关闭RTP媒体流
3. 保存录音文件到对象存储
4. 创建通话记录：
   - 来电号码
   - 被叫号码
   - 通话时长
   - 录音文件URL
   - 所属方案ID
5. 异步处理任务：
   - 录音转文字（ASR）
   - 生成对话摘要（LLM）
   - 识别来电意图
6. 推送通知给用户：
   - App推送
   - 微信服务号消息
   - 短信通知（可选）

### 3.5 声音克隆流程

```
用户 → 进入声音克隆 → 选择克隆方式
                          ↓
                    录制音频样本
                          ↓
                    上传音频文件
                          ↓
                    音频质量检测
                          ↓
                    提交训练任务
                          ↓
                    后台模型训练
                          ↓
                    训练完成通知
                          ↓
                    试听克隆音色
                          ↓
                    保存到音色库
```

**详细步骤**：

#### 方式1：在线录制
1. 用户进入"声音克隆"页面
2. 点击"开始录制"
3. 系统显示朗读文本（约1-2分钟）
4. 用户朗读文本，系统实时录音
5. 录制完成，可试听和重录
6. 确认后上传音频

#### 方式2：上传文件
1. 用户选择"上传音频文件"
2. 选择本地音频文件（支持MP3、WAV、M4A）
3. 上传到服务器

#### 音频质量检测
1. 检测音频时长（需1-3分钟）
2. 检测采样率（建议16kHz以上）
3. 检测噪声水平
4. 检测音量大小
5. 如不合格，提示用户重新录制

#### 模型训练
1. 音频预处理：
   - 降噪处理
   - 音量归一化
   - 格式转换
2. 提交到声音克隆服务（火山引擎/讯飞）
3. 创建训练任务记录
4. 后台异步训练（约1-2小时）
5. 定期查询训练状态
6. 训练完成，保存模型ID

#### 完成与使用
1. 推送通知用户训练完成
2. 用户进入"我的音色库"
3. 试听克隆音色效果
4. 可在代接方案中选择使用
5. 支持删除和重新训练

### 3.6 通话记录查看流程

```
用户 → 进入通话记录 → 查看记录列表
                          ↓
                    点击某条记录
                          ↓
                    进入详情页
                          ↓
        ┌───────────────────────────────────┐
        │  播放录音 | 查看文字 | 查看摘要   │
        └───────────────────────────────────┘
                          ↓
                    执行操作
                          ↓
        ┌───────────────────────────────────┐
        │  回拨 | 发短信 | 删除 | 加黑名单   │
        └───────────────────────────────────┘
```

**详细步骤**：

#### 记录列表
1. 用户进入"通话记录"页面
2. 显示所有代接记录，按时间倒序
3. 每条记录显示：
   - 来电号码（脱敏显示：138****1234）
   - 来电归属地（如：北京移动）
   - 代接时间（如：今天 14:30）
   - 通话时长（如：1分30秒）
   - 所属方案（如：工作模式）
   - 摘要标签（如：快递、推销、朋友）
4. 支持筛选：
   - 按日期筛选
   - 按方案筛选
   - 按号码筛选
5. 支持搜索：按号码或内容搜索

#### 详情页
1. 点击某条记录进入详情
2. **基本信息**：
   - 完整来电号码
   - 来电归属地
   - 通话开始时间
   - 通话结束时间
   - 通话时长
   - 所属方案
3. **录音播放**：
   - 音频播放器
   - 支持播放/暂停
   - 支持倍速播放（0.5x、1x、1.5x、2x）
   - 支持进度条拖动
   - 显示当前播放时间/总时长
4. **对话文字**：
   - 显示完整对话内容
   - 区分AI和来电方
   - 显示每句话的时间戳
   - 支持复制文字
5. **智能摘要**：
   - AI生成的通话摘要
   - 识别来电意图
   - 提取关键信息
   - 例如："快递员通知包裹已放在门口"
6. **操作按钮**：
   - **回拨**：直接拨打该号码
   - **发短信**：发送短信给该号码
   - **删除记录**：删除该通话记录
   - **加入黑名单**：将号码加入黑名单
   - **分享**：分享录音或文字

---

## 4. 功能模块详细设计

### 4.1 用户账户模块

#### 4.1.1 注册登录
**功能点**：
- 手机号验证码登录
- 微信一键登录
- 自动创建账户
- JWT Token认证

**数据表**：
- users（用户表）
- user_auth（用户认证表）
- sms_codes（短信验证码表）

**接口**：
- POST /api/auth/send-code - 发送验证码
- POST /api/auth/login - 登录
- POST /api/auth/wechat-login - 微信登录
- POST /api/auth/logout - 登出
- GET /api/auth/refresh - 刷新Token

#### 4.1.2 个人信息管理
**功能点**：
- 查看个人信息
- 修改昵称
- 上传头像
- 绑定/解绑手机号
- 修改密码
- 注销账号

**接口**：
- GET /api/user/profile - 获取个人信息
- PUT /api/user/profile - 更新个人信息
- POST /api/user/avatar - 上传头像
- PUT /api/user/password - 修改密码
- DELETE /api/user/account - 注销账号

### 4.2 号码管理模块

#### 4.2.1 号码绑定
**功能点**：
- 添加手机号
- 验证手机号归属
- 分配转接号码
- 显示呼叫转移设置指引
- 检测转移状态
- 删除号码

**数据表**：
- user_phones（用户手机号表）
- forward_numbers（转接号码池表）

**接口**：
- GET /api/phones - 获取号码列表
- POST /api/phones - 添加号码
- DELETE /api/phones/:id - 删除号码
- GET /api/phones/:id/forward-guide - 获取转移设置指引
- GET /api/phones/:id/status - 检测转移状态

#### 4.2.2 呼叫转移管理
**功能点**：
- 生成转移设置代码
- 区分运营商（移动、联通、电信）
- 测试转移是否生效
- 提供取消转移代码

**转移代码规则**：
```
开启无条件转移：**21*转接号码#
开启无应答转移：**61*转接号码#
开启遇忙转移：**67*转接号码#
开启不可及转移：**62*转接号码#
取消所有转移：##002#
查询转移状态：*#21#
```

### 4.3 代接方案模块

#### 4.3.1 方案管理
**功能点**：
- 创建方案
- 编辑方案
- 删除方案
- 启用/禁用方案
- 方案列表查询
- 方案详情查询

**数据表**：
- answer_schemes（代接方案表）
- scheme_keywords（方案关键词表）

**接口**：
- GET /api/schemes - 获取方案列表
- POST /api/schemes - 创建方案
- GET /api/schemes/:id - 获取方案详情
- PUT /api/schemes/:id - 更新方案
- DELETE /api/schemes/:id - 删除方案
- POST /api/schemes/:id/enable - 启用方案
- POST /api/schemes/:id/disable - 禁用方案

#### 4.3.2 AI配置
**配置项**：
- **AI提示词**：
  - 人设描述（如"我是某某的秘书"）
  - 语气风格（严肃、幽默、温和）
  - 回复策略（简洁、详细）
  - 特殊指令（如"不要透露主人信息"）

- **开场白**：
  - 固定文本
  - 支持变量（如{caller_name}、{time}）
  - 示例："您好，这里是{user_name}的AI助手，请问有什么可以帮您？"

- **关键词回复**：
  - 关键词列表
  - 对应回复内容
  - 匹配模式（精确匹配、模糊匹配）
  - 优先级设置

- **兜底策略**：
  - 使用固定回复
  - AI自由发挥
  - 转人工（预留）

#### 4.3.3 音色配置
**系统音色**：
- 男声：成熟男声、青年男声、磁性男声
- 女声：甜美女声、知性女声、温柔女声
- 童声：男童、女童
- 方言：粤语、四川话、东北话（可选）

**克隆音色**：
- 从"我的音色库"选择
- 显示音色名称和创建时间
- 支持试听

**接口**：
- GET /api/voices/system - 获取系统音色列表
- GET /api/voices/cloned - 获取克隆音色列表
- POST /api/voices/preview - 试听音色

### 4.4 通话记录模块

#### 4.4.1 记录管理
**功能点**：
- 记录列表查询
- 记录详情查询
- 记录删除
- 记录搜索
- 记录筛选
- 记录导出

**数据表**：
- call_records（通话记录表）
- call_transcripts（通话文字表）
- call_summaries（通话摘要表）

**接口**：
- GET /api/calls - 获取通话记录列表
- GET /api/calls/:id - 获取通话详情
- DELETE /api/calls/:id - 删除记录
- GET /api/calls/:id/audio - 获取录音文件
- GET /api/calls/:id/transcript - 获取文字记录
- GET /api/calls/:id/summary - 获取摘要
- POST /api/calls/export - 导出记录

#### 4.4.2 录音处理
**功能点**：
- 录音文件存储
- 录音在线播放
- 录音下载
- 录音转文字
- 录音质量优化

**处理流程**：
1. 通话结束，保存原始录音
2. 音频格式转换（统一为MP3）
3. 音频压缩（降低存储成本）
4. 上传到对象存储
5. 生成播放URL（带签名，有效期24小时）
6. 异步转文字（ASR）
7. 保存文字记录

#### 4.4.3 智能摘要
**功能点**：
- 自动生成摘要
- 识别来电意图
- 提取关键信息
- 情感分析
- 重要性评分

**摘要生成逻辑**：
1. 获取通话文字记录
2. 调用LLM生成摘要
3. 提示词模板：
```
请根据以下通话内容生成简短摘要（不超过50字）：
- 来电者身份
- 来电目的
- 关键信息
- 是否需要回复

通话内容：
{transcript}
```
4. 保存摘要结果
5. 识别意图标签（快递、推销、朋友、工作等）

### 4.5 声音克隆模块

#### 4.5.1 音频录制
**功能点**：
- 在线录音
- 实时波形显示
- 录音暂停/继续
- 录音试听
- 重新录制

**技术实现**：
- 使用Web Audio API / React Native Audio
- 采样率：16kHz
- 格式：WAV / PCM
- 时长：1-3分钟
- 实时音量检测

#### 4.5.2 音频上传
**功能点**：
- 选择本地文件
- 支持格式：MP3、WAV、M4A、AAC
- 文件大小限制：50MB
- 上传进度显示
- 断点续传

#### 4.5.3 质量检测
**检测项**：
- 时长检测：1-3分钟
- 采样率检测：≥16kHz
- 噪声检测：信噪比≥20dB
- 音量检测：-20dB ~ -3dB
- 静音检测：静音时长<10%

**不合格处理**：
- 显示具体问题
- 提供优化建议
- 支持重新录制/上传

#### 4.5.4 模型训练
**功能点**：
- 提交训练任务
- 训练进度查询
- 训练状态通知
- 训练失败重试

**数据表**：
- voice_clones（声音克隆表）
- clone_tasks（训练任务表）

**接口**：
- POST /api/voices/clone - 提交克隆任务
- GET /api/voices/clone/:id - 查询训练状态
- GET /api/voices/my-voices - 获取我的音色库
- DELETE /api/voices/:id - 删除音色
- POST /api/voices/:id/preview - 试听音色

**训练流程**：
1. 用户提交音频文件
2. 创建训练任务记录
3. 调用声音克隆服务API
4. 定期轮询训练状态
5. 训练完成，保存模型ID
6. 推送通知用户
7. 音色可用于代接方案

### 4.6 通知模块

#### 4.6.1 推送通知
**通知场景**：
- 新的代接通话
- 声音克隆完成
- 重要来电提醒
- 系统公告

**通知渠道**：
- App推送（极光推送/个推）
- 微信服务号消息
- 短信通知（可选）

**通知内容**：
```json
{
  "title": "您有新的代接通话",
  "content": "来自138****1234的通话，摘要：快递已放在门口",
  "type": "call",
  "data": {
    "call_id": "123456",
    "caller": "138****1234",
    "summary": "快递已放在门口"
  },
  "timestamp": 1234567890
}
```

#### 4.6.2 通知设置
**功能点**：
- 开启/关闭推送
- 选择通知渠道
- 设置免打扰时段
- 通知声音设置
- 震动设置

**接口**：
- GET /api/notifications/settings - 获取通知设置
- PUT /api/notifications/settings - 更新通知设置
- GET /api/notifications - 获取通知列表
- PUT /api/notifications/:id/read - 标记已读
- DELETE /api/notifications/:id - 删除通知

### 4.7 会员模块

#### 4.7.1 套餐管理
**套餐类型**：

**免费版**：
- 代接时长：每月30分钟
- 音色克隆：1个
- 录音保存：7天
- 文字转录：每月100次
- 方案数量：3个

**专业版**（¥19.9/月）：
- 代接时长：每月300分钟
- 音色克隆：5个
- 录音保存：30天
- 文字转录：每月1000次
- 方案数量：10个
- 优先客服支持

**企业版**（¥99/月）：
- 代接时长：无限制
- 音色克隆：无限制
- 录音保存：永久
- 文字转录：无限制
- 方案数量：无限制
- 专属客服
- API接口

#### 4.7.2 订单管理
**功能点**：
- 购买套餐
- 续费
- 升级/降级
- 退款
- 发票管理

**数据表**：
- subscriptions（订阅表）
- orders（订单表）
- invoices（发票表）

**接口**：
- GET /api/subscriptions/plans - 获取套餐列表
- POST /api/subscriptions/purchase - 购买套餐
- GET /api/subscriptions/current - 获取当前订阅
- POST /api/subscriptions/cancel - 取消订阅
- GET /api/orders - 获取订单列表
- POST /api/invoices - 申请发票

---

## 5. 技术实现方案

### 5.1 SIP通话实现

#### 5.1.1 SIP服务器搭建
**技术选型**：
- 自研SIP协议栈（基于Go）
- 支持RFC 3261标准
- UDP/TCP传输
- 注册、呼叫、挂断等基本功能

**核心组件**：
```
SIP Server
├── Transport Layer（传输层）
│   ├── UDP Transport
│   ├── TCP Transport
│   └── TLS Transport
├── Transaction Layer（事务层）
│   ├── Client Transaction
│   └── Server Transaction
├── Dialog Layer（对话层）
│   ├── Dialog Manager
│   └── Session Manager
└── Application Layer（应用层）
    ├── Call Handler
    ├── Registration Handler
    └── Media Handler
```

#### 5.1.2 媒体处理
**RTP音频流**：
- 编解码：PCMU、PCMA、Opus
- 采样率：8kHz、16kHz
- 打包：20ms一帧
- 缓冲：100ms抖动缓冲

**音频处理管道**：
```
接收RTP → 解码 → 重采样 → VAD检测 → ASR识别
                                          ↓
发送RTP ← 编码 ← 音频合成 ← TTS合成 ← LLM生成
```

### 5.2 AI服务集成

#### 5.2.1 ASR（语音识别）
**提供商选择**：
- 主要：火山引擎ASR
- 备用：腾讯云ASR、讯飞ASR

**集成方式**：
- WebSocket实时流式识别
- 支持中英文混合
- 识别准确率：>95%
- 识别延迟：<500ms

**优化策略**：
- VAD预处理，减少无效识别
- 音频质量检测
- 自动重试机制
- 结果后处理（标点、数字转换）

#### 5.2.2 TTS（语音合成）
**提供商选择**：
- 主要：火山引擎TTS
- 备用：腾讯云TTS、讯飞TTS

**集成方式**：
- HTTP API调用
- 流式合成
- 支持多种音色
- 合成延迟：<300ms

**优化策略**：
- 文本预处理（多音字、数字）
- 分段合成（长文本）
- 音频缓存（常用语）
- 并发控制

#### 5.2.3 LLM（对话生成）
**模型选择**：
- 主要：GPT-4-turbo
- 备用：Claude-3、DeepSeek

**Prompt设计**：
```
你是{user_name}的AI助手，负责代接电话。

人设：{ai_prompt}

对话历史：
{conversation_history}

用户说：{user_input}

请根据以下规则回复：
1. 保持人设，语气自然
2. 回复简洁，不超过50字
3. 如果是重要事项，提醒对方留言
4. 如果是骚扰电话，礼貌拒绝

你的回复：
```

**优化策略**：
- 上下文管理（保留最近5轮对话）
- 流式输出（降低首字延迟）
- 结果缓存（相似问题）
- 超时控制（3秒）

### 5.3 录音存储方案

#### 5.3.1 存储架构
**存储选型**：
- 对象存储：阿里云OSS / 七牛云
- 热数据：最近7天，标准存储
- 温数据：7-30天，低频存储
- 冷数据：30天以上，归档存储

**存储结构**：
```
bucket/
├── recordings/
│   ├── 2024/
│   │   ├── 01/
│   │   │   ├── 31/
│   │   │   │   ├── user_123_call_456.mp3
│   │   │   │   └── user_123_call_457.mp3
```

#### 5.3.2 录音处理流程
1. **实时录音**：
   - 接收RTP音频流
   - 实时写入本地临时文件
   - 格式：PCM 16kHz 16bit

2. **后处理**：
   - 格式转换：PCM → MP3
   - 音频压缩：比特率64kbps
   - 降噪处理（可选）
   - 音量归一化

3. **上传存储**：
   - 上传到对象存储
   - 生成访问URL
   - 设置访问权限（私有）
   - 删除本地临时文件

4. **URL签名**：
   - 生成带签名的访问URL
   - 有效期：24小时
   - 防盗链保护

### 5.4 实时性能优化

#### 5.4.1 延迟优化
**目标延迟**：
- 端到端延迟：<2秒
- ASR识别延迟：<500ms
- LLM生成延迟：<1.5秒
- TTS合成延迟：<300ms

**优化措施**：
1. **VAD优化**：
   - 快速检测语音结束
   - 减少等待时间
   - 降低误触发

2. **并行处理**：
   - ASR和LLM并行
   - TTS流式合成
   - 边合成边播放

3. **预加载**：
   - 开场白预合成
   - 常用回复预合成
   - 音色模型预加载

4. **缓存策略**：
   - LLM结果缓存
   - TTS音频缓存
   - 用户配置缓存

#### 5.4.2 并发处理
**并发能力**：
- 单机并发：1000通话
- 集群并发：10000+通话

**资源分配**：
- 每通话占用：
  - CPU：0.5核
  - 内存：50MB
  - 带宽：64kbps

**负载均衡**：
- 基于最少连接数
- 健康检查
- 自动故障转移

---

## 6. 数据库设计

### 6.1 核心数据表

#### 6.1.1 用户表（users）
```sql
CREATE TABLE users (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    phone VARCHAR(20) UNIQUE COMMENT '手机号',
    nickname VARCHAR(50) COMMENT '昵称',
    avatar VARCHAR(255) COMMENT '头像URL',
    wechat_openid VARCHAR(100) COMMENT '微信OpenID',
    status TINYINT DEFAULT 1 COMMENT '状态：1正常 2禁用',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    INDEX idx_phone (phone),
    INDEX idx_wechat (wechat_openid)
) COMMENT='用户表';
```

#### 6.1.2 用户手机号表（user_phones）
```sql
CREATE TABLE user_phones (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    user_id BIGINT NOT NULL COMMENT '用户ID',
    phone VARCHAR(20) NOT NULL COMMENT '手机号',
    forward_number VARCHAR(20) COMMENT '转接号码',
    operator VARCHAR(20) COMMENT '运营商：移动/联通/电信',
    status TINYINT DEFAULT 1 COMMENT '状态：1正常 2已删除',
    forward_status TINYINT DEFAULT 0 COMMENT '转移状态：0未设置 1已设置',
    last_check_time TIMESTAMP COMMENT '最后检测时间',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    INDEX idx_user (user_id),
    INDEX idx_phone (phone),
    INDEX idx_forward (forward_number)
) COMMENT='用户手机号表';
```

#### 6.1.3 代接方案表（answer_schemes）
```sql
CREATE TABLE answer_schemes (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    user_id BIGINT NOT NULL COMMENT '用户ID',
    name VARCHAR(100) NOT NULL COMMENT '方案名称',
    ai_prompt TEXT COMMENT 'AI提示词',
    opening_text VARCHAR(500) COMMENT '开场白',
    fallback_strategy TINYINT DEFAULT 1 COMMENT '兜底策略：1固定回复 2AI发挥',
    fallback_text TEXT COMMENT '兜底回复内容',
    voice_type TINYINT DEFAULT 1 COMMENT '音色类型：1系统 2克隆',
    voice_id VARCHAR(100) COMMENT '音色ID',
    recording_enabled TINYINT DEFAULT 1 COMMENT '是否录音：0否 1是',
    recording_mode TINYINT DEFAULT 1 COMMENT '录音模式：1全程 2仅留言',
    message_duration INT DEFAULT 20 COMMENT '留言时长（秒）',
    is_enabled TINYINT DEFAULT 0 COMMENT '是否启用：0否 1是',
    phone_ids JSON COMMENT '生效号码ID列表',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    INDEX idx_user (user_id),
    INDEX idx_enabled (is_enabled)
) COMMENT='代接方案表';
```

#### 6.1.4 方案关键词表（scheme_keywords）
```sql
CREATE TABLE scheme_keywords (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    scheme_id BIGINT NOT NULL COMMENT '方案ID',
    keyword VARCHAR(100) NOT NULL COMMENT '关键词',
    reply_text TEXT NOT NULL COMMENT '回复内容',
    match_type TINYINT DEFAULT 1 COMMENT '匹配类型：1精确 2模糊',
    priority INT DEFAULT 0 COMMENT '优先级',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_scheme (scheme_id)
) COMMENT='方案关键词表';
```

#### 6.1.5 通话记录表（call_records）
```sql
CREATE TABLE call_records (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    user_id BIGINT NOT NULL COMMENT '用户ID',
    scheme_id BIGINT COMMENT '方案ID',
    caller_number VARCHAR(20) NOT NULL COMMENT '来电号码',
    called_number VARCHAR(20) NOT NULL COMMENT '被叫号码',
    caller_location VARCHAR(100) COMMENT '来电归属地',
    call_start_time TIMESTAMP COMMENT '通话开始时间',
    call_end_time TIMESTAMP COMMENT '通话结束时间',
    call_duration INT COMMENT '通话时长（秒）',
    recording_url VARCHAR(500) COMMENT '录音文件URL',
    recording_duration INT COMMENT '录音时长（秒）',
    transcript_status TINYINT DEFAULT 0 COMMENT '转录状态：0未转录 1转录中 2已完成 3失败',
    summary_status TINYINT DEFAULT 0 COMMENT '摘要状态：0未生成 1生成中 2已完成 3失败',
    intent_tag VARCHAR(50) COMMENT '意图标签',
    importance_score INT DEFAULT 0 COMMENT '重要性评分：0-100',
    is_read TINYINT DEFAULT 0 COMMENT '是否已读：0未读 1已读',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    INDEX idx_user (user_id),
    INDEX idx_caller (caller_number),
    INDEX idx_time (call_start_time),
    INDEX idx_read (is_read)
) COMMENT='通话记录表';
```

#### 6.1.6 通话文字表（call_transcripts）
```sql
CREATE TABLE call_transcripts (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    call_id BIGINT NOT NULL COMMENT '通话记录ID',
    speaker TINYINT NOT NULL COMMENT '说话人：1AI 2来电方',
    text TEXT NOT NULL COMMENT '文字内容',
    start_time DECIMAL(10,2) COMMENT '开始时间（秒）',
    end_time DECIMAL(10,2) COMMENT '结束时间（秒）',
    confidence DECIMAL(5,2) COMMENT '置信度',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_call (call_id)
) COMMENT='通话文字表';
```

#### 6.1.7 通话摘要表（call_summaries）
```sql
CREATE TABLE call_summaries (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    call_id BIGINT NOT NULL UNIQUE COMMENT '通话记录ID',
    summary TEXT COMMENT '摘要内容',
    caller_identity VARCHAR(100) COMMENT '来电者身份',
    call_purpose VARCHAR(200) COMMENT '来电目的',
    key_info TEXT COMMENT '关键信息',
    need_callback TINYINT DEFAULT 0 COMMENT '是否需要回复：0否 1是',
    sentiment VARCHAR(20) COMMENT '情感：positive/neutral/negative',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_call (call_id)
) COMMENT='通话摘要表';
```

#### 6.1.8 声音克隆表（voice_clones）
```sql
CREATE TABLE voice_clones (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    user_id BIGINT NOT NULL COMMENT '用户ID',
    name VARCHAR(100) NOT NULL COMMENT '音色名称',
    audio_url VARCHAR(500) COMMENT '样本音频URL',
    model_id VARCHAR(100) COMMENT '模型ID',
    provider VARCHAR(50) COMMENT '提供商：volcengine/xunfei',
    status TINYINT DEFAULT 0 COMMENT '状态：0训练中 1成功 2失败',
    error_message TEXT COMMENT '错误信息',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    INDEX idx_user (user_id),
    INDEX idx_status (status)
) COMMENT='声音克隆表';
```

#### 6.1.9 订阅表（subscriptions）
```sql
CREATE TABLE subscriptions (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    user_id BIGINT NOT NULL COMMENT '用户ID',
    plan_type TINYINT NOT NULL COMMENT '套餐类型：1免费 2专业 3企业',
    start_date DATE NOT NULL COMMENT '开始日期',
    end_date DATE NOT NULL COMMENT '结束日期',
    status TINYINT DEFAULT 1 COMMENT '状态：1正常 2已过期 3已取消',
    auto_renew TINYINT DEFAULT 0 COMMENT '自动续费：0否 1是',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    INDEX idx_user (user_id),
    INDEX idx_status (status),
    INDEX idx_end_date (end_date)
) COMMENT='订阅表';
```

#### 6.1.10 使用量统计表（usage_stats）
```sql
CREATE TABLE usage_stats (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    user_id BIGINT NOT NULL COMMENT '用户ID',
    stat_date DATE NOT NULL COMMENT '统计日期',
    call_duration INT DEFAULT 0 COMMENT '通话时长（秒）',
    call_count INT DEFAULT 0 COMMENT '通话次数',
    transcript_count INT DEFAULT 0 COMMENT '转录次数',
    storage_used BIGINT DEFAULT 0 COMMENT '存储使用量（字节）',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    UNIQUE KEY uk_user_date (user_id, stat_date),
    INDEX idx_date (stat_date)
) COMMENT='使用量统计表';
```

---

## 7. 接口设计

### 7.1 认证接口

#### POST /api/auth/send-code
发送验证码

**请求参数**：
```json
{
  "phone": "13800138000"
}
```

**响应**：
```json
{
  "code": 200,
  "msg": "验证码已发送",
  "data": {
    "expire_time": 300
  }
}
```

#### POST /api/auth/login
登录

**请求参数**：
```json
{
  "phone": "13800138000",
  "code": "123456"
}
```

**响应**：
```json
{
  "code": 200,
  "msg": "登录成功",
  "data": {
    "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
    "user": {
      "id": 123,
      "phone": "13800138000",
      "nickname": "用户123",
      "avatar": "https://..."
    }
  }
}
```

### 7.2 方案接口

#### GET /api/schemes
获取方案列表

**响应**：
```json
{
  "code": 200,
  "msg": "成功",
  "data": {
    "list": [
      {
        "id": 1,
        "name": "工作模式",
        "is_enabled": 1,
        "voice_type": 1,
        "phone_count": 2,
        "created_at": "2024-01-31 10:00:00"
      }
    ],
    "total": 10
  }
}
```

#### POST /api/schemes
创建方案

**请求参数**：
```json
{
  "name": "工作模式",
  "ai_prompt": "我是某某的秘书，语气专业",
  "opening_text": "您好，这里是{user_name}的AI助手",
  "voice_type": 1,
  "voice_id": "voice_001",
  "recording_enabled": 1,
  "recording_mode": 1,
  "message_duration": 20,
  "phone_ids": [1, 2],
  "keywords": [
    {
      "keyword": "快递",
      "reply_text": "请放在门口",
      "match_type": 1
    }
  ]
}
```

**响应**：
```json
{
  "code": 200,
  "msg": "创建成功",
  "data": {
    "id": 1
  }
}
```

### 7.3 通话记录接口

#### GET /api/calls
获取通话记录列表

**请求参数**：
```
page=1&limit=20&scheme_id=1&start_date=2024-01-01&end_date=2024-01-31
```

**响应**：
```json
{
  "code": 200,
  "msg": "成功",
  "data": {
    "list": [
      {
        "id": 1,
        "caller_number": "138****1234",
        "caller_location": "北京移动",
        "call_start_time": "2024-01-31 14:30:00",
        "call_duration": 90,
        "scheme_name": "工作模式",
        "intent_tag": "快递",
        "summary": "快递已放在门口",
        "is_read": 0
      }
    ],
    "total": 100,
    "page": 1,
    "limit": 20
  }
}
```

#### GET /api/calls/:id
获取通话详情

**响应**：
```json
{
  "code": 200,
  "msg": "成功",
  "data": {
    "id": 1,
    "caller_number": "13800138000",
    "caller_location": "北京移动",
    "call_start_time": "2024-01-31 14:30:00",
    "call_end_time": "2024-01-31 14:31:30",
    "call_duration": 90,
    "recording_url": "https://...",
    "recording_duration": 85,
    "scheme_name": "工作模式",
    "transcript": [
      {
        "speaker": 1,
        "text": "您好，这里是某某的AI助手",
        "start_time": 0.0,
        "end_time": 2.5
      },
      {
        "speaker": 2,
        "text": "你好，我是快递员",
        "start_time": 3.0,
        "end_time": 5.0
      }
    ],
    "summary": {
      "summary": "快递员通知包裹已放在门口",
      "caller_identity": "快递员",
      "call_purpose": "通知包裹送达",
      "key_info": "包裹放在门口",
      "need_callback": 0,
      "sentiment": "neutral"
    }
  }
}
```

### 7.4 声音克隆接口

#### POST /api/voices/clone
提交克隆任务

**请求参数**：
```json
{
  "name": "我的声音",
  "audio_url": "https://...",
  "provider": "volcengine"
}
```

**响应**：
```json
{
  "code": 200,
  "msg": "任务已提交",
  "data": {
    "id": 1,
    "status": 0,
    "estimated_time": 7200
  }
}
```

#### GET /api/voices/clone/:id
查询训练状态

**响应**：
```json
{
  "code": 200,
  "msg": "成功",
  "data": {
    "id": 1,
    "name": "我的声音",
    "status": 1,
    "model_id": "model_123",
    "created_at": "2024-01-31 10:00:00",
    "updated_at": "2024-01-31 12:00:00"
  }
}
```

---

## 8. 部署与运维

### 8.1 部署架构

#### 8.1.1 单机部署
**适用场景**：测试环境、小规模使用

**配置要求**：
- CPU：4核
- 内存：8GB
- 存储：100GB SSD
- 带宽：10Mbps

**部署方式**：
```bash
# 使用Docker Compose
docker-compose up -d

# 服务列表
- shengdai-server（主服务）
- mysql（数据库）
- redis（缓存）
- nginx（反向代理）
```

#### 8.1.2 集群部署
**适用场景**：生产环境、大规模使用

**架构**：
```
负载均衡（Nginx）
    ↓
应用服务器集群（3台）
    ↓
数据库主从（1主2从）
    ↓
Redis集群（3节点）
    ↓
对象存储（OSS）
```

**配置要求**：
- 应用服务器：8核16GB × 3
- 数据库服务器：16核32GB × 3
- Redis服务器：4核8GB × 3
- 带宽：100Mbps

### 8.2 监控告警

#### 8.2.1 监控指标
**系统指标**：
- CPU使用率
- 内存使用率
- 磁盘使用率
- 网络流量

**应用指标**：
- QPS（每秒请求数）
- 响应时间
- 错误率
- 并发通话数

**业务指标**：
- 日活用户数
- 通话成功率
- 平均通话时长
- ASR识别准确率
- TTS合成成功率

#### 8.2.2 告警规则
- CPU使用率 > 80%
- 内存使用率 > 85%
- 磁盘使用率 > 90%
- API错误率 > 5%
- 通话失败率 > 10%
- 数据库连接数 > 80%

### 8.3 日志管理

#### 8.3.1 日志分类
- **访问日志**：记录所有API请求
- **错误日志**：记录系统错误
- **业务日志**：记录业务操作
- **通话日志**：记录通话详情
- **审计日志**：记录敏感操作

#### 8.3.2 日志存储
- 本地文件：最近7天
- 日志服务：30天
- 归档存储：永久保存

---

## 9. 项目开发计划

### 9.1 开发阶段

#### 第一阶段：基础功能（4周）
**Week 1-2：用户系统**
- 用户注册登录
- 个人信息管理
- 号码绑定
- 呼叫转移设置

**Week 3-4：代接方案**
- 方案创建编辑
- AI配置
- 音色选择
- 方案启用

#### 第二阶段：核心功能（6周）
**Week 5-7：SIP通话**
- SIP服务器搭建
- 呼叫接入
- 媒体处理
- 录音功能

**Week 8-10：AI集成**
- ASR集成
- TTS集成
- LLM集成
- 对话管理

#### 第三阶段：增值功能（4周）
**Week 11-12：声音克隆**
- 音频录制
- 质量检测
- 模型训练
- 音色管理

**Week 13-14：通话记录**
- 记录列表
- 详情展示
- 录音播放
- 文字转录
- 智能摘要

#### 第四阶段：完善优化（4周）
**Week 15-16：会员系统**
- 套餐管理
- 订单支付
- 使用量统计
- 权限控制

**Week 17-18：测试优化**
- 功能测试
- 性能测试
- 安全测试
- Bug修复
- 性能优化

### 9.2 上线计划

#### 9.2.1 灰度发布
- 内部测试：1周
- 小范围测试：100用户，1周
- 扩大测试：1000用户，2周
- 全量发布

#### 9.2.2 运营推广
- 产品宣传
- 用户教育
- 客服支持
- 数据分析
- 持续优化

---

## 10. 总结

**声代**是一款创新的智能AI电话代接产品，通过AI技术解决用户无法接听电话的痛点。项目采用现代化的技术栈，具备完善的功能设计和清晰的开发计划。

**核心优势**：
- 智能化：AI自动应答，个性化配置
- 真实性：声音克隆，使用用户自己的声音
- 便捷性：一键启用，自动记录
- 智能化：自动摘要，快速了解来电意图

**技术亮点**：
- 自研SIP协议栈，稳定可靠
- 多AI服务集成，灵活切换
- 实时音频处理，低延迟
- 完善的监控告警，保障稳定性

项目预计18周完成开发，经过充分测试后上线运营。

---

**文档版本**：v1.0
**更新日期**：2024-01-31
**项目名称**：声代（智能AI代接系统）
