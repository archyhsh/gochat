package main

import (
	"context"
	"flag"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/archyhsh/gochat/internal/relation/handler"
	"github.com/archyhsh/gochat/internal/relation/service"
	"github.com/archyhsh/gochat/pkg/auth"
	"github.com/archyhsh/gochat/pkg/config"
	"github.com/archyhsh/gochat/pkg/db"
	"github.com/archyhsh/gochat/pkg/kafka"
	"github.com/archyhsh/gochat/pkg/logger"
	"github.com/archyhsh/gochat/pkg/response"
	"github.com/gorilla/mux"
)

func main() {
	configPath := flag.String("config", "configs/relation.yaml", "config file path")
	flag.Parse()
	cfg := config.MustLoad(*configPath)
	log := logger.New(cfg.Log.Level, cfg.Log.Format)
	log.Info("Starting Relation Service",
		"name", cfg.Server.Name,
		"addr", cfg.Server.Addr,
	)
	dbConfig := &db.Config{
		Host:            cfg.MySQL.Host,
		Port:            cfg.MySQL.Port,
		User:            cfg.MySQL.User,
		Password:        cfg.MySQL.Password,
		Database:        cfg.MySQL.Database,
		MaxOpenConns:    cfg.MySQL.MaxOpenConns,
		MaxIdleConns:    cfg.MySQL.MaxIdleConns,
		ConnMaxLifetime: cfg.MySQL.ConnMaxLifetime,
	}
	database, err := db.NewMySQL(dbConfig)
	if err != nil {
		log.Fatal("Failed to connect to MySQL", "error", err)
	}
	log.Info("Connected to MySQL")
	jwtManager := auth.NewJWTManager(cfg.JWT.Secret, cfg.JWT.ExpireTime)
	relationService := service.NewRelationService(database, log)
	relationHandler := handler.NewRelationHandler(relationService, log)

	// 创建 Kafka producer 用于发布关系变更事件
	var kafkaProducer *kafka.Producer
	if len(cfg.Kafka.Brokers) > 0 {
		producer, err := kafka.NewProducer(cfg.Kafka.Brokers, cfg.Kafka.Topics.Relation)
		if err != nil {
			log.Warn("Failed to create Kafka producer, event publishing disabled", "error", err)
		} else {
			kafkaProducer = producer
			relationService.SetProducer(producer)
			log.Info("Kafka producer created", "brokers", cfg.Kafka.Brokers, "topic", cfg.Kafka.Topics.Relation)
		}
	}

	router := mux.NewRouter()
	router.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		response.Success(w, map[string]string{"status": "ok"})
	}).Methods("GET")
	api := router.PathPrefix("/api/v1").Subrouter()
	api.Use(handler.AuthMiddleware(jwtManager))
	api.HandleFunc("/friend/apply", relationHandler.Apply).Methods("POST")
	api.HandleFunc("/friend/apply/handle", relationHandler.HandleApply).Methods("POST")
	api.HandleFunc("/friend/apply/list", relationHandler.GetApplyList).Methods("GET")
	api.HandleFunc("/friends", relationHandler.GetFriendList).Methods("GET")
	api.HandleFunc("/friends/{id:[0-9]+}", relationHandler.DeleteFriend).Methods("DELETE")
	api.HandleFunc("/friends/{id:[0-9]+}/block", relationHandler.BlockFriend).Methods("POST")
	api.HandleFunc("/friends/{id:[0-9]+}/unblock", relationHandler.UnblockFriend).Methods("POST")
	api.HandleFunc("/friends/remark", relationHandler.UpdateRemark).Methods("PUT")

	corsHandler := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusOK)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
	httpServer := &http.Server{
		Addr:         cfg.Server.Addr,
		Handler:      corsHandler(router),
		ReadTimeout:  time.Duration(cfg.Server.ReadTimeout) * time.Second,
		WriteTimeout: time.Duration(cfg.Server.WriteTimeout) * time.Second,
	}

	go func() {
		log.Info("HTTP server listening", "addr", cfg.Server.Addr)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal("Failed to start server", "error", err)
		}
	}()
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Info("Shutting down Relation Service...")
	if kafkaProducer != nil {
		kafkaProducer.Close()
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := httpServer.Shutdown(ctx); err != nil {
		log.Error("Server forced to shutdown", "error", err)
	}
	log.Info("Relation Service stopped")
}
