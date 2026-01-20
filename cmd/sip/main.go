package main

import (
	"flag"
	"log"
	"os"

	"github.com/LingByte/SoulNexus/cmd/bootstrap"
	"github.com/LingByte/SoulNexus/pkg/config"
	"github.com/LingByte/SoulNexus/pkg/logger"
	"github.com/LingByte/SoulNexus/pkg/sip"
	"go.uber.org/zap"
)

func main() {
	// Print Banner
	if err := bootstrap.PrintBannerFromFile("banner.txt"); err != nil {
		log.Fatalf("unload banner: %v", err)
	}
	// Load Global Configuration
	if err := config.Load(); err != nil {
		panic("config load failed: " + err.Error())
	}
	// Load Log Configuration
	err := logger.Init(&config.GlobalConfig.Log, config.GlobalConfig.Mode)
	if err != nil {
		panic(err)
	}

	initSQL := flag.String("init-sql", "", "path to database init .sql script (optional)")
	db, err := bootstrap.SetupDatabase(os.Stdout, &bootstrap.Options{
		InitSQLPath: *initSQL,                             // Can be specified via --init-sql
		AutoMigrate: false,                                // Whether to migrate entities
		SeedNonProd: os.Getenv("APP_ENV") != "production", // Non-production default configuration
	})
	if err != nil {
		logger.Error("database setup failed", zap.Error(err))
		return
	}

	server := sip.NewSipServer(10000)
	server.SetDBConfig(db)
	defer server.Close()
	// 检查命令行参数，如果有 -call 参数则发起呼叫
	if len(os.Args) > 2 && os.Args[1] == "-call" {
		targetURI := os.Args[2]
		log.Printf("准备发起呼叫到: %s", targetURI)
		server.Start(5060, targetURI)
	}
	server.Start(5060, "")
}
