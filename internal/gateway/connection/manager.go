package connection

import (
	"encoding/json"
	"errors"
	"sync"

	"github.com/yourusername/gochat/pkg/logger"
)

var (
	ErrSendBufferFull     = errors.New("send buffer full")
	ErrConnectionNotFound = errors.New("connection not found")
	ErrUserNotOnline      = errors.New("user not online")
)

// Manager 连接管理器
type Manager struct {
	connections    sync.Map // connID -> *Connection
	userConns      sync.Map // userID -> sync.Map[connID]*Connection
	register       chan *Connection
	unregister     chan *Connection
	broadcast      chan *BroadcastMessage
	messageHandler MessageHandler
	logger         logger.Logger
	closeChan      chan struct{}
	closeOnce      sync.Once
}

// BroadcastMessage 广播消息
type BroadcastMessage struct {
	UserIDs []int64 // 目标用户列表，为空则广播给所有人
	Data    []byte
}

// MessageHandler 消息处理接口
type MessageHandler interface {
	Handle(conn *Connection, data []byte) error
}

// NewManager 创建连接管理器
func NewManager(log logger.Logger) *Manager {
	m := &Manager{
		register:   make(chan *Connection, 256),
		unregister: make(chan *Connection, 256),
		broadcast:  make(chan *BroadcastMessage, 256),
		logger:     log,
		closeChan:  make(chan struct{}),
	}
	return m
}

// SetMessageHandler 设置消息处理器
func (m *Manager) SetMessageHandler(handler MessageHandler) {
	m.messageHandler = handler
}

// Start 启动管理器
func (m *Manager) Start() {
	go m.run()
}

// run 运行管理器主循环
func (m *Manager) run() {
	for {
		select {
		case conn := <-m.register:
			m.handleRegister(conn)

		case conn := <-m.unregister:
			m.handleUnregister(conn)

		case msg := <-m.broadcast:
			m.handleBroadcast(msg)

		case <-m.closeChan:
			m.handleShutdown()
			return
		}
	}
}

// handleRegister 处理连接注册
func (m *Manager) handleRegister(conn *Connection) {
	// 存储连接
	m.connections.Store(conn.ID, conn)

	// 存储用户连接映射（支持多端登录）
	userConnsInterface, _ := m.userConns.LoadOrStore(conn.UserID, &sync.Map{})
	userConns := userConnsInterface.(*sync.Map)
	userConns.Store(conn.ID, conn)

	m.logger.Info("Connection registered",
		"connID", conn.ID,
		"userID", conn.UserID,
		"platform", conn.Platform,
	)
}

// handleUnregister 处理连接注销
func (m *Manager) handleUnregister(conn *Connection) {
	if _, ok := m.connections.LoadAndDelete(conn.ID); ok {
		// 从用户连接映射中删除
		if userConnsInterface, ok := m.userConns.Load(conn.UserID); ok {
			userConns := userConnsInterface.(*sync.Map)
			userConns.Delete(conn.ID)

			// 检查用户是否还有其他连接
			empty := true
			userConns.Range(func(key, value interface{}) bool {
				empty = false
				return false
			})
			if empty {
				m.userConns.Delete(conn.UserID)
			}
		}

		// 关闭发送 channel
		close(conn.Send)

		m.logger.Info("Connection unregistered",
			"connID", conn.ID,
			"userID", conn.UserID,
		)
	}
}

// handleBroadcast 处理广播消息
func (m *Manager) handleBroadcast(msg *BroadcastMessage) {
	if len(msg.UserIDs) == 0 {
		// 广播给所有连接
		m.connections.Range(func(key, value interface{}) bool {
			conn := value.(*Connection)
			select {
			case conn.Send <- msg.Data:
			default:
				// 发送缓冲区满，跳过
			}
			return true
		})
	} else {
		// 发送给指定用户
		for _, userID := range msg.UserIDs {
			m.SendToUser(userID, msg.Data)
		}
	}
}

// handleShutdown 处理关闭
func (m *Manager) handleShutdown() {
	m.connections.Range(func(key, value interface{}) bool {
		conn := value.(*Connection)
		close(conn.Send)
		conn.Close()
		return true
	})
}

// Register 注册连接
func (m *Manager) Register(conn *Connection) {
	m.register <- conn
}

// Unregister 注销连接
func (m *Manager) Unregister(conn *Connection) {
	m.unregister <- conn
}

// Broadcast 广播消息
func (m *Manager) Broadcast(data []byte) {
	m.broadcast <- &BroadcastMessage{Data: data}
}

// BroadcastToUsers 广播给指定用户
func (m *Manager) BroadcastToUsers(userIDs []int64, data []byte) {
	m.broadcast <- &BroadcastMessage{UserIDs: userIDs, Data: data}
}

// SendToUser 发送消息给用户（所有设备）
func (m *Manager) SendToUser(userID int64, data []byte) error {
	userConnsInterface, ok := m.userConns.Load(userID)
	if !ok {
		return ErrUserNotOnline
	}

	userConns := userConnsInterface.(*sync.Map)
	userConns.Range(func(key, value interface{}) bool {
		conn := value.(*Connection)
		select {
		case conn.Send <- data:
		default:
			// 发送缓冲区满
		}
		return true
	})

	return nil
}

// SendToConnection 发送消息给指定连接
func (m *Manager) SendToConnection(connID string, data []byte) error {
	connInterface, ok := m.connections.Load(connID)
	if !ok {
		return ErrConnectionNotFound
	}

	conn := connInterface.(*Connection)
	select {
	case conn.Send <- data:
		return nil
	default:
		return ErrSendBufferFull
	}
}

// IsUserOnline 检查用户是否在线
func (m *Manager) IsUserOnline(userID int64) bool {
	_, ok := m.userConns.Load(userID)
	return ok
}

// GetUserConnections 获取用户所有连接
func (m *Manager) GetUserConnections(userID int64) []*Connection {
	userConnsInterface, ok := m.userConns.Load(userID)
	if !ok {
		return nil
	}

	var conns []*Connection
	userConns := userConnsInterface.(*sync.Map)
	userConns.Range(func(key, value interface{}) bool {
		conns = append(conns, value.(*Connection))
		return true
	})

	return conns
}

// GetOnlineUserCount 获取在线用户数
func (m *Manager) GetOnlineUserCount() int {
	count := 0
	m.userConns.Range(func(key, value interface{}) bool {
		count++
		return true
	})
	return count
}

// GetConnectionCount 获取连接数
func (m *Manager) GetConnectionCount() int {
	count := 0
	m.connections.Range(func(key, value interface{}) bool {
		count++
		return true
	})
	return count
}

// HandleMessage 处理收到的消息
func (m *Manager) HandleMessage(conn *Connection, data []byte) {
	if m.messageHandler != nil {
		if err := m.messageHandler.Handle(conn, data); err != nil {
			m.logger.Error("Failed to handle message",
				"connID", conn.ID,
				"error", err,
			)
		}
	}
}

// Shutdown 关闭管理器
func (m *Manager) Shutdown() {
	m.closeOnce.Do(func() {
		close(m.closeChan)
	})
}

// Message 消息结构
type Message struct {
	Type string          `json:"type"`
	Data json.RawMessage `json:"data"`
}
