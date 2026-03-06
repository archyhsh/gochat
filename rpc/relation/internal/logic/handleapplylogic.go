package logic

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/archyhsh/gochat/rpc/pb"
	"github.com/archyhsh/gochat/rpc/relation/internal/svc"
	"github.com/archyhsh/gochat/rpc/relation/model"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/zeromicro/go-zero/core/logx"
)

type HandleApplyLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewHandleApplyLogic(ctx context.Context, svcCtx *svc.ServiceContext) *HandleApplyLogic {
	return &HandleApplyLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *HandleApplyLogic) HandleApply(in *pb.HandleApplyRequest) (*pb.HandleApplyResponse, error) {
	apply, err := l.svcCtx.FriendApplyModel.FindOne(l.ctx, in.ApplyId)
	if err != nil {
		if err == model.ErrNotFound {
			return nil, status.Error(codes.NotFound, "cannot find apply message")
		} else {
			return nil, status.Error(codes.Internal, "failed to find apply message")
		}
	}
	if in.Accept {
		apply.Status = 1
		err = l.svcCtx.FriendshipModel.InsertFriendshipByUserIdFriendId(l.ctx, apply.FromUserId, apply.ToUserId)
		if err != nil {
			return nil, err
		}

		if l.svcCtx.Producer != nil {
			event := map[string]interface{}{
				"type":         "friend_event",
				"action":       "accept",
				"from_user_id": apply.FromUserId,
				"to_user_id":   apply.ToUserId,
				"timestamp":    time.Now().Unix(),
			}
			data, _ := json.Marshal(event)
			key := fmt.Sprintf("friend_%d_%d", apply.FromUserId, apply.ToUserId)
			_ = l.svcCtx.Producer.Send([]byte(key), data)
		}
	} else {
		apply.Status = 2
	}
	err = l.svcCtx.FriendApplyModel.Update(l.ctx, apply)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to handle apply")
	}
	return &pb.HandleApplyResponse{Base: &pb.BaseResponse{Code: 200, Message: "Success"}}, nil
}
