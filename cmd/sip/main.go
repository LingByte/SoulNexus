package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/LingByte/SoulNexus/internal/sipapp"
	"github.com/LingByte/SoulNexus/pkg/config"
	"github.com/LingByte/SoulNexus/pkg/logger"
	"go.uber.org/zap"
)

func main() {
	host := flag.String("host", "0.0.0.0", "SIP UDP listen host")
	port := flag.Int("port", 5060, "SIP UDP listen port")
	localIP := flag.String("local-ip", "127.0.0.1", "SDP c= line IP (RTP reachable from your phone)")
	flag.Parse()

	logDir := filepath.Join(".", "logs")
	_ = os.MkdirAll(logDir, 0o755)
	_ = logger.Init(&logger.LogConfig{
		Level:      "info",
		Filename:   filepath.Join(logDir, "sip.log"),
		MaxSize:    50,
		MaxAge:     14,
		MaxBackups: 5,
		Daily:      true,
	}, "dev")

	if err := config.Load(); err != nil && logger.Lg != nil {
		logger.Lg.Warn("sip: config.Load failed", zap.Error(err))
	}

	em, err := sipapp.Start(sipapp.Config{
		Host:    *host,
		Port:    *port,
		LocalIP: *localIP,
		DB:      nil,
	})
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "sip: start failed: %v\n", err)
		os.Exit(1)
	}

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	em.Shutdown(ctx)
}
