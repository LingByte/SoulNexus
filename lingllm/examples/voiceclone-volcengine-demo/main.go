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
	godotenv.Load()

	appID := flag.String("app-id", envOr("VOLCENGINE_APP_ID", "5525383952"), "火山引擎 App ID")
	token := flag.String("token", envOr("VOLCENGINE_TOKEN", ""), "火山引擎 Access Token")
	audioFile := flag.String("audio", "input.wav", "训练音频路径（wav 16kHz mono 推荐）")
	speakerID := flag.String("speaker-id", "", "控制台创建的 Speaker ID，如 S_xxx")
	trainText := flag.String("train-text", "喂您好，听得到我说话吗？", "训练音频对应的念诵文本")
	synthesizeText := flag.String("text", "喂您好，听得到我说话吗？", "合成测试文本")
	outputFile := flag.String("output", "output.wav", "合成输出路径")
	resourceID := flag.String("resource-id", "seed-icl-1.0", "Resource-Id: seed-icl-1.0 / seed-icl-2.0")
	modelType := flag.Int("model-type", 1, "训练 model_type: 1=ICL1.0, 4=ICL2.0, 5=ICL3.0")
	queryOnly := flag.Bool("query-only", false, "仅查询训练状态")
	flag.Parse()

	if *token == "" {
		fmt.Println("错误: 需要 -token 或环境变量 VOLCENGINE_TOKEN")
		os.Exit(1)
	}
	if *speakerID == "" {
		fmt.Println("错误: 需要 -speaker-id（在火山控制台「声音复刻」里创建/购买音色后获得 S_ 开头 ID）")
		os.Exit(1)
	}

	ctx := context.Background()
	service := voiceclone.NewVolcengineCloneService(voiceclone.VolcengineCloneConfig{
		AppID:      *appID,
		Token:      *token,
		Cluster:    "volcano_icl",
		ResourceID: *resourceID,
		ModelType:  *modelType,
		Encoding:   "pcm",
		SampleRate: 16000,
		SpeedRatio: 1.0,
	})
	fmt.Printf("✓ 火山克隆服务 app=%s speaker=%s resource=%s model_type=%d\n",
		*appID, *speakerID, *resourceID, *modelType)

	if *queryOnly {
		queryTrainingStatus(ctx, service, *speakerID)
		return
	}

	if *audioFile == "" {
		fmt.Printf("\n=== 直接合成模式 ===\n")
		synthesizeAndSave(ctx, service, *speakerID, *synthesizeText, *outputFile)
		return
	}

	fmt.Printf("\n=== 训练流程 ===\n")
	fmt.Printf("1. 上传训练音频...\n")
	fmt.Printf("   文件: %s\n", *audioFile)
	fmt.Printf("   念诵文本: %s\n", *trainText)

	f, err := os.Open(*audioFile)
	if err != nil {
		fmt.Printf("✗ 打开音频失败: %v\n", err)
		os.Exit(1)
	}
	defer f.Close()

	if err := service.SubmitAudio(ctx, &voiceclone.SubmitAudioRequest{
		TaskID:    fmt.Sprintf("speaker_id:%s:wav", *speakerID),
		AudioFile: f,
		Language:  "zh",
		TrainText: *trainText,
	}); err != nil {
		fmt.Printf("✗ 提交音频失败: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("✓ 音频提交成功\n")

	fmt.Printf("\n2. 等待训练完成...\n")
	queryTrainingStatus(ctx, service, *speakerID)

	fmt.Printf("\n3. 合成测试...\n")
	synthesizeAndSave(ctx, service, *speakerID, *synthesizeText, *outputFile)

	fmt.Printf("\n✓ 完成！voice-demo 配置示例:\n")
	fmt.Printf(`export TTS_PROVIDER=volcengine_clone
export TTS_CONFIG_JSON='{
  "provider": "volcengine_clone",
  "appId": "%s",
  "accessToken": "<your-token>",
  "cluster": "volcano_icl",
  "resourceId": "%s",
  "modelType": %d,
  "assetId": "%s",
  "encoding": "pcm",
  "sampleRate": 16000,
  "streaming": true
}'
`, *appID, *resourceID, *modelType, *speakerID)
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func queryTrainingStatus(ctx context.Context, service voiceclone.VoiceCloneService, speakerID string) {
	const maxAttempts = 60
	for i := 0; i < maxAttempts; i++ {
		status, err := service.QueryTaskStatus(ctx, speakerID)
		if err != nil {
			fmt.Printf("✗ 查询失败: %v\n", err)
			os.Exit(1)
		}
		switch status.Status {
		case voiceclone.TrainingStatusSuccess:
			fmt.Printf("✓ 训练成功 speaker=%s\n", status.AssetID)
			return
		case voiceclone.TrainingStatusFailed:
			fmt.Printf("✗ 训练失败: %s\n", status.FailedDesc)
			os.Exit(1)
		case voiceclone.TrainingStatusQueued, voiceclone.TrainingStatusInProgress:
			fmt.Printf("  [%d/%d] 训练中...\n", i+1, maxAttempts)
		}
		time.Sleep(2 * time.Second)
	}
	fmt.Printf("⚠ 训练超时，稍后可用 -query-only -speaker-id %s 再查\n", speakerID)
}

func synthesizeAndSave(ctx context.Context, service voiceclone.VoiceCloneService, speakerID, text, outputPath string) {
	fmt.Printf("   合成文本: %s\n", text)
	resp, err := service.Synthesize(ctx, &voiceclone.SynthesizeRequest{
		AssetID:  speakerID,
		Text:     text,
		Language: "zh",
	})
	if err != nil {
		fmt.Printf("✗ 合成失败: %v\n", err)
		os.Exit(1)
	}
	dir := filepath.Dir(outputPath)
	if dir != "." && dir != "" {
		_ = os.MkdirAll(dir, 0755)
	}
	if err := os.WriteFile(outputPath, resp.AudioData, 0644); err != nil {
		fmt.Printf("✗ 保存失败: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("✓ 已保存 %s (%d bytes, %d Hz)\n", outputPath, len(resp.AudioData), resp.SampleRate)
}
