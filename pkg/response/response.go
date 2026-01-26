package response

import (
	"encoding/json"
	"net/http"
)

// Response 统一响应结构
type Response struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// 响应码
const (
	CodeSuccess      = 0
	CodeBadRequest   = 400
	CodeUnauthorized = 401
	CodeForbidden    = 403
	CodeNotFound     = 404
	CodeServerError  = 500
)

// 响应消息
var messages = map[int]string{
	CodeSuccess:      "success",
	CodeBadRequest:   "bad request",
	CodeUnauthorized: "unauthorized",
	CodeForbidden:    "forbidden",
	CodeNotFound:     "not found",
	CodeServerError:  "internal server error",
}

// JSON 返回 JSON 响应
func JSON(w http.ResponseWriter, statusCode int, resp *Response) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(resp)
}

// Success 成功响应
func Success(w http.ResponseWriter, data interface{}) {
	JSON(w, http.StatusOK, &Response{
		Code:    CodeSuccess,
		Message: messages[CodeSuccess],
		Data:    data,
	})
}

// Error 错误响应
func Error(w http.ResponseWriter, statusCode int, code int, message string) {
	JSON(w, statusCode, &Response{
		Code:    code,
		Message: message,
	})
}

// BadRequest 400 错误
func BadRequest(w http.ResponseWriter, message string) {
	if message == "" {
		message = messages[CodeBadRequest]
	}
	Error(w, http.StatusBadRequest, CodeBadRequest, message)
}

// Unauthorized 401 错误
func Unauthorized(w http.ResponseWriter, message string) {
	if message == "" {
		message = messages[CodeUnauthorized]
	}
	Error(w, http.StatusUnauthorized, CodeUnauthorized, message)
}

// Forbidden 403 错误
func Forbidden(w http.ResponseWriter, message string) {
	if message == "" {
		message = messages[CodeForbidden]
	}
	Error(w, http.StatusForbidden, CodeForbidden, message)
}

// NotFound 404 错误
func NotFound(w http.ResponseWriter, message string) {
	if message == "" {
		message = messages[CodeNotFound]
	}
	Error(w, http.StatusNotFound, CodeNotFound, message)
}

// ServerError 500 错误
func ServerError(w http.ResponseWriter, message string) {
	if message == "" {
		message = messages[CodeServerError]
	}
	Error(w, http.StatusInternalServerError, CodeServerError, message)
}
