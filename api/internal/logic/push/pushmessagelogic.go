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
	data := map[string]interface{}{
		"msg_id":          req.MsgId,
		"conversation_id": req.ConversationId,
		"sender_id":       req.SenderId,
		"content":         req.Content,
		"msg_type":        req.MsgType,
		"timestamp":       req.Timestamp,
	}
	jsonData, _ := json.Marshal(data)

	for _, uid := range req.UserIds {
		if val, ok := l.svcCtx.Conns.Load(uid); ok {
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
