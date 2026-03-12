package workflow

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os/exec"
	"strings"
	"time"

	"github.com/LingByte/SoulNexus/internal/models"
)

// PluginNode 插件节点
type PluginNode struct {
	Node
	Plugin  *models.NodePlugin       // 插件定义
	Config  map[string]interface{}   // 用户配置
	Runtime models.NodePluginRuntime // 运行时配置
}

func (p *PluginNode) Base() *Node {
	return &p.Node
}

func (p *PluginNode) Run(ctx *WorkflowContext) ([]string, error) {
	ctx.AddLog("info", fmt.Sprintf("开始执行插件节点: %s", p.Plugin.DisplayName), p.ID, p.Name)

	// 准备输入数据
	inputs := make(map[string]interface{})
	for _, port := range p.Plugin.Definition.Inputs {
		if value, exists := ctx.ResolveValue(port.Name); exists {
			inputs[port.Name] = value
		} else if port.Required {
			return nil, fmt.Errorf("必需的输入参数 %s 未提供", port.Name)
		} else if port.Default != nil {
			inputs[port.Name] = port.Default
		}
	}

	// 合并用户配置
	for key, value := range p.Config {
		inputs[key] = value
	}

	// 根据运行时类型执行
	var result map[string]interface{}
	var err error

	switch p.Runtime.Type {
	case "script":
		result, err = p.executeScript(ctx, inputs)
	case "http":
		result, err = p.executeHTTP(ctx, inputs)
	case "builtin":
		result, err = p.executeBuiltin(ctx, inputs)
	default:
		return nil, fmt.Errorf("不支持的运行时类型: %s", p.Runtime.Type)
	}

	if err != nil {
		ctx.AddLog("error", fmt.Sprintf("插件执行失败: %v", err), p.ID, p.Name)
		return nil, err
	}

	// 处理输出数据
	for _, port := range p.Plugin.Definition.Outputs {
		if value, exists := result[port.Name]; exists {
			ctx.SetData(fmt.Sprintf("%s.%s", p.ID, port.Name), value)
		}
	}

	ctx.AddLog("success", "插件执行完成", p.ID, p.Name)
	return p.NextNodes, nil
}

// executeScript 执行脚本类型插件
func (p *PluginNode) executeScript(ctx *WorkflowContext, inputs map[string]interface{}) (map[string]interface{}, error) {
	script, ok := p.Runtime.Config["script"].(string)
	if !ok {
		return nil, fmt.Errorf("脚本配置不存在")
	}

	language, ok := p.Runtime.Config["language"].(string)
	if !ok {
		language = "javascript" // 默认使用JavaScript
	}

	// 准备输入JSON
	inputJSON, err := json.Marshal(inputs)
	if err != nil {
		return nil, fmt.Errorf("序列化输入数据失败: %v", err)
	}

	var cmd *exec.Cmd
	var result []byte

	switch language {
	case "javascript", "js":
		// 使用Node.js执行JavaScript
		fullScript := fmt.Sprintf(`
			const inputs = %s;
			const main = %s;
			const result = main(inputs);
			console.log(JSON.stringify(result));
		`, string(inputJSON), script)

		cmd = exec.Command("node", "-e", fullScript)

	case "python", "py":
		// 使用Python执行
		fullScript := fmt.Sprintf(`
import json
import sys

inputs = %s

%s

if __name__ == "__main__":
    result = main(inputs)
    print(json.dumps(result))
		`, string(inputJSON), script)

		cmd = exec.Command("python3", "-c", fullScript)

	case "go":
		// Go脚本需要特殊处理，这里简化实现
		return nil, fmt.Errorf("Go脚本执行暂未实现")

	default:
		return nil, fmt.Errorf("不支持的脚本语言: %s", language)
	}

	// 设置超时
	timeout := time.Duration(p.Runtime.Timeout) * time.Second
	if timeout <= 0 {
		timeout = 30 * time.Second
	}

	// 执行命令
	ctx.AddLog("info", fmt.Sprintf("执行%s脚本", language), p.ID, p.Name)

	done := make(chan error, 1)
	go func() {
		result, err = cmd.Output()
		done <- err
	}()

	select {
	case err := <-done:
		if err != nil {
			return nil, fmt.Errorf("脚本执行失败: %v", err)
		}
	case <-time.After(timeout):
		cmd.Process.Kill()
		return nil, fmt.Errorf("脚本执行超时")
	}

	// 解析结果
	var output map[string]interface{}
	if err := json.Unmarshal(result, &output); err != nil {
		return nil, fmt.Errorf("解析脚本输出失败: %v", err)
	}

	return output, nil
}

