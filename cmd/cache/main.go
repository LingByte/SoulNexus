package main

import (
	"fmt"

	"github.com/LingByte/SoulNexus/internal/config"
	"github.com/gin-gonic/gin"
)

func main() {
	cacheConfig, err := config.LoadCacheConfig()
	if err != nil {
		panic(fmt.Sprintf("load cache config error %v", err))
	}
	engine := gin.Default()
	engine.Run(fmt.Sprintf(":%d", cacheConfig.Port))
}
