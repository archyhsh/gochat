package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/yourusername/gochat/internal/gateway/server"
	"github.com/yourusername/gochat/pkg/config"
	"github.com/yourusername/gochat/pkg/logger"
)

var (
	configFile = flag.String("config", "configs/gateway.yaml", "config file path")
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
	log.Info("Starting Gateway Service", "version", version)

	// 创建 WebSocket 服务器
	srv := server.New(cfg, log)

	// 启动 HTTP 服务器
	httpServer := &http.Server{
		Addr:         cfg.Server.Addr,
		Handler:      srv.Router(),
		ReadTimeout:  time.Duration(cfg.Server.ReadTimeout) * time.Second,
		WriteTimeout: time.Duration(cfg.Server.WriteTimeout) * time.Second,
	}

	// 启动服务
	go func() {
		log.Info("Gateway listening", "addr", cfg.Server.Addr)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error("Failed to start server", "error", err)
			os.Exit(1)
		}
	}()

	// 优雅关闭
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// 关闭所有连接
	srv.Shutdown()

	if err := httpServer.Shutdown(ctx); err != nil {
		log.Error("Server forced to shutdown", "error", err)
	}

	log.Info("Server exited")
}
