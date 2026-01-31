package service

import (
	"github.com/archyhsh/gochat/pkg/logger"
	"gorm.io/gorm"
)

type RelationChecker struct {
	db     *gorm.DB
	logger logger.Logger
}

type Friendship struct {
	ID       int64 `gorm:"primaryKey"`
	UserID   int64 `gorm:"column:user_id"`
	FriendID int64 `gorm:"column:friend_id"`
	Status   int   `gorm:"column:status"`
}

const (
	FriendStatusNormal  = 0
	FriendStatusBlocked = 1
)

func NewRelationChecker(db *gorm.DB, log logger.Logger) *RelationChecker {
	return &RelationChecker{
		db:     db,
		logger: log,
	}
}

func (Friendship) TableName() string {
	return "friendship"
}

func (r *RelationChecker) IsFriend(userID, friendID int64) bool {
	if r.db == nil {
		r.logger.Warn("RelationChecker: database not initialized")
		return true
	}
	var count int64
	err := r.db.Model(&Friendship{}).
		Where("user_id = ? AND friend_id = ?", userID, friendID).
		Count(&count).Error
	if err != nil {
		r.logger.Error("Failed to check friendship", "userID", userID, "friendID", friendID, "error", err)
		return true
	}
	return count > 0
}

func (r *RelationChecker) IsBlocked(userID, friendID int64) bool {
	if r.db == nil {
		return false
	}
	var count int64
	err := r.db.Model(&Friendship{}).
		Where("user_id = ? AND friend_id = ? AND status = ?", friendID, userID, FriendStatusBlocked).
		Count(&count).Error
	if err != nil {
		r.logger.Error("Failed to check blocked status", "userID", userID, "friendID", friendID, "error", err)
		return false
	}
	return count > 0
}

func (r *RelationChecker) HasBlocked(userID, friendID int64) bool {
	if r.db == nil {
		return false
	}
	var count int64
	err := r.db.Model(&Friendship{}).
		Where("user_id = ? AND friend_id = ? AND status = ?", userID, friendID, FriendStatusBlocked).
		Count(&count).Error
	if err != nil {
		r.logger.Error("Failed to check if user blocked friend", "userID", userID, "friendID", friendID, "error", err)
		return false
	}
	return count > 0
}

func (r *RelationChecker) GetBlockStatus(senderID, receiverID int64) (senderBlocked, receiverBlocked bool) {
	senderBlocked = r.HasBlocked(senderID, receiverID)
	receiverBlocked = r.IsBlocked(senderID, receiverID)
	return
}

func (r *RelationChecker) CanSendMessage(senderID, receiverID int64) (bool, string) {
	if senderID == receiverID {
		return false, "cannot send message to yourself"
	}
	if !r.IsFriend(senderID, receiverID) {
		return false, "not friends with this user"
	}
	if r.IsBlocked(senderID, receiverID) {
		return false, "you have been blocked by this user"
	}
	return true, ""
}
