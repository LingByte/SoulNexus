package config

import "github.com/LingByte/SoulNexus/pkg/utils"

type CacheConfig struct {
	Port int `env:"CACHE_PORT"`
}

func LoadCacheConfig() (*CacheConfig, error) {
	return &CacheConfig{
		Port: int(utils.GetIntEnv("CACHE_PORT")),
	}, nil
}
