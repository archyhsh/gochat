package model

import (
	"time"
)

type Message struct {
	ID             int64     `gorm:"primaryKey;autoIncrement" json:"id"`
	MsgID          string    `gorm:"column:msg_id;type:varchar(64);uniqueIndex" json:"msg_id"`
	ConversationID string    `gorm:"column:conversation_id;type:varchar(64);index" json:"conversation_id"`
	SenderID       int64     `gorm:"column:sender_id;index" json:"sender_id"`
	ReceiverID     int64     `gorm:"column:receiver_id" json:"receiver_id"`
	GroupID        int64     `gorm:"column:group_id" json:"group_id"`
	MsgType        int       `gorm:"column:msg_type" json:"msg_type"` // 1文本 2图片 3文件 4语音 5视频 6系统
	Content        string    `gorm:"column:content;type:text" json:"content"`
	Status         int       `gorm:"column:status;default:0" json:"status"` // 0正常 1撤回
	CreatedAt      time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
}

type MessageRead struct {
	ID     int64     `gorm:"primaryKey;autoIncrement" json:"id"`
	MsgID  string    `gorm:"column:msg_id;type:varchar(64);uniqueIndex:uk_msg_user" json:"msg_id"`
	UserID int64     `gorm:"column:user_id;uniqueIndex:uk_msg_user;index" json:"user_id"`
	ReadAt time.Time `gorm:"column:read_at;autoCreateTime" json:"read_at"`
}

type KafkaChatMessage struct {
	Type    string   `json:"type"`
	Data    ChatData `json:"data"`
	TraceID string   `json:"trace_id,omitempty"`
}

type ChatData struct {
	MsgID          string `json:"msg_id"`
	ConversationID string `json:"conversation_id"`
	SenderID       int64  `json:"sender_id"`
	ReceiverID     int64  `json:"receiver_id,omitempty"`
	GroupID        int64  `json:"group_id,omitempty"`
	MsgType        int    `json:"msg_type"`
	Content        string `json:"content"`
	Timestamp      int64  `json:"timestamp"`
}

func (Message) TableName() string {
	return "message_" + time.Now().Format("200601")
}

func (MessageRead) TableName() string {
	return "message_read"
}

func (c *ChatData) ToMessage() *Message {
	return &Message{
		MsgID:          c.MsgID,
		ConversationID: c.ConversationID,
		SenderID:       c.SenderID,
		ReceiverID:     c.ReceiverID,
		GroupID:        c.GroupID,
		MsgType:        c.MsgType,
		Content:        c.Content,
		Status:         0,
	}
}

func MessageTableName(t time.Time) string {
	return "message_" + t.Format("200601")
}
