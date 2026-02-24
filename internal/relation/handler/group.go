package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/archyhsh/gochat/internal/relation/model"
	"github.com/archyhsh/gochat/internal/relation/service"
	"github.com/archyhsh/gochat/pkg/logger"
	"github.com/archyhsh/gochat/pkg/response"
	"github.com/gorilla/mux"
)

type GroupHandler struct {
	groupService *service.GroupService
	logger       logger.Logger
}

func NewGroupHandler(groupService *service.GroupService, log logger.Logger) *GroupHandler {
	return &GroupHandler{
		groupService: groupService,
		logger:       log,
	}
}

func (h *GroupHandler) CreateGroup(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("user_id")
	if userID == nil {
		response.Unauthorized(w, "unauthorized")
		return
	}

	var req model.CreateGroupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "invalid request body")
		return
	}

	if len(req.Name) == 0 || len(req.Name) > 100 {
		response.BadRequest(w, "group name must be 1-100 characters")
		return
	}

	group, err := h.groupService.CreateGroup(userID.(int64), &req)
	if err != nil {
		if errors.Is(err, service.ErrGroupAlreadyExists) {
			response.BadRequest(w, "group name already exists")
			return
		}
		h.logger.Error("CreateGroup failed", "error", err)
		response.ServerError(w, "failed to create group")
		return
	}

	response.Success(w, group)
}

func (h *GroupHandler) GetGroupList(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("user_id")
	if userID == nil {
		response.Unauthorized(w, "unauthorized")
		return
	}

	groups, err := h.groupService.GetGroupList(userID.(int64))
	if err != nil {
		h.logger.Error("GetGroupList failed", "error", err)
		response.ServerError(w, "failed to get group list")
		return
	}

	response.Success(w, groups)
}

func (h *GroupHandler) GetGroupInfo(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("user_id")
	if userID == nil {
		response.Unauthorized(w, "unauthorized")
		return
	}

	vars := mux.Vars(r)
	groupIDStr := vars["id"]
	groupID, err := strconv.ParseInt(groupIDStr, 10, 64)
	if err != nil {
		response.BadRequest(w, "invalid group id")
		return
	}

	group, err := h.groupService.GetGroupInfo(groupID, userID.(int64))
	if err != nil {
		switch {
		case errors.Is(err, service.ErrGroupNotFound):
			response.NotFound(w, "group not found")
		default:
			h.logger.Error("GetGroupInfo failed", "error", err)
			response.ServerError(w, "failed to get group info")
		}
		return
	}

	response.Success(w, group)
}

func (h *GroupHandler) GetGroupMembers(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("user_id")
	if userID == nil {
		response.Unauthorized(w, "unauthorized")
		return
	}

	vars := mux.Vars(r)
	groupIDStr := vars["id"]
	groupID, err := strconv.ParseInt(groupIDStr, 10, 64)
	if err != nil {
		response.BadRequest(w, "invalid group id")
		return
	}

	members, err := h.groupService.GetGroupMembers(groupID)
	if err != nil {
		h.logger.Error("GetGroupMembers failed", "error", err)
		response.ServerError(w, "failed to get group members")
		return
	}

	response.Success(w, members)
}

func (h *GroupHandler) JoinGroup(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("user_id")
	if userID == nil {
		response.Unauthorized(w, "unauthorized")
		return
	}

	vars := mux.Vars(r)
	groupIDStr := vars["id"]
	groupID, err := strconv.ParseInt(groupIDStr, 10, 64)
	if err != nil {
		response.BadRequest(w, "invalid group id")
		return
	}

	var req model.JoinGroupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "invalid request body")
		return
	}

	err = h.groupService.JoinGroup(groupID, userID.(int64), req.Message)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrGroupNotFound):
			response.NotFound(w, "group not found")
		case errors.Is(err, service.ErrGroupAlreadyJoined):
			response.BadRequest(w, "already joined the group")
		case errors.Is(err, service.ErrGroupFull):
			response.BadRequest(w, "group is full")
		default:
			h.logger.Error("JoinGroup failed", "error", err)
			response.ServerError(w, "failed to join group")
		}
		return
	}

	response.Success(w, map[string]string{"message": "joined group successfully"})
}

func (h *GroupHandler) QuitGroup(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("user_id")
	if userID == nil {
		response.Unauthorized(w, "unauthorized")
		return
	}

	vars := mux.Vars(r)
	groupIDStr := vars["id"]
	groupID, err := strconv.ParseInt(groupIDStr, 10, 64)
	if err != nil {
		response.BadRequest(w, "invalid group id")
		return
	}

	err = h.groupService.QuitGroup(groupID, userID.(int64))
	if err != nil {
		switch {
		case errors.Is(err, service.ErrGroupNotFound):
			response.NotFound(w, "group not found")
		case errors.Is(err, service.ErrGroupNotJoined):
			response.BadRequest(w, "not joined the group")
		case errors.Is(err, service.ErrCannotQuitOwner):
			response.BadRequest(w, "owner cannot quit, please dismiss the group")
		default:
			h.logger.Error("QuitGroup failed", "error", err)
			response.ServerError(w, "failed to quit group")
		}
		return
	}

	response.Success(w, map[string]string{"message": "quit group successfully"})
}

