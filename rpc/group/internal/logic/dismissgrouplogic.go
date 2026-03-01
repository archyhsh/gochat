package logic

import (
	"context"
	"encoding/json"
	"strconv"
	"time"

	"github.com/archyhsh/gochat/rpc/group/internal/svc"
	"github.com/archyhsh/gochat/rpc/group/model"
	"github.com/archyhsh/gochat/rpc/pb"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/zeromicro/go-zero/core/logx"
)

type DismissGroupLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewDismissGroupLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DismissGroupLogic {
	return &DismissGroupLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *DismissGroupLogic) DismissGroup(in *pb.DismissGroupRequest) (*pb.DismissGroupResponse, error) {
	err := l.svcCtx.GroupModel.Update(l.ctx, &model.Group{
		Id:     in.GroupId,
		Status: 0,
	})
	if err != nil {
		l.Errorf("Failed to dismiss group in DB: %v", err)
		return nil, status.Error(codes.Internal, "failed to dismiss group")
	}

	if l.svcCtx.Config.Producer != nil {
		event := map[string]interface{}{
			"type":      "group_event",
			"action":    "dismiss",
			"group_id":  in.GroupId,
			"timestamp": time.Now().Unix(),
		}
		data, _ := json.Marshal(event)
		key := strconv.FormatInt(in.GroupId, 10)
		_ = l.svcCtx.Config.Producer.Send([]byte(key), data)
	}

	return &pb.DismissGroupResponse{
		Base: &pb.BaseResponse{Code: 200, Message: "Success"},
	}, nil
}
