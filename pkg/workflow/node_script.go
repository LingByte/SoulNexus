package workflow

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"text/template"
	"time"

	"github.com/LingByte/SoulNexus/pkg/workflow/libs"
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
				ctx.AddLog("info", fmt.Sprintf("Input: %s", string(inputJSON)), s.ID, s.Name)
			}
		}
	} else {
		fmt.Printf("%s\n", message)
	}

	// 检查脚本是否为空
	if strings.TrimSpace(s.Script) == "" {
		errMsg := "Script is empty"
		if ctx != nil {
			ctx.AddLog("error", errMsg, s.ID, s.Name)
		}
		return nil, fmt.Errorf(errMsg)
	}

	runtime := s.Runtime
	if runtime == nil {
		runtime = defaultGoScriptRuntime
	}

	if ctx != nil {
		ctx.AddLog("info", "Starting script runtime...", s.ID, s.Name)
	}

	result, err := runtime(ctx, s.Script, inputs)
	if err != nil {
		errMsg := fmt.Sprintf("Script execution failed: %s", err.Error())
		if ctx != nil {
			ctx.AddLog("error", errMsg, s.ID, s.Name)
		}
		return nil, err
	}
	if result == nil {
		result = map[string]interface{}{}
	}

	// Log output parameters with their values
	if ctx != nil && len(result) > 0 {
		outputJSON, err := json.Marshal(result)
		if err == nil {
			ctx.AddLog("info", fmt.Sprintf("Output: %s", string(outputJSON)), s.ID, s.Name)
		}
		ctx.AddLog("success", "Script executed successfully", s.ID, s.Name)
	} else if ctx != nil {
		ctx.AddLog("success", "Script executed successfully (no output)", s.ID, s.Name)
	}

	return result, nil
}

func (s *ScriptNode) Base() *Node {
	return &s.Node
}

func (s *ScriptNode) Run(ctx *WorkflowContext) ([]string, error) {
	inputs, err := s.Node.PrepareInputs(ctx)
	if err != nil {
		if ctx != nil {
			ctx.AddLog("error", fmt.Sprintf("Failed to prepare inputs: %v", err), s.ID, s.Name)
		}
		return nil, err
	}
	result, err := s.ExecuteScript(ctx, inputs)
	if err != nil {
		if ctx != nil {
			ctx.AddLog("error", fmt.Sprintf("Script execution error: %v", err), s.ID, s.Name)
		}
		return nil, err
	}
	s.Node.PersistOutputs(ctx, result)
	if ctx != nil {
		ctx.AddLog("success", fmt.Sprintf("Script node completed, next nodes: %v", s.NextNodes), s.ID, s.Name)
	}
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
		// 等待读取完成（带超时）
		select {
		case <-outputDone:
			// 读取完成
		case <-time.After(1 * time.Second):
			// 超时，继续
		}
		// 关闭读端
		rOut.Close()
		rErr.Close()
	}()

	// 现在创建 interp，此时 os.Stdout 已经被重定向
	i := interp.New(interp.Options{})
	i.Use(stdlib.Symbols)

	// 执行脚本 - 使用带日志函数和库的包装
	wrapped := wrapScriptWithLibraries(script, ctx)
	if _, err := i.Eval(wrapped); err != nil {
		// 检查是否是小写库名称的错误
		errMsg := err.Error()
		if strings.Contains(errMsg, "undefined: stringLib") ||
			strings.Contains(errMsg, "undefined: mathLib") ||
			strings.Contains(errMsg, "undefined: timeLib") ||
			strings.Contains(errMsg, "undefined: cryptoLib") ||
			strings.Contains(errMsg, "undefined: regexLib") ||
			strings.Contains(errMsg, "undefined: validationLib") ||
			strings.Contains(errMsg, "undefined: arrayLib") ||
			strings.Contains(errMsg, "undefined: httpLib") ||
			strings.Contains(errMsg, "cannot convert") && strings.Contains(errMsg, "main.stringLib") ||
			strings.Contains(errMsg, "cannot convert") && strings.Contains(errMsg, "main.mathLib") ||
			strings.Contains(errMsg, "cannot convert") && strings.Contains(errMsg, "main.timeLib") ||
			strings.Contains(errMsg, "cannot convert") && strings.Contains(errMsg, "main.cryptoLib") ||
			strings.Contains(errMsg, "cannot convert") && strings.Contains(errMsg, "main.regexLib") ||
			strings.Contains(errMsg, "cannot convert") && strings.Contains(errMsg, "main.validationLib") ||
			strings.Contains(errMsg, "cannot convert") && strings.Contains(errMsg, "main.arrayLib") ||
			strings.Contains(errMsg, "cannot convert") && strings.Contains(errMsg, "main.httpLib") {
			return nil, fmt.Errorf("script evaluation failed: library names must be UPPERCASE (e.g., StringLib, MathLib, TimeLib, ValidationLib, HttpLib, etc.). Error: %w", err)
		}

		// 如果注入失败，尝试使用原始脚本
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

// wrapScriptWithLibraries 包装脚本并注入库对象和日志函数
func wrapScriptWithLibraries(src string, ctx *WorkflowContext) string {
	userScript := removePackageAndImports(src)

	// 获取所有库的代码
	libsCode := libs.GetAllLibsCode()

	// 脚本模板 - 使用 //go:embed 嵌入的库代码
	scriptTemplate := `package main
import (
	"crypto/md5"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"
)

{{.LibsCode}}

func log(args ...interface{}) {
	for _, arg := range args {
		fmt.Print(arg)
	}
	fmt.Println()
}

{{.UserScript}}
`

	// 注意：库实例已在 .gox 文件中定义为大写变量（StringLib, MathLib 等）
	// 用户脚本应该使用大写的库名称，例如：StringLib.ToUpper("hello")

	tmpl, err := template.New("script").Parse(scriptTemplate)
	if err != nil {
		return src
	}

	var buf bytes.Buffer
	err = tmpl.Execute(&buf, map[string]interface{}{
		"LibsCode":   libsCode,
		"UserScript": userScript,
	})
	if err != nil {
		return src
	}

	return buf.String()
}

// removePackageAndImports 移除脚本中的 package 和 import 声明
func removePackageAndImports(src string) string {
	lines := strings.Split(src, "\n")
	var result []string
	inImport := false
	skipNextLines := 0

	for i, line := range lines {
		trimmedLine := strings.TrimSpace(line)

		// 跳过 package 声明
		if strings.HasPrefix(trimmedLine, "package ") {
			continue
		}

		// 跳过 import 块
		if strings.HasPrefix(trimmedLine, "import") {
			if strings.Contains(trimmedLine, "(") {
				inImport = true
			}
			continue
		}

		if inImport {
			if strings.Contains(trimmedLine, ")") {
				inImport = false
			}
			continue
		}

		// 跳过空行（在开头）
		if i < len(lines)-1 && trimmedLine == "" && skipNextLines < 5 {
			skipNextLines++
			continue
		}

		result = append(result, line)
	}

	return strings.Join(result, "\n")
}
