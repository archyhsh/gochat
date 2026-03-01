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
	// TODO: Get ownerID from context or request metadata
	ownerID := int64(1)
	var groupID int64

	// Use TransactCtx to ensure atomicity of group creation and adding owner as member
	err := l.svcCtx.SqlConn.TransactCtx(l.ctx, func(ctx context.Context, session sqlx.Session) error {
		res, err := l.svcCtx.GroupModel.Insert(ctx, &model.Group{
			Name:        in.Name,
			Avatar:      in.Avatar,
			Description: in.Description,
			OwnerId:     ownerID,
			MaxMembers:  500,
			MemberCount: 1,
			Status:      1,
		})
		if err != nil {
			return err
		}
		groupID, err = res.LastInsertId()
		if err != nil {
			return err
		}
		_, err = l.svcCtx.GroupMemberModel.Insert(ctx, &model.GroupMember{
			GroupId:  groupID,
			UserId:   ownerID,
			Role:     2,
			JoinedAt: time.Now(),
		})
		return err
	})

	if err != nil {
		l.Errorf("Failed to create group: %v", err)
		return nil, status.Error(codes.Internal, "failed to create group")
	}

	if l.svcCtx.Config.Producer != nil {
		event := map[string]interface{}{
			"type":      "group_event",
			"action":    "create",
			"group_id":  groupID,
			"owner_id":  ownerID,
			"timestamp": time.Now().Unix(),
		}
		data, _ := json.Marshal(event)
		key := strconv.FormatInt(groupID, 10)
		_ = l.svcCtx.Config.Producer.Send([]byte(key), data)
	}

	return &pb.CreateGroupResponse{
		Base: &pb.BaseResponse{Code: 200, Message: "Success"},
		Group: &pb.Group{
			Id:          groupID,
			Name:        in.Name,
			Avatar:      in.Avatar,
			Description: in.Description,
			OwnerId:     ownerID,
		},
	}, nil
}
