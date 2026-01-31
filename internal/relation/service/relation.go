package service

import (
	"encoding/json"
	"errors"

	"github.com/archyhsh/gochat/internal/relation/model"
	"github.com/archyhsh/gochat/pkg/kafka"
	"github.com/archyhsh/gochat/pkg/logger"
	"gorm.io/gorm"
)

type RelationService struct {
	db       *gorm.DB
	logger   logger.Logger
	producer *kafka.Producer
}

var (
	ErrAlreadyFriends    = errors.New("already friends")
	ErrApplyNotFound     = errors.New("apply not found")
	ErrApplyAlreadyExist = errors.New("apply already exists")
	ErrFriendNotFound    = errors.New("friend not found")
	ErrCannotAddSelf     = errors.New("cannot add yourself as friend")
	ErrAlreadyBlocked    = errors.New("user is blocked")
)

func NewRelationService(db *gorm.DB, log logger.Logger) *RelationService {
	return &RelationService{
		db:     db,
		logger: log,
	}
}

func (s *RelationService) SetProducer(producer *kafka.Producer) {
	s.producer = producer
}

func (s *RelationService) publishEvent(event *model.RelationEvent) {
	if s.producer == nil {
		return
	}
	data, err := json.Marshal(event)
	if err != nil {
		s.logger.Error("Failed to marshal relation event", "error", err)
		return
	}
	if err := s.producer.Send([]byte(event.Type), data); err != nil {
		s.logger.Error("Failed to publish relation event", "error", err)
	} else {
		s.logger.Info("Relation event published", "type", event.Type, "userID", event.UserID, "peerID", event.PeerID)
	}
}

func (s *RelationService) Apply(fromUserID, toUserID int64, message string) error {
	if fromUserID == toUserID {
		return ErrCannotAddSelf
	}
	var existing model.Friendship
	if err := s.db.Where("user_id = ? AND friend_id = ?", fromUserID, toUserID).First(&existing).Error; err == nil {
		return ErrAlreadyFriends
	}
	var existApply model.FriendApply
	if err := s.db.Where("from_user_id = ? AND to_user_id = ? AND status = ?",
		fromUserID, toUserID, model.ApplyStatusPending).First(&existApply).Error; err == nil {
		return ErrApplyAlreadyExist
	}
	apply := &model.FriendApply{
		FromUserID: fromUserID,
		ToUserID:   toUserID,
		Message:    message,
		Status:     model.ApplyStatusPending,
	}
	if err := s.db.Create(apply).Error; err != nil {
		s.logger.Error("Failed to create friend apply", "from", fromUserID, "to", toUserID, "error", err)
		return err
	}
	s.logger.Info("Friend apply created", "from", fromUserID, "to", toUserID)
	return nil
}

func (s *RelationService) HandleApply(userID, applyID int64, accept bool) error {
	var apply model.FriendApply
	if err := s.db.First(&apply, applyID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrApplyNotFound
		}
		return err
	}
	if apply.ToUserID != userID {
		return ErrApplyNotFound
	}
	if apply.Status != model.ApplyStatusPending {
		return errors.New("apply already handled")
	}
	newStatus := model.ApplyStatusRejected
	if accept {
		newStatus = model.ApplyStatusAccepted
	}
	if err := s.db.Model(&apply).Update("status", newStatus).Error; err != nil {
		return err
	}
	if accept {
		return s.createFriendship(apply.FromUserID, apply.ToUserID)
	}
	s.logger.Info("Friend apply handled", "applyID", applyID, "accept", accept)
	return nil
}

func (s *RelationService) createFriendship(userID1, userID2 int64) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		friendship1 := &model.Friendship{
			UserID:   userID1,
			FriendID: userID2,
			Status:   model.FriendStatusNormal,
		}
		if err := tx.Create(friendship1).Error; err != nil {
			return err
		}
		friendship2 := &model.Friendship{
			UserID:   userID2,
			FriendID: userID1,
			Status:   model.FriendStatusNormal,
		}
		if err := tx.Create(friendship2).Error; err != nil {
			return err
		}
		s.logger.Info("Friendship created", "user1", userID1, "user2", userID2)
		return nil
	})
}

