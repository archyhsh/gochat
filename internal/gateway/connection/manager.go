package connection

import (
	"encoding/json"
	"errors"
	"sync"

	"github.com/archyhsh/gochat/pkg/logger"
)

var (
	ErrSendBufferFull     = errors.New("send buffer full")
	ErrConnectionNotFound = errors.New("connection not found")
	ErrUserNotOnline      = errors.New("user not online")
)

type Manager struct {
	connections    sync.Map
	userConns      sync.Map
	register       chan *Connection
	unregister     chan *Connection
	broadcast      chan *BroadcastMessage
	messageHandler MessageHandler
	logger         logger.Logger
	closeChan      chan struct{}
	closeOnce      sync.Once
}

type BroadcastMessage struct {
	UserIDs []int64
	Data    []byte
}

type MessageHandler interface {
	Handle(conn *Connection, data []byte) error
}

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

func (m *Manager) SetMessageHandler(handler MessageHandler) {
	m.messageHandler = handler
}

func (m *Manager) Start() {
	go m.run()
}

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

func (m *Manager) handleRegister(conn *Connection) {
	m.connections.Store(conn.ID, conn)

	userConnsInterface, _ := m.userConns.LoadOrStore(conn.UserID, &sync.Map{})
	userConns := userConnsInterface.(*sync.Map)
	userConns.Store(conn.ID, conn)

	m.logger.Info("Connection registered",
		"connID", conn.ID,
		"userID", conn.UserID,
		"platform", conn.Platform,
	)
}

func (m *Manager) handleUnregister(conn *Connection) {
	if _, ok := m.connections.LoadAndDelete(conn.ID); ok {
		if userConnsInterface, ok := m.userConns.Load(conn.UserID); ok {
			userConns := userConnsInterface.(*sync.Map)
			userConns.Delete(conn.ID)
			empty := true
			userConns.Range(func(key, value interface{}) bool {
				empty = false
				return false
			})
			if empty {
				m.userConns.Delete(conn.UserID)
			}
		}
		close(conn.Send)
		m.logger.Info("Connection unregistered",
			"connID", conn.ID,
			"userID", conn.UserID,
		)
	}
}

func (m *Manager) handleBroadcast(msg *BroadcastMessage) {
	if len(msg.UserIDs) == 0 {
		m.connections.Range(func(key, value interface{}) bool {
			conn := value.(*Connection)
			select {
			case conn.Send <- msg.Data:
			default:
				m.logger.Warn("Broadcast buffer full, message dropped",
					"connID", conn.ID,
					"userID", conn.UserID,
				)
			}
			return true
		})
	} else {
		for _, userID := range msg.UserIDs {
			m.SendToUser(userID, msg.Data)
		}
	}
}

func (m *Manager) handleShutdown() {
	m.connections.Range(func(key, value interface{}) bool {
		conn := value.(*Connection)
		close(conn.Send)
		conn.Close()
		return true
	})
}

func (m *Manager) Register(conn *Connection) {
	m.register <- conn
}

func (m *Manager) Unregister(conn *Connection) {
	m.unregister <- conn
}

func (m *Manager) Broadcast(data []byte) {
	m.broadcast <- &BroadcastMessage{Data: data}
}

func (m *Manager) BroadcastToUsers(userIDs []int64, data []byte) {
	m.broadcast <- &BroadcastMessage{UserIDs: userIDs, Data: data}
}

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
			m.logger.Warn("SendToUser buffer full, message dropped",
				"connID", conn.ID,
				"userID", userID,
			)
		}
		return true
	})

	return nil
}

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

func (m *Manager) IsUserOnline(userID int64) bool {
	_, ok := m.userConns.Load(userID)
	return ok
}

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

func (m *Manager) GetOnlineUserCount() int {
	count := 0
	m.userConns.Range(func(key, value interface{}) bool {
		count++
		return true
	})
	return count
}

func (m *Manager) GetConnectionCount() int {
	count := 0
	m.connections.Range(func(key, value interface{}) bool {
		count++
		return true
	})
	return count
}

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

func (m *Manager) Shutdown() {
	m.closeOnce.Do(func() {
		close(m.closeChan)
	})
}

func (m *Manager) Logger() logger.Logger {
	return m.logger
}

type Message struct {
	Type string          `json:"type"`
	Data json.RawMessage `json:"data"`
}
