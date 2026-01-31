package server

import (
	"context"
	"net/http"
	"strconv"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"gorm.io/gorm"

	"github.com/archyhsh/gochat/internal/gateway/connection"
	"github.com/archyhsh/gochat/internal/gateway/consumer"
	"github.com/archyhsh/gochat/internal/gateway/handler"
	"github.com/archyhsh/gochat/internal/gateway/service"
	"github.com/archyhsh/gochat/pkg/auth"
	"github.com/archyhsh/gochat/pkg/config"
	"github.com/archyhsh/gochat/pkg/db"
	"github.com/archyhsh/gochat/pkg/kafka"
	"github.com/archyhsh/gochat/pkg/logger"
	"github.com/archyhsh/gochat/pkg/response"
)

type Server struct {
	config                 *config.Config
	logger                 logger.Logger
	db                     *gorm.DB
	manager                *connection.Manager
	jwtManager             *auth.JWTManager
	producer               *kafka.Producer
	relationChecker        *service.RelationChecker
	relationConsumerCancel context.CancelFunc
	upgrader               websocket.Upgrader
	router                 *mux.Router
}

func New(cfg *config.Config, log logger.Logger) *Server {
	s := &Server{
		config:     cfg,
		logger:     log,
		manager:    connection.NewManager(log),
		jwtManager: auth.NewJWTManager(cfg.JWT.Secret, cfg.JWT.ExpireTime),
		upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			CheckOrigin: func(r *http.Request) bool {
				// TODO: stricter restrictions for origin website => need a whitelist or somewhat
				return true
			},
		},
	}

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
		log.Warn("Failed to connect to MySQL, friend check will be disabled",
			"error", err,
		)
	} else {
		s.db = database
		s.relationChecker = service.NewRelationChecker(database, log)
		log.Info("MySQL connected for relation check")
	}
	producer, err := kafka.NewProducer(cfg.Kafka.Brokers, cfg.Kafka.Topics.Message)
	if err != nil {
		log.Warn("Failed to create Kafka producer, messages will not be persisted",
			"error", err,
		)
	} else {
		s.producer = producer
		log.Info("Kafka producer initialized",
			"brokers", cfg.Kafka.Brokers,
			"topic", cfg.Kafka.Topics.Message,
		)
	}
	if len(cfg.Kafka.Brokers) > 0 && cfg.Kafka.Topics.Relation != "" {
		relationConsumer := consumer.NewRelationEventConsumer(s.manager, log)
		kafkaRelationConsumer, err := kafka.NewConsumer(
			cfg.Kafka.Brokers,
			"gateway-relation-consumer",
			[]string{cfg.Kafka.Topics.Relation},
			relationConsumer,
		)
		if err != nil {
			log.Warn("Failed to create relation event consumer", "error", err)
		} else {
			// Start relation event consumer in background
			ctx, cancel := context.WithCancel(context.Background())
			go func() {
				if err := kafkaRelationConsumer.Start(ctx); err != nil {
					log.Error("Relation event consumer error", "error", err)
				}
			}()
			log.Info("Relation event consumer started", "topic", cfg.Kafka.Topics.Relation)
			// Store cancel function for graceful shutdown
			s.relationConsumerCancel = cancel
		}
	}

	msgHandler := handler.NewMessageHandler(s.manager, s.producer, s.relationChecker, log)
	s.manager.SetMessageHandler(msgHandler)
	s.manager.Start()
	s.initRouter()
	return s
}

func (s *Server) initRouter() {
	s.router = mux.NewRouter()
	s.router.HandleFunc("/health", s.healthHandler).Methods("GET")
	s.router.HandleFunc("/ws", s.wsHandler).Methods("GET")
	s.router.HandleFunc("/api/v1/stats", s.statsHandler).Methods("GET")
	s.router.HandleFunc("/api/v1/test/token", s.testTokenHandler).Methods("GET")
}

func (s *Server) Router() http.Handler {
	return s.router
}

func (s *Server) healthHandler(w http.ResponseWriter, r *http.Request) {
	response.Success(w, map[string]string{
		"status": "ok",
	})
}

func (s *Server) wsHandler(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")
	if token == "" {
		response.Unauthorized(w, "token required")
		return
	}
	claims, err := s.jwtManager.ParseToken(token)
	if err != nil {
		s.logger.Warn("Invalid token", "error", err)
		response.Unauthorized(w, "invalid token")
		return
	}
	deviceID := r.URL.Query().Get("device_id")
	if deviceID == "" {
		deviceID = uuid.New().String()
	}
	platform := r.URL.Query().Get("platform")
	if platform == "" {
		platform = "web"
	}
	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		s.logger.Error("Failed to upgrade connection", "error", err)
		return
	}
	connID := uuid.New().String()
	c := connection.NewConnection(connID, claims.UserID, deviceID, platform, conn, s.manager)
	s.manager.Register(c)
	go c.WritePump()
	go c.ReadPump()
	s.logger.Info("New WebSocket connection",
		"connID", connID,
		"userID", claims.UserID,
		"platform", platform,
	)
}

func (s *Server) statsHandler(w http.ResponseWriter, r *http.Request) {
	stats := map[string]interface{}{
		"online_users":      s.manager.GetOnlineUserCount(),
		"total_connections": s.manager.GetConnectionCount(),
	}
	response.Success(w, stats)
}

func (s *Server) Shutdown() {
	s.logger.Info("Shutting down gateway server...")
	if s.relationConsumerCancel != nil {
		s.relationConsumerCancel()
	}
	s.manager.Shutdown()
	if s.producer != nil {
		s.producer.Close()
	}
}

// testTokenHandler 生成测试 Token (仅用于开发测试)
func (s *Server) testTokenHandler(w http.ResponseWriter, r *http.Request) {
	userIDStr := r.URL.Query().Get("user_id")
	username := r.URL.Query().Get("username")

	if userIDStr == "" {
		userIDStr = "1"
	}
	if username == "" {
		username = "test_user"
	}

	userID, err := strconv.ParseInt(userIDStr, 10, 64)
	if err != nil {
		response.BadRequest(w, "invalid user_id")
		return
	}

	token, err := s.jwtManager.GenerateToken(userID, username)
	if err != nil {
		response.ServerError(w, "failed to generate token")
		return
	}

	response.Success(w, map[string]interface{}{
		"token":    token,
		"user_id":  userID,
		"username": username,
	})
}
