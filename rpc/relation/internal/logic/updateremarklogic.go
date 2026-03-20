package logic

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/archyhsh/gochat/rpc/pb"
	"github.com/archyhsh/gochat/rpc/relation/internal/svc"
	"github.com/archyhsh/gochat/rpc/relation/model"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"github.com/zeromicro/go-zero/core/logx"
)

type UpdateRemarkLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewUpdateRemarkLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UpdateRemarkLogic {
	return &UpdateRemarkLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *UpdateRemarkLogic) UpdateRemark(in *pb.UpdateRemarkRequest) (*pb.UpdateRemarkResponse, error) {
	md, ok := metadata.FromIncomingContext(l.ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "missing metadata")
	}
	userIdStrs := md.Get("user_id")
	if len(userIdStrs) == 0 {
		return nil, status.Error(codes.Unauthenticated, "user_id not found in metadata")
	}
	userId, err := strconv.ParseInt(userIdStrs[0], 10, 64)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "invalid user_id in metadata")
	}

	if userId == in.FriendId {
		return nil, status.Error(codes.InvalidArgument, "cannot block yourself")
	}
	friendship, err := l.svcCtx.FriendshipModel.FindOneByUserIdFriendId(l.ctx, userId, in.FriendId)
	if err != nil {
		if err == model.ErrNotFound {
			return nil, status.Error(codes.NotFound, "target is not your friend")
		}
		return nil, status.Error(codes.Internal, "failed to query friendship")
	}

	friendship.Remark = in.Remark
	friendship.Version = time.Now().UnixNano()
	err = l.svcCtx.FriendshipModel.Update(l.ctx, friendship)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to update remark")
	}

	// Send version update signal to self (multi-device sync)
	if l.svcCtx.Producer != nil {
		event := map[string]interface{}{
			"type":      "friend_event",
			"action":    "update_remark",
			"user_id":   userId,
			"friend_id": in.FriendId,
			"remark":    in.Remark,
			"version":   friendship.Version,
			"timestamp": time.Now().Unix(),
		}
		data, _ := json.Marshal(event)
		key := fmt.Sprintf("remark_%d_%d", userId, in.FriendId)
		_ = l.svcCtx.Producer.Send(l.ctx, []byte(key), data)
	}

	return &pb.UpdateRemarkResponse{
		Base:    &pb.BaseResponse{Code: 200, Message: "Success"},
		Version: friendship.Version,
	}, nil
}
