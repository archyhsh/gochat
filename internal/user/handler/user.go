package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/archyhsh/gochat/internal/user/model"
	"github.com/archyhsh/gochat/internal/user/service"
	"github.com/archyhsh/gochat/pkg/auth"
	"github.com/archyhsh/gochat/pkg/logger"
	"github.com/archyhsh/gochat/pkg/response"
)

type UserHandler struct {
	userService *service.UserService
	jwtManager  *auth.JWTManager
	logger      logger.Logger
}

func NewUserHandler(userService *service.UserService, jwtManager *auth.JWTManager, log logger.Logger) *UserHandler {
	return &UserHandler{
		userService: userService,
		jwtManager:  jwtManager,
		logger:      log,
	}
}

func (h *UserHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req model.RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "invalid request body")
		return
	}
	if len(req.Username) < 3 || len(req.Username) > 50 {
		response.BadRequest(w, "username must be 3-50 characters")
		return
	}
	if len(req.Password) < 6 || len(req.Password) > 50 {
		response.BadRequest(w, "password must be 6-50 characters")
		return
	}
	if len(req.Nickname) < 1 || len(req.Nickname) > 50 {
		response.BadRequest(w, "nickname must be 1-50 characters")
		return
	}
	user, err := h.userService.Register(&req)
	if err != nil {
		if errors.Is(err, service.ErrUserAlreadyExists) {
			response.BadRequest(w, "username already exists")
			return
		}
		h.logger.Error("Register failed", "error", err)
		response.ServerError(w, "register failed")
		return
	}
	response.Success(w, user)
}

func (h *UserHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req model.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "invalid request body")
		return
	}
	if req.Username == "" || req.Password == "" {
		response.BadRequest(w, "username and password required")
		return
	}
	loginResp, err := h.userService.Login(&req)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrUserNotFound):
			response.Unauthorized(w, "user not found")
		case errors.Is(err, service.ErrInvalidPassword):
			response.Unauthorized(w, "invalid password")
		case errors.Is(err, service.ErrUserDisabled):
			response.Forbidden(w, "user is disabled")
		default:
			h.logger.Error("Login failed", "error", err)
			response.ServerError(w, "login failed")
		}
		return
	}
	response.Success(w, loginResp)
}

func (h *UserHandler) GetUser(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	userIDStr := vars["id"]
	userID, err := strconv.ParseInt(userIDStr, 10, 64)
	if err != nil {
		response.BadRequest(w, "invalid user id")
		return
	}
	user, err := h.userService.GetUserByID(userID)
	if err != nil {
		if errors.Is(err, service.ErrUserNotFound) {
			response.NotFound(w, "user not found")
			return
		}
		response.ServerError(w, "failed to get user")
		return
	}
	response.Success(w, user)
}

func (h *UserHandler) GetCurrentUser(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("user_id")
	if userID == nil {
		response.Unauthorized(w, "unauthorized")
		return
	}
	user, err := h.userService.GetUserByID(userID.(int64))
	if err != nil {
		if errors.Is(err, service.ErrUserNotFound) {
			response.NotFound(w, "user not found")
			return
		}
		response.ServerError(w, "failed to get user")
		return
	}
	response.Success(w, user)
}

func (h *UserHandler) UpdateUser(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("user_id")
	if userID == nil {
		response.Unauthorized(w, "unauthorized")
		return
	}
	var req model.UpdateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "invalid request body")
		return
	}
	user, err := h.userService.UpdateUser(userID.(int64), &req)
	if err != nil {
		if errors.Is(err, service.ErrUserNotFound) {
			response.NotFound(w, "user not found")
			return
		}
		response.ServerError(w, "failed to update user")
		return
	}
	response.Success(w, user)
}

func (h *UserHandler) SearchUsers(w http.ResponseWriter, r *http.Request) {
	keyword := r.URL.Query().Get("keyword")
	limitStr := r.URL.Query().Get("limit")
	limit := 20
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 100 {
			limit = l
		}
	}
	users, err := h.userService.SearchUsers(keyword, limit)
	if err != nil {
		response.ServerError(w, "failed to search users")
		return
	}
	response.Success(w, users)
}
