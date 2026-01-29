package main

import (
	"context"
	"flag"
	"os"
	"os/signal"
	"syscall"

	"github.com/archyhsh/gochat/internal/message/consumer"
	"github.com/archyhsh/gochat/internal/message/service"
	"github.com/archyhsh/gochat/pkg/config"
	"github.com/archyhsh/gochat/pkg/db"
	"github.com/archyhsh/gochat/pkg/kafka"
	"github.com/archyhsh/gochat/pkg/logger"
)

func main() {
	configPath := flag.String("config", "configs/message.yaml", "config file path")
	flag.Parse()

	cfg := config.MustLoad(*configPath)
	log := logger.New(cfg.Log.Level, cfg.Log.Format)
	log.Info("Starting Message Service",
		"name", cfg.Server.Name,
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
	msgService := service.NewMessageService(database, log)
	msgConsumer := consumer.NewMessageConsumer(msgService, log)
	topics := []string{cfg.Kafka.Topics.Message}
	kafkaConsumer, err := kafka.NewConsumer(
		cfg.Kafka.Brokers,
		cfg.Kafka.ConsumerGroup,
		topics,
		msgConsumer,
	)
	if err != nil {
		log.Fatal("Failed to create Kafka consumer", "error", err)
	}
	log.Info("Kafka consumer created",
		"brokers", cfg.Kafka.Brokers,
		"group", cfg.Kafka.ConsumerGroup,
		"topics", topics,
	)
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		if err := kafkaConsumer.Start(ctx); err != nil {
			log.Error("Kafka consumer error", "error", err)
		}
	}()
	log.Info("Message Service started, waiting for messages...")
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info("Shutting down Message Service...")
	cancel()
	kafkaConsumer.Close()
	log.Info("Message Service stopped")
}
