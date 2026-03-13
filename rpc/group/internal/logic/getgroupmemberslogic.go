package logic

import (
	"context"

	"fmt"
	"github.com/archyhsh/gochat/rpc/group/internal/svc"
	"github.com/archyhsh/gochat/rpc/pb"
	"github.com/archyhsh/gochat/rpc/user/userservice"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetGroupMembersLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetGroupMembersLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetGroupMembersLogic {
	return &GetGroupMembersLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *GetGroupMembersLogic) GetGroupMembers(in *pb.GetGroupMembersRequest) (*pb.GetGroupMembersResponse, error) {
	members, err := l.svcCtx.GroupMemberModel.FindMembersByGroupId(l.ctx, in.GroupId)
	if err != nil {
		l.Errorf("GetGroupMembers failed to query DB: groupID=%d, error=%v", in.GroupId, err)
		return nil, status.Error(codes.Internal, "failed to query group members")
	}

	if len(members) == 0 {
		return &pb.GetGroupMembersResponse{
			Base: &pb.BaseResponse{Code: 200, Message: "Success"},
		}, nil
	}

	var uids []int64
	for _, m := range members {
		uids = append(uids, m.UserId)
	}

	userResp, err := l.svcCtx.UserRpc.GetUsersByIds(l.ctx, &userservice.GetUsersByIdsRequest{
		UserIds: uids,
	})

	userMap := make(map[int64]*pb.User)
	if err == nil && userResp != nil {
		for _, u := range userResp.Users {
			userMap[u.Id] = u
		}
	} else {
		l.Errorf("Scheme C: Failed to batch fetch users: %v", err)
	}

	var pbMembers []*pb.GroupMember
	for _, m := range members {
		avatar := ""
		globalNick := ""
		if u, ok := userMap[m.UserId]; ok {
			globalNick = u.Nickname
			avatar = u.Avatar
		}

		displayNick := globalNick
		if m.Nickname != "" {
			if globalNick != "" && m.Nickname != globalNick {
				displayNick = fmt.Sprintf("%s (%s)", m.Nickname, globalNick)
			} else {
				displayNick = m.Nickname
			}
		}

		if displayNick == "" {
			displayNick = fmt.Sprintf("User %d", m.UserId)
		}

		pbMembers = append(pbMembers, &pb.GroupMember{
			UserId:      m.UserId,
			Nickname:    displayNick,
			Avatar:      avatar,
			Role:        int32(m.Role),
			JoinedAt:    m.JoinedAt.Unix(),
			InfoVersion: m.InfoVersion,
		})
	}

	return &pb.GetGroupMembersResponse{
		Base:    &pb.BaseResponse{Code: 200, Message: "Success"},
		Members: pbMembers,
	}, nil
}