// executeHTTP 执行HTTP类型插件
func (p *PluginNode) executeHTTP(ctx *WorkflowContext, inputs map[string]interface{}) (map[string]interface{}, error) {
	url, ok := p.Runtime.Config["url"].(string)
	if !ok {
		return nil, fmt.Errorf("HTTP URL配置不存在")
	}

	method, ok := p.Runtime.Config["method"].(string)
	if !ok {
		method = "POST"
	}

	// 准备请求数据
	requestData, err := json.Marshal(inputs)
	if err != nil {
		return nil, fmt.Errorf("序列化请求数据失败: %v", err)
	}

	// 创建HTTP请求
	req, err := http.NewRequest(method, url, bytes.NewBuffer(requestData))
	if err != nil {
		return nil, fmt.Errorf("创建HTTP请求失败: %v", err)
	}

	// 设置请求头
	req.Header.Set("Content-Type", "application/json")
	if headers, ok := p.Runtime.Config["headers"].(map[string]interface{}); ok {
		for key, value := range headers {
			if strValue, ok := value.(string); ok {
				req.Header.Set(key, strValue)
			}
		}
	}

	// 设置超时
	timeout := time.Duration(p.Runtime.Timeout) * time.Second
	if timeout <= 0 {
		timeout = 30 * time.Second
	}

	client := &http.Client{Timeout: timeout}

	ctx.AddLog("info", fmt.Sprintf("发送HTTP请求到: %s", url), p.ID, p.Name)

	// 发送请求
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("HTTP请求失败: %v", err)
	}
	defer resp.Body.Close()

	// 读取响应
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取HTTP响应失败: %v", err)
	}

	// 检查状态码
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("HTTP请求失败，状态码: %d, 响应: %s", resp.StatusCode, string(body))
	}

	// 解析响应
	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		// 如果不是JSON，返回原始文本
		result = map[string]interface{}{
			"response":   string(body),
			"statusCode": resp.StatusCode,
		}
	}

	return result, nil
}

// executeBuiltin 执行内置类型插件
func (p *PluginNode) executeBuiltin(ctx *WorkflowContext, inputs map[string]interface{}) (map[string]interface{}, error) {
	handler, ok := p.Runtime.Config["handler"].(string)
	if !ok {
		return nil, fmt.Errorf("内置处理器配置不存在")
	}

	// 这里可以注册各种内置处理器
	switch handler {
	case "data_transform":
		return p.executeDataTransform(ctx, inputs)
	case "text_process":
		return p.executeTextProcess(ctx, inputs)
	case "math_calculate":
		return p.executeMathCalculate(ctx, inputs)
	default:
		return nil, fmt.Errorf("不支持的内置处理器: %s", handler)
	}
}

// executeDataTransform 数据转换处理器
func (p *PluginNode) executeDataTransform(ctx *WorkflowContext, inputs map[string]interface{}) (map[string]interface{}, error) {
	operation, ok := p.Runtime.Config["operation"].(string)
	if !ok {
		return nil, fmt.Errorf("数据转换操作未指定")
	}

	data, exists := inputs["data"]
	if !exists {
		return nil, fmt.Errorf("输入数据不存在")
	}

	var result interface{}

	switch operation {
	case "to_json":
		jsonData, err := json.Marshal(data)
		if err != nil {
			return nil, fmt.Errorf("转换为JSON失败: %v", err)
		}
		result = string(jsonData)

	case "from_json":
		if strData, ok := data.(string); ok {
			var jsonResult interface{}
			if err := json.Unmarshal([]byte(strData), &jsonResult); err != nil {
				return nil, fmt.Errorf("解析JSON失败: %v", err)
			}
			result = jsonResult
		} else {
			return nil, fmt.Errorf("输入数据不是字符串")
		}

	case "to_string":
		result = fmt.Sprintf("%v", data)

	default:
		return nil, fmt.Errorf("不支持的数据转换操作: %s", operation)
	}

	return map[string]interface{}{
		"result": result,
	}, nil
}

