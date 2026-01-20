# MCP 模块快速开始指南

## 1. 启动服务器

```bash
# 启动 MCP 服务器（SSE 模式，端口 3001）
go run cmd/mcp/main.go --transport sse --port 3001
```

服务器启动后会显示：
```
Starting MCP server {"transport": "sse", "port": "3001", "mode": "development"}
MCP 工具已注册 {"name": "echo", "description": "回显输入的文本"}
...
SSE server listening {"port": "3001"}
```

## 2. 使用客户端测试

**注意**：客户端会自动在 URL 后添加 `/sse` 路径。如果手动指定 URL，请确保包含 `/sse` 路径，例如：`http://localhost:3001/sse`

### 列出所有工具

```bash
go run cmd/mcp-client/main.go -tool list
```

### 测试各个工具

#### Echo 工具
```bash
go run cmd/mcp-client/main.go -tool echo -args '{"text":"Hello MCP!"}'
```

#### 计算器工具
```bash
# 简单计算
go run cmd/mcp-client/main.go -tool calculator -args '{"expression":"2+2"}'

# 复杂计算
go run cmd/mcp-client/main.go -tool calculator -args '{"expression":"10*5+20/4"}'
```

#### 获取当前时间
```bash
# 默认格式
go run cmd/mcp-client/main.go -tool get_current_time

# 指定格式和时区
go run cmd/mcp-client/main.go -tool get_current_time -args '{"format":"2006-01-02 15:04:05","timezone":"Asia/Shanghai"}'
```

#### 文本处理
```bash
# 转大写
go run cmd/mcp-client/main.go -tool text_process -args '{"text":"hello world","operation":"uppercase"}'

# 转小写
go run cmd/mcp-client/main.go -tool text_process -args '{"text":"HELLO WORLD","operation":"lowercase"}'

# 反转文本
go run cmd/mcp-client/main.go -tool text_process -args '{"text":"Hello","operation":"reverse"}'

# 统计信息
go run cmd/mcp-client/main.go -tool text_process -args '{"text":"Hello World\nThis is a test","operation":"count"}'
```

#### JSON 格式化
```bash
go run cmd/mcp-client/main.go -tool json_format -args '{"json":"{\"name\":\"test\",\"value\":123,\"items\":[1,2,3]}"}'
```

#### 随机数生成
```bash
go run cmd/mcp-client/main.go -tool random_number -args '{"min":1,"max":100}'
```

#### URL 编码/解码
```bash
# 编码
go run cmd/mcp-client/main.go -tool url_encode -args '{"text":"hello world","operation":"encode"}'

# 解码
go run cmd/mcp-client/main.go -tool url_encode -args '{"text":"hello%20world","operation":"decode"}'
```

#### 正则表达式匹配
```bash
go run cmd/mcp-client/main.go -tool regex_match -args '{"text":"Hello123World","pattern":"[0-9]+"}'
```

## 3. 自定义工具

在 `cmd/mcp/main.go` 中添加自定义工具：

```go
mcpServer.RegisterTool(
    "my_custom_tool",
    "我的自定义工具描述",
    func(arguments map[string]any) (*mcp.CallToolResult, error) {
        // 获取参数
        param, err := lingechoMCP.SafeGetString(arguments, "param", true)
        if err != nil {
            return lingechoMCP.ErrorResponse(400, err.Error()), nil
        }
        
        // 处理逻辑
        result := "处理结果: " + param
        
        // 返回结果
        return lingechoMCP.SuccessResponse(map[string]interface{}{
            "result": result,
        }), nil
    },
    mcp.WithString(
        "param",
        mcp.Description("参数描述"),
        mcp.Required(),
    ),
)
```

## 4. 集成到主服务器

如果需要将 MCP 服务器集成到 LingEcho 主服务器中，可以在 `cmd/server/main.go` 中添加：

```go
import (
    lingechoMCP "github.com/LingByte/SoulNexus/pkg/mcp"
    "github.com/mark3labs/mcp-go/server"
)

// 在 main 函数中
mcpServer := lingechoMCP.NewMCPServer(&lingechoMCP.Config{
    Name:                   "LingEcho/mcp",
    Version:                "1.0.0",
    Logger:                 log,
    EnableLogging:          true,
    EnableToolCapabilities: true,
})

// 注册工具
lingechoMCP.RegisterDefaultTools(mcpServer)

// 启动 SSE 服务器（在单独的 goroutine 中）
go func() {
    sseServer := server.NewSSEServer(mcpServer.GetServer())
    if err := sseServer.Start(":3001"); err != nil {
        logger.Error("MCP server error", zap.Error(err))
    }
}()
```

## 5. 故障排查

### 服务器无法启动
- 检查端口是否被占用：`lsof -i :3001`
- 检查日志输出中的错误信息

### 客户端连接失败
- 确认服务器已启动
- 检查 URL 是否正确：`http://localhost:3001`
- 检查防火墙设置

### 工具调用失败
- 检查参数格式是否正确（JSON 格式）
- 查看服务器日志中的错误信息
- 使用 `-tool list` 确认工具名称正确

