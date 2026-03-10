package config

import (
	"fmt"
	"os"
	"strconv"
)

type Config struct {
	Env               string
	PostgresURL       string
	GatewayPort       int
	CoreGRPCPort      int
	MatchingGRPCPort  int
	CoreGRPCAddr      string
	MatchingGRPCAddr  string
	ServiceRoleSecret string
}

func Load() (*Config, error) {
	cfg := &Config{
		Env:               getEnv("APP_ENV", "dev"),
		PostgresURL:       os.Getenv("POSTGRES_URL"),
		GatewayPort:       getEnvInt("GATEWAY_PORT", 8080),
		CoreGRPCPort:      getEnvInt("CORE_GRPC_PORT", 9091),
		MatchingGRPCPort:  getEnvInt("MATCHING_GRPC_PORT", 9092),
		CoreGRPCAddr:      getEnv("CORE_GRPC_ADDR", "127.0.0.1:9091"),
		MatchingGRPCAddr:  getEnv("MATCHING_GRPC_ADDR", "127.0.0.1:9092"),
		ServiceRoleSecret: os.Getenv("SERVICE_ROLE_SECRET"),
	}

	if cfg.PostgresURL == "" {
		return nil, fmt.Errorf("POSTGRES_URL is required")
	}
	return cfg, nil
}

func getEnv(key, fallback string) string {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	return v
}

func getEnvInt(key string, fallback int) int {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return fallback
	}
	return n
}