// executeTextProcess 文本处理器
func (p *PluginNode) executeTextProcess(ctx *WorkflowContext, inputs map[string]interface{}) (map[string]interface{}, error) {
	operation, ok := p.Runtime.Config["operation"].(string)
	if !ok {
		return nil, fmt.Errorf("文本处理操作未指定")
	}

	text, exists := inputs["text"]
	if !exists {
		return nil, fmt.Errorf("输入文本不存在")
	}

	textStr, ok := text.(string)
	if !ok {
		textStr = fmt.Sprintf("%v", text)
	}

	var result string

	switch operation {
	case "upper":
		result = strings.ToUpper(textStr)
	case "lower":
		result = strings.ToLower(textStr)
	case "trim":
		result = strings.TrimSpace(textStr)
	case "reverse":
		runes := []rune(textStr)
		for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {
			runes[i], runes[j] = runes[j], runes[i]
		}
		result = string(runes)
	default:
		return nil, fmt.Errorf("不支持的文本处理操作: %s", operation)
	}

	return map[string]interface{}{
		"result": result,
		"length": len(result),
	}, nil
}

// executeMathCalculate 数学计算处理器
func (p *PluginNode) executeMathCalculate(ctx *WorkflowContext, inputs map[string]interface{}) (map[string]interface{}, error) {
	operation, ok := p.Runtime.Config["operation"].(string)
	if !ok {
		return nil, fmt.Errorf("数学计算操作未指定")
	}

	a, aExists := inputs["a"]
	b, bExists := inputs["b"]

	if !aExists {
		return nil, fmt.Errorf("输入参数a不存在")
	}

	// 转换为float64
	var aFloat, bFloat float64
	var err error

	if aFloat, err = toFloat64(a); err != nil {
		return nil, fmt.Errorf("参数a不是有效数字: %v", err)
	}

	if bExists {
		if bFloat, err = toFloat64(b); err != nil {
			return nil, fmt.Errorf("参数b不是有效数字: %v", err)
		}
	}

	var result float64

	switch operation {
	case "add":
		if !bExists {
			return nil, fmt.Errorf("加法运算需要参数b")
		}
		result = aFloat + bFloat
	case "subtract":
		if !bExists {
			return nil, fmt.Errorf("减法运算需要参数b")
		}
		result = aFloat - bFloat
	case "multiply":
		if !bExists {
			return nil, fmt.Errorf("乘法运算需要参数b")
		}
		result = aFloat * bFloat
	case "divide":
		if !bExists {
			return nil, fmt.Errorf("除法运算需要参数b")
		}
		if bFloat == 0 {
			return nil, fmt.Errorf("除数不能为零")
		}
		result = aFloat / bFloat
	case "abs":
		if aFloat < 0 {
			result = -aFloat
		} else {
			result = aFloat
		}
	default:
		return nil, fmt.Errorf("不支持的数学运算: %s", operation)
	}

	return map[string]interface{}{
		"result": result,
	}, nil
}

// toFloat64 将interface{}转换为float64
func toFloat64(v interface{}) (float64, error) {
	switch val := v.(type) {
	case float64:
		return val, nil
	case float32:
		return float64(val), nil
	case int:
		return float64(val), nil
	case int32:
		return float64(val), nil
	case int64:
		return float64(val), nil
	case string:
		return 0, fmt.Errorf("字符串不能转换为数字: %s", val)
	default:
		return 0, fmt.Errorf("不支持的数字类型: %T", v)
	}
}
