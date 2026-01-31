package service

import (
	"errors"

	"github.com/archyhsh/gochat/internal/user/model"
	"github.com/archyhsh/gochat/pkg/auth"
	"github.com/archyhsh/gochat/pkg/logger"
	"gorm.io/gorm"
)

type UserService struct {
	db         *gorm.DB
	jwtManager *auth.JWTManager
	logger     logger.Logger
}

var (
	ErrUserNotFound      = errors.New("user not found")
	ErrUserAlreadyExists = errors.New("username already exists")
	ErrInvalidPassword   = errors.New("invalid password")
	ErrUserDisabled      = errors.New("user is disabled")
)

func NewUserService(db *gorm.DB, jwtManager *auth.JWTManager, log logger.Logger) *UserService {
	return &UserService{
		db:         db,
		jwtManager: jwtManager,
		logger:     log,
	}
}

func (s *UserService) Register(req *model.RegisterRequest) (*model.User, error) {
	var existing model.User
	if err := s.db.Where("username = ?", req.Username).First(&existing).Error; err == nil {
		return nil, ErrUserAlreadyExists
	}
	user := &model.User{
		Username: req.Username,
		Nickname: req.Nickname,
		Status:   1,
	}
	if err := user.SetPassword(req.Password); err != nil {
		return nil, err
	}
	if err := s.db.Create(user).Error; err != nil {
		s.logger.Error("Failed to create user", "username", req.Username, "error", err)
		return nil, err
	}
	s.logger.Info("User registered", "userID", user.ID, "username", user.Username)
	return user, nil
}

func (s *UserService) Login(req *model.LoginRequest) (*model.LoginResponse, error) {
	var user model.User
	if err := s.db.Where("username = ?", req.Username).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}
	if user.Status == 0 {
		return nil, ErrUserDisabled
	}
	if !user.CheckPassword(req.Password) {
		return nil, ErrInvalidPassword
	}
	token, err := s.jwtManager.GenerateToken(user.ID, user.Username)
	if err != nil {
		s.logger.Error("Failed to generate token", "userID", user.ID, "error", err)
		return nil, err
	}
	s.logger.Info("User logged in", "userID", user.ID, "username", user.Username)
	return &model.LoginResponse{
		Token:         token,
		UserBasicInfo: user.ToBasicInfo(),
	}, nil
}

func (s *UserService) GetUserByID(userID int64) (*model.User, error) {
	var user model.User
	if err := s.db.First(&user, userID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}
	return &user, nil
}

func (s *UserService) GetUserByUsername(username string) (*model.User, error) {
	var user model.User
	if err := s.db.Where("username = ?", username).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}
	return &user, nil
}

func (s *UserService) UpdateUser(userID int64, req *model.UpdateUserRequest) (*model.User, error) {
	var user model.User
	if err := s.db.First(&user, userID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}
	updates := make(map[string]interface{})
	if req.Nickname != "" {
		updates["nickname"] = req.Nickname
	}
	if req.Avatar != "" {
		updates["avatar"] = req.Avatar
	}
	if req.Phone != "" {
		updates["phone"] = req.Phone
	}
	if req.Email != "" {
		updates["email"] = req.Email
	}
	if req.Gender > 0 {
		updates["gender"] = req.Gender
	}

	if len(updates) > 0 {
		if err := s.db.Model(&user).Updates(updates).Error; err != nil {
			return nil, err
		}
	}
	s.db.First(&user, userID)
	return &user, nil
}

func (s *UserService) SearchUsers(keyword string, limit int) ([]model.User, error) {
	var users []model.User
	query := s.db.Where("status = ?", 1)

	if keyword != "" {
		query = query.Where("username LIKE ? OR nickname LIKE ?", "%"+keyword+"%", "%"+keyword+"%")
	}
	if err := query.Limit(limit).Find(&users).Error; err != nil {
		return nil, err
	}
	return users, nil
}
