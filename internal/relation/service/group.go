package service

import (
	"encoding/json"
	"errors"
	"time"

	"github.com/archyhsh/gochat/internal/relation/model"
	"github.com/archyhsh/gochat/pkg/kafka"
	"github.com/archyhsh/gochat/pkg/logger"
	"gorm.io/gorm"
)

type GroupService struct {
	db       *gorm.DB
	logger   logger.Logger
	producer *kafka.Producer
}

var (
	ErrGroupNotFound      = errors.New("group not found")
	ErrGroupAlreadyJoined = errors.New("already joined the group")
	ErrGroupNotJoined     = errors.New("not joined the group")
	ErrNotGroupMember     = errors.New("not a group member")
	ErrGroupFull          = errors.New("group is full")
	ErrNotGroupOwner      = errors.New("not the group owner")
	ErrNotGroupAdmin      = errors.New("not the group admin")
	ErrCannotKickOwner    = errors.New("cannot kick the group owner")
	ErrCannotQuitOwner    = errors.New("owner cannot quit, please dismiss the group")
	ErrGroupAlreadyExists = errors.New("group already exists")
	ErrInvalidMemberID    = errors.New("invalid member id")
	ErrMutedInGroup       = errors.New("muted in this group")
)

func NewGroupService(db *gorm.DB, log logger.Logger) *GroupService {
	return &GroupService{
		db:     db,
		logger: log,
	}
}

func (s *GroupService) SetProducer(producer *kafka.Producer) {
	s.producer = producer
}

func (s *GroupService) publishGroupEvent(event *model.GroupEvent) {
	if s.producer == nil {
		return
	}
	data, err := json.Marshal(event)
	if err != nil {
		s.logger.Error("Failed to marshal group event", "error", err)
		return
	}
	if err := s.producer.Send([]byte(event.Type), data); err != nil {
		s.logger.Error("Failed to publish group event", "error", err)
	} else {
		s.logger.Info("Group event published",
			"type", event.Type,
			"groupID", event.GroupID,
			"userID", event.UserID,
			"action", event.Action,
		)
	}
}

func (s *GroupService) CreateGroup(ownerID int64, req *model.CreateGroupRequest) (*model.Group, error) {
	if len(req.Name) == 0 || len(req.Name) > 100 {
		return nil, errors.New("group name must be 1-100 characters")
	}

	group := &model.Group{
		Name:        req.Name,
		Avatar:      req.Avatar,
		Description: req.Description,
		OwnerID:     ownerID,
		MaxMembers:  500,
		MemberCount: 1,
		Status:      model.GroupStatusNormal,
	}

	if err := s.db.Create(group).Error; err != nil {
		s.logger.Error("Failed to create group", "error", err)
		return nil, errors.New("failed to create group")
	}

	ownerMember := &model.GroupMember{
		GroupID:  group.ID,
		UserID:   ownerID,
		Role:     model.GroupRoleOwner,
		Nickname: "",
		JoinedAt: time.Now(),
	}
	if err := s.db.Create(ownerMember).Error; err != nil {
		s.logger.Error("Failed to create group owner member", "error", err)
	}

	if len(req.MemberIDs) > 0 {
		for _, memberID := range req.MemberIDs {
			if memberID == ownerID {
				continue
			}
			if err := s.JoinGroup(group.ID, memberID, ""); err != nil {
				s.logger.Warn("Failed to add member to group", "groupID", group.ID, "userID", memberID, "error", err)
			}
		}
	}

	s.logger.Info("Group created", "groupID", group.ID, "ownerID", ownerID, "name", group.Name)
	return group, nil
}

func (s *GroupService) GetGroupList(userID int64) ([]model.GroupInfo, error) {
	var groups []model.GroupInfo
	err := s.db.Table("group_member gm").
		Select(`g.id, g.name, g.avatar, g.description, g.owner_id, g.max_members,
				g.member_count, g.announcement, g.created_at,
				1 as is_joined, gm.role as my_role`).
		Joins("LEFT JOIN `group` g ON g.id = gm.group_id").
		Where("gm.user_id = ? AND g.status = ?", userID, model.GroupStatusNormal).
		Order("g.created_at DESC").
		Scan(&groups).Error
	if err != nil {
		s.logger.Error("Failed to get group list", "error", err)
		return nil, err
	}
	for i := range groups {
		var ownerName string
		s.db.Table("user").Select("nickname").Where("id = ?", groups[i].OwnerID).Scan(&ownerName)
		groups[i].OwnerName = ownerName
		groups[i].IsJoined = true
	}
	return groups, nil
}

