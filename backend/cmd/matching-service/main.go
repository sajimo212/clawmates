package main

import (
	"fmt"
	"net"

	matchingv1 "clawmates/backend/gen/matching/v1"
	"clawmates/backend/internal/config"
	"clawmates/backend/internal/logging"
	transportgrpc "clawmates/backend/internal/transport/grpc"

	"go.uber.org/zap"
	"google.golang.org/grpc"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		panic(err)
	}

	logger, err := logging.New(cfg.Env)
	if err != nil {
		panic(err)
	}
	defer func() { _ = logger.Sync() }()

	lis, err := net.Listen("tcp", ":"+itoa(cfg.MatchingGRPCPort))
	if err != nil {
		logger.Fatal("listen matching grpc", zap.Error(err), zap.Int("port", cfg.MatchingGRPCPort))
	}

	s := grpc.NewServer()
	matchingv1.RegisterMatchingServiceServer(s, transportgrpc.NewMatchingServer())
	logger.Info("matching-service listening", zap.Int("port", cfg.MatchingGRPCPort))

	if err := s.Serve(lis); err != nil {
		logger.Fatal("serve matching grpc", zap.Error(err))
	}
}

func itoa(v int) string {
	return fmt.Sprintf("%d", v)
}
