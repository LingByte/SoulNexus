package main

import (
	"github.com/LingByte/SoulNexus/pkg/logger"
	sip1 "github.com/LingByte/SoulNexus/pkg/sip"
	"go.uber.org/zap"
)

func main() {
	// 初始化 logger，避免 LLMHandler 内部调用时 nil panic
	logger.Lg, _ = zap.NewDevelopment()

	server, err := sip1.NewSipServer(10000, 5060, nil)
	if err != nil {
		panic(err)
	}
	defer server.Close()
	server.Start()
	select {}
}