func (s *GroupService) GetGroupInfo(groupID, userID int64) (*model.GroupInfo, error) {
	var group model.Group
	if err := s.db.First(&group, groupID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrGroupNotFound
		}
		return nil, err
	}

	if group.Status != model.GroupStatusNormal {
		return nil, ErrGroupNotFound
	}

	var ownerName string
	s.db.Table("user").Select("nickname").Where("id = ?", group.OwnerID).Scan(&ownerName)

	var myRole int
	var isJoined bool
	var member model.GroupMember
	if err := s.db.Where("group_id = ? AND user_id = ?", groupID, userID).First(&member).Error; err == nil {
		isJoined = true
		myRole = member.Role
	} else {
		myRole = -1
		isJoined = false
	}

	return &model.GroupInfo{
		ID:           group.ID,
		Name:         group.Name,
		Avatar:       group.Avatar,
		Description:  group.Description,
		OwnerID:      group.OwnerID,
		OwnerName:    ownerName,
		MaxMembers:   group.MaxMembers,
		MemberCount:  group.MemberCount,
		Announcement: group.Announcement,
		CreatedAt:    group.CreatedAt.Format("2006-01-02 15:04:05"),
		IsJoined:     isJoined,
		MyRole:       myRole,
	}, nil
}

func (s *GroupService) GetGroupMembers(groupID int64) ([]model.GroupMemberInfo, error) {
	var members []model.GroupMemberInfo
	err := s.db.Table("group_member gm").
		Select(`gm.user_id, u.username, u.nickname, u.avatar, gm.role,
				gm.nickname as nickname_in, gm.muted_until, gm.joined_at`).
		Joins("LEFT JOIN `user` u ON u.id = gm.user_id").
		Where("gm.group_id = ?", groupID).
		Order("gm.role DESC, gm.joined_at ASC").
		Scan(&members).Error
	if err != nil {
		s.logger.Error("Failed to get group members", "groupID", groupID, "error", err)
		return nil, err
	}
	return members, nil
}

func (s *GroupService) JoinGroup(groupID, userID int64, message string) error {
	var group model.Group
	if err := s.db.First(&group, groupID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrGroupNotFound
		}
		return err
	}

	if group.Status != model.GroupStatusNormal {
		return ErrGroupNotFound
	}

	var existingMember model.GroupMember
	if err := s.db.Where("group_id = ? AND user_id = ?", groupID, userID).First(&existingMember).Error; err == nil {
		return ErrGroupAlreadyJoined
	}

	if group.MemberCount >= group.MaxMembers {
		return ErrGroupFull
	}

	member := &model.GroupMember{
		GroupID:  groupID,
		UserID:   userID,
		Role:     model.GroupRoleMember,
		Nickname: "",
		JoinedAt: time.Now(),
	}
	if err := s.db.Create(member).Error; err != nil {
		s.logger.Error("Failed to join group", "error", err)
		return errors.New("failed to join group")
	}

	s.db.Model(&group).Update("member_count", gorm.Expr("member_count + 1"))

	s.publishGroupEvent(&model.GroupEvent{
		Type:      model.GroupEventTypeJoin,
		GroupID:   groupID,
		UserID:    userID,
		Action:    "joined",
		Timestamp: time.Now().UnixMilli(),
	})

	s.logger.Info("User joined group", "groupID", groupID, "userID", userID)
	return nil
}

func (s *GroupService) QuitGroup(groupID, userID int64) error {
	var group model.Group
	if err := s.db.First(&group, groupID).Error; err != nil {
		return ErrGroupNotFound
	}

	if group.OwnerID == userID {
		return ErrCannotQuitOwner
	}

	var member model.GroupMember
	if err := s.db.Where("group_id = ? AND user_id = ?", groupID, userID).First(&member).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrGroupNotJoined
		}
		return err
	}

	if err := s.db.Delete(&member).Error; err != nil {
		s.logger.Error("Failed to quit group", "error", err)
		return errors.New("failed to quit group")
	}

	s.db.Model(&group).Update("member_count", gorm.Expr("member_count - 1"))

	s.publishGroupEvent(&model.GroupEvent{
		Type:      model.GroupEventTypeQuit,
		GroupID:   groupID,
		UserID:    userID,
		Action:    "quit",
		Timestamp: time.Now().UnixMilli(),
	})

	s.logger.Info("User quit group", "groupID", groupID, "userID", userID)
	return nil
}

