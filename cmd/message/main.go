package main

import (
	"context"
	"flag"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gorilla/mux"

	"github.com/archyhsh/gochat/internal/message/consumer"
	"github.com/archyhsh/gochat/internal/message/handler"
	"github.com/archyhsh/gochat/internal/message/service"
	"github.com/archyhsh/gochat/pkg/auth"
	"github.com/archyhsh/gochat/pkg/config"
	"github.com/archyhsh/gochat/pkg/db"
	"github.com/archyhsh/gochat/pkg/kafka"
	"github.com/archyhsh/gochat/pkg/logger"
	"github.com/archyhsh/gochat/pkg/response"
)

func main() {
	configPath := flag.String("config", "configs/message.yaml", "config file path")
	flag.Parse()

	cfg := config.MustLoad(*configPath)
	log := logger.New(cfg.Log.Level, cfg.Log.Format)
	log.Info("Starting Message Service",
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
	msgService := service.NewMessageService(database, log)
	msgHandler := handler.NewMessageHandler(msgService, log)
	msgConsumer := consumer.NewMessageConsumer(msgService, log)
	topics := []string{cfg.Kafka.Topics.Message}
	kafkaConsumer, err := kafka.NewConsumer(
		cfg.Kafka.Brokers,
		cfg.Kafka.ConsumerGroup,
		topics,
		msgConsumer,
	)
	if err != nil {
		log.Warn("Failed to create Kafka consumer, message persistence disabled",
			"error", err,
		)
	} else {
		log.Info("Kafka consumer created",
			"brokers", cfg.Kafka.Brokers,
			"group", cfg.Kafka.ConsumerGroup,
			"topics", topics,
		)
	}
	router := mux.NewRouter()
	router.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		response.Success(w, map[string]string{"status": "ok"})
	}).Methods("GET")
	api := router.PathPrefix("/api/v1").Subrouter()
	api.Use(handler.AuthMiddleware(jwtManager))
	api.HandleFunc("/messages", msgHandler.GetMessages).Methods("GET")
	api.HandleFunc("/messages/{msg_id}", msgHandler.GetMessageByID).Methods("GET")
	api.HandleFunc("/conversations", msgHandler.GetConversations).Methods("GET")
	api.HandleFunc("/conversations/read", msgHandler.ClearUnread).Methods("PUT")
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
	ctx, cancel := context.WithCancel(context.Background())
	if kafkaConsumer != nil {
		go func() {
			if err := kafkaConsumer.Start(ctx); err != nil {
				log.Error("Kafka consumer error", "error", err)
			}
		}()
	}
	go func() {
		log.Info("HTTP server listening", "addr", cfg.Server.Addr)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal("Failed to start server", "error", err)
		}
	}()

	log.Info("Message Service started")

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info("Shutting down Message Service...")

	cancel()
	if kafkaConsumer != nil {
		kafkaConsumer.Close()
	}
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		log.Error("Server forced to shutdown", "error", err)
	}
	log.Info("Message Service stopped")
}
