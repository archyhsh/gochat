package main

import (
	"context"
	"flag"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/archyhsh/gochat/internal/gateway/server"
	"github.com/archyhsh/gochat/pkg/config"
	"github.com/archyhsh/gochat/pkg/logger"
)

func main() {
	configPath := flag.String("config", "configs/gateway.yaml", "config file path")
	flag.Parse()

	// 加载配置
	cfg := config.MustLoad(*configPath)

	// 初始化日志
	log := logger.New(cfg.Log.Level, cfg.Log.Format)
	log.Info("Starting Gateway Server",
		"name", cfg.Server.Name,
		"addr", cfg.Server.Addr,
	)

	// 创建服务
	srv := server.New(cfg, log)

	// 静态文件服务 (前端演示页面)
	fs := http.FileServer(http.Dir("web/static"))
	mux := http.NewServeMux()
	mux.Handle("/", fs)
	mux.Handle("/health", srv.Router())
	mux.Handle("/ws", srv.Router())
	mux.Handle("/api/", srv.Router())

	// HTTP 服务器
	httpServer := &http.Server{
		Addr:         cfg.Server.Addr,
		Handler:      mux,
		ReadTimeout:  time.Duration(cfg.Server.ReadTimeout) * time.Second,
		WriteTimeout: time.Duration(cfg.Server.WriteTimeout) * time.Second,
	}

	// 启动服务
	go func() {
		log.Info("HTTP server listening", "addr", cfg.Server.Addr)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal("Failed to start server", "error", err)
		}
	}()

	// 优雅关闭
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	srv.Shutdown()
	if err := httpServer.Shutdown(ctx); err != nil {
		log.Error("Server forced to shutdown", "error", err)
	}

	log.Info("Server exited")
}
