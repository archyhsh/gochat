package model

import (
	"time"
)

type Group struct {
	ID           int64     `gorm:"primaryKey;autoIncrement" json:"id"`
	Name         string    `gorm:"column:name;type:varchar(100);not null" json:"name"`
	Avatar       string    `gorm:"column:avatar;type:varchar(255);default:''" json:"avatar"`
	Description  string    `gorm:"column:description;type:varchar(500);default:''" json:"description"`
	OwnerID      int64     `gorm:"column:owner_id;not null;index" json:"owner_id"`
	MaxMembers   int       `gorm:"column:max_members;default:500" json:"max_members"`
	MemberCount  int       `gorm:"column:member_count;default:1" json:"member_count"`
	Announcement string    `gorm:"column:announcement;type:varchar(1000);default:''" json:"announcement"`
	Status       int       `gorm:"column:status;default:1;index" json:"status"`
	CreatedAt    time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt    time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

type GroupMember struct {
	ID         int64      `gorm:"primaryKey;autoIncrement" json:"id"`
	GroupID    int64      `gorm:"column:group_id;not null;index" json:"group_id"`
	UserID     int64      `gorm:"column:user_id;not null;index" json:"user_id"`
	Role       int        `gorm:"column:role;default:0" json:"role"`
	Nickname   string     `gorm:"column:nickname;type:varchar(50);default:''" json:"nickname"`
	MutedUntil *time.Time `gorm:"column:muted_until;null" json:"muted_until"`
	JoinedAt   time.Time  `gorm:"column:joined_at;autoCreateTime" json:"joined_at"`
	CreatedAt  time.Time  `gorm:"column:created_at;autoCreateTime" json:"created_at"`
}

type GroupRequest struct {
	ID        int64     `gorm:"primaryKey;autoIncrement" json:"id"`
	GroupID   int64     `gorm:"column:group_id;not null;index" json:"group_id"`
	UserID    int64     `gorm:"column:user_id;not null;index" json:"user_id"`
	Message   string    `gorm:"column:message;type:varchar(255);default:''" json:"message"`
	Status    int       `gorm:"column:status;default:0;index" json:"status"`
	CreatedAt time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

type GroupInfo struct {
	ID           int64  `json:"id"`
	Name         string `json:"name"`
	Avatar       string `json:"avatar"`
	Description  string `json:"description"`
	OwnerID      int64  `json:"owner_id"`
	OwnerName    string `json:"owner_name"`
	MaxMembers   int    `json:"max_members"`
	MemberCount  int    `json:"member_count"`
	Announcement string `json:"announcement"`
	CreatedAt    string `json:"created_at"`
	IsJoined     bool   `json:"is_joined"`
	MyRole       int    `json:"my_role"`
}

type GroupMemberInfo struct {
	UserID     int64      `json:"user_id"`
	Username   string     `json:"username"`
	Nickname   string     `json:"nickname"`
	Avatar     string     `json:"avatar"`
	Role       int        `json:"role"`
	NicknameIn string     `json:"nickname_in"`
	MutedUntil *time.Time `json:"muted_until"`
	JoinedAt   string     `json:"joined_at"`
}

type GroupEvent struct {
	Type      string `json:"type"`
	GroupID   int64  `json:"group_id"`
	UserID    int64  `json:"user_id,omitempty"`
	Action    string `json:"action"`
	Content   string `json:"content,omitempty"`
	Timestamp int64  `json:"timestamp"`
}

type CreateGroupRequest struct {
	Name        string  `json:"name" binding:"required,min=1,max=100"`
	Avatar      string  `json:"avatar"`
	Description string  `json:"description" binding:"max=500"`
	MemberIDs   []int64 `json:"member_ids"`
}

type JoinGroupRequest struct {
	Message string `json:"message"`
}

type UpdateAnnouncementRequest struct {
	Content string `json:"content" binding:"required,max=1000"`
}

type GroupMessage struct {
	MsgID      string  `json:"msg_id"`
	GroupID    int64   `json:"group_id"`
	SenderID   int64   `json:"sender_id"`
	SenderName string  `json:"sender_name"`
	MsgType    int     `json:"msg_type"`
	Content    string  `json:"content"`
	Mentions   []int64 `json:"mentions"`
	Timestamp  int64   `json:"timestamp"`
}

type SendGroupMessageRequest struct {
	MsgType  int     `json:"msg_type" binding:"required,min=1,max=10"`
	Content  string  `json:"content" binding:"required"`
	Mentions []int64 `json:"mentions"`
}

type RecallGroupMessageRequest struct {
	MsgID string `json:"msg_id" binding:"required"`
}

const (
	GroupStatusNormal  = 1
	GroupStatusDismiss = 0
)

const (
	GroupRoleMember = 0
	GroupRoleAdmin  = 1
	GroupRoleOwner  = 2
)

const (
	GroupRequestStatusPending  = 0
	GroupRequestStatusAccepted = 1
	GroupRequestStatusRejected = 2
)

const (
	GroupEventTypeCreate  = "create"
	GroupEventTypeJoin    = "join"
	GroupEventTypeQuit    = "quit"
	GroupEventTypeKick    = "kick"
	GroupEventTypeDismiss = "dismiss"
	GroupEventTypeNotice  = "notice"
)

func (Group) TableName() string {
	return "group"
}

func (GroupMember) TableName() string {
	return "group_member"
}

func (GroupRequest) TableName() string {
	return "group_request"
}
