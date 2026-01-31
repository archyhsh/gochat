package model

import (
	"time"

	"github.com/archyhsh/gochat/pkg/types"
)

type Friendship struct {
	ID        int64     `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID    int64     `gorm:"column:user_id;index" json:"user_id"`
	FriendID  int64     `gorm:"column:friend_id;index" json:"friend_id"`
	Remark    string    `gorm:"column:remark;type:varchar(50)" json:"remark"`
	Status    int       `gorm:"column:status;default:0" json:"status"`
	CreatedAt time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

type FriendApply struct {
	ID         int64     `gorm:"primaryKey;autoIncrement" json:"id"`
	FromUserID int64     `gorm:"column:from_user_id;index" json:"from_user_id"`
	ToUserID   int64     `gorm:"column:to_user_id;index" json:"to_user_id"`
	Message    string    `gorm:"column:message;type:varchar(255)" json:"message"`
	Status     int       `gorm:"column:status;default:0;index" json:"status"`
	CreatedAt  time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt  time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

type FriendInfo struct {
	types.UserBasicInfo
	Remark        string `json:"remark"`
	Status        int    `json:"status"`
	BlockedByPeer int    `json:"blocked_by_peer"`
	CreatedAt     string `json:"created_at"`
}

type ApplyInfo struct {
	ID int64 `json:"id"`
	types.UserBasicInfo
	Message   string `json:"message"`
	Status    int    `json:"status"`
	CreatedAt string `json:"created_at"`
}

type ApplyRequest struct {
	ToUserID int64  `json:"to_user_id" binding:"required"`
	Message  string `json:"message"`
}

type HandleApplyRequest struct {
	ApplyID int64 `json:"apply_id" binding:"required"`
	Accept  bool  `json:"accept"`
}

type UpdateRemarkRequest struct {
	FriendID int64  `json:"friend_id" binding:"required"`
	Remark   string `json:"remark"`
}

const (
	ApplyStatusPending  = 0
	ApplyStatusAccepted = 1
	ApplyStatusRejected = 2
)
const (
	FriendStatusNormal  = 0
	FriendStatusBlocked = 1
)

func (Friendship) TableName() string {
	return "friendship"
}

func (FriendApply) TableName() string {
	return "friend_apply"
}
