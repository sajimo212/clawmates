package main

import (
	"context"
	"fmt"
	"net"
	"time"

	corev1 "clawmates/backend/gen/core/v1"
	matchingv1 "clawmates/backend/gen/matching/v1"
	"clawmates/backend/internal/config"
	"clawmates/backend/internal/db"
	"clawmates/backend/internal/logging"
	"clawmates/backend/internal/repository"
	"clawmates/backend/internal/service"
	transportgrpc "clawmates/backend/internal/transport/grpc"

	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
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

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	pool, err := db.NewPool(ctx, cfg.PostgresURL)
	if err != nil {
		logger.Fatal("connect postgres", zap.Error(err))
	}
	defer pool.Close()

	matchingConn, err := grpc.NewClient(cfg.MatchingGRPCAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		logger.Fatal("dial matching grpc", zap.Error(err), zap.String("addr", cfg.MatchingGRPCAddr))
	}
	defer matchingConn.Close()

	repo := repository.NewPostgresRepository(pool)
	coreService := service.NewCoreService(repo, matchingv1.NewMatchingServiceClient(matchingConn))
	coreServer := transportgrpc.NewCoreServer(coreService)

	lis, err := net.Listen("tcp", ":"+itoa(cfg.CoreGRPCPort))
	if err != nil {
		logger.Fatal("listen core grpc", zap.Error(err), zap.Int("port", cfg.CoreGRPCPort))
	}

	s := grpc.NewServer()
	corev1.RegisterCoreServiceServer(s, coreServer)
	logger.Info("core-service listening", zap.Int("port", cfg.CoreGRPCPort))

	if err := s.Serve(lis); err != nil {
		logger.Fatal("serve core grpc", zap.Error(err))
	}
}

func itoa(v int) string {
	return fmt.Sprintf("%d", v)
}
