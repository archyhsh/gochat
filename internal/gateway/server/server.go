package server

import (
	"net/http"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"

	"github.com/yourusername/gochat/internal/gateway/connection"
	"github.com/yourusername/gochat/internal/gateway/handler"
	"github.com/yourusername/gochat/pkg/auth"
	"github.com/yourusername/gochat/pkg/config"
	"github.com/yourusername/gochat/pkg/logger"
	"github.com/yourusername/gochat/pkg/response"
)

// Server Gateway 服务器
type Server struct {
	config     *config.Config
	logger     logger.Logger
	manager    *connection.Manager
	jwtManager *auth.JWTManager
	upgrader   websocket.Upgrader
	router     *mux.Router
}

// New 创建 Gateway 服务器
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
				// TODO: 在生产环境中实现更严格的检查
				return true
			},
		},
	}

	// 设置消息处理器
	msgHandler := handler.NewMessageHandler(s.manager, log)
	s.manager.SetMessageHandler(msgHandler)

	// 启动连接管理器
	s.manager.Start()

	// 初始化路由
	s.initRouter()

	return s
}

// initRouter 初始化路由
func (s *Server) initRouter() {
	s.router = mux.NewRouter()

	// 健康检查
	s.router.HandleFunc("/health", s.healthHandler).Methods("GET")

	// WebSocket 连接
	s.router.HandleFunc("/ws", s.wsHandler).Methods("GET")

	// 状态接口
	s.router.HandleFunc("/api/v1/stats", s.statsHandler).Methods("GET")
}

// Router 返回路由器
func (s *Server) Router() http.Handler {
	return s.router
}

// healthHandler 健康检查
func (s *Server) healthHandler(w http.ResponseWriter, r *http.Request) {
	response.Success(w, map[string]string{
		"status": "ok",
	})
}

// wsHandler WebSocket 连接处理
func (s *Server) wsHandler(w http.ResponseWriter, r *http.Request) {
	// 获取 token
	token := r.URL.Query().Get("token")
	if token == "" {
		response.Unauthorized(w, "token required")
		return
	}

	// 验证 token
	claims, err := s.jwtManager.ParseToken(token)
	if err != nil {
		s.logger.Warn("Invalid token", "error", err)
		response.Unauthorized(w, "invalid token")
		return
	}

	// 获取设备信息
	deviceID := r.URL.Query().Get("device_id")
	if deviceID == "" {
		deviceID = uuid.New().String()
	}
	platform := r.URL.Query().Get("platform")
	if platform == "" {
		platform = "web"
	}

	// 升级连接
	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		s.logger.Error("Failed to upgrade connection", "error", err)
		return
	}

	// 创建连接对象
	connID := uuid.New().String()
	c := connection.NewConnection(connID, claims.UserID, deviceID, platform, conn, s.manager)

	// 注册连接
	s.manager.Register(c)

	// 启动读写协程
	go c.WritePump()
	go c.ReadPump()

	s.logger.Info("New WebSocket connection",
		"connID", connID,
		"userID", claims.UserID,
		"platform", platform,
	)
}

// statsHandler 统计信息
func (s *Server) statsHandler(w http.ResponseWriter, r *http.Request) {
	stats := map[string]interface{}{
		"online_users":      s.manager.GetOnlineUserCount(),
		"total_connections": s.manager.GetConnectionCount(),
	}
	response.Success(w, stats)
}

// Shutdown 关闭服务器
func (s *Server) Shutdown() {
	s.logger.Info("Shutting down gateway server...")
	s.manager.Shutdown()
}