func (s *RelationService) GetFriendList(userID int64) ([]model.FriendInfo, error) {
	var friends []model.FriendInfo
	err := s.db.Table("friendship f").
		Select(`u.id as user_id, u.username, u.nickname, u.avatar, 
				f.remark, f.status, f.created_at,
				IFNULL(f2.status, 0) as blocked_by_peer`).
		Joins("LEFT JOIN user u ON f.friend_id = u.id").
		Joins("LEFT JOIN friendship f2 ON f2.user_id = f.friend_id AND f2.friend_id = f.user_id").
		Where("f.user_id = ?", userID).
		Order("f.created_at DESC").
		Scan(&friends).Error
	if err != nil {
		return nil, err
	}
	return friends, nil
}

func (s *RelationService) GetApplyList(userID int64) ([]model.ApplyInfo, error) {
	var applies []model.ApplyInfo
	err := s.db.Table("friend_apply a").
		Select("a.id, a.from_user_id as user_id, u.username, u.nickname, u.avatar, a.message, a.status, a.created_at").
		Joins("LEFT JOIN user u ON a.from_user_id = u.id").
		Where("a.to_user_id = ?", userID).
		Order("a.created_at DESC").
		Scan(&applies).Error
	if err != nil {
		return nil, err
	}
	return applies, nil
}

func (s *RelationService) DeleteFriend(userID, friendID int64) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("user_id = ? AND friend_id = ?", userID, friendID).Delete(&model.Friendship{}).Error; err != nil {
			return err
		}
		if err := tx.Where("user_id = ? AND friend_id = ?", friendID, userID).Delete(&model.Friendship{}).Error; err != nil {
			return err
		}
		s.logger.Info("Friendship deleted", "user", userID, "friend", friendID)
		s.publishEvent(&model.RelationEvent{
			Type:   "friend",
			UserID: userID,
			PeerID: friendID,
			Action: "deleted",
		})

		return nil
	})
}

func (s *RelationService) BlockFriend(userID, friendID int64) error {
	result := s.db.Model(&model.Friendship{}).
		Where("user_id = ? AND friend_id = ?", userID, friendID).
		Update("status", model.FriendStatusBlocked)
	if result.RowsAffected == 0 {
		return ErrFriendNotFound
	}
	if result.Error != nil {
		return result.Error
	}
	s.logger.Info("Friend blocked", "user", userID, "friend", friendID)
	s.publishEvent(&model.RelationEvent{
		Type:   "block",
		UserID: userID,
		PeerID: friendID,
		Action: "blocked",
	})
	return nil
}

func (s *RelationService) UnblockFriend(userID, friendID int64) error {
	result := s.db.Model(&model.Friendship{}).
		Where("user_id = ? AND friend_id = ?", userID, friendID).
		Update("status", model.FriendStatusNormal)
	if result.RowsAffected == 0 {
		return ErrFriendNotFound
	}
	if result.Error != nil {
		return result.Error
	}
	s.logger.Info("Friend unblocked", "user", userID, "friend", friendID)
	s.publishEvent(&model.RelationEvent{
		Type:   "block",
		UserID: userID,
		PeerID: friendID,
		Action: "unblocked",
	})
	return nil
}

func (s *RelationService) UpdateRemark(userID, friendID int64, remark string) error {
	result := s.db.Model(&model.Friendship{}).
		Where("user_id = ? AND friend_id = ?", userID, friendID).
		Update("remark", remark)
	if result.RowsAffected == 0 {
		return ErrFriendNotFound
	}
	return result.Error
}

func (s *RelationService) IsFriend(userID, friendID int64) bool {
	var count int64
	s.db.Model(&model.Friendship{}).
		Where("user_id = ? AND friend_id = ? AND status = ?", userID, friendID, model.FriendStatusNormal).
		Count(&count)
	return count > 0
}
