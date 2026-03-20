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

type DeleteFriendLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewDeleteFriendLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DeleteFriendLogic {
	return &DeleteFriendLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *DeleteFriendLogic) DeleteFriend(in *pb.DeleteFriendRequest) (*pb.DeleteFriendResponse, error) {
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
		return nil, status.Error(codes.InvalidArgument, "cannot delete yourself")
	}
	_, err = l.svcCtx.FriendshipModel.FindOneByUserIdFriendId(l.ctx, userId, in.FriendId)
	if err != nil {
		if err == model.ErrNotFound {
			return nil, status.Error(codes.NotFound, "target is not your friend")
		}
		return nil, status.Error(codes.Internal, "database error: "+err.Error())
	}

	err = l.svcCtx.FriendshipModel.DeleteFriendshipByUserIdFriendId(l.ctx, userId, in.FriendId)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to delete friend")
	}

	// Send delete signal via Kafka
	if l.svcCtx.Producer != nil {
		event := map[string]interface{}{
			"type":         "friend_event",
			"action":       "delete",
			"from_user_id": userId,
			"to_user_id":   in.FriendId,
			"timestamp":    time.Now().Unix(),
		}
		data, _ := json.Marshal(event)
		key := fmt.Sprintf("friend_%d_%d", userId, in.FriendId)
		_ = l.svcCtx.Producer.Send(l.ctx, []byte(key), data)
	}

	return &pb.DeleteFriendResponse{Base: &pb.BaseResponse{Code: 200, Message: "Success"}}, nil
}
