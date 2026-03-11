package dispatch

import (
	"context"
	"errors"

	"github.com/archyhsh/gochat/api/internal/svc"
	"github.com/archyhsh/gochat/api/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type DispatchLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewDispatchLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DispatchLogic {
	return &DispatchLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *DispatchLogic) Dispatch(req *types.DispatchRequest) (resp *types.DispatchResponse, err error) {
	l.Infof("Dispatching request for service: %s, action: %s", req.Service, req.Action)

	// In a real production system, this would use gRPC reflection or a pre-defined
	// mapping of services to their respective RPC clients.
	// For now, we provide the architectural hook for future hot-deployments.

	switch req.Service {
	case "user", "message", "group", "relation":
		// These are already handled by dedicated routes.
		// This entry point is reserved for NEW services not yet in the static routing table.
		return &types.DispatchResponse{
			Data: "{\"message\": \"Service is already natively supported. Please use standard routes.\"}",
		}, nil
	default:
		// Attempt dynamic discovery via Etcd (Concept)
		addr, err := l.svcCtx.Router.Find(l.ctx, 0) // Placeholder for service-level lookup
		if err != nil || addr == "" {
			return nil, errors.New("service not found in dynamic registry")
		}

		return &types.DispatchResponse{
			Data: "{\"message\": \"Dynamic dispatch successful (simulated)\"}",
		}, nil
	}
}
