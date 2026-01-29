package connection

import (
	"encoding/json"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

type Connection struct {
	ID        string
	UserID    int64
	DeviceID  string
	Platform  string
	Conn      *websocket.Conn
	Send      chan []byte
	closeChan chan struct{}
	closeOnce sync.Once
	manager   *Manager
}

func NewConnection(id string, userID int64, deviceID string, platform string, conn *websocket.Conn, manager *Manager) *Connection {
	return &Connection{
		ID:        id,
		UserID:    userID,
		DeviceID:  deviceID,
		Platform:  platform,
		Conn:      conn,
		Send:      make(chan []byte, 256),
		closeChan: make(chan struct{}),
		manager:   manager,
	}
}

func (c *Connection) ReadPump() {
	defer func() {
		c.manager.Unregister(c)
		c.Close()
	}()

	c.Conn.SetReadLimit(maxMessageSize)
	c.Conn.SetReadDeadline(time.Now().Add(pongWait))
	c.Conn.SetPongHandler(func(string) error {
		c.Conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		_, message, err := c.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				c.manager.Logger().Warn("WebSocket unexpected close",
					"connID", c.ID,
					"userID", c.UserID,
					"deviceID", c.DeviceID,
					"platform", c.Platform,
					"error", err,
				)
			} else {
				c.manager.Logger().Debug("WebSocket connection closed",
					"connID", c.ID,
					"userID", c.UserID,
					"reason", err.Error(),
				)
			}
			break
		}

		c.manager.HandleMessage(c, message)
	}
}

func (c *Connection) WritePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.Close()
	}()

	for {
		select {
		case message, ok := <-c.Send:
			c.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.Conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			n := len(c.Send)
			for i := 0; i < n; i++ {
				w.Write(newline)
				w.Write(<-c.Send)
			}

			if err := w.Close(); err != nil {
				return
			}

		case <-ticker.C:
			c.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}

		case <-c.closeChan:
			return
		}
	}
}

func (c *Connection) SendMessage(msg interface{}) error {
	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	select {
	case c.Send <- data:
		return nil
	default:
		return ErrSendBufferFull
	}
}

func (c *Connection) Close() {
	c.closeOnce.Do(func() {
		close(c.closeChan)
		c.Conn.Close()
	})
}

const (
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = (pongWait * 9) / 10
	maxMessageSize = 65536
)

var newline = []byte{'\n'}
