package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/archyhsh/gochat/internal/relation/model"
	"github.com/archyhsh/gochat/internal/relation/service"
	"github.com/archyhsh/gochat/pkg/logger"
	"github.com/archyhsh/gochat/pkg/response"
)

type RelationHandler struct {
	relationService *service.RelationService
	logger          logger.Logger
}

func NewRelationHandler(relationService *service.RelationService, log logger.Logger) *RelationHandler {
	return &RelationHandler{
		relationService: relationService,
		logger:          log,
	}
}

func (h *RelationHandler) Apply(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("user_id")
	if userID == nil {
		response.Unauthorized(w, "unauthorized")
		return
	}
	var req model.ApplyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "invalid request body")
		return
	}
	if req.ToUserID == 0 {
		response.BadRequest(w, "to_user_id is required")
		return
	}
	err := h.relationService.Apply(userID.(int64), req.ToUserID, req.Message)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrCannotAddSelf):
			response.BadRequest(w, "cannot add yourself as friend")
		case errors.Is(err, service.ErrAlreadyFriends):
			response.BadRequest(w, "already friends")
		case errors.Is(err, service.ErrApplyAlreadyExist):
			response.BadRequest(w, "apply already exists")
		default:
			h.logger.Error("Apply failed", "error", err)
			response.ServerError(w, "apply failed")
		}
		return
	}
	response.Success(w, map[string]string{"message": "apply sent"})
}

func (h *RelationHandler) HandleApply(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("user_id")
	if userID == nil {
		response.Unauthorized(w, "unauthorized")
		return
	}
	var req model.HandleApplyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "invalid request body")
		return
	}
	if req.ApplyID == 0 {
		response.BadRequest(w, "apply_id is required")
		return
	}
	err := h.relationService.HandleApply(userID.(int64), req.ApplyID, req.Accept)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrApplyNotFound):
			response.NotFound(w, "apply not found")
		default:
			h.logger.Error("HandleApply failed", "error", err)
			response.ServerError(w, "handle apply failed")
		}
		return
	}
	action := "rejected"
	if req.Accept {
		action = "accepted"
	}
	response.Success(w, map[string]string{"message": "apply " + action})
}

func (h *RelationHandler) GetFriendList(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("user_id")
	if userID == nil {
		response.Unauthorized(w, "unauthorized")
		return
	}
	friends, err := h.relationService.GetFriendList(userID.(int64))
	if err != nil {
		h.logger.Error("GetFriendList failed", "error", err)
		response.ServerError(w, "failed to get friend list")
		return
	}
	response.Success(w, friends)
}

func (h *RelationHandler) GetApplyList(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("user_id")
	if userID == nil {
		response.Unauthorized(w, "unauthorized")
		return
	}
	applies, err := h.relationService.GetApplyList(userID.(int64))
	if err != nil {
		h.logger.Error("GetApplyList failed", "error", err)
		response.ServerError(w, "failed to get apply list")
		return
	}
	response.Success(w, applies)
}

func (h *RelationHandler) DeleteFriend(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("user_id")
	if userID == nil {
		response.Unauthorized(w, "unauthorized")
		return
	}
	vars := mux.Vars(r)
	friendIDStr := vars["id"]
	friendID, err := strconv.ParseInt(friendIDStr, 10, 64)
	if err != nil {
		response.BadRequest(w, "invalid friend id")
		return
	}
	if err := h.relationService.DeleteFriend(userID.(int64), friendID); err != nil {
		h.logger.Error("DeleteFriend failed", "error", err)
		response.ServerError(w, "failed to delete friend")
		return
	}
	response.Success(w, map[string]string{"message": "friend deleted"})
}

func (h *RelationHandler) BlockFriend(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("user_id")
	if userID == nil {
		response.Unauthorized(w, "unauthorized")
		return
	}
	vars := mux.Vars(r)
	friendIDStr := vars["id"]
	friendID, err := strconv.ParseInt(friendIDStr, 10, 64)
	if err != nil {
		response.BadRequest(w, "invalid friend id")
		return
	}
	if err := h.relationService.BlockFriend(userID.(int64), friendID); err != nil {
		if errors.Is(err, service.ErrFriendNotFound) {
			response.NotFound(w, "friend not found")
			return
		}
		h.logger.Error("BlockFriend failed", "error", err)
		response.ServerError(w, "failed to block friend")
		return
	}
	response.Success(w, map[string]string{"message": "friend blocked"})
}

func (h *RelationHandler) UnblockFriend(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("user_id")
	if userID == nil {
		response.Unauthorized(w, "unauthorized")
		return
	}
	vars := mux.Vars(r)
	friendIDStr := vars["id"]
	friendID, err := strconv.ParseInt(friendIDStr, 10, 64)
	if err != nil {
		response.BadRequest(w, "invalid friend id")
		return
	}

	if err := h.relationService.UnblockFriend(userID.(int64), friendID); err != nil {
		if errors.Is(err, service.ErrFriendNotFound) {
			response.NotFound(w, "friend not found")
			return
		}
		h.logger.Error("UnblockFriend failed", "error", err)
		response.ServerError(w, "failed to unblock friend")
		return
	}

	response.Success(w, map[string]string{"message": "friend unblocked"})
}

func (h *RelationHandler) UpdateRemark(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("user_id")
	if userID == nil {
		response.Unauthorized(w, "unauthorized")
		return
	}
	var req model.UpdateRemarkRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "invalid request body")
		return
	}
	if req.FriendID == 0 {
		response.BadRequest(w, "friend_id is required")
		return
	}
	if err := h.relationService.UpdateRemark(userID.(int64), req.FriendID, req.Remark); err != nil {
		if errors.Is(err, service.ErrFriendNotFound) {
			response.NotFound(w, "friend not found")
			return
		}
		h.logger.Error("UpdateRemark failed", "error", err)
		response.ServerError(w, "failed to update remark")
		return
	}
	response.Success(w, map[string]string{"message": "remark updated"})
}
