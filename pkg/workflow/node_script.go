package workflow

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/traefik/yaegi/interp"
	"github.com/traefik/yaegi/stdlib"
)

// ScriptNode allows custom scripted logic
type ScriptNode struct {
	Node
	Script        string
	Runtime       func(ctx *WorkflowContext, script string, inputs map[string]interface{}) (map[string]interface{}, error)
	LastResultKey string
}

func (s *ScriptNode) ExecuteScript(ctx *WorkflowContext, inputs map[string]interface{}) (map[string]interface{}, error) {
	message := fmt.Sprintf("Executing script node %s", s.Name)
	if ctx != nil {
		ctx.AddLog("info", message, s.ID, s.Name)
		// Log input parameters with their values
		if len(inputs) > 0 {
			inputJSON, err := json.Marshal(inputs)
			if err == nil {
				ctx.AddLog("debug", fmt.Sprintf("Script inputs: %s", string(inputJSON)), s.ID, s.Name)
			} else {
				ctx.AddLog("debug", fmt.Sprintf("Script inputs: %d parameter(s)", len(inputs)), s.ID, s.Name)
			}
		}
	} else {
		fmt.Printf("%s\n", message)
	}
	runtime := s.Runtime
	if runtime == nil {
		runtime = defaultGoScriptRuntime
	}
	result, err := runtime(ctx, s.Script, inputs)
	if err != nil {
		ctx.AddLog("error", fmt.Sprintf("Script execution failed: %s", err.Error()), s.ID, s.Name)
		return nil, err
	}
	if result == nil {
		result = map[string]interface{}{}
	}

	// Log output parameters with their values
	if ctx != nil && len(result) > 0 {
		outputJSON, err := json.Marshal(result)
		if err == nil {
			ctx.AddLog("debug", fmt.Sprintf("Script outputs: %s", string(outputJSON)), s.ID, s.Name)
		} else {
			ctx.AddLog("debug", fmt.Sprintf("Script outputs: %d result(s)", len(result)), s.ID, s.Name)
		}
		ctx.AddLog("success", "Script executed successfully", s.ID, s.Name)
	}

	return result, nil
}

func (s *ScriptNode) Base() *Node {
	return &s.Node
}

func (s *ScriptNode) Run(ctx *WorkflowContext) ([]string, error) {
	inputs, err := s.Node.PrepareInputs(ctx)
	if err != nil {
		return nil, err
	}
	result, err := s.ExecuteScript(ctx, inputs)
	if err != nil {
		return nil, err
	}
	s.Node.PersistOutputs(ctx, result)
	return s.NextNodes, nil
}

// CustomWriter 是一个自定义的 Writer，用于捕获脚本输出
type CustomWriter struct {
	buf    *bytes.Buffer
	ctx    *WorkflowContext
	prefix string
}

func (cw *CustomWriter) Write(p []byte) (n int, err error) {
	n = len(p)
	if cw.buf != nil {
		cw.buf.Write(p)
	}
	// 实时发送到日志
	if cw.ctx != nil && len(p) > 0 {
		text := string(p)
		lines := strings.Split(strings.TrimRight(text, "\n"), "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line != "" {
				// Get current node ID and name from context if available
				nodeID := cw.ctx.CurrentNode
				nodeName := ""
				if nodeID != "" {
					// Try to get node name from context if available
					nodeName = "Script"
				}
				cw.ctx.AddLog("info", fmt.Sprintf("[Script %s] %s", cw.prefix, line), nodeID, nodeName)
			}
		}
	}
	return n, nil
}

func defaultGoScriptRuntime(ctx *WorkflowContext, script string, inputs map[string]interface{}) (map[string]interface{}, error) {
	// 创建自定义 Writer 来捕获输出
	var stdoutBuf bytes.Buffer
	var stderrBuf bytes.Buffer
	stdoutWriter := &CustomWriter{buf: &stdoutBuf, ctx: ctx, prefix: "Output"}
	stderrWriter := &CustomWriter{buf: &stderrBuf, ctx: ctx, prefix: "Error"}

	// 保存原始输出
	originalStdout := os.Stdout
	originalStderr := os.Stderr

	// 创建管道来捕获输出
	rOut, wOut, _ := os.Pipe()
	rErr, wErr, _ := os.Pipe()

	// 在 goroutine 中读取并处理输出
	outputDone := make(chan struct{})
	go func() {
		defer close(outputDone)
		// 从管道读取并写入到自定义 Writer
		io.Copy(stdoutWriter, rOut)
		io.Copy(stderrWriter, rErr)
	}()

	// 在创建 interp 之前设置标准输出，确保 yaegi 使用重定向的 stdout
	os.Stdout = wOut
	os.Stderr = wErr

	// 确保在函数返回前恢复标准输出
	defer func() {
		// 先关闭写端，触发读取完成
		wOut.Close()
		wErr.Close()
		// 恢复标准输出
		os.Stdout = originalStdout
		os.Stderr = originalStderr
		// 等待读取完成
		<-outputDone
		// 关闭读端
		rOut.Close()
		rErr.Close()
	}()

	// 现在创建 interp，此时 os.Stdout 已经被重定向
	i := interp.New(interp.Options{})
	i.Use(stdlib.Symbols)

	// 执行脚本 - 使用带日志函数的包装
	wrapped := wrapScriptWithLog(script, ctx)
	if _, err := i.Eval(wrapped); err != nil {
		// 如果注入日志函数失败，尝试使用原始脚本
		wrapped = wrapScript(script)
		if _, err2 := i.Eval(wrapped); err2 != nil {
			return nil, fmt.Errorf("script evaluation failed: %w (original: %v)", err2, err)
		}
	}

	v, err := i.Eval("Run")
	if err != nil {
		return nil, fmt.Errorf("script must define Run function: %w", err)
	}
	runFunc, ok := v.Interface().(func(map[string]interface{}) (map[string]interface{}, error))
	if !ok {
		return nil, fmt.Errorf("Run function signature mismatch")
	}

	// 执行 Run 函数
	result, err := runFunc(inputs)

	// 确保所有缓冲的输出都被刷新
	wOut.Sync()
	wErr.Sync()

	// 确保所有输出都被处理
	// 由于使用了 CustomWriter，输出已经实时发送到日志了

	return result, err
}

func wrapScript(src string) string {
	trimmed := strings.TrimSpace(src)
	if strings.HasPrefix(trimmed, "package") {
		return trimmed
	}
	builder := strings.Builder{}
	builder.WriteString("package main\n")
	builder.WriteString(trimmed)
	return builder.String()
}

// wrapScriptWithLog 包装脚本并注入日志函数
func wrapScriptWithLog(src string, ctx *WorkflowContext) string {
	trimmed := strings.TrimSpace(src)

	// 注入日志函数的代码
	logFuncCode := `
// 注入的日志函数，用于替代 fmt.Println
func log(args ...interface{}) {
	for _, arg := range args {
		fmt.Print(arg)
	}
	fmt.Println()
}
`

	builder := strings.Builder{}
	if strings.HasPrefix(trimmed, "package") {
		// 如果已经有 package 声明，在 package 后添加日志函数
		lines := strings.Split(trimmed, "\n")
		builder.WriteString(lines[0] + "\n")
		builder.WriteString(logFuncCode)
		for i := 1; i < len(lines); i++ {
			builder.WriteString(lines[i] + "\n")
		}
	} else {
		builder.WriteString("package main\n")
		builder.WriteString(logFuncCode)
		builder.WriteString(trimmed)
	}
	return builder.String()
}
