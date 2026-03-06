package manager

import (
	"errors"
	"sync"
)

var (
	ErrConnectionNotFound = errors.New("connection not found")
	ErrUserNotOnline      = errors.New("user not online")
)

type Connection interface {
	GetID() string
	GetUserID() int64
	GetPlatform() string
	SendMessage(msg interface{}) error
	Close() error
}

type Manager struct {
	connections sync.Map // connID -> Connection
	userConns   sync.Map // userID -> map[connID]Connection
}

func NewManager() *Manager {
	return &Manager{}
}

func (m *Manager) Register(conn Connection) {
	m.connections.Store(conn.GetID(), conn)

	userConnsInterface, _ := m.userConns.LoadOrStore(conn.GetUserID(), &sync.Map{})
	userConns := userConnsInterface.(*sync.Map)
	userConns.Store(conn.GetID(), conn)
}

func (m *Manager) Unregister(conn Connection) {
	if _, ok := m.connections.LoadAndDelete(conn.GetID()); ok {
		if userConnsInterface, ok := m.userConns.Load(conn.GetUserID()); ok {
			userConns := userConnsInterface.(*sync.Map)
			userConns.Delete(conn.GetID())
			empty := true
			userConns.Range(func(key, value interface{}) bool {
				empty = false
				return false
			})
			if empty {
				m.userConns.Delete(conn.GetUserID())
			}
		}
	}
}

func (m *Manager) SendToUser(userID int64, msg interface{}) error {
	userConnsInterface, ok := m.userConns.Load(userID)
	if !ok {
		return ErrUserNotOnline
	}

	userConns := userConnsInterface.(*sync.Map)
	userConns.Range(func(key, value interface{}) bool {
		conn := value.(Connection)
		_ = conn.SendMessage(msg)
		return true
	})

	return nil
}

func (m *Manager) SendToConnection(connID string, msg interface{}) error {
	connInterface, ok := m.connections.Load(connID)
	if !ok {
		return ErrConnectionNotFound
	}

	conn := connInterface.(Connection)
	return conn.SendMessage(msg)
}

func (m *Manager) IsUserOnline(userID int64) bool {
	_, ok := m.userConns.Load(userID)
	return ok
}

func (m *Manager) GetUserConnections(userID int64) []Connection {
	userConnsInterface, ok := m.userConns.Load(userID)
	if !ok {
		return nil
	}

	var conns []Connection
	userConns := userConnsInterface.(*sync.Map)
	userConns.Range(func(key, value interface{}) bool {
		conns = append(conns, value.(Connection))
		return true
	})

	return conns
}
