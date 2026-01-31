package model

import (
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/archyhsh/gochat/pkg/types"
)

type User struct {
	ID        int64     `gorm:"primaryKey;autoIncrement" json:"id"`
	Username  string    `gorm:"column:username;type:varchar(50);uniqueIndex" json:"username"`
	Password  string    `gorm:"column:password;type:varchar(255)" json:"-"`
	Nickname  string    `gorm:"column:nickname;type:varchar(50)" json:"nickname"`
	Avatar    string    `gorm:"column:avatar;type:varchar(255)" json:"avatar"`
	Phone     string    `gorm:"column:phone;type:varchar(20);index" json:"phone"`
	Email     string    `gorm:"column:email;type:varchar(100);index" json:"email"`
	Gender    int       `gorm:"column:gender;default:0" json:"gender"`
	Status    int       `gorm:"column:status;default:1" json:"status"`
	CreatedAt time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

type UserDevice struct {
	ID           int64     `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID       int64     `gorm:"column:user_id;index" json:"user_id"`
	DeviceID     string    `gorm:"column:device_id;type:varchar(100)" json:"device_id"`
	Platform     string    `gorm:"column:platform;type:varchar(20)" json:"platform"`
	PushToken    string    `gorm:"column:push_token;type:varchar(255)" json:"push_token"`
	LastActiveAt time.Time `gorm:"column:last_active_at" json:"last_active_at"`
	CreatedAt    time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt    time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

type RegisterRequest struct {
	Username string `json:"username" binding:"required,min=3,max=50"`
	Password string `json:"password" binding:"required,min=6,max=50"`
	Nickname string `json:"nickname" binding:"required,min=1,max=50"`
}

type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type LoginResponse struct {
	Token string `json:"token"`
	types.UserBasicInfo
}

type UpdateUserRequest struct {
	Nickname string `json:"nickname"`
	Avatar   string `json:"avatar"`
	Phone    string `json:"phone"`
	Email    string `json:"email"`
	Gender   int    `json:"gender"`
}

func (User) TableName() string {
	return "user"
}

func (UserDevice) TableName() string {
	return "user_device"
}

func (u *User) SetPassword(password string) error {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	u.Password = string(hash)
	return nil
}

func (u *User) CheckPassword(password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(u.Password), []byte(password))
	return err == nil
}

func (u *User) ToBasicInfo() types.UserBasicInfo {
	return types.UserBasicInfo{
		UserID:   u.ID,
		Username: u.Username,
		Nickname: u.Nickname,
		Avatar:   u.Avatar,
	}
}