func (s *GroupService) KickGroupMember(groupID, operatorID, targetID int64) error {
	var group model.Group
	if err := s.db.First(&group, groupID).Error; err != nil {
		return ErrGroupNotFound
	}

	var operator model.GroupMember
	if err := s.db.Where("group_id = ? AND user_id = ?", groupID, operatorID).First(&operator).Error; err != nil {
		return ErrNotGroupMember
	}

	if operator.Role != model.GroupRoleOwner && operator.Role != model.GroupRoleAdmin {
		return ErrNotGroupAdmin
	}

	var target model.GroupMember
	if err := s.db.Where("group_id = ? AND user_id = ?", groupID, targetID).First(&target).Error; err != nil {
		return ErrGroupNotJoined
	}

	if target.Role == model.GroupRoleOwner {
		return ErrCannotKickOwner
	}

	if operator.Role == model.GroupRoleAdmin && target.Role == model.GroupRoleAdmin {
		return ErrCannotKickOwner
	}

	if err := s.db.Delete(&target).Error; err != nil {
		s.logger.Error("Failed to kick member", "error", err)
		return errors.New("failed to kick member")
	}

	s.db.Model(&group).Update("member_count", gorm.Expr("member_count - 1"))

	s.publishGroupEvent(&model.GroupEvent{
		Type:      model.GroupEventTypeKick,
		GroupID:   groupID,
		UserID:    targetID,
		Action:    "kicked",
		Timestamp: time.Now().UnixMilli(),
	})

	s.logger.Info("Member kicked from group", "groupID", groupID, "operatorID", operatorID, "targetID", targetID)
	return nil
}

func (s *GroupService) DismissGroup(groupID, userID int64) error {
	var group model.Group
	if err := s.db.First(&group, groupID).Error; err != nil {
		return ErrGroupNotFound
	}

	if group.OwnerID != userID {
		return ErrNotGroupOwner
	}

	s.publishGroupEvent(&model.GroupEvent{
		Type:      model.GroupEventTypeDismiss,
		GroupID:   groupID,
		UserID:    userID,
		Action:    "dismissed",
		Timestamp: time.Now().UnixMilli(),
	})

	if err := s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("group_id = ?", groupID).Delete(&model.GroupMember{}).Error; err != nil {
			return err
		}
		if err := tx.Model(&group).Update("status", model.GroupStatusDismiss).Error; err != nil {
			return err
		}
		return nil
	}); err != nil {
		s.logger.Error("Failed to dismiss group", "error", err)
		return errors.New("failed to dismiss group")
	}

	s.logger.Info("Group dismissed", "groupID", groupID, "ownerID", userID)
	return nil
}

func (s *GroupService) UpdateAnnouncement(groupID, userID int64, content string) error {
	var group model.Group
	if err := s.db.First(&group, groupID).Error; err != nil {
		return ErrGroupNotFound
	}

	var member model.GroupMember
	if err := s.db.Where("group_id = ? AND user_id = ?", groupID, userID).First(&member).Error; err != nil {
		return ErrGroupNotJoined
	}

	if member.Role != model.GroupRoleOwner && member.Role != model.GroupRoleAdmin {
		return ErrNotGroupAdmin
	}

	if err := s.db.Model(&group).Update("announcement", content).Error; err != nil {
		s.logger.Error("Failed to update announcement", "error", err)
		return errors.New("failed to update announcement")
	}

	var senderName string
	s.db.Table("user").Select("nickname").Where("id = ?", userID).Scan(&senderName)

	s.publishGroupEvent(&model.GroupEvent{
		Type:      model.GroupEventTypeNotice,
		GroupID:   groupID,
		UserID:    userID,
		Action:    "announcement",
		Content:   content,
		Timestamp: time.Now().UnixMilli(),
	})

	s.logger.Info("Group announcement updated", "groupID", groupID, "userID", userID)
	return nil
}

func (s *GroupService) IsGroupMember(groupID, userID int64) bool {
	var count int64
	s.db.Model(&model.GroupMember{}).
		Where("group_id = ? AND user_id = ?", groupID, userID).
		Count(&count)
	return count > 0
}

func (s *GroupService) GetUserRole(groupID, userID int64) int {
	var member model.GroupMember
	if err := s.db.Where("group_id = ? AND user_id = ?", groupID, userID).First(&member).Error; err != nil {
		return -1
	}
	return member.Role
}

func (s *GroupService) IsMuted(groupID, userID int64) bool {
	var member model.GroupMember
	if err := s.db.Where("group_id = ? AND user_id = ?", groupID, userID).First(&member).Error; err != nil {
		return false
	}
	if member.MutedUntil.IsZero() {
		return false
	}
	return member.MutedUntil.After(time.Now())
}
