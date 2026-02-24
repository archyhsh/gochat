package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"

	"github.com/archyhsh/gochat/internal/message/service"
	"github.com/archyhsh/gochat/pkg/logger"
	"github.com/archyhsh/gochat/pkg/response"
)

type MessageHandler struct {
	messageService *service.MessageService
	logger         logger.Logger
}

func NewMessageHandler(messageService *service.MessageService, log logger.Logger) *MessageHandler {
	return &MessageHandler{
		messageService: messageService,
		logger:         log,
	}
}

func (h *MessageHandler) GetMessages(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("user_id")
	if userID == nil {
		response.Unauthorized(w, "unauthorized")
		return
	}
	conversationID := r.URL.Query().Get("conversation_id")
	if conversationID == "" {
		response.BadRequest(w, "conversation_id is required")
		return
	}
	limit := 20
	offset := 0
	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 && parsed <= 100 {
			limit = parsed
		}
	}
	if o := r.URL.Query().Get("offset"); o != "" {
		if parsed, err := strconv.Atoi(o); err == nil && parsed >= 0 {
			offset = parsed
		}
	}

	allowed, err := h.messageService.CheckPermission(userID.(int64), conversationID)
	if err != nil {
		h.logger.Error("Permission check error", "userID", userID, "conversationID", conversationID, "error", err)
		response.ServerError(w, "failed to check permission")
		return
	}
	if !allowed {
		h.logger.Warn("Permission denied", "userID", userID, "conversationID", conversationID)
		response.Forbidden(w, "permission denied")
		return
	}

	h.logger.Debug("Loading messages", "conversationID", conversationID, "limit", limit, "offset", offset)
	messages, err := h.messageService.GetConversationMessages(conversationID, limit, offset)
	if err != nil {
		h.logger.Error("Failed to get messages",
			"conversationID", conversationID,
			"error", err,
		)
		response.ServerError(w, "failed to get messages")
		return
	}
	response.Success(w, map[string]interface{}{
		"messages": messages,
		"limit":    limit,
		"offset":   offset,
	})
}

func (h *MessageHandler) GetConversations(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("user_id")
	if userID == nil {
		response.Unauthorized(w, "unauthorized")
		return
	}
	limit := 20
	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 && parsed <= 50 {
			limit = parsed
		}
	}
	conversations, err := h.messageService.GetUserConversations(userID.(int64), limit)
	if err != nil {
		h.logger.Error("Failed to get conversations",
			"userID", userID,
			"error", err,
		)
		response.ServerError(w, "failed to get conversations")
		return
	}
	response.Success(w, conversations)
}

func (h *MessageHandler) ClearUnread(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("user_id")
	if userID == nil {
		response.Unauthorized(w, "unauthorized")
		return
	}

	var req struct {
		ConversationID string `json:"conversation_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "invalid request")
		return
	}

	if err := h.messageService.ClearUnread(userID.(int64), req.ConversationID); err != nil {
		h.logger.Error("Failed to clear unread", "error", err)
		response.ServerError(w, "failed to clear unread")
		return
	}

	response.Success(w, nil)
}

func (h *MessageHandler) GetMessageByID(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("user_id")
	if userID == nil {
		response.Unauthorized(w, "unauthorized")
		return
	}
	vars := mux.Vars(r)
	msgID := vars["msg_id"]
	if msgID == "" {
		response.BadRequest(w, "msg_id is required")
		return
	}
	message, err := h.messageService.GetMessageByID(msgID)
	if err != nil {
		response.NotFound(w, "message not found")
		return
	}
	response.Success(w, message)
}
