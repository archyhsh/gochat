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
	// 1. Fetch Group Details
	group, err := l.svcCtx.GroupModel.FindOne(l.ctx, in.GroupId)
	if err != nil {
		if err == model.ErrNotFound {
			return nil, status.Error(codes.NotFound, "group not found")
		}
		return nil, status.Error(codes.Internal, err.Error())
	}

	// 2. Member Limit Check
	maxMembers := group.MaxMembers
	if maxMembers == 0 {
		maxMembers = 500
	}
	if group.MemberCount+int64(len(in.MemberIds)) > maxMembers {
		return nil, status.Error(codes.ResourceExhausted, "invitation would exceed group member limit")
	}

	// 3. Transactional Update
	err = l.svcCtx.SqlConn.TransactCtx(l.ctx, func(ctx context.Context, session sqlx.Session) error {
		now := time.Now()
		version := now.UnixNano()
		addedCount := 0

		for _, memberId := range in.MemberIds {
			// Check if already a member
			exists, err := l.svcCtx.GroupMemberModel.FindMemberByGroupIdAndUserId(ctx, in.GroupId, memberId)
			if err != nil && err != model.ErrNotFound {
				return err
			}
			if exists != nil && exists.Id > 0 {
				continue
			}

			// Insert into Group Member
			_, err = l.svcCtx.GroupMemberModel.Insert(ctx, &model.GroupMember{
				GroupId:  in.GroupId,
				UserId:   memberId,
				Role:     1, // Regular member
				JoinedAt: now,
			})
			if err != nil {
				return err
			}
			addedCount++
		}

		if addedCount > 0 {
			// Update Group Count and Version
			group.MemberCount += int64(addedCount)
			group.MetaVersion = version
			return l.svcCtx.GroupModel.Update(ctx, group)
		}
		return nil
	})

	if err != nil {
		l.Errorf("InviteMembers failed: %v", err)
		return nil, status.Error(codes.Internal, "internal database error")
	}

	// 4. Async Notification via Kafka
	if l.svcCtx.Producer != nil {
		event := map[string]interface{}{
			"type":       "group_event",
			"action":     "invite",
			"group_id":   in.GroupId,
			"user_id":    0, // System/Inviter (could pass actor_id if available)
			"member_ids": in.MemberIds,
			"version":    group.MetaVersion,
			"timestamp":  time.Now().Unix(),
		}
		data, _ := json.Marshal(event)
		_ = l.svcCtx.Producer.Send(l.ctx, []byte(strconv.FormatInt(in.GroupId, 10)), data)
	}

	return &pb.InviteMembersResponse{
		Base: &pb.BaseResponse{Code: 200, Message: "Success"},
	}, nil
}
