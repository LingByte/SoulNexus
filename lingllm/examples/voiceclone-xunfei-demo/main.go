package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/LingByte/lingllm/voiceclone"
	"github.com/joho/godotenv"
)

func main() {
	// 加载环境变量
	godotenv.Load()

	// 命令行参数
	appID := flag.String("app-id", os.Getenv("XUNFEI_APP_ID"), "讯飞 App ID")
	apiKey := flag.String("api-key", os.Getenv("XUNFEI_API_KEY"), "讯飞 API Key")
	audioFile := flag.String("audio", "", "音频文件路径（用于训练）")
	taskName := flag.String("task-name", "voice-clone-demo", "训练任务名称")
	synthesizeText := flag.String("text", "你好，这是一个音色克隆的演示。", "要合成的文本")
	assetID := flag.String("asset-id", "", "已训练的音色 ID（用于合成，如果提供则跳过训练）")
	outputFile := flag.String("output", "output.wav", "输出音频文件路径")
	flag.Parse()

	// 验证必需参数
	if *appID == "" || *apiKey == "" {
		fmt.Println("错误: 需要提供 XUNFEI_APP_ID 和 XUNFEI_API_KEY")
		fmt.Println("可以通过环境变量或命令行参数 -app-id 和 -api-key 提供")
		os.Exit(1)
	}

	ctx := context.Background()

	// 创建讯飞克隆服务
	config := voiceclone.XunfeiCloneConfig{
		AppID:         *appID,
		APIKey:        *apiKey,
		EngineVersion: "omni_v1",  // 多风格版
		VCN:           "x6_clone", // 多风格克隆音色
	}

	service := voiceclone.NewXunfeiCloneService(config)
	fmt.Printf("✓ 已创建讯飞克隆服务\n")

	// 如果提供了 assetID，直接进行合成
	if *assetID != "" {
		fmt.Printf("\n=== 直接合成模式 ===\n")
		fmt.Printf("使用已训练的音色 ID: %s\n", *assetID)
		synthesizeAndSave(ctx, service, *assetID, *synthesizeText, *outputFile)
		return
	}

	// 否则进行完整的训练流程
	if *audioFile == "" {
		fmt.Println("错误: 需要提供音频文件路径（-audio）或已训练的音色 ID（-asset-id）")
		os.Exit(1)
	}

	fmt.Printf("\n=== 训练流程 ===\n")

	// 1. 创建训练任务
	fmt.Printf("\n1. 创建训练任务...\n")
	createReq := &voiceclone.CreateTaskRequest{
		TaskName:      *taskName,
		Sex:           1, // 1=男, 2=女
		AgeGroup:      2, // 1=儿童, 2=青年, 3=中年, 4=中老年
		Language:      "zh",
		ResourceType:  12, // 12=一句话复刻
		EngineVersion: "omni_v1",
		Denoise:       1,   // 开启降噪
		MosRatio:      0.5, // 音频质量检测阈值
	}

	taskResp, err := service.CreateTask(ctx, createReq)
	if err != nil {
		fmt.Printf("✗ 创建任务失败: %v\n", err)
		os.Exit(1)
	}
	taskID := taskResp.TaskID
	fmt.Printf("✓ 任务创建成功，任务 ID: %s\n", taskID)

	// 2. 获取训练文本
	fmt.Printf("\n2. 获取训练文本...\n")
	// 讯飞的训练文本 ID（这里使用示例 ID，实际需要从讯飞获取）
	textID := int64(1) // 示例文本 ID
	trainingText, err := service.GetTrainingTexts(ctx, textID)
	if err != nil {
		fmt.Printf("⚠ 获取训练文本失败: %v\n", err)
		fmt.Printf("  继续使用默认文本进行训练\n")
	} else {
		fmt.Printf("✓ 获取训练文本成功\n")
		fmt.Printf("  文本 ID: %d, 名称: %s\n", trainingText.TextID, trainingText.TextName)
		if len(trainingText.Segments) > 0 {
			fmt.Printf("  文本段落数: %d\n", len(trainingText.Segments))
			for i, seg := range trainingText.Segments {
				fmt.Printf("    [%d] %v: %s\n", i, seg.SegID, seg.SegText)
			}
		}
	}

	// 3. 提交音频文件
	fmt.Printf("\n3. 提交音频文件进行训练...\n")
	audioFileHandle, err := os.Open(*audioFile)
	if err != nil {
		fmt.Printf("✗ 打开音频文件失败: %v\n", err)
		os.Exit(1)
	}
	defer audioFileHandle.Close()

	submitReq := &voiceclone.SubmitAudioRequest{
		TaskID:    taskID,
		TextID:    textID,
		TextSegID: 1, // 使用第一个文本段落
		AudioFile: audioFileHandle,
		Language:  "zh",
		MosRatio:  0.5,
	}

	if err := service.SubmitAudio(ctx, submitReq); err != nil {
		fmt.Printf("✗ 提交音频失败: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("✓ 音频提交成功\n")

	// 4. 轮询查询训练状态
	fmt.Printf("\n4. 查询训练状态...\n")
	maxAttempts := 60 // 最多轮询 60 次
	pollInterval := 2 * time.Second
	var finalAssetID string

	for i := 0; i < maxAttempts; i++ {
		status, err := service.QueryTaskStatus(ctx, taskID)
		if err != nil {
			fmt.Printf("✗ 查询状态失败: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("  [%d/%d] 状态: ", i+1, maxAttempts)
		switch status.Status {
		case voiceclone.TrainingStatusQueued:
			fmt.Printf("排队中 (进度: %.1f%%)\n", status.Progress)
		case voiceclone.TrainingStatusInProgress:
			fmt.Printf("训练中 (进度: %.1f%%)\n", status.Progress)
		case voiceclone.TrainingStatusSuccess:
			fmt.Printf("✓ 训练成功\n")
			finalAssetID = status.AssetID
			fmt.Printf("  音色 ID: %s\n", status.AssetID)
			fmt.Printf("  音库 ID: %s\n", status.TrainVID)
			break
		case voiceclone.TrainingStatusFailed:
			fmt.Printf("✗ 训练失败\n")
			fmt.Printf("  失败原因: %s\n", status.FailedDesc)
			os.Exit(1)
		}

		if status.Status == voiceclone.TrainingStatusSuccess {
			break
		}

		if i < maxAttempts-1 {
			time.Sleep(pollInterval)
		}
	}

	if finalAssetID == "" {
		fmt.Printf("✗ 训练超时，未能获取音色 ID\n")
		os.Exit(1)
	}

	// 5. 使用训练好的音色进行合成
	fmt.Printf("\n5. 使用训练好的音色进行合成...\n")
	synthesizeAndSave(ctx, service, finalAssetID, *synthesizeText, *outputFile)

	fmt.Printf("\n✓ 完整流程执行成功！\n")
	fmt.Printf("  音色 ID: %s (可用于后续合成)\n", finalAssetID)
}

func synthesizeAndSave(ctx context.Context, service voiceclone.VoiceCloneService, assetID, text, outputPath string) {
	fmt.Printf("  文本: %s\n", text)

	synthesizeReq := &voiceclone.SynthesizeRequest{
		AssetID:  assetID,
		Text:     text,
		Language: "zh",
	}

	resp, err := service.Synthesize(ctx, synthesizeReq)
	if err != nil {
		fmt.Printf("✗ 合成失败: %v\n", err)
		os.Exit(1)
	}

	// 保存音频文件
	outputDir := filepath.Dir(outputPath)
	if outputDir != "." && outputDir != "" {
		os.MkdirAll(outputDir, 0755)
	}

	if err := os.WriteFile(outputPath, resp.AudioData, 0644); err != nil {
		fmt.Printf("✗ 保存音频文件失败: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✓ 合成成功\n")
	fmt.Printf("  输出文件: %s\n", outputPath)
	fmt.Printf("  格式: %s, 采样率: %d Hz, 时长: %.2f 秒\n",
		resp.Format, resp.SampleRate, resp.Duration)
}
