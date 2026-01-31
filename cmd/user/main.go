package main

import (
	"context"
	"flag"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/archyhsh/gochat/internal/user/handler"
	"github.com/archyhsh/gochat/internal/user/service"
	"github.com/archyhsh/gochat/pkg/auth"
	"github.com/archyhsh/gochat/pkg/config"
	"github.com/archyhsh/gochat/pkg/db"
	"github.com/archyhsh/gochat/pkg/logger"
	"github.com/archyhsh/gochat/pkg/response"
	"github.com/gorilla/mux"
)

func main() {
	configPath := flag.String("config", "configs/user.yaml", "config file path")
	flag.Parse()
	cfg := config.MustLoad(*configPath)
	log := logger.New(cfg.Log.Level, cfg.Log.Format)
	log.Info("Starting User Service",
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
	userService := service.NewUserService(database, jwtManager, log)
	userHandler := handler.NewUserHandler(userService, jwtManager, log)
	router := mux.NewRouter()
	router.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		response.Success(w, map[string]string{"status": "ok"})
	}).Methods("GET")
	router.HandleFunc("/api/v1/register", userHandler.Register).Methods("POST")
	router.HandleFunc("/api/v1/login", userHandler.Login).Methods("POST")
	router.HandleFunc("/api/v1/users/search", userHandler.SearchUsers).Methods("GET")
	router.HandleFunc("/api/v1/users/{id:[0-9]+}", userHandler.GetUser).Methods("GET")
	authRouter := router.PathPrefix("/api/v1").Subrouter()
	authRouter.Use(handler.AuthMiddleware(jwtManager))
	authRouter.HandleFunc("/user/me", userHandler.GetCurrentUser).Methods("GET")
	authRouter.HandleFunc("/user/me", userHandler.UpdateUser).Methods("PUT")

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
	log.Info("Shutting down User Service...")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := httpServer.Shutdown(ctx); err != nil {
		log.Error("Server forced to shutdown", "error", err)
	}
	log.Info("User Service stopped")
}
