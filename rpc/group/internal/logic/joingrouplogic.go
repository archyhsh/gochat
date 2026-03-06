package logic

import (
	"context"
	"encoding/json"
	"strconv"
	"time"

	"github.com/archyhsh/gochat/rpc/group/internal/svc"
	"github.com/archyhsh/gochat/rpc/group/model"
	"github.com/archyhsh/gochat/rpc/pb"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"github.com/zeromicro/go-zero/core/logx"
)

type JoinGroupLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewJoinGroupLogic(ctx context.Context, svcCtx *svc.ServiceContext) *JoinGroupLogic {
	return &JoinGroupLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *JoinGroupLogic) JoinGroup(in *pb.JoinGroupRequest) (*pb.JoinGroupResponse, error) {
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
	group, err := l.svcCtx.GroupModel.FindOne(l.ctx, in.GroupId)
	if err != nil {
		if err == model.ErrNotFound {
			return &pb.JoinGroupResponse{
				Base: &pb.BaseResponse{Code: 404, Message: "Group not found"},
			}, nil
		}
		return nil, status.Error(codes.Internal, "failed to query group")
	}

	if group.Status == 0 {
		return &pb.JoinGroupResponse{
			Base: &pb.BaseResponse{Code: 400, Message: "Group is dismissed"},
		}, nil
	}

	member, err := l.svcCtx.GroupMemberModel.FindMemberByGroupIdAndUserId(l.ctx, in.GroupId, userId)
	if err != nil && err != model.ErrNotFound {
		return nil, status.Error(codes.Internal, "failed to check member status")
	}
	if member != nil && member.Id > 0 {
		return &pb.JoinGroupResponse{
			Base: &pb.BaseResponse{Code: 400, Message: "Already a member"},
		}, nil
	}

	if group.MemberCount >= group.MaxMembers {
		return &pb.JoinGroupResponse{
			Base: &pb.BaseResponse{Code: 400, Message: "Group is full"},
		}, nil
	}

	err = l.svcCtx.SqlConn.TransactCtx(l.ctx, func(ctx context.Context, session sqlx.Session) error {
		_, err := l.svcCtx.GroupMemberModel.Insert(ctx, &model.GroupMember{
			GroupId:  in.GroupId,
			UserId:   userId,
			Role:     0,
			JoinedAt: time.Now(),
		})
		if err != nil {
			return err
		}
		_, err = session.ExecCtx(ctx, "update `group` set member_count = member_count + 1 where id = ?", in.GroupId)
		return err
	})

	if err != nil {
		l.Errorf("Transactional join group failed: %v", err)
		return nil, status.Error(codes.Internal, "failed to join group")
	}

	if l.svcCtx.Producer != nil {
		event := map[string]interface{}{
			"type":      "group_event",
			"action":    "join",
			"group_id":  in.GroupId,
			"user_id":   userId,
			"intro":     in.Message,
			"timestamp": time.Now().Unix(),
		}
		data, _ := json.Marshal(event)
		key := strconv.FormatInt(in.GroupId, 10)
		_ = l.svcCtx.Producer.Send([]byte(key), data)
	}

	return &pb.JoinGroupResponse{
		Base: &pb.BaseResponse{Code: 200, Message: "Successfully joined group"},
	}, nil
}
