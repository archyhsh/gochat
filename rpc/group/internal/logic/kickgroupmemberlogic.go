package logic

import (
	"context"
	"encoding/json"
	"strconv"
	"time"

	"github.com/archyhsh/gochat/rpc/group/internal/svc"
	"github.com/archyhsh/gochat/rpc/pb"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

type KickGroupMemberLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewKickGroupMemberLogic(ctx context.Context, svcCtx *svc.ServiceContext) *KickGroupMemberLogic {
	return &KickGroupMemberLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *KickGroupMemberLogic) KickGroupMember(in *pb.KickGroupMemberRequest) (*pb.KickGroupMemberResponse, error) {
	// only group owner can kick members
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
	if userId == in.MemberId {
		return nil, status.Error(codes.InvalidArgument, "cannot kick yourself")
	}
	member, _ := l.svcCtx.GroupMemberModel.FindMemberByGroupIdAndUserId(l.ctx, in.GroupId, userId)
	if member == nil || member.Role != 2 {
		return nil, status.Error(codes.PermissionDenied, "only group owner can kick members")
	}
	err = l.svcCtx.SqlConn.TransactCtx(l.ctx, func(ctx context.Context, session sqlx.Session) error {
		member, err := l.svcCtx.GroupMemberModel.FindMemberByGroupIdAndUserId(ctx, in.GroupId, in.MemberId)
		if err != nil {
			return err
		}
		err = l.svcCtx.GroupMemberModel.Delete(ctx, member.Id)
		if err != nil {
			return err
		}
		_, err = session.ExecCtx(ctx, "update `group` set member_count = member_count - 1 where id = ?", in.GroupId)
		return err
	})

	if err != nil {
		l.Errorf("Failed to kick member: %v", err)
		return nil, status.Error(codes.Internal, "failed to kick member")
	}
	if l.svcCtx.Producer != nil {
		event := map[string]interface{}{
			"type":      "group_event",
			"action":    "kick",
			"group_id":  in.GroupId,
			"user_id":   in.MemberId,
			"timestamp": time.Now().Unix(),
		}
		data, _ := json.Marshal(event)
		key := strconv.FormatInt(in.GroupId, 10)
		_ = l.svcCtx.Producer.Send(l.ctx, []byte(key), data)
	}

	return &pb.KickGroupMemberResponse{
		Base: &pb.BaseResponse{Code: 200, Message: "Success"},
	}, nil
}
