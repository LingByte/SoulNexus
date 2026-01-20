# LingEcho MCP 模块

这是一个通用的 MCP (Model Context Protocol) 服务器模块，基于 `github.com/mark3labs/mcp-go` 库实现。

## 功能特性

- ✅ 通用的 MCP 服务器封装
- ✅ 支持工具注册和管理
- ✅ 自动错误处理和 panic 恢复
- ✅ 支持 SSE 和 stdio 两种传输方式
- ✅ 集成 LingEcho 日志系统
- ✅ 提供便捷的参数验证辅助函数

## 快速开始

### 1. 创建 MCP 服务器

```go
import (
    "github.com/LingByte/SoulNexus/pkg/mcp"
    "github.com/mark3labs/mcp-go/mcp"
    "go.uber.org/zap"
)

// 创建服务器实例
mcpServer := mcp.NewMCPServer(&mcp.Config{
    Name:                   "LingEcho/mcp",
    Version:                "1.0.0",
    Logger:                 zap.L(),
    EnableLogging:          true,
    EnableToolCapabilities: true,
})
```

### 2. 注册工具

```go
// 注册一个简单的工具
mcpServer.RegisterTool(
    "echo",
    "回显输入的文本",
    func(arguments map[string]any) (*mcp.CallToolResult, error) {
        text, err := mcp.SafeGetString(arguments, "text", true)
        if err != nil {
            return mcp.ErrorResponse(400, err.Error()), nil
        }
        return mcp.TextResponse(text), nil
    },
    mcp.WithString(
        "text",
        mcp.Description("要回显的文本"),
        mcp.Required(),
    ),
)
```

### 3. 启动服务器

#### SSE 模式（HTTP）

```go
import "github.com/mark3labs/mcp-go/server"

sseServer := server.NewSSEServer(mcpServer.GetServer())
if err := sseServer.Start(":3001"); err != nil {
    log.Fatal(err)
}
```

#### stdio 模式

```go
import "github.com/mark3labs/mcp-go/server"

if err := server.ServeStdio(mcpServer.GetServer()); err != nil {
    log.Fatal(err)
}
```

## 辅助函数

### 参数获取

- `SafeGetString(arguments, key, required)` - 安全获取字符串参数
- `SafeGetNumber(arguments, key, required, defaultValue)` - 安全获取数字参数
- `SafeGetBool(arguments, key, required, defaultValue)` - 安全获取布尔参数

### 响应创建

- `ErrorResponse(code, message, details...)` - 创建错误响应
- `SuccessResponse(data)` - 创建成功响应（带数据）
- `TextResponse(text)` - 创建简单文本响应

## 默认工具

服务器默认注册了以下通用工具：

1. **echo** - 回显输入的文本
2. **calculator** - 执行数学表达式计算（+、-、*、/、%、^）
3. **get_current_time** - 获取当前时间，支持多种格式和时区
4. **text_process** - 文本处理（大小写转换、反转、统计、去空格）
5. **json_format** - JSON 格式化与验证
6. **random_number** - 生成指定范围内的随机数
7. **url_encode** - URL 编码/解码
8. **regex_match** - 正则表达式匹配

## 客户端使用

### 命令行客户端

项目提供了命令行客户端示例 `cmd/mcp-client/main.go`：

```bash
# 列出所有可用工具
go run cmd/mcp-client/main.go -tool list

# 调用 echo 工具
go run cmd/mcp-client/main.go -tool echo -args '{"text":"Hello World"}'

# 调用计算器工具
go run cmd/mcp-client/main.go -tool calculator -args '{"expression":"2+2*3"}'

# 获取当前时间
go run cmd/mcp-client/main.go -tool get_current_time -args '{"format":"2006-01-02 15:04:05","timezone":"Asia/Shanghai"}'

# 文本处理
go run cmd/mcp-client/main.go -tool text_process -args '{"text":"Hello World","operation":"uppercase"}'

# JSON 格式化
go run cmd/mcp-client/main.go -tool json_format -args '{"json":"{\"name\":\"test\",\"value\":123}"}'
```

### 编程方式使用

```go
import (
    "context"
    "github.com/mark3labs/mcp-go/client"
    "github.com/mark3labs/mcp-go/client/transport"
    "github.com/mark3labs/mcp-go/mcp"
)

// 创建客户端（注意：URL 需要包含 /sse 路径）
httpTransport, _ := transport.NewSSE("http://localhost:3001/sse")
mcpClient := client.NewClient(httpTransport)
mcpClient.Start(context.Background())
defer mcpClient.Close()

// 初始化
initRequest := mcp.InitializeRequest{}
initRequest.Params.ProtocolVersion = mcp.LATEST_PROTOCOL_VERSION
initRequest.Params.Capabilities = mcp.ClientCapabilities{}
mcpClient.Initialize(context.Background(), initRequest)

// 调用工具
request := mcp.CallToolRequest{}
request.Params.Name = "echo"
request.Params.Arguments = map[string]any{"text": "Hello"}
result, _ := mcpClient.CallTool(context.Background(), request)
```

## 完整示例

参考 `cmd/mcp/main.go` 查看完整的独立服务器示例。

## 与 voicefox-mcp-releases 的区别

本模块是通用的 MCP 实现，不包含任何特定业务逻辑（如停车场、水务等特定服务）。你可以：

1. 使用 `RegisterTool` 方法注册自己的工具
2. 使用 `RegisterDefaultTools` 注册默认工具集
3. 使用辅助函数简化参数验证和响应创建
4. 集成到 LingEcho 主服务器中，或作为独立服务运行

## 注意事项

- 工具处理函数会自动捕获 panic 并返回友好错误
- 所有工具调用都会记录日志（Debug 级别）
- 建议使用提供的辅助函数进行参数验证，避免类型错误
- 计算器工具支持基本运算，复杂表达式建议使用专门的表达式解析库

