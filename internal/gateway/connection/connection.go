package connection

import (
	"encoding/json"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// Connection WebSocket 连接
type Connection struct {
	ID        string          // 连接 ID
	UserID    int64           // 用户 ID
	DeviceID  string          // 设备 ID
	Platform  string          // 平台 (web/ios/android)
	Conn      *websocket.Conn // WebSocket 连接
	Send      chan []byte     // 发送消息 channel
	closeChan chan struct{}   // 关闭信号
	closeOnce sync.Once       // 确保只关闭一次
	manager   *Manager        // 连接管理器引用
}

// NewConnection 创建新连接
func NewConnection(id string, userID int64, deviceID, platform string, conn *websocket.Conn, manager *Manager) *Connection {
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

// ReadPump 读取消息协程
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
				// 记录异常关闭
			}
			break
		}

		// 处理收到的消息
		c.manager.HandleMessage(c, message)
	}
}

// WritePump 发送消息协程
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
				// 管理器关闭了 channel
				c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.Conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			// 批量发送队列中的消息
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

// SendMessage 发送消息
func (c *Connection) SendMessage(msg interface{}) error {
	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	select {
	case c.Send <- data:
		return nil
	default:
		// channel 已满，丢弃消息
		return ErrSendBufferFull
	}
}

// Close 关闭连接
func (c *Connection) Close() {
	c.closeOnce.Do(func() {
		close(c.closeChan)
		c.Conn.Close()
	})
}

// 常量定义
const (
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = (pongWait * 9) / 10
	maxMessageSize = 65536
)

var newline = []byte{'\n'}