func (h *GroupHandler) KickGroupMember(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("user_id")
	if userID == nil {
		response.Unauthorized(w, "unauthorized")
		return
	}

	vars := mux.Vars(r)
	groupIDStr := vars["id"]
	groupID, err := strconv.ParseInt(groupIDStr, 10, 64)
	if err != nil {
		response.BadRequest(w, "invalid group id")
		return
	}

	memberIDStr := vars["member_id"]
	memberID, err := strconv.ParseInt(memberIDStr, 10, 64)
	if err != nil {
		response.BadRequest(w, "invalid member id")
		return
	}

	err = h.groupService.KickGroupMember(groupID, userID.(int64), memberID)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrGroupNotFound):
			response.NotFound(w, "group not found")
		case errors.Is(err, service.ErrGroupNotJoined):
			response.NotFound(w, "member not in group")
		case errors.Is(err, service.ErrNotGroupOwner):
			response.Forbidden(w, "only owner can kick members")
		case errors.Is(err, service.ErrNotGroupAdmin):
			response.Forbidden(w, "only admin can kick members")
		case errors.Is(err, service.ErrCannotKickOwner):
			response.BadRequest(w, "cannot kick the group owner")
		default:
			h.logger.Error("KickGroupMember failed", "error", err)
			response.ServerError(w, "failed to kick member")
		}
		return
	}

	response.Success(w, map[string]string{"message": "member kicked successfully"})
}

func (h *GroupHandler) DismissGroup(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("user_id")
	if userID == nil {
		response.Unauthorized(w, "unauthorized")
		return
	}

	vars := mux.Vars(r)
	groupIDStr := vars["id"]
	groupID, err := strconv.ParseInt(groupIDStr, 10, 64)
	if err != nil {
		response.BadRequest(w, "invalid group id")
		return
	}

	err = h.groupService.DismissGroup(groupID, userID.(int64))
	if err != nil {
		switch {
		case errors.Is(err, service.ErrGroupNotFound):
			response.NotFound(w, "group not found")
		case errors.Is(err, service.ErrNotGroupOwner):
			response.Forbidden(w, "only the owner can dismiss the group")
		default:
			h.logger.Error("DismissGroup failed", "error", err)
			response.ServerError(w, "failed to dismiss group")
		}
		return
	}

	response.Success(w, map[string]string{"message": "group dismissed successfully"})
}

func (h *GroupHandler) UpdateAnnouncement(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("user_id")
	if userID == nil {
		response.Unauthorized(w, "unauthorized")
		return
	}

	vars := mux.Vars(r)
	groupIDStr := vars["id"]
	groupID, err := strconv.ParseInt(groupIDStr, 10, 64)
	if err != nil {
		response.BadRequest(w, "invalid group id")
		return
	}

	var req model.UpdateAnnouncementRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "invalid request body")
		return
	}

	if len(req.Content) > 1000 {
		response.BadRequest(w, "announcement must be less than 1000 characters")
		return
	}

	err = h.groupService.UpdateAnnouncement(groupID, userID.(int64), req.Content)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrGroupNotFound):
			response.NotFound(w, "group not found")
		case errors.Is(err, service.ErrGroupNotJoined):
			response.BadRequest(w, "not a group member")
		case errors.Is(err, service.ErrNotGroupOwner):
			response.Forbidden(w, "only owner can update announcement")
		case errors.Is(err, service.ErrNotGroupAdmin):
			response.Forbidden(w, "only admin can update announcement")
		default:
			h.logger.Error("UpdateAnnouncement failed", "error", err)
			response.ServerError(w, "failed to update announcement")
		}
		return
	}

	response.Success(w, map[string]string{"message": "announcement updated successfully"})
}

func (h *GroupHandler) GetAnnouncement(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("user_id")
	if userID == nil {
		response.Unauthorized(w, "unauthorized")
		return
	}

	vars := mux.Vars(r)
	groupIDStr := vars["id"]
	groupID, err := strconv.ParseInt(groupIDStr, 10, 64)
	if err != nil {
		response.BadRequest(w, "invalid group id")
		return
	}

	group, err := h.groupService.GetGroupInfo(groupID, userID.(int64))
	if err != nil {
		switch {
		case errors.Is(err, service.ErrGroupNotFound):
			response.NotFound(w, "group not found")
		default:
			h.logger.Error("GetAnnouncement failed", "error", err)
			response.ServerError(w, "failed to get announcement")
		}
		return
	}

	response.Success(w, map[string]interface{}{
		"group_id":     group.ID,
		"announcement": group.Announcement,
	})
}

func (h *GroupHandler) SearchGroups(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("user_id")
	if userID == nil {
		response.Unauthorized(w, "unauthorized")
		return
	}

	keyword := r.URL.Query().Get("keyword")
	if keyword == "" {
		response.BadRequest(w, "keyword is required")
		return
	}

	groups, err := h.groupService.SearchGroups(keyword, userID.(int64))
	if err != nil {
		h.logger.Error("SearchGroups failed", "error", err)
		response.ServerError(w, "failed to search groups")
		return
	}

	response.Success(w, groups)
}

func (h *GroupHandler) InviteMembers(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("user_id")
	if userID == nil {
		response.Unauthorized(w, "unauthorized")
		return
	}

	vars := mux.Vars(r)
	groupID, _ := strconv.ParseInt(vars["id"], 10, 64)

	var req struct {
		MemberIDs []int64 `json:"member_ids"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "invalid request body")
		return
	}

	if err := h.groupService.InviteMembers(groupID, userID.(int64), req.MemberIDs); err != nil {
		switch {
		case errors.Is(err, service.ErrGroupNotFound):
			response.NotFound(w, "group not found")
		case errors.Is(err, service.ErrNotGroupMember):
			response.Forbidden(w, "only members can invite others")
		default:
			h.logger.Error("InviteMembers failed", "error", err)
			response.ServerError(w, "failed to invite members")
		}
		return
	}

	response.Success(w, map[string]string{"message": "members invited successfully"})
}
