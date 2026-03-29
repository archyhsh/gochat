package websocket

import (
	"context"
	"sync"

	"github.com/archyhsh/gochat/api/internal/svc"
	"github.com/gorilla/websocket"
	"github.com/zeromicro/go-zero/core/logx"
)

type WsLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewWsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *WsLogic {
	return &WsLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *WsLogic) OnConnect(userId int64, conn *websocket.Conn) {
	actual, _ := l.svcCtx.Conns.LoadOrStore(userId, &sync.Map{})
	m := actual.(*sync.Map)
	m.Store(conn, struct{}{})

	// Register in global router
	if err := l.svcCtx.Router.Register(l.ctx, userId); err != nil {
		l.Errorf("Router register error for user %d: %v", userId, err)
	}
}

func (l *WsLogic) OnDisconnect(userId int64, conn *websocket.Conn) {
	if val, ok := l.svcCtx.Conns.Load(userId); ok {
		m := val.(*sync.Map)
		m.Delete(conn)

		isEmpty := true
		m.Range(func(key, value interface{}) bool {
			isEmpty = false
			return false
		})

		if isEmpty {
			l.svcCtx.Conns.Delete(userId)
			_ = l.svcCtx.Router.Unregister(l.ctx, userId)
		}
	}
}

func (l *WsLogic) HandleHeartbeat(userId int64) {
	// Renew lease in Redis
	if err := l.svcCtx.Router.Register(l.ctx, userId); err != nil {
		l.Errorf("Router renewal failed for user %d: %v", userId, err)
	}
}
