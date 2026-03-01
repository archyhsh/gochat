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
	"google.golang.org/grpc/status"

	"github.com/zeromicro/go-zero/core/logx"
)

type InviteMembersLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewInviteMembersLogic(ctx context.Context, svcCtx *svc.ServiceContext) *InviteMembersLogic {
	return &InviteMembersLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *InviteMembersLogic) InviteMembers(in *pb.InviteMembersRequest) (*pb.InviteMembersResponse, error) {
	err := l.svcCtx.SqlConn.TransactCtx(l.ctx, func(ctx context.Context, session sqlx.Session) error {
		for _, memberId := range in.MemberIds {
			// Check if already a member
			exists, err := l.svcCtx.GroupMemberModel.FindMemberByGroupIdAndUserId(ctx, in.GroupId, memberId)
			if err != nil && err != model.ErrNotFound {
				return err
			}
			if exists != nil && exists.Id > 0 {
				continue
			}
			_, err = l.svcCtx.GroupMemberModel.Insert(ctx, &model.GroupMember{
				GroupId:  in.GroupId,
				UserId:   memberId,
				Role:     0,
				JoinedAt: time.Now(),
			})
			if err != nil {
				return status.Error(codes.Internal, "Insert group member failed")
			}
			_, err = session.ExecCtx(ctx, "update `group` set member_count = member_count + 1 where id = ?", in.GroupId)
			if err != nil {
				return status.Error(codes.Internal, "update member count failed")
			}
		}
		return nil
	})

	if err != nil {
		l.Errorf("InviteMembers DB transaction failed: %v", err)
		return &pb.InviteMembersResponse{
			Base: &pb.BaseResponse{Code: 500, Message: "Internal server error"},
		}, nil
	}

	if l.svcCtx.Config.Producer != nil {
		event := map[string]interface{}{
			"type":       "group_event",
			"action":     "invite",
			"group_id":   in.GroupId,
			"member_ids": in.MemberIds,
			"timestamp":  time.Now().Unix(),
		}
		data, _ := json.Marshal(event)
		key := strconv.FormatInt(in.GroupId, 10)
		err = l.svcCtx.Config.Producer.Send([]byte(key), data)
		if err != nil {
			l.Errorf("Failed to send Kafka event for InviteMembers: %v", err)
		}
	}

	return &pb.InviteMembersResponse{
		Base: &pb.BaseResponse{Code: 200, Message: "Success"},
	}, nil
}
