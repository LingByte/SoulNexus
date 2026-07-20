// ASR Denoiser Demo: 演示如何在 ASR 管道中使用降噪器
//
// 本示例展示了两种降噪实现的使用方法：
//  1. Simple Denoiser - 轻量级降噪 (AEC + AGC)
//  2. RNNoise Denoiser - 高质量神经网络降噪 (需要 rnnoise build tag)
//
// 使用方法:
//
//	# 使用 Simple 降噪器
//	go run ./examples/voice-demo/asr-denoise-demo
//
//	# 使用 RNNoise 降噪器 (需要安装 librnnoise)
//	go run -tags rnnoise ./examples/voice-demo/asr-denoise-demo -denoiser rnnoise
package main

import (
	"context"
	"flag"
	"fmt"
	"log"

	"github.com/LingByte/lingllm/protocol/voice/asr"
)

func main() {
	denoiserType := flag.String("denoiser", "simple", "降噪器类型: none, simple, rnnoise")
	flag.Parse()

	// 创建降噪器工厂
	factory := asr.NewDenoiserFactory()

	// 显示可用的降噪器类型
	fmt.Println("=== 可用的降噪器类型 ===")
	availableTypes := factory.GetAvailableDenoiserTypes()
	for _, t := range availableTypes {
		fmt.Printf("  - %s\n", t)
	}
	fmt.Println()

	// 根据用户选择创建降噪器
	fmt.Printf("使用降噪器: %s\n", *denoiserType)

	var component interface{}
	var err error

	switch *denoiserType {
	case "none":
		component, err = factory.CreateDenoiser(asr.DenoiserTypeNone, nil)
		if err != nil {
			log.Fatalf("创建无降噪组件失败: %v", err)
		}
		fmt.Println("✓ 无降噪组件创建成功")

	case "simple":
		config := &asr.SimpleDenoiserConfig{
			AECEnable:     true,
			AGCEnable:     true,
			SampleRate:    16000,
			Channels:      1,
			BitsPerSample: 16,
		}
		component, err = factory.CreateDenoiser(asr.DenoiserTypeSimple, config)
		if err != nil {
			log.Fatalf("创建 Simple 降噪组件失败: %v", err)
		}
		fmt.Println("✓ Simple 降噪组件创建成功")
		fmt.Println("  - AEC (回声消除): 启用")
		fmt.Println("  - AGC (自动增益控制): 启用")
		fmt.Println("  - 采样率: 16000 Hz")
		fmt.Println("  - 声道数: 1")
		fmt.Println("  - 位深: 16-bit")

	case "rnnoise":
		component, err = factory.CreateDenoiser(asr.DenoiserTypeRNNoise, nil)
		if err != nil {
			log.Fatalf("创建 RNNoise 降噪组件失败: %v", err)
		}
		fmt.Println("✓ RNNoise 降噪组件创建成功")
		fmt.Println("  - 采样率: 48000 Hz (固定)")
		fmt.Println("  - 声道数: 1 (固定)")
		fmt.Println("  - 位深: 16-bit (固定)")
		fmt.Println("  - 帧大小: 960 bytes (480 samples)")

	default:
		log.Fatalf("未知的降噪器类型: %s", *denoiserType)
	}

	fmt.Println()

	// 演示处理 PCM 数据
	if component != nil {
		fmt.Println("=== 演示音频处理 ===")
		demonstrateProcessing(component)
	}

	// 清理资源
	if component != nil {
		if closer, ok := component.(interface{ Close() error }); ok {
			if err := closer.Close(); err != nil {
				log.Printf("关闭组件失败: %v", err)
			} else {
				fmt.Println("\n✓ 组件已正确关闭")
			}
		}
	}
}

// demonstrateProcessing 演示音频处理
func demonstrateProcessing(component interface{}) {
	// 创建模拟的 PCM 数据 (1024 字节 = 512 个 16-bit 样本)
	pcmData := make([]byte, 1024)
	for i := 0; i < len(pcmData); i++ {
		pcmData[i] = byte(i % 256)
	}

	fmt.Printf("输入数据大小: %d 字节\n", len(pcmData))

	// 处理数据
	ctx := context.Background()
	output, ok, err := component.(interface {
		Process(ctx interface{}, data interface{}) (interface{}, bool, error)
	}).Process(ctx, pcmData)

	if err != nil {
		log.Printf("处理失败: %v", err)
		return
	}

	if !ok {
		fmt.Println("处理被中断")
		return
	}

	outputData, ok := output.([]byte)
	if !ok {
		fmt.Printf("输出类型错误: %T\n", output)
		return
	}

	fmt.Printf("输出数据大小: %d 字节\n", len(outputData))
	fmt.Println("✓ 音频处理成功")
}

// 在 ASR 管道中使用降噪器的示例
func exampleASRPipeline() {
	fmt.Println("\n=== ASR 管道集成示例 ===")
	fmt.Print(`
// 1. 创建降噪器工厂
factory := asr.NewDenoiserFactory()

// 2. 创建 Simple 降噪器组件
config := &asr.SimpleDenoiserConfig{
    AECEnable:     true,
    AGCEnable:     true,
    SampleRate:    16000,
    Channels:      1,
    BitsPerSample: 16,
}
denoiserComponent, err := factory.CreateDenoiser(asr.DenoiserTypeSimple, config)
if err != nil {
    log.Fatal(err)
}
defer denoiserComponent.(*asr.SimpleDenoiserComponent).Close()

// 3. 在 ASR 引擎中使用
engine := asr.NewEngine()
engine.AddComponent(denoiserComponent.(*asr.SimpleDenoiserComponent))
engine.AddComponent(vad)
engine.AddComponent(recognizer)
// ...
`)
}
