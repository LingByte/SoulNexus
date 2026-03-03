# AI自由回答功能

## 功能概述

AI自由回答功能允许用户在代接方案中控制AI助手的回答行为。启用后,AI除了完成预设的开场白、关键词回复等任务外,还会对拨号者的对话进行自由回答。

## 功能特性

### 1. 启用AI自由回答
- **启用时**: AI会使用LLM对用户的对话进行智能回复
- **禁用时**: AI只会执行预设任务(开场白、关键词回复、兜底回复)

### 2. 回复优先级
无论是否启用AI自由回答,回复优先级如下:
1. **关键词回复** - 如果匹配到关键词,优先使用关键词回复
2. **AI自由回答** - 如果启用且未匹配关键词,使用LLM生成回复
3. **兜底回复** - 如果禁用AI自由回答或LLM失败,使用兜底回复

## 数据库变更

### 新增字段
```sql
ALTER TABLE `sip_users` ADD COLUMN `ai_free_response` TINYINT(1) NOT NULL DEFAULT 1 COMMENT '是否启用AI自由回答' AFTER `fallback_message`;
```

### 字段说明
- **字段名**: `ai_free_response`
- **类型**: TINYINT(1) (布尔值)
- **默认值**: 1 (启用)
- **位置**: 在 `fallback_message` 字段之后

## 后端实现

### 1. 模型更新
在 `server/internal/models/sip_user.go` 中添加:
```go
AIFreeResponse  bool  `json:"aiFreeResponse" gorm:"default:true"`  // 是否启用AI自由回答
```

### 2. 对话处理逻辑
在 `server/pkg/sip/voice_conversation_handler.go` 中更新对话处理流程:
- 检查关键词回复
- 如果启用AI自由回答,调用LLM
- 否则使用兜底回复或跳过

## App端实现

### 1. API类型定义
在 `app/src/services/api/scheme.ts` 中添加:
```typescript
aiFreeResponse?: boolean;
```

### 2. 界面更新
在 `app/src/screens/CallCenter/SchemeTab.tsx` 中:
- 添加复选框控件
- 显示功能说明
- 在表单提交时包含该字段

### 3. 用户界面
```
☑ 启用AI自由回答
  启用后,AI除了完成预设任务外,还会对拨号者的对话进行自由回答
```

## 使用场景

### 场景1: 完全自动化客服
- **启用AI自由回答**: ✅
- **配置**: 开场白 + AI自由回答
- **效果**: AI可以回答各种问题,提供灵活的对话体验

### 场景2: 严格脚本式回复
- **启用AI自由回答**: ❌
- **配置**: 开场白 + 关键词回复 + 兜底回复
- **效果**: AI只按预设脚本回复,更可控

### 场景3: 混合模式
- **启用AI自由回答**: ✅
- **配置**: 开场白 + 关键词回复 + AI自由回答 + 兜底回复
- **效果**: 重要问题用关键词回复,其他问题AI自由回答

## 迁移说明

### 运行迁移
```bash
cd server
mysql -u root -p your_database < migrations/add_ai_free_response_to_sip_users.sql
```

### 默认行为
- 新创建的方案默认启用AI自由回答
- 现有方案会自动设置为启用(默认值为1)

## 注意事项

1. **LLM成本**: 启用AI自由回答会增加LLM API调用次数
2. **回复质量**: 依赖于配置的LLM模型和系统提示词
3. **兜底机制**: 建议配置兜底回复,以应对LLM失败的情况
4. **关键词优先**: 关键词回复始终优先于AI自由回答

## 测试建议

1. 创建测试方案,启用AI自由回答
2. 拨打测试电话,验证AI回复行为
3. 测试关键词回复优先级
4. 测试LLM失败时的兜底回复
5. 禁用AI自由回答,验证只使用预设回复

## 相关文件

### 后端
- `server/internal/models/sip_user.go` - 数据模型
- `server/pkg/sip/voice_conversation_handler.go` - 对话处理
- `server/migrations/add_ai_free_response_to_sip_users.sql` - 数据库迁移

### App端
- `app/src/services/api/scheme.ts` - API类型定义
- `app/src/screens/CallCenter/SchemeTab.tsx` - 方案管理界面
