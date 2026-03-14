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
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

type CreateGroupLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewCreateGroupLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CreateGroupLogic {
	return &CreateGroupLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *CreateGroupLogic) CreateGroup(in *pb.CreateGroupRequest) (*pb.CreateGroupResponse, error) {
	// create a new group in group model
	md, ok := metadata.FromIncomingContext(l.ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "missing metadata")
	}
	userIdStrs := md.Get("user_id")
	if len(userIdStrs) == 0 {
		return nil, status.Error(codes.Unauthenticated, "user_id not found in metadata")
	}
	ownerId, err := strconv.ParseInt(userIdStrs[0], 10, 64)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "invalid user_id in metadata")
	}
	var groupId int64

	// Use TransactCtx to ensure atomicity of group creation and adding owner as member
	err = l.svcCtx.SqlConn.TransactCtx(l.ctx, func(ctx context.Context, session sqlx.Session) error {
		res, err := l.svcCtx.GroupModel.Insert(ctx, &model.Group{
			Name:        in.Name,
			Avatar:      in.Avatar,
			Description: in.Description,
			OwnerId:     ownerId,
			MaxMembers:  500,
			MemberCount: 1,
			Status:      1,
		})
		if err != nil {
			return err
		}
		groupId, err = res.LastInsertId()
		if err != nil {
			return err
		}
		_, err = l.svcCtx.GroupMemberModel.Insert(ctx, &model.GroupMember{
			GroupId:  groupId,
			UserId:   ownerId,
			Role:     2,
			JoinedAt: time.Now(),
		})
		return err
	})

	if err != nil {
		l.Errorf("Failed to create group: %v", err)
		return nil, status.Error(codes.Internal, "failed to create group")
	}

	if l.svcCtx.Producer != nil {
		event := map[string]interface{}{
			"type":      "group_event",
			"action":    "create",
			"group_id":  groupId,
			"user_id":   ownerId,
			"timestamp": time.Now().Unix(),
		}
		data, _ := json.Marshal(event)
		key := strconv.FormatInt(groupId, 10)
		_ = l.svcCtx.Producer.Send([]byte(key), data)
	}

	return &pb.CreateGroupResponse{
		Base: &pb.BaseResponse{Code: 200, Message: "Success"},
		Group: &pb.Group{
			Id:          groupId,
			Name:        in.Name,
			Avatar:      in.Avatar,
			Description: in.Description,
			OwnerId:     ownerId,
		},
	}, nil
}
