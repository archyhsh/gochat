package push

import (
	"context"
	"encoding/json"
	"sync"

	"github.com/archyhsh/gochat/api/internal/svc"
	"github.com/archyhsh/gochat/api/internal/types"

	"github.com/gorilla/websocket"
	"github.com/zeromicro/go-zero/core/logx"
)

type PushMessageLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewPushMessageLogic(ctx context.Context, svcCtx *svc.ServiceContext) *PushMessageLogic {
	return &PushMessageLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *PushMessageLogic) PushMessage(req *types.PushRequest) (resp *types.PushResponse, err error) {
	// Base data shared by all targets
	baseData := map[string]interface{}{
		"msg_id":              req.MsgId,
		"conversation_id":     req.ConversationId,
		"sender_id":           req.SenderId,
		"content":             req.Content,
		"msg_type":            req.MsgType,
		"timestamp":           req.Timestamp,
		"sender_info_version": req.SenderInfoVersion,
		"group_meta_version":  req.GroupMetaVersion,
		"relation_version":    req.RelationVersion,
		"sequence":            req.Sequence,
	}

	for _, uid := range req.UserIds {
		if val, ok := l.svcCtx.Conns.Load(uid); ok {
			// Deep copy or extend for specific user unread count
			userData := make(map[string]interface{})
			for k, v := range baseData {
				userData[k] = v
			}

			if req.UnreadMap != nil {
				if count, ok := req.UnreadMap[uid]; ok {
					userData["unread_count"] = count
				}
			}

			jsonData, _ := json.Marshal(userData)
			m := val.(*sync.Map)
			m.Range(func(key, value interface{}) bool {
				conn := key.(*websocket.Conn)
				_ = conn.WriteMessage(websocket.TextMessage, jsonData)
				return true
			})
		}
	}

	return &types.PushResponse{
		Success: true,
	}, nil
}
