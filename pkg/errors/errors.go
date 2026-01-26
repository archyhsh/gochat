package errors

import (
	"fmt"
)

// AppError 应用错误
type AppError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Err     error  `json:"-"`
}

func (e *AppError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("[%d] %s: %v", e.Code, e.Message, e.Err)
	}
	return fmt.Sprintf("[%d] %s", e.Code, e.Message)
}

func (e *AppError) Unwrap() error {
	return e.Err
}

// 错误码定义
const (
	// 通用错误 1000-1999
	ErrCodeUnknown       = 1000
	ErrCodeInvalidParam  = 1001
	ErrCodeUnauthorized  = 1002
	ErrCodeForbidden     = 1003
	ErrCodeNotFound      = 1004
	ErrCodeInternalError = 1005

	// 用户相关错误 2000-2999
	ErrCodeUserNotFound    = 2001
	ErrCodeUserExists      = 2002
	ErrCodeInvalidPassword = 2003
	ErrCodeInvalidToken    = 2004
	ErrCodeTokenExpired    = 2005

	// 消息相关错误 3000-3999
	ErrCodeMessageNotFound     = 3001
	ErrCodeMessageSendFailed   = 3002
	ErrCodeMessageRecallFailed = 3003

	// 好友相关错误 4000-4999
	ErrCodeFriendNotFound      = 4001
	ErrCodeFriendExists        = 4002
	ErrCodeFriendBlocked       = 4003
	ErrCodeFriendApplyNotFound = 4004

	// 群组相关错误 5000-5999
	ErrCodeGroupNotFound  = 5001
	ErrCodeGroupFull      = 5002
	ErrCodeNotGroupMember = 5003
	ErrCodeNotGroupAdmin  = 5004
	ErrCodeGroupMuted     = 5005
)

// 预定义错误
var (
	ErrUnknown       = &AppError{Code: ErrCodeUnknown, Message: "unknown error"}
	ErrInvalidParam  = &AppError{Code: ErrCodeInvalidParam, Message: "invalid parameter"}
	ErrUnauthorized  = &AppError{Code: ErrCodeUnauthorized, Message: "unauthorized"}
	ErrForbidden     = &AppError{Code: ErrCodeForbidden, Message: "forbidden"}
	ErrNotFound      = &AppError{Code: ErrCodeNotFound, Message: "not found"}
	ErrInternalError = &AppError{Code: ErrCodeInternalError, Message: "internal server error"}

	ErrUserNotFound    = &AppError{Code: ErrCodeUserNotFound, Message: "user not found"}
	ErrUserExists      = &AppError{Code: ErrCodeUserExists, Message: "user already exists"}
	ErrInvalidPassword = &AppError{Code: ErrCodeInvalidPassword, Message: "invalid password"}
	ErrInvalidToken    = &AppError{Code: ErrCodeInvalidToken, Message: "invalid token"}
	ErrTokenExpired    = &AppError{Code: ErrCodeTokenExpired, Message: "token expired"}

	ErrMessageNotFound     = &AppError{Code: ErrCodeMessageNotFound, Message: "message not found"}
	ErrMessageSendFailed   = &AppError{Code: ErrCodeMessageSendFailed, Message: "message send failed"}
	ErrMessageRecallFailed = &AppError{Code: ErrCodeMessageRecallFailed, Message: "message recall failed"}

	ErrFriendNotFound      = &AppError{Code: ErrCodeFriendNotFound, Message: "friend not found"}
	ErrFriendExists        = &AppError{Code: ErrCodeFriendExists, Message: "friend already exists"}
	ErrFriendBlocked       = &AppError{Code: ErrCodeFriendBlocked, Message: "friend blocked"}
	ErrFriendApplyNotFound = &AppError{Code: ErrCodeFriendApplyNotFound, Message: "friend apply not found"}

	ErrGroupNotFound  = &AppError{Code: ErrCodeGroupNotFound, Message: "group not found"}
	ErrGroupFull      = &AppError{Code: ErrCodeGroupFull, Message: "group is full"}
	ErrNotGroupMember = &AppError{Code: ErrCodeNotGroupMember, Message: "not group member"}
	ErrNotGroupAdmin  = &AppError{Code: ErrCodeNotGroupAdmin, Message: "not group admin"}
	ErrGroupMuted     = &AppError{Code: ErrCodeGroupMuted, Message: "you are muted in this group"}
)

// New 创建新错误
func New(code int, message string) *AppError {
	return &AppError{Code: code, Message: message}
}

// Wrap 包装错误
func Wrap(err error, code int, message string) *AppError {
	return &AppError{Code: code, Message: message, Err: err}
}

// WrapWithError 用已有的 AppError 包装
func WrapWithError(appErr *AppError, err error) *AppError {
	return &AppError{
		Code:    appErr.Code,
		Message: appErr.Message,
		Err:     err,
	}
}
