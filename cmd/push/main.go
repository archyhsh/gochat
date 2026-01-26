package main

import (
	"context"
	"flag"
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"

	"google.golang.org/grpc"

	"github.com/yourusername/gochat/pkg/config"
	"github.com/yourusername/gochat/pkg/logger"
)

var (
	configFile = flag.String("config", "configs/push.yaml", "config file path")
	version    = "1.0.0"
)

func main() {
	flag.Parse()

	// 初始化配置
	cfg, err := config.Load(*configFile)
	if err != nil {
		fmt.Printf("Failed to load config: %v\n", err)
		os.Exit(1)
	}

	// 初始化日志
	log := logger.New(cfg.Log.Level, cfg.Log.Format)
	log.Info("Starting Push Service", "version", version)

	// 创建 gRPC 服务器
	grpcServer := grpc.NewServer()

	// TODO: 注册推送服务
	// pb.RegisterPushServiceServer(grpcServer, pushService)

	// 启动 gRPC 服务
	lis, err := net.Listen("tcp", cfg.Server.GRPCAddr)
	if err != nil {
		log.Error("Failed to listen", "error", err)
		os.Exit(1)
	}

	go func() {
		log.Info("Push Service listening", "addr", cfg.Server.GRPCAddr)
		if err := grpcServer.Serve(lis); err != nil {
			log.Error("Failed to serve", "error", err)
			os.Exit(1)
		}
	}()

	// 优雅关闭
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info("Shutting down server...")

	_, cancel := context.WithTimeout(context.Background(), 30)
	defer cancel()

	grpcServer.GracefulStop()

	log.Info("Server exited")
}
