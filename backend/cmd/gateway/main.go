package main

import (
	"fmt"
	"net/http"
	"time"

	corev1 "clawmates/backend/gen/core/v1"
	"clawmates/backend/internal/config"
	"clawmates/backend/internal/logging"
	httptransport "clawmates/backend/internal/transport/http"

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

	coreConn, err := grpc.NewClient(cfg.CoreGRPCAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		logger.Fatal("dial core grpc", zap.Error(err), zap.String("addr", cfg.CoreGRPCAddr))
	}
	defer coreConn.Close()

	gateway := httptransport.NewGateway(corev1.NewCoreServiceClient(coreConn), cfg.ServiceRoleSecret)
	handler := gateway.Handler()

	srv := &http.Server{
		Addr:              ":" + itoa(cfg.GatewayPort),
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
	}

	logger.Info("gateway listening", zap.Int("port", cfg.GatewayPort))
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		logger.Fatal("serve gateway", zap.Error(err))
	}
}

func itoa(v int) string {
	return fmt.Sprintf("%d", v)
}
